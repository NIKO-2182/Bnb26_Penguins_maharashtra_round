# FILE: d:/hacks/BitNBuilds/Makefile
# PURPOSE: Task automation aliases for Docker and local development
# INPUTS / OUTPUTS: N/A
# DEPENDS ON: docker-compose.yml, pnpm, go
# USED BY: Developers, Operators
# RULES: Keep targets simple and clean
# DO NOT: N/A

up:
	docker compose up -d --build

down:
	docker compose down

test:
	go test ./api/... ./sim/...

dev-web:
	pnpm --dir web dev

reset:
	docker compose down
	docker compose up -d --build
