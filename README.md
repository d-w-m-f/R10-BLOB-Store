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

O que precisa estar no ar:

| Componente | Porta | Como sobe |
| --- | --- | --- |
| PostgreSQL | 5432 | `docker compose` (ciclo de vida próprio) |
| Gateway | 8080 | `./scripts/r10 up` |
| 4 daemons wkr10 | 8081-8084 | `./scripts/r10 up` |
| Frontend Angular | 4200 | `pnpm start` (opcional; o e2e não usa) |

São **4 daemons**, não um: a topologia (`services.ClusterTopology`) define 3 workers de
bloco e 1 inline, e o placement precisa de 12 máquinas distintas para um stripe 8+4.
Com um worker só, todo upload com Erasure Coding falha.

### Subida normal (mantém o que já está armazenado)

```bash
docker compose -f docker/docker-compose.yml up -d
./scripts/r10 up
./scripts/r10 preflight
```

### Subida limpa (APAGA todos os blobs armazenados)

```bash
docker compose -f docker/docker-compose.yml up -d
./scripts/r10 up
./scripts/r10 migrate     # só é necessário após mudança de schema; inofensivo no resto
./scripts/r10 reset       # destrói os blobs e re-provisiona 4 workers / 38 máquinas
./scripts/r10 preflight
```

**A ordem importa num ponto:** o Postgres precisa estar no ar antes do `r10 up`, porque o
gateway encerra na hora se não conseguir conectar. Se o gateway aparecer faltando no
preflight, quase sempre é isso -- confira `/tmp/r10_logs/gateway.log`.

Na dúvida, rode só `./scripts/r10 preflight`: ele diz exatamente o que está errado e traz
o comando de correção na própria mensagem de falha.

`r10 up` compila os dois binários, derruba o que estiver rodando (é seguro repetir), sobe
os 5 processos em background e joga os logs em `/tmp/r10_logs/`. O terminal fica livre.
`./scripts/r10 down` derruba tudo.

### Rodando um serviço isolado (debug)

Útil para acompanhar o log de um processo em foreground:

```bash
cd apps/gateway && go run ./cmd/gateway                    # gateway
cd apps/wkr10 && PORT=8082 WORKER_NAME=wkr10_2 go run ./cmd/worker   # um daemon
```

`PORT` e `WORKER_NAME` sobrescrevem o `.env` (godotenv nunca sobrescreve variável já
definida). As portas precisam bater com `services.ClusterTopology`.

### Frontend SPA (Control Plane)

```bash
cd apps/web
pnpm install
pnpm run start     # http://localhost:4200
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
