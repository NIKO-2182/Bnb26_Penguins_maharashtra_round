#!/usr/bin/env bash
# fairdrop-demo.sh — end-to-end proof run
# Usage: bash fairdrop-demo.sh [fcfs|fair|both]
set -u

MODE_ARG="${1:-both}"

hr() { printf '\n\033[1m%s\033[0m\n' "$1"; }
wait_ready() {
  for i in $(seq 1 30); do
    if curl -sf -m 3 localhost:8080/health >/dev/null 2>&1; then return 0; fi
    sleep 1
  done
  echo "api not reachable on :8080 -- is the stack up?" >&2
  return 1
}

reset_run() {
  MODE=$1
  # Clear sticky admission tickets too, or queue positions keep climbing
  # across demo runs instead of restarting at 1.
  export XDG_RUNTIME_DIR=/run/user/1000
  local cid
  cid=$(docker ps --format '{{.ID}} {{.Image}}' | grep redis | awk '{print $1}' | head -1)
  [ -n "$cid" ] && docker exec "$cid" redis-cli --scan --pattern 'ticket:user:*' \
    | xargs -r docker exec -i "$cid" redis-cli DEL >/dev/null 2>&1

  # Only reset here. The mode is set by /run itself now -- setting it separately
  # is what allowed the sim's label and the API's actual mode to drift apart.
  curl -sf -m 10 -XPOST localhost:8080/admin/reset >/dev/null
}

run_sim() {
  SEED=${2:-42}
  BOTS=${3:-2000}
  HUMANS=${4:-500}
  SEATS=${5:-500}
  curl -sf -m 15 -XPOST localhost:8090/run -H 'Content-Type: application/json' \
    -d "{\"scenario\":\"flash_sale\",\"mode\":\"$MODE\",\"seed\":$SEED,\"bots\":$BOTS,\"humans\":$HUMANS,\"seats\":$SEATS}" \
    >/dev/null
}

report() {
  curl -sf -m 30 localhost:8080/results -o "/tmp/fd_$MODE.json"
  MODE="$MODE" python3 - <<'PY'
import json, os
m = os.environ['MODE']
d = json.load(open('/tmp/fd_%s.json' % m))
r = d['runs'][0]
print(f"  seats -> humans {r['seats_humans']:>4}   bots {r['seats_bots']:>4}")
print(f"  BOT SHARE            {r['bot_share']*100:>6.1f}%")
print(f"  human win rate       {r['human_win_rate']*100:>6.1f}%   (of humans actually judged)")
print(f"  BOT ADVANTAGE RATIO  {r['bot_advantage_ratio']:>6.3f}     <- target 1.000")
print(f"  precision / recall   {r['precision']:.3f} / {r['recall']:.3f}")
print(f"  humans denied        {r['humans_denied']:>4}    (lost to sold_out: {r['humans_lost_to_sold_out']})")
print(f"  INVARIANTS  oversell {r['oversell_count']}  duplicates {r['duplicate_count']}")
print(f"  LEDGER CHAIN ok={d['chain_ok']}  {d['chain_checked']} events verified  head={d.get('chain_head','')[:16]}")
PY
}

wait_ready || exit 1

if [ "$MODE_ARG" != "both" ]; then
  hr "RUN: mode=$MODE_ARG"
  reset_run "$MODE_ARG"; run_sim "$MODE_ARG"; sleep 115; report
  exit 0
fi

hr "1/2  CONTROL ARM  --  mode=fcfs (defences OFF)"
reset_run fcfs; run_sim fcfs; sleep 115; report

hr "2/2  TREATMENT ARM  --  mode=fair (defences ON)"
reset_run fair; run_sim fair; sleep 115; report

hr "Ablation complete"
echo "  Bot seats should collapse to 0 in fair mode."
echo "  Ratio 0.000 means OVER-correction, not parity -- see README notes."