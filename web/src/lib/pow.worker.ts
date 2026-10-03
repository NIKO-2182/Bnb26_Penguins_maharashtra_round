/**
 * File: web/src/lib/pow.worker.ts
 * Purpose: Proof-of-work solver run off the main thread. Finds a decimal
 *          nonce such that SHA-256(challenge + nonce) has at least
 *          `difficulty` leading zero bits. Uses a synchronous SHA-256 so the
 *          hot loop avoids per-hash promise overhead from crypto.subtle.
 * Protocol:
 *   in:  { challenge: string, difficulty: number }
 *   out: { type: "progress", attempts, hashRate }
 *        { type: "done", nonce, hash, attempts, ms }
 */

const K = new Uint32Array([
  0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
  0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
  0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
  0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
  0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
  0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
  0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
  0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
])

const W = new Uint32Array(64)
const H = new Uint32Array(8)

function sha256(bytes: Uint8Array): Uint32Array {
  const len = bytes.length
  const padded = new Uint8Array(((len + 9 + 63) >> 6) << 6)
  padded.set(bytes)
  padded[len] = 0x80
  const view = new DataView(padded.buffer)
  view.setUint32(padded.length - 4, len * 8)
  view.setUint32(padded.length - 8, Math.floor((len * 8) / 0x100000000))

  H[0] = 0x6a09e667; H[1] = 0xbb67ae85; H[2] = 0x3c6ef372; H[3] = 0xa54ff53a
  H[4] = 0x510e527f; H[5] = 0x9b05688c; H[6] = 0x1f83d9ab; H[7] = 0x5be0cd19

  for (let off = 0; off < padded.length; off += 64) {
    for (let i = 0; i < 16; i++) W[i] = view.getUint32(off + i * 4)
    for (let i = 16; i < 64; i++) {
      const w15 = W[i - 15], w2 = W[i - 2]
      const s0 = ((w15 >>> 7) | (w15 << 25)) ^ ((w15 >>> 18) | (w15 << 14)) ^ (w15 >>> 3)
      const s1 = ((w2 >>> 17) | (w2 << 15)) ^ ((w2 >>> 19) | (w2 << 13)) ^ (w2 >>> 10)
      W[i] = (W[i - 16] + s0 + W[i - 7] + s1) | 0
    }
    let a = H[0], b = H[1], c = H[2], d = H[3], e = H[4], f = H[5], g = H[6], h = H[7]
    for (let i = 0; i < 64; i++) {
      const S1 = ((e >>> 6) | (e << 26)) ^ ((e >>> 11) | (e << 21)) ^ ((e >>> 25) | (e << 7))
      const ch = (e & f) ^ (~e & g)
      const t1 = (h + S1 + ch + K[i] + W[i]) | 0
      const S0 = ((a >>> 2) | (a << 30)) ^ ((a >>> 13) | (a << 19)) ^ ((a >>> 22) | (a << 10))
      const maj = (a & b) ^ (a & c) ^ (b & c)
      const t2 = (S0 + maj) | 0
      h = g; g = f; f = e; e = (d + t1) | 0
      d = c; c = b; b = a; a = (t1 + t2) | 0
    }
    H[0] += a; H[1] += b; H[2] += c; H[3] += d; H[4] += e; H[5] += f; H[6] += g; H[7] += h
  }
  return H
}

function leadingZeroBits(h: Uint32Array): number {
  let bits = 0
  for (let i = 0; i < 8; i++) {
    if (h[i] === 0) {
      bits += 32
      continue
    }
    return bits + Math.clz32(h[i])
  }
  return bits
}

function toHex(h: Uint32Array): string {
  let s = ""
  for (let i = 0; i < 8; i++) s += (h[i] >>> 0).toString(16).padStart(8, "0")
  return s
}

const encoder = new TextEncoder()

self.onmessage = (ev: MessageEvent<{ challenge: string; difficulty: number }>) => {
  const { challenge, difficulty } = ev.data
  const start = performance.now()
  let lastReport = start
  let attempts = 0
  for (let nonce = 0; ; nonce++) {
    const h = sha256(encoder.encode(challenge + nonce))
    attempts++
    if (leadingZeroBits(h) >= difficulty) {
      self.postMessage({ type: "done", nonce: String(nonce), hash: toHex(h), attempts, ms: performance.now() - start })
      return
    }
    if ((attempts & 1023) === 0) {
      const now = performance.now()
      if (now - lastReport > 80) {
        lastReport = now
        self.postMessage({ type: "progress", attempts, hashRate: attempts / ((now - start) / 1000) })
      }
    }
  }
}
