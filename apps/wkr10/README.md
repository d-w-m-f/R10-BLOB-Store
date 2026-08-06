# R10 Blob Store - Worker Daemon (wkr10)

O **`wkr10` (`apps/wkr10`)** é o músculo de I/O e o servidor de armazenamento físico do R10 Blob Store. Construído nativamente em **Go**, este microsserviço opera como um Daemon multiplexado com foco em transferência em altíssima velocidade, zero alocação desnecessária de RAM e controle inteligente do Sistema de Arquivos do Linux.

---

## 🧭 Sumário das Funcionalidades Implementadas

### 1. Arquitetura de Multiplexação de Daemons (1:N)
- **Densidade de Recursos:** Em vez de forçar o sistema operacional a gerenciar dezenas de instâncias ou portas pesadas de rede abertos no host para simular nós distribuídos, um único processo Go do `wkr10` atua como **Daemon Multiplexador**, capaz de gerenciar simultaneamente dezenas de **Máquinas Lógicas** independentes (ex: 4 workers servindo 38 máquinas).
- Cada máquina gerenciada reside isolada no diretório apontado pela variável de ambiente (`CLUSTER_ROOT_DIR`), sob um namespace criptográfico randômico de 8 caracteres alfanuméricos (ex: `/tmp/r10_cluster/machine_aB12XyZ9`).

### 2. Storage Engines I/O (Block + Offset - `internal/io/`)
O worker não salva dados "da mesma forma" para todos os tipos de payload. Ele emprega a interface estrita `StorageEngine` para alternar entre dois motores mecânicos distintos de manipulação em disco:

- 📦 **Block Engine (`block_engine.go`):**
  - **Uso de Negócio:** Projetada para arquivos extensos ou para os shards brutos de 32MB de Erasure Coding.
  - **Funcionamento:** Grava cada payload diretamente como um arquivo binário autônomo dentro do subdiretório `/chunks/` da respectiva máquina (ex: `machine_X/chunks/a1b2c3...dat`).
  - **Retorno:** O `PhysicalOffset` sempre volta como `0`, pois se trata de arquivo dedicado e independente onde o sistema operacional resolve bem a fragmentação.
- ⚡ **Inline Engine - LSM-Tree Style (`inline_engine.go`):**
  - **Uso de Negócio:** Projetada para receber micro-objetos e arquivos de menor escala (<128KB).
  - **O Problema de Engenharia Resolvido:** Se gravássemos milhões de fotos de 12KB em arquivos individuais no SO, iríamos **esgotar todos os Inodes do disco** em poucas horas e arruinar a performance por gravação aleatória.
  - **Funcionamento (Append-Only):** Esta engine **não cria arquivos avulsos**. Ela mantém aberto em modo de anexo contínuo (`os.O_APPEND`) um arquivo mestre gigante chamado `volume_01.dat`. Ao receber novos bytes, concatena-os ao final da estrutura (transformando a gravação em puro I/O Sequencial de velocidade insana).
  - **Retorno:** Antes do append, o sistema executa um `.Stat()` para aferir o tamanho do arquivo no milissegundo da escrita. Esse valor exato é o **`PhysicalOffset`** devolvido ao Gateway para viabilizar leituras precisas no futuro!

### 3. Concorrência Atômica e Milimétrica em RAM
- Para impedir que uploads que chegam em requisições paralelas misturem bytes e corrompam o volume contínuo de append-only, a Inline Engine injeta uma estrutura de bloqueio granular com **`sync.Mutex` dinamicamente mapeada por namespace via `sync.Map`**.
- Duas chamadas simultâneas direcionadas para o mesmo volume de uma máquina específica esperam ordenadamente na fila daquela trava de memória local sem gargalar as threads de outras máquinas rodando no mesmo worker.

### 4. Zero-Copy & Low-RAM Data Streaming (`internal/handlers/chunk_handler.go`)
- **O Fim da Despesa com Base64/JSON:** O Worker recusa o tráfego de binários serializados como strings Base64 no body de payloads JSON (o que consumiria na memória RAM até 3 vezes o peso original do arquivo para parseamento e decodificação).
- **Socket-to-Disk Direct Pipe:** A transferência se processa recebendo o body bruto diretamente do protocolo TCP (`c.Request.Body`), descendo ao disco através da primitiva **`io.Copy()`** do Go sem armazenamento em buffers intermediários da aplicação.
- **Metadados via HTTP Headers:** Os parâmetros identificadores viajam fora do body através dos headers HTTP customizados:
  - `X-Chunk-ID`: Carrega o UUID exclusivo do bloco a ser inserido.
  - `X-Chunk-Size`: Informa a dimensão nominal em bytes para validação de fluxo.

---

## 📡 Rotas REST Implementadas (`cmd/worker/main.go`)

Todas as requisições autenticadas e pré-processadas são roteadas do Gateway em direção a estes dois canais I/O especializados do Worker:

- `GET /ping`: Monitoramento de vivência (Heartbeat) retornando o nome do daemon e o status da porta.
- `POST /api/v1/machines/:machine_namespace/chunks`: Canal para gravação de arquivos sob a mecânica da **Block Engine**.
- `POST /api/v1/machines/:machine_namespace/append`: Canal para gravação de arquivos sequenciais sob a mecânica da **Inline Engine (Append-Only)**.

---

## 🚀 Como Executar o Worker Localmente

O binário lê automaticamente as configurações de ambiente através da biblioteca `godotenv`. As 3 variáveis fundamentais estão mapeadas no arquivo `.env` (e no template `.env.example`):
1. `PORT`: Porta de binding HTTP (ex: `8081`).
2. `CLUSTER_ROOT_DIR`: Pasta raiz onde repousam as pastas que representam as máquinas do cluster (ex: `/tmp/r10_cluster`).
3. `WORKER_NAME`: Identificador nominal da instância do daemon no banco (ex: `wkr10_1`).

```bash
# Navegar até o workspace do worker
cd apps/wkr10

# Criar arquivo .env a partir do exemplo (se ainda não existir)
cp -n .env.example .env

# Garantir resolução de pacotes e módulos Go
go mod tidy

# Inicializar o daemon multiplexado apontando para o cluster local (.env carregado automaticamente)
go run cmd/worker/main.go
```
