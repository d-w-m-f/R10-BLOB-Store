# R10 Gateway - Especificação do Banco de Dados Relacional (PostgreSQL via Schemas DDD)

Este documento detalha o esquema relacional de tabelas do **R10 Blob Store Gateway**. A estrutura é mapeada via **GORM** a partir do diretório `internal/models/` e provisionada automaticamente no PostgreSQL ao executar o script de inicialização do cluster (`go run scripts/bootstrap.go`).

Para manter o máximo padrão de engenharia e escalabilidade horizontal (seguindo os conceitos de *Domain-Driven Design* em Monólitos Modulares e isolamento de Bounded Contexts), **as tabelas não habitam o schema padrão `public`**. O banco é estritamente particionado em **3 Schemas Dedicados**: `infra`, `storage` e `control_plane`.

O banco opera sob chaves primárias do tipo **UUID** (`gen_random_uuid()`), garantindo exclusividade criptográfica na rede distribuída e facilitando referências seguras entre os microsserviços e esquemas.

---

## 1. Schema: `infra` (Infraestrutura & Topologia do Cluster)

Este namespace concentra os recursos de rede, servidores físicos/virtuais e dispositivos de disco que formam a nossa **Topologia de Multiplexação de Daemons (1:N)**.

### `infra.workers`
Representa um processo daemon ativo na rede responsável pelas operações de I/O em disco.

| Campo | Tipo no Banco (PG) | Mapeamento JSON | Descrição & Regras de Negócio |
| :--- | :--- | :--- | :--- |
| **`id`** | `UUID` | `worker_uuid` | **Chave Primaria** (PK). Gerada via `gen_random_uuid()`. |
| `name` | `VARCHAR(255)` | `worker_name` | Nome único do nó de processamento (ex: `wkr10_1`). Índice Único. |
| `capacity_mb`| `BIGINT` | `worker_capacity_mb` | Capacidade máxima total de armazenamento de todas as suas máquinas. |
| `used_mb` | `BIGINT` | `worker_used_mb` | Volume em MB atualmente consumido (padrão: 0). |
| `status` | `VARCHAR(50)` | `worker_status` | Enum contendo o status operacional: `active` ou `inactive`. |
| `created_at` | `TIMESTAMP` | `worker_created_at` | Carimbo de data/hora da inicialização do daemon. |
| `updated_at` | `TIMESTAMP` | `worker_updated_at` | Última atualização de metadados ou heartbeat. |

### `infra.machines`
Representa uma unidade de alocação física ou virtual (namespace criptográfico) supervisionada por um Worker.

| Campo | Tipo no Banco (PG) | Mapeamento JSON | Descrição & Regras de Negócio |
| :--- | :--- | :--- | :--- |
| **`id`** | `UUID` | `machine_uuid` | **Chave Primaria** (PK). |
| `name` | `VARCHAR(255)` | `machine_name` | Namespace aleatório alfanumérico de 8 caracteres (ex: `machine_aB12XyZ9`). |
| `type` | `VARCHAR(50)` | `machine_type` | Tipo de Storage Engine que rodará nela: `block` (avulso) ou `inline` (LSM-Tree/Append-Only). |
| **`worker_id`**| `UUID` | `machine_worker_id` | **Chave Estrangeira** (FK) apontando para `infra.workers`. |
| `created_at` | `TIMESTAMP` | `machine_created_at` | Data e hora de criação no bootstrap. |
| `updated_at` | `TIMESTAMP` | `machine_updated_at` | Data e hora da última modificação. |

### `infra.discs`
Representa um dispositivo ou partição de armazenamento montado e atrelado a uma Machine.

| Campo | Tipo no Banco (PG) | Mapeamento JSON | Descrição & Regras de Negócio |
| :--- | :--- | :--- | :--- |
| **`id`** | `UUID` | `disc_uuid` | **Chave Primaria** (PK). |
| `serial_number`| `VARCHAR(255)`| `disc_serial_number`| Número de série exclusivo do disco. Índice Único. |
| **`machine_id`**| `UUID` | `disc_machine_id` | **Chave Estrangeira** (FK) associando o disco à tabela `infra.machines`. |
| `capacity_mb` | `BIGINT` | `disc_capacity_mb`| Capacidade máxima em MB do disco físico/lógico. |
| `used_mb` | `BIGINT` | `disc_used_mb` | Ocupação de disco em MB (padrão: 0). |
| `status` | `VARCHAR(50)` | `disc_status` | Status operacional do dispositivo: `active` ou `inactive`. |

---

## 2. Schema: `storage` (Armazenamento de Objetos & Metadados)

Esta pasta lógica delimita o negócio central (Blob Core). Abriga clientes, metadados de arquivos inteiriços, fracionamentos de Erasure Coding (Reed-Solomon) e coordenadas mecânicas da **Block+Offset Engine**.

### `storage.users`
Proprietários autenticados de objetos (Blobs) no sistema R10.

| Campo | Tipo no Banco (PG) | Mapeamento JSON | Descrição & Regras de Negócio |
| :--- | :--- | :--- | :--- |
| **`id`** | `UUID` | `user_uuid` | **Chave Primaria** (PK). |
| `email` | `VARCHAR(255)`| `email` | Endereço de e-mail com Índice Único obrigatório. |
| `name` | `VARCHAR(255)`| `name` | Nome de exibição do usuário ou tenant da nuvem. |

### `storage.memblock32`
Buffer lógico de 32MB destinado a agrupar arquivos de tamanho intermediário (128KB a 32MB - Caso 2) em um lote contínuo antes de gerar os Shards por Erasure Coding.

| Campo | Tipo no Banco (PG) | Mapeamento JSON | Descrição & Regras de Negócio |
| :--- | :--- | :--- | :--- |
| **`id`** | `UUID` | `memblock_uuid` | **Chave Primaria** (PK). |
| `current_size` | `BIGINT` | `current_size` | Volume atual acumulado em bytes no buffer (padrão: 0). |
| `max_size` | `BIGINT` | `max_size` | Teto de estresse do buffer (padrão: 33554432 bytes / 32MB). |
| `status` | `VARCHAR(50)` | `status` | Estado do lote: `active`, `sealed` ou `encoded`. |

### `storage.blobs`
Representação lógica do arquivo ou objeto integral submetido ao Gateway.

| Campo | Tipo no Banco (PG) | Mapeamento JSON | Descrição & Regras de Negócio |
| :--- | :--- | :--- | :--- |
| **`id`** | `UUID` | `blob_uuid` | **Chave Primaria** (PK). Identificador universal do arquivo. |
| **`owner_id`**| `UUID` | `blob_owner_id` | **Chave Estrangeira** (FK) apontando para `storage.users`. |
| **`memblock32_id`**| `UUID (nullable)` | `memblock32_id`| **FK Opcional** apontando para um lote `storage.memblock32`. |
| `size` | `BIGINT` | `blob_size` | Tamanho nominal do arquivo original em bytes. |
| `checksum` | `VARCHAR(255)`| `blob_checksum` | Hash criptográfico de validação contra corrupção silenciosa. |
| `checksum_alg` | `VARCHAR(50)` | `blob_checksum_alg`| Algoritmo de hash (ex: `sha256`, `md5`). |
| `mime_type` | `VARCHAR(255)`| `blob_mime_type` | Tipo de conteúdo HTTP (ex: `image/png`, `video/mp4`). |
| `filename` | `VARCHAR(1024)`| `blob_filename` | Nome original do arquivo submetido. |
| `old_metadata`| `JSONB` | `blob_old_metadata`| Atributos arbitrários customizados no formato JSONB. |
| `deleted` | `BOOLEAN` | `blob_deleted` | Flag binária marcando deleção lógica (padrão: `false`). |

### `storage.blob_chunks`
Representa um Shard físico (dado ou paridade) de Reed-Solomon atribuído a um worker e disco.

> [!IMPORTANT]
> **Diferenciação de Offsets (A Engenharia I/O)**: O campo `LogicalOffset` representa a posição em bytes do fragmento dentro do arquivo original do cliente. Já `PhysicalPath` e `PhysicalOffset` indicam exatamente onde e no exato byte em que a Storage Engine do Worker salvou o shard (por exemplo, dentro do arquivo de volume de append sequencial `volume_01.dat`).

| Campo | Tipo no Banco (PG) | Mapeamento JSON | Descrição & Regras de Negócio |
| :--- | :--- | :--- | :--- |
| **`id`** | `UUID` | `blob_chunk_id` | **Chave Primaria** (PK). |
| **`blob_id`**| `UUID` | `blob_id` | **Chave Estrangeira** (FK) apontando para o `storage.blobs` mestre. |
| **`disc_id`**| `UUID` | `disc_id` | **Chave Estrangeira** (FK) indicando o dispositivo `infra.discs`. |
| **`worker_id`**| `UUID` | `worker_id` | **Chave Estrangeira** (FK) indicando o nó `infra.workers`. |
| `checksum` | `VARCHAR(255)`| `blob_checksum` | Hash individual deste fragmento/shard específico. |
| `size` | `BIGINT` | `blob_size` | Quantidade de bytes gravados neste bloco. |
| **`logical_offset`**| `BIGINT`| `logical_offset` | Ponto de partida em relação ao arquivo original do usuário. |
| **`physical_path`**| `VARCHAR(255)`| `physical_path`| Caminho do arquivo gravado (`volume_01.dat` no Inline ou `chunks/uuid.dat` no Block). |
| **`physical_offset`**| `BIGINT`| `physical_offset`| Byte preciso de início do payload binário no filesystem. |

---

## 3. Schema: `control_plane` (Orquestração Assíncrona & Resiliência)

Este namespace operacional sustenta a automação não-bloqueante via Goroutines para que ações administrativas do cluster rodem nos bastidores de forma reativa com o Frontend Angular sem causar congestionamento nas conexões de rede do usuário ou de uploads na nuvem.

### `control_plane.jobs`
Armazena tarefas de longa duração para permitir o *polling* contínuo de status via HTTP 202 Accepted na SPA.

| Campo | Tipo no Banco (PG) | Mapeamento JSON | Descrição & Regras de Negócio |
| :--- | :--- | :--- | :--- |
| **`id`** | `UUID` | `job_uuid` | **Chave Primaria** (PK). Retornada instantaneamente no início do job. |
| `type` | `VARCHAR(50)` | `job_type` | Natureza do trabalho em background: `bootstrap` ou `reset`. |
| `status` | `VARCHAR(50)` | `job_status` | Máquina de estados do job: `pending`, `running`, `success` ou `failed`. |
| `error` | `TEXT` | `job_error` | Contém stacktrace ou explicação em caso de falha mecânica. |
| `created_at`| `TIMESTAMP` | `job_created_at` | Horário de enfileiramento da tarefa. |
| `updated_at`| `TIMESTAMP` | `job_updated_at` | Horário da conclusão ou mudança de estado. |

### `control_plane.backups`
Registra cópias e espelhamentos agendados entre os dispositivos da nuvem para efeitos de redundância passiva.

| Campo | Tipo no Banco (PG) | Mapeamento JSON | Descrição & Regras de Negócio |
| :--- | :--- | :--- | :--- |
| **`id`** | `UUID` | `backup_id` | **Chave Primaria** (PK). |
| `serial_disc_copied_from`| `VARCHAR(255)`| `serial_disc_copied_from`| Número de série do disco origem em `infra.discs` (Indexado). |
| `serial_disc_copied_to` | `VARCHAR(255)`| `serial_disc_copied_to` | Número de série do disco destino em `infra.discs` (Indexado). |
| `created_at` | `TIMESTAMP` | `backup_created_at`| Horário de início da clonagem do volume. |
| `updated_at` | `TIMESTAMP` | `backup_updated_at`| Horário de finalização do backup. |
