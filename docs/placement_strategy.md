# R10 BLOB Store: Placement Strategy (Distribuição de Escrita)

A **Placement Strategy** (ou Estratégia de Distribuição/Colocação) é o algoritmo responsável por decidir em quais *Workers* (máquinas físicas) os *Chunks* de Erasure Coding (ou de paridade) serão armazenados de fato. 

O principal objetivo dessa etapa é garantir **alta disponibilidade e tolerância a falhas**. Nunca podemos permitir que múltiplos fragmentos do mesmo arquivo fiquem na mesma máquina física, pois a falha dessa máquina invalidaria toda a matemática de redundância do Erasure Coding.

Nossa arquitetura roda sob um cluster massivo e requer que escolhamos dinamicamente um grupo de Workers para cada gravação.

## A Heurística de Seleção de 12 Máquinas (8+4)

Dado o volume alto de máquinas (ex: 38 máquinas no laboratório), a heurística adotada equilibra **distribuição de espaço livre** com **espalhamento aleatório** para evitar afunilamento (onde todos os uploads batem sequencialmente nas mesmas máquinas vazias, causando *bottleneck* de IOPS).

### Algoritmo: O Grupo dos 12

Para selecionar as 12 máquinas exclusivas necessárias para o Erasure Coding (8 dados + 4 paridades), o Gateway executa os seguintes passos lógicos no Banco de Dados:

1. **Filtro Primário:**
   Filtra apenas os *Workers* que possuem status ativo (`active`) e que sejam do tipo correto (`block` para chuking pesado).
   
2. **Top 16 por Espaço Livre:**
   O Gateway ordena todos os Workers elegíveis pelo espaço livre restante em disco (`CapacityMB - UsedMB`) em ordem decrescente, e **seleciona o Top 16**.
   
3. **Seleção Determinística (6 Máquinas):**
   Das 16 máquinas escolhidas, as **6 primeiras (com absolutamente o maior espaço livre no cluster todo)** são selecionadas obrigatoriamente. Isso garante que máquinas muito vazias sempre sejam priorizadas para desafogar o cluster.
   
4. **Seleção Randômica (6 Máquinas):**
   Sobram 10 máquinas desse pódio inicial de 16. Destas 10, o Gateway **seleciona aleatoriamente 6**. Isso garante entropia na distribuição de I/O, impedindo que uploads simultâneos sempre engarrafe os mesmos 12 discos.

**Resultado Final:**
Um array de 12 `Worker` structs únicos e perfeitamente capacitados para receber os 12 Chunks fragmentados via requisições HTTP individuais.

## Encapsulamento
No Gateway, essa lógica está rigorosamente isolada no `placement_service.go`, tornando o algorítmo altamente modular. Caso no futuro queiramos implementar *Rack Awareness* geográfico ou *Data Center Awareness*, basta injetar uma nova heurística na mesma interface de serviço.
