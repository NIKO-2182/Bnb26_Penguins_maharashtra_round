up:
	docker compose up -d

down:
	docker compose down

test:
	go test ./api/.../...

loadtest:
	# TODO: implementation

matrix:
	./scripts/run_matrix.sh

reset:
	docker compose down
	docker compose up -d
