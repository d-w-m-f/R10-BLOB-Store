# R10 Blob Store

<div align="left">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/Angular%2018-DD0031?style=for-the-badge&logo=angular&logoColor=white" alt="Angular" />
  <img src="https://img.shields.io/badge/PostgreSQL-316192?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker" />
  <img src="https://img.shields.io/badge/TypeScript-007ACC?style=for-the-badge&logo=typescript&logoColor=white" alt="TypeScript" />
  <img src="https://img.shields.io/badge/SCSS-CC6699?style=for-the-badge&logo=sass&logoColor=white" alt="SCSS" />
</div>

Sistema de armazenamento de objetos distribuído de alta disponibilidade e performance. Implementa tolerância a falhas via Erasure Coding (Reed-Solomon 8:4) no Gateway com verificação de integridade matemática em tempo real (Bracketing). Utiliza orquestração assíncrona de cluster via Control Plane dedicado e arquitetura de workers multiplexados baseada em storage engines Block e Inline (LSM-Tree Append-Only), maximizando o throughput de I/O e a resiliência dos dados.

---

## Dependências

Certifique-se de possuir os seguintes ambientes e binários instalados no seu sistema operacional:

- **[Go](https://golang.org/dl/)**: v1.21+ (Compilação e execução do Gateway e Workers)
- **[Node.js & PNPM](https://pnpm.io/)**: v20+ / Corepack habilitado (Gestão da aplicação Web)
- **[Docker & Docker Compose](https://www.docker.com/)**: (Orquestração do banco de dados relacional)

---

## Setup & Execução

### 1. Banco de Dados (Docker Compose)

```bash
# Start PostgreSQL container in background
docker compose -f docker/docker-compose.yml up -d

# Verify container health and exposure on port 5432
docker compose -f docker/docker-compose.yml ps
```

### 2. Gateway API & Orquestrador

```bash
# Enter gateway application workspace
cd apps/gateway

# Download and resolve Go module dependencies
go mod tidy

# Execute database migrations and register schema models
go run scripts/bootstrap.go

# Start REST gateway server on port 8080
go run cmd/gateway/main.go
```

### 3. Worker Multiplexado (wkr10)

```bash
# Open a new terminal session and enter worker workspace
cd apps/wkr10

# Copy environment template if not present (.env is auto-loaded by godotenv)
cp -n .env.example .env

# Download and resolve Go module dependencies
go mod tidy

# Start multiplexed storage daemon
go run cmd/worker/main.go
```

### 4. Frontend SPA (Control Plane)

```bash
# Open a new terminal session and enter web application workspace
cd apps/web

# Install frontend UI dependencies via PNPM
pnpm install

# Launch Angular dev server and interface at http://localhost:4200
pnpm run start
```
