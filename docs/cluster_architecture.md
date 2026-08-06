# Arquitetura do Worker (wkr10) e Cluster de Desenvolvimento Local

O `wkr10` é o *storage daemon* responsável por realizar as escritas diretas no hardware (ou pastas simuladas). Para garantir um ambiente de desenvolvimento robusto sem depender de orquestração complexa (como 5 instâncias no Docker Compose), adotamos uma simulação de hardware no próprio host (máquina local).

## Topologia de Hardware Simulado

Nossa rede de desenvolvimento requer **5 máquinas lógicas**, que rodam sob o mesmo IP (`localhost`) mas em **portas distintas** e operam em **pastas diferentes**, como se fossem discos físicos completamente apartados.

A estrutura é definida assim:

| Worker / Machine | Tipo   | Porta | Diretório Físico (Mount)         | Papel (Role)                |
|------------------|--------|-------|----------------------------------|-----------------------------|
| Machine 1        | Block  | 9001  | `/tmp/r10_cluster/machine_1/`    | Armazena Chunks de 32MB     |
| Machine 2        | Block  | 9002  | `/tmp/r10_cluster/machine_2/`    | Armazena Chunks de 32MB     |
| Machine 3        | Block  | 9003  | `/tmp/r10_cluster/machine_3/`    | Armazena Chunks de 32MB     |
| Machine 4        | Block  | 9004  | `/tmp/r10_cluster/machine_4/`    | Armazena Chunks de 32MB     |
| Machine 5        | Inline | 9005  | `/tmp/r10_cluster/machine_5/`    | Arquivos pequenos (<128KB)  |

Para subir o cluster localmente, o desenvolvedor iniciará 5 binários do `wkr10` com as seguintes variáveis de ambiente:
```bash
# Exemplo subindo o Worker da Máquina 1
MACHINE_ID="<UUID-NO-BANCO>" DATA_DIR="/tmp/r10_cluster/machine_1" PORT="9001" MACHINE_TYPE="block" ./wkr10
```

## Estratégias de Escrita e I/O (I/O Engines)

A depender do tipo de máquina (`MACHINE_TYPE`), o Worker implementa motores de I/O drásticamente diferentes para evitar esgotamento de Inodes e melhorar velocidade:

### 1. Engine: Block Machines (Casos > 32MB e Erasure Coding)
Para as máquinas de tipo **Block**, o sistema trabalha com arquivos grandes (Chunks).
Como a própria natureza de um Chunk de Erasure Coding (ou a fatia inteira de 32MB de um arquivo maior) já tem um tamanho excelente para o disco rígido padrão (ext4/xfs), **escrevemos arquivos estáticos normalmente**.
- **Padrão de Nome:** `/data/chunk_<chunk_uuid>.dat`
- **Por que?** Evita a complexidade de alocação de blocos grandes e lida muito bem com exclusão (basta deletar o arquivo de 32MB e o SO cuida do resto).

### 2. Engine: Inline Machines (Casos < 128KB)
Para a máquina de tipo **Inline**, lidar com milhões de fotos/textos de 12KB esgotaria os _inodes_ da partição muito rapidamente e geraria uma fragmentação imensa no File System.
A solução adotada é o **Block + Offset (Estilo Haystack/SeaweedFS)**.
- O Worker inicia abrindo um **Volume Contíguo** enorme (ex: 10GB) chamado `inline_volume_01.dat`.
- Quando a foto de 12KB chega, o worker não cria um arquivo novo no Linux. Ele abre o `inline_volume_01.dat`, aponta o cursor para o final (Append-Only) e despeja os 12KB lá.
- Ele devolve para o Gateway: `VolumeID: 01`, `Offset: 1.000.450`, `Size: 12.000 bytes`.
- **Por que?** O I/O é estritamente sequencial (super rápido em HDDs tradicionais e otimizado em SSDs), e gastamos apenas 1 Inode para milhares de arquivos!

## Inicialização e Reset (Bootstrapping)

A orquestração não será feita via Docker para esses 5 nós na fase atual de desenvolvimento. Foram criados dois scripts no pacote do `gateway` (`scripts/setup_cluster.go` e `scripts/reset_cluster.go`).
- **Setup:** Popula as tabelas do PostgreSQL (`Machines`, `Discs`, `Workers`) com os UUIDs corretos das 5 máquinas virtuais. Em seguida, gera fisicamente as pastas de montagem (`/tmp/r10_cluster/`).
- **Reset:** Limpa todas as instâncias de tabelas relacionadas à topologia (Workers, Machines, Discs) e faz um `rm -rf` no cluster local simulado para testes reprodutíveis.
