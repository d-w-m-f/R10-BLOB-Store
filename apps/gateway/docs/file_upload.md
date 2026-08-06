# Arquitetura de Upload e Orquestração de EC (Erasure Coding) no Gateway

## A Filosofia

O R10 Blob Store é desenhado para lidar com arquivos gigantes (vários GBs). Para garantir a resiliência tanto da rede do usuário quanto da memória do nosso servidor Gateway, utilizamos uma **Arquitetura de Upload em Pedaços (Chunked Resumable Upload)**.

Ao invés de carregar o arquivo inteiro na memória RAM, o Gateway trabalha **exclusivamente com Streams e I/O de disco temporário**, processando a matemática do Erasure Coding de forma assíncrona.

## 1. O Tamanho Matemático (Slices vs Chunks)

Existe uma diferença fundamental entre os pedaços que viajam na rede e os pedaços finais que são salvos nos discos dos Workers.

- **HTTP Slice (Rede):** 8MB. Este é o tamanho de cada pacote que o Frontend envia via requisições PUT sucessivas.
- **Backend Chunk (Armazenamento):** 32MB. Este é o tamanho do bloco lógico (Case 3) no qual o Erasure Coding (8+4) atua.

A mágica está no alinhamento: **8MB x 4 Slices = 32MB**.

## 2. O Fluxo de Processamento (Streaming para o Disco Local)

### Passo A: Handshake de Inicialização
1. O Angular avisa que vai subir o arquivo "video_10gb.mp4".
2. O Gateway cria um `Upload_ID` e aloca uma pasta no diretório de *Staging* temporário local (ex: `/tmp/r10_uploads/{upload_id}`).

### Passo B: Recepção Contínua e Escrita no Disco (Staging)
1. O Angular começa a enviar as partes de 8MB.
2. A requisição HTTP bate no controlador do Gin.
3. O Gin abre um `io.Reader` do *body* da requisição HTTP e liga em um `io.Writer` que aponta para um arquivo temporário no diretório de Staging. A memória RAM gasta é apenas o pequeno buffer de leitura do Go (kilobytes).

### Passo C: O Disparo Assíncrono do Erasure Coding
1. Quando a **Parte 4** termina de ser recebida e escrita no disco, o diretório daquele upload atinge exatos **32MB**.
2. O Gateway percebe isso instantaneamente. Ele aciona um gatilho e delega esse "bloco de 32MB" recém-formado para uma **Goroutine de Processamento (Worker Interno do Gateway)**.
3. O Frontend não precisa parar. O cliente já começa a enviar a **Parte 5** (que abre um novo ciclo de acumulação de 32MB) enquanto o bloco anterior é processado.

### Passo D: O Destino Final
1. A Goroutine do Gateway lê os 32MB consolidados do disco de staging.
2. Ela passa o bloco pelo algoritmo **Reed-Solomon (8+4)**.
3. O algoritmo gera 12 Pedaços.
4. O Gateway envia esses pedaços para 12 `Workers` (máquinas `block`) distintos.
5. Após a confirmação de escrita, a Goroutine **deleta o bloco de 32MB do diretório temporário**, liberando espaço.

## Resumo dos Benefícios

- **Prevenção de OOM (Out of Memory):** Arquivos de 10GB não usam 10GB de RAM, apenas espaço contido e temporário no SSD local do Docker.
- **Resiliência:** Se a rede cair, perde-se no máximo 8MB de tráfego.
- **Pipeline Paralelo:** Enquanto partes novas estão subindo pela rede lenta, partes antigas estão sendo encriptadas, calculadas e despachadas em alta velocidade para os discos internos.
