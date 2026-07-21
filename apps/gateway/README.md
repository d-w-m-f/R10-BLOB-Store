# R10 Blob Store - Gateway

## estrutura de pastas

Inspirado no guia de estrutura de pastas da [golang-standards](https://github.com/golang-standards/project-layout/blob/master/README_ptBR.md)

A seguir, uma descrição de cada pasta:

### /api

Especificações OpenAPI/Swagger, arquivos de esquema JSON, arquivos de definição de protocolo.

### /build 

Empacotamento e integração contínua.

Coloque suas configurações de pacote e scripts em nuvem (AMI), contêiner (Docker), sistema operacional (deb, rpm, pkg) no diretório /build/package.

Coloque suas configurações e scripts de CI (travis, circle, drone) no diretório /build/ci. 

### /cmd

Principais aplicações para este projeto.

O nome do diretório para cada aplicação deve corresponder ao nome do executável que você deseja ter (ex. /cmd/gateway).

Não coloque muitos códigos no diretório da aplicação. Se você acha que o código pode ser importado e usado em outros projetos, ele deve estar no diretório /pkg. Se o código não for reutilizável ou se você não quiser que outros o reutilizem, coloque esse código no diretório /internal.

### /deployments

IaaS, PaaS, configurações e modelos de implantação de orquestração de sistema e contêiner (docker-compose, kubernetes / helm, mesos, terraform, bosh).

### /docs
Documentos do projeto e do usuário (além da documentação gerada pelo godoc).

### /internal

Aplicação privada e código de bibliotecas. Este é o código que você não quer que outras pessoas importem em suas aplicações ou bibliotecas. Observe que esse padrão de layout é imposto pelo próprio compilador Go. Veja o Go 1.4 release notes para mais detalhes. Observe que você não está limitado ao diretório internal de nível superior. Você pode ter mais de um diretório internal em qualquer nível da árvore do seu projeto.

Opcionalmente, você pode adicionar um pouco de estrutura extra aos seus pacotes internos para separar o seu código interno compartilhado e não compartilhado. Não é obrigatório (especialmente para projetos menores), mas é bom ter dicas visuais que mostram o uso pretendido do pacote. Seu atual código da aplicação pode ir para o diretório /internal/app (ex. /internal/app/myapp) e o código compartilhado por essas aplicações no diretório /internal/pkg (ex. /internal/pkg/myprivlib).


### /pkg

Código de bibliotecas que podem ser usados por aplicativos externos (ex. /pkg/mypubliclib). Outros projetos irão importar essas bibliotecas esperando que funcionem, então pense duas vezes antes de colocar algo aqui :-) Observe que o diretório internal é a melhor maneira de garantir que seus pacotes privados não sejam importáveis porque é imposto pelo Go. O diretório /pkg contudo é uma boa maneira de comunicar explicitamente que o código naquele diretório é seguro para uso. I'll take pkg over internal A postagem no blog de Travis Jeffery fornece uma boa visão geral dos diretórios pkg e internal, e quando pode fazer sentido usá-los.

Não há problema em não usá-lo se o projeto do seu aplicativo for muito pequeno e onde um nível extra de aninhamento não agrega muito valor (a menos que você realmente queira :-)). Pense nisso quando estiver ficando grande o suficiente e seu diretório raiz ficar muito ocupado (especialmente se você tiver muitos componentes de aplicativos não Go).


### /test

Aplicações de testes externos adicionais e dados de teste. Sinta-se à vontade para estruturar o diretório /test da maneira que quiser. Para projetos maiores, faz sentido ter um subdiretório de dados. Por exemplo, você pode ter /test/data ou /test/testdata se precisar que o Go ignore o que está naquele diretório. Observe que o Go também irá ignorar diretórios ou arquivos que começam com "." ou "_", para que você tenha mais flexibilidade em termos de como nomear seu diretório de dados de teste.


### /web
Componentes específicos de aplicativos da Web: ativos estáticos da Web, modelos do lado do servidor e SPAs.


