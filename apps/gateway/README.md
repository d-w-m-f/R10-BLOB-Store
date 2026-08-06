# R10 Blob Store - Gateway Service

O **Gateway (`apps/gateway`)** é o cérebro e orquestrador do ecossistema R10 Blob Store. Construído em **Go**, operando com o framework web **Gin** e o ORM **GORM**, o Gateway centraliza a inteligência de rede, a autenticação de clientes, o faturamento de redundância matemática, o catálogo de dados no PostgreSQL e o controle de infraestrutura do cluster.

---

## 🧭 Sumário das Funcionalidades Implementadas

### 1. Resiliência Matemática e Erasure Coding (`services/erasure_service.go`)
- **Reed-Solomon 8:4:** Em vez de triplicar arquivos cegamente no disco (RAID 1), o Gateway segmenta o payload do usuário em **8 shards de dados + 4 shards de paridade**, oferecendo tolerância a perda de até 4 nós simultâneos com apenas 50% de custo adicional de armazenamento.
- **Bracketing (A Validação dos 5 Checkpoints):** Antes de autorizar o despacho do arquivo para a rede, o serviço executa uma verificação algorítmica síncrona. Pelo *Princípio de Superposição Linear* no Corpo de Galois $GF(2^8)$, ele testa estritamente **5 combinações determinísticas de perda ($m+1=5$)** para comprovar com 100% de precisão que a matriz é invertível e imune a dados corrompidos.

### 2. Control Plane & Orquestração Assíncrona (`controllers/management_controller.go`)
- **Background Jobs Não-Bloqueantes:** Tarefas administrativas longas (como formar centenas de pastas ou formatar o cluster) não congelam a conexão HTTP do usuário nem dão timeout.
- O Gateway registra um modelo `Job` no PostgreSQL e dispara uma **Goroutine assíncrona** que trabalha em background nos bastidores (`services/cluster_orchestration.go`), respondendo de imediato com um código HTTP **`202 Accepted` + `JobID`** para que o Frontend execute um *polling* leve e reativo.

### 3. Gerenciamento de Metadados e Catálogo SQL (`internal/models/`)
- Mapeia as **9 tabelas fundamentais** no PostgreSQL com Chaves Primárias baseadas em **UUIDs criptográficos** (`gen_random_uuid()`):
  - *Infraestrutura:* `workers`, `machines`, `discs`.
  - *Negócio & Armazenamento:* `users`, `blobs`, `blob_chunks`, `memblock32` (buffer de agrupamento 32MB para arquivos intermediários).
  - *Orquestração:* `jobs` e `backups`.
- **Rastreabilidade Físico vs Lógico:** O model `BlobChunk` distingue milimetricamente o `LogicalOffset` (posição do byte no arquivo real do cliente) do dueto `PhysicalPath` e `PhysicalOffset` (em qual arquivo e em qual byte do disco do Worker o fragmento foi apendado).

---

## 📡 Rotas REST e APIs Disponibilizadas

### 🛠️ API do Control Plane (`/api/v1/management/`)
- `GET /stats`: Agrega o consumo ao vivo de disco, somando a capacidade total vs. utilizada no cluster (`CapacityMB` vs `UsedMB`).
- `GET /workers`: Exibe a visibilidade e o estado operacional dos Daemons e das Máquinas sob supervisão deles.
- `POST /bootstrap`: Dispara o Job assíncrono para criar tabelas, formatar discos e popular o laboratório local de teste.
- `POST /reset`: Dispara o Job assíncrono para truncar com segurança relacional o cluster e deletar recursivamente o filesystem.
- `GET /jobs/:id`: Endpoint de polling reativo consumido pelo Angular para atualizar o loading visual das rotinas de infraestrutura.

### 📦 API de Ingestão de Arquivos (`/api/v1/uploads/`)
- Rotas para gerenciar fluxos de upload por partes (Chunked & Resumable Uploads), conectando a recepção de blocos HTTP diretamente à engine de segmentação Reed-Solomon e despacho aos Workers.

---

## ⚙️ Scripts CLI & Automação (`scripts/`)
O Gateway também embarca ferramentas de terminal para operação autoconhecedora sem depender da API web:
- `go run scripts/bootstrap.go`: Roda as migrações GORM para provisionar e atualizar as tabelas do PostgreSQL.
- `go run scripts/setup_cluster.go`: Aplica o padrão *Config Object* para construir no disco do host uma topologia multiplexada de alta densidade computacional (**4 Workers operando 38 Máquinas Lógicas** com namespaces alfanuméricos de 8 caracteres em `/tmp/r10_cluster`).
- `go run scripts/reset_cluster.go`: Limpa o cluster local executando as deleções SQL na "Ordem de Cascata" correta (`Discs ➔ Machines ➔ Workers`) para evitar erros de violação de Foreign Key antes de remover o diretório mestre no SO.

---

## 🚀 Como Executar o Gateway Localmente

```bash
# Navegar até o diretório do serviço Gateway
cd apps/gateway

# Garantir dependências Go atualizadas e conectadas
go mod tidy

# Rodar o bootstrap de tabelas (exige PostgreSQL rodando no Docker na porta 5432)
go run scripts/bootstrap.go

# Inicializar o servidor REST na porta 8080
go run cmd/gateway/main.go
```
