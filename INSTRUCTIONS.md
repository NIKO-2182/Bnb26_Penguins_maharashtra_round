// FILE: d:/hacks/BitNBuilds/INSTRUCTIONS.md
// PURPOSE: Environment setup, execution instructions, container lifecycle, and Makefile reference
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: docker-compose.yml, Makefile, web/package.json
// USED BY: Developers, Operators, Evaluation agents
// RULES: Detailed operational guide for stack management
// DO NOT: Remove Makefile or container command references

# Environment Setup & Execution Instructions: Fair Drop System

This document provides step-by-step instructions for running the Fair Drop backend, frontend dashboard, each containerized microservice, and quick commands via `Makefile`.

---

## 1. What is the Makefile & Why Use It?

The **`Makefile`** is a task automation file located in the root of the project. It provides shortcut commands (aliases) for common development and container operations. 

Instead of remembering and typing long CLI commands like `docker compose up -d --build` or `pnpm --dir web dev`, you can run simple `make` shortcuts (if `make` is installed on your OS).

> **Note for Windows Users**: On Windows PowerShell, if `make` is not installed, run the **Executed Command** column directly in your terminal (e.g. `docker compose up -d --build`).

| Makefile Target | Executed Command (Windows / Direct) | Description |
| :--- | :--- | :--- |
| `make up` | `docker compose up -d --build` | Builds & starts all stack containers in detached mode |
| `make down` | `docker compose down` | Stops and removes all running containers and networks |
| `make dev-web` | `pnpm --dir web dev` | Launches the local frontend dev server (Vite + React) |
| `make test` | `go test ./api/... ./sim/...` | Executes all backend Go unit & integration tests |
| `make reset` | `docker compose down && docker compose up -d --build` | Rebuilds and restarts the entire container stack fresh |

---

## 2. Container Services Overview

The application consists of five dockerized services managed via `docker-compose.yml`:

| Service | Container Name | Port | Description |
| :--- | :--- | :--- | :--- |
| **Redis** | `bitnbuilds-redis-1` | `6379` | Primary in-memory store for CAS claims, sessions, & trust scores |
| **PostgreSQL** | `bitnbuilds-db-1` | `5432` | Relational store for user profiles & persistence |
| **API Backend** | `bitnbuilds-api-1` | `8080` | Core Go API service handling claims, trust verification, & endpoints |
| **Simulator** | `bitnbuilds-sim-1` | `8090` | Load generator simulating human vs. bot traffic scenarios |
| **Web Dashboard**| `bitnbuilds-web-1` | `3000` | Vite + React + Tailwind CSS dashboard UI (Nginx reverse proxy) |

---

## 3. Frontend Up & Down Tasks

### Option A: Running Frontend via Docker Container

#### 1. Bring Frontend Up
To start only the containerized web frontend (or rebuild its static image):
```bash
docker compose up -d --build web
```
*(Or bring up the entire stack with `make up`)*

#### 2. Bring Frontend Down
To stop only the frontend container while keeping backend/database running:
```bash
docker compose stop web
```
To remove the frontend container completely:
```bash
docker compose rm -f web
```
*(Or bring down all services with `make down`)*

#### 3. View Frontend Container Logs
```bash
docker compose logs -f web
```

---

### Option B: Local Frontend Development (Without Docker)

#### 1. Start Local Frontend Dev Server
Run the local Vite + React dev server:
```bash
make dev-web
```
*(Alternative manual commands: `pnpm --dir web dev` or `cd web && pnpm dev`)*

The local frontend dev server will start at `http://localhost:3000` with HMR (Hot Module Replacement) enabled.

#### 2. Stop Local Frontend Dev Server
To stop the dev server:
- Focus the terminal window running `pnpm dev`.
- Press **`Ctrl + C`** to send the interrupt signal and stop the server.

---

## 4. Quickstart: Full Stack Management

### Start All Services (Docker)
```bash
make up
```
*(Or: `docker compose up -d --build`)*

**Access points:**
- **Web Dashboard**: `http://localhost:3000`
- **Backend API**: `http://localhost:8080`
- **Simulator Service**: `http://localhost:8090`

### Check Container Health & Status
```bash
docker compose ps
```

### View Combined System Logs
```bash
docker compose logs -f
```

### Stop All Services
```bash
make down
```
*(Or: `docker compose down`)*

---

## 5. Individual Microservice Commands

### A. Infrastructure Only (Redis + Postgres DB)
```bash
docker compose up -d redis db
```

### B. Backend API Container
* **Up**: `docker compose up -d --build api`
* **Down**: `docker compose stop api`
* **Logs**: `docker compose logs -f api`

### C. Simulator Container
* **Up**: `docker compose up -d --build sim`
* **Down**: `docker compose stop sim`
* **Logs**: `docker compose logs -f sim`

---

## 6. Local Backend Development (Go CLI)

To run the Go API backend locally on your host machine:

1. **Start Redis & Postgres containers**:
   ```bash
   docker compose up -d redis db
   ```

2. **Set Environment Variables**:
   * **PowerShell (Windows)**:
     ```powershell
     $env:REDIS_URL="localhost:6379"
     $env:DB_URL="postgres://user:password@localhost:5432/fairdrop"
     $env:PORT="8080"
     $env:HMAC_SECRET="dev-secret-key"
     ```
   * **Bash (Linux/Mac)**:
     ```bash
     export REDIS_URL="localhost:6379"
     export DB_URL="postgres://user:password@localhost:5432/fairdrop"
     export PORT="8080"
     export HMAC_SECRET="dev-secret-key"
     ```

3. **Run API & Tests**:
   ```bash
   make test              # Run test suite
   go run ./api/main.go   # Start API server
   ```

---

## 7. Verifying API Endpoints

Once the API backend is active on `http://localhost:8080`:

* **Health Check**: `curl http://localhost:8080/health`
* **Join Session**: `curl -X POST http://localhost:8080/join -H "Content-Type: application/json" -d '{"username":"user1"}'`
* **Live Metrics**: `curl http://localhost:8080/metrics`
* **Ledger Audit**: `curl "http://localhost:8080/ledger?limit=10"`
* **Reset State**: `curl -X POST http://localhost:8080/admin/reset`

---

## 8. Abuse & Trust Signal Engine Architecture

The backend includes a multi-layered behavior evaluation engine in `api/internal/abuse/`:

| Module | Location | Description |
| :--- | :--- | :--- |
| **Signal Extractor** | `api/internal/abuse/signals.go` | Extracts timing variance across requests, sustained burst counts ($>3$), IP/subnet session density, and exact PoW solve durations. |
| **Fairness Controller** | `api/internal/abuse/budget.go` | Label-free proxy controller tracking borderline trust score ratios ($0.2\text{--}0.4$) to adjust score thresholds dynamically while maintaining 100% ground-truth isolation. |
| **Trust Manager** | `api/internal/abuse/trust.go` | Combines PoW solve speed scaling ($<100\text{ms}$ soft penalty vs $1\text{s}\text{--}15\text{s}$ human bonus), timing variance, burst guards, and shared IP ceilings ($>150$ per IP). |
| **Unit Tests** | `api/internal/abuse/trust_test.go` | Table-driven unit test suite asserting score ranges across human (`human_normal`, `human_slow`, `human_frustrated`, `human_shared_ip`) and bot profiles (`bot_naive`, `bot_solver`). |

### Running Unit Tests
```bash
go test ./api/internal/abuse/...
```

