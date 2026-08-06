# Bracketing: Garantindo Integridade Matemática Pós-Sharding

O *Bracketing* é o processo que o Gateway executa logo após quebrar o arquivo original em Shards (Erasure Coding) para garantir que a decodificação matemática (a reconstrução do arquivo a partir dos pedaços de paridade) ocorra com 100% de precisão. O objetivo é assegurar que nunca perderemos dados antes de comitar os pedaços para os Workers.

No nosso cluster (configurado com RS 8+4), dividimos o dado em **8 Data Shards** (índices de 0 a 7) e **4 Parity Shards** (índices de 8 a 11).

## A Abordagem Determinística (As 5 Combinações)

Em vez de testar aleatoriamente ou computar todas as 162 permutações possíveis de perdas, adotamos uma verificação determinística, selecionando exatamente 5 combinações críticas. 

Durante a verificação em memória, nós intencionalmente anulamos (simulando destruição de nós) 4 shards de dados e pedimos para a biblioteca reconstituí-los baseada nas paridades. As 5 combinações de "sobraviventes" testadas são:

1. **Test 1:** Shards Paridade (8, 9, 10, 11) + Shards Dados (4, 5, 6, 7)
2. **Test 2:** Shards Paridade (8, 9, 10, 11) + Shards Dados (3, 5, 6, 7)
3. **Test 3:** Shards Paridade (8, 9, 10, 11) + Shards Dados (2, 5, 6, 7)
4. **Test 4:** Shards Paridade (8, 9, 10, 11) + Shards Dados (1, 5, 6, 7)
5. **Test 5:** Shards Paridade (8, 9, 10, 11) + Shards Dados (0, 5, 6, 7)

## Por que 5 testes garantem a Matriz de Decodificação Inteira?

O Erasure Coding utiliza operações em matrizes sobre o Corpo de Galois `GF(2^8)`. A matriz geradora tem 12 linhas (8 para identidade, 4 para paridade).
Para decodificar e reconstruir o arquivo, forma-se uma submatriz quadrada 8x8 contendo as linhas dos shards sobreviventes, a qual é invertida. 

O grande perigo são os Shards de Paridade conterem falhas na sua transformação linear.
Fixando as 4 Paridades e os Dados 5, 6 e 7 como "presentes", o teste exige a presença de exatamente **um** shard do grupo {0, 1, 2, 3, 4}.
Ao iterarmos forçando que apenas UM fique fora da falha (e os outros quatro sejam perdidos e reconstruídos), estamos efetivamente testando as 5 permutações de exclusão de subconjuntos de 4 elementos dentro de um grupo de 5.

Como o sistema é uma transformação linear, estamos testando a reconstrução de cada um dos vetores base desse sub-espaço individualmente. Se a matriz for perfeitamente inversível para essas 5 permutações que abrangem os vetores {0, 1, 2, 3, 4}, o **Princípio de Superposição Linear** garante que qualquer combinação linear (qualquer outra combinação de perda) será válida. 

Convertemos assim a necessidade de checar exaustivamente 162 combinações complexas em apenas **5 testes independentes e contíguos de inversão O(n)**, garantindo integridade matemática total na velocidade da luz.
