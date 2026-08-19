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
go run ./scripts/bootstrap

# Start REST gateway server on port 8080
go run ./cmd/gateway
```

### 3. Workers Multiplexados (wkr10)

O cluster simulado roda **4 daemons** (3 de bloco + 1 inline), um por porta, conforme
`services.ClusterTopology` no Gateway. Cada daemon multiplexa dezenas de máquinas lógicas.

```bash
cd apps/wkr10
go mod tidy

# Um terminal por daemon (PORT e WORKER_NAME sobrescrevem o .env)
PORT=8081 WORKER_NAME=wkr10_1 go run ./cmd/worker   # block
PORT=8082 WORKER_NAME=wkr10_2 go run ./cmd/worker   # block
PORT=8083 WORKER_NAME=wkr10_3 go run ./cmd/worker   # block
PORT=8084 WORKER_NAME=wkr10_4 go run ./cmd/worker   # inline
```

Ou suba tudo (4 workers + gateway) de uma vez, com logs em `/tmp/r10_logs`:

```bash
./scripts/r10 up      # subir
./scripts/r10 down    # derrubar
```

### 3.1. Provisionar o cluster

Antes do primeiro upload é preciso criar os workers, as máquinas lógicas e os discos:

```bash
# Via Control Plane (assíncrono, retorna job_id)
curl -X POST http://localhost:8080/api/v1/management/bootstrap

# Ou via CLI (síncrono)
cd apps/gateway && go run ./scripts/setup_cluster
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


---

## API

| Método | Rota | Descrição |
| --- | --- | --- |
| `POST` | `/api/v1/uploads/init` | Abre um upload (`filename`, `total_size`, `content_type`) e devolve `upload_id` |
| `PUT` | `/api/v1/uploads/:upload_id/parts/:n` | Envia a n-ésima fatia de 8MB como binário puro |
| `POST` | `/api/v1/uploads/:upload_id/complete` | Monta, aplica Erasure Coding, distribui e cataloga o blob |
| `GET` | `/api/v1/files` | Lista o catálogo |
| `GET` | `/api/v1/files/:blob_id` | Metadados do blob + placement dos chunks |
| `GET` | `/api/v1/files/:blob_id/download` | Remonta o arquivo a partir dos chunks e devolve os bytes originais |
| `DELETE` | `/api/v1/files/:blob_id` | Deleção lógica |
| `GET` | `/api/v1/management/cluster` \| `/workers` | Estatísticas do cluster |
| `POST` | `/api/v1/management/bootstrap` \| `/reset` | Provisiona / destrói a topologia (assíncrono, `job_id`) |
| `GET` | `/api/v1/management/jobs/:job_id` | Status de um job |

### Roteamento de armazenamento

O Gateway decide o destino pelo tamanho do arquivo, conforme `docs/erasure-coding.md`:

| Tamanho | Caso | Destino |
| --- | --- | --- |
| `< 128KB` | Caso 1 | Máquina **inline**: append no volume `volume_01.dat`, offset físico gravado no catálogo |
| `128KB – 32MB` | Caso 2 | Um chunk único numa máquina **block**, sem Erasure Coding |
| `> 32MB` | Caso 3 | Blocos cheios de 32MB viram 12 shards (Reed-Solomon 8+4) em 12 máquinas distintas; o resto vira Caso 2 |

---

## Testes

```bash
# Testes unitários (bracketing / Reed-Solomon)
cd apps/gateway && go test ./...

# Verificação end-to-end -- ./scripts/r10 é o entrypoint único
./scripts/r10 preflight   # confere que o ambiente está são E atualizado (rode sempre antes)
./scripts/r10 test        # matriz de casos: placement + round trip byte a byte
./scripts/r10 test --full # inclui os casos de fronteira mais lentos (32MB exato, multi-stripe)
./scripts/r10 fault       # injeção de falhas contra o caminho de Erasure Coding
./scripts/r10 all         # preflight -> test -> fault

./scripts/r10 explain <blob-id>   # como um blob está fisicamente armazenado
```

A matriz de casos é **dado**, não código: veja `scripts/e2e_cases.json`. Os resultados
saem como JSON lines em `/tmp/r10_logs/{matrix,fault}.jsonl`.

O e2e **não** roda a cada commit -- é pesado demais. Ele roda localmente sob demanda e,
futuramente, no estágio 2 de um pipeline de CI de 3 estágios, como job único, logo antes
do merge para a main.

Falhas históricas de e2e ficam catalogadas em [`docs/failure-catalog.md`](docs/failure-catalog.md);
leia antes de investigar do zero, e registre lá o que for novo.
