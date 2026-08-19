gateway lida com:

service discovery
integra bd (auth, users, machines)
service discovery
pode integrar com kafka
pode integrar futuramente com serviços terceiros ou crescer
vai ter que fazer o bracketing

gateway:

conversa por rest com web
conversa por rest com o storages
pode conversar por grpc com o storages no futuro
pode conversar por rest provavelmente com outros serviços de terceiros no futuro.


docs
pkg
platform/
test/
test/bddleia:
https://github.com/golang-standards/project-layout/blob/master/README_ptBR.md
e me diga como eu deveria estruturar o meu projeto de backend aí gateway em go, que:
- vai conversar por rest com API
- vai conversar com microsserviços por rest, e talvez grpc no futuro
- vai conversar com banco de dados, futuramente talvez brokers, e tem que ter flexibilidade de expandir pra integrar com outras coisas
- vai fazer service discovery
- vai fazer rate limiting

por baixo esses são os requisitos do meu gateay, numa 'conta de papel de pao'. E ai, o que me diz?