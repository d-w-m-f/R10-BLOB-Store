# R10 BLOB Store - Frontend (Angular)

Bem-vindo ao frontend do R10 BLOB Store! Este projeto foi desenvolvido em **Angular 18+**, focado em um design premium (glassmorphism), cores vibrantes (turquesa, ciano, azul escuro) e alta performance utilizando Standalone Components e as novas reatividades do framework.

## 🚀 Como Rodar o Projeto

1. **Instale as dependências**:
   Navegue até a pasta `apps/web` e instale os pacotes necessários:
   ```bash
   cd apps/web
   pnpm install
   ```

2. **Inicie o servidor de desenvolvimento**:
   ```bash
   pnpm start
   ```
   > O comando chama o `ng serve` do Angular CLI.
   > Acesse `http://localhost:4200` no seu navegador para ver a aplicação rodando.

## 📦 Dependências e Bibliotecas

- **Angular Core & Router**: `v18.x.x`
- **RxJS**: Gerenciamento de eventos e fluxos assíncronos (embutido).
- **Estilização**: SCSS puro. Não utilizamos TailwindCSS, apenas variáveis CSS e flexbox/grid no arquivo `styles.scss` e escopo de componentes.
- **Ícones**: [Phosphor Icons](https://phosphoricons.com/) (importados via CDN no `index.html`). 

---

## 📚 Tutorial Angular para Iniciantes (Vindos do Next.js/React)

O Angular é de fato um framework "opinionated" (opinativo) e "batteries included" (com tudo incluso). Ele te fornece a estrutura completa, sem que você precise tomar dezenas de decisões sobre quais bibliotecas usar para tarefas comuns.

Aqui vai um comparativo e uma explicação de como as coisas funcionam aqui, feitas para você que está acostumado com ecossistemas como React/Next.js.

### 1. Standalone Components (A Nova Era do Angular)
Antigamente, o Angular usava arquivos chamados `NgModules` para agrupar componentes. Hoje (a partir da v14/v15, e consolidado na v18), usamos **Standalone Components**. Eles se parecem mais com componentes React: cada componente declara no próprio arquivo (`imports: [...]`) o que ele precisa para funcionar.

### 2. Rotas (`app.routes.ts`) vs App/Pages Router (Next.js)
Enquanto no Next.js o roteamento é baseado em arquivos (pastas `app` ou `pages`), no Angular o roteamento é definido de forma **declarativa em um arquivo TypeScript** (o `app.routes.ts`).
- Aqui nós configuramos um *Lazy Loading* nativo. Reparou no `loadComponent: () => import(...)`? Isso significa que o Angular vai dividir seu código (Code Splitting) automaticamente e só carregará a página de `upload` quando o usuário realmente acessar a rota `/dashboard/upload`.

### 3. Validação de Formulários (Reactive Forms vs Zod)
No Next.js/React, você normalmente uniria um `react-hook-form` com `zod`. No Angular, isso já vem "na caixa" com os **Reactive Forms** (`@angular/forms`).
- No arquivo `login.component.ts`, usamos o `FormBuilder` para criar o grupo de campos (`email` e `password`).
- As regras (como `Validators.required` ou `Validators.email`) já vêm embutidas. 
- O HTML do formulário é "bindado" (conectado) diretamente ao objeto TypeScript usando a diretiva `[formGroup]="loginForm"`.

### 4. Requisições HTTP (HttpClient vs Axios)
Em vez de instalar o `axios` ou usar o `fetch` puro, o Angular fornece o **HttpClient** (`@angular/common/http`).
- Ele não retorna `Promises` por padrão, mas sim **Observables** (via RxJS). Observables são poderosos: você pode cancelar uma requisição, repetir caso falhe e encadear operações de forma limpa.
- No momento, nosso `AuthService` está usando Promises simuladas (`mock`) para facilitar o entendimento imediato, mas no futuro substituiremos por injetar o `HttpClient` e fazer um `.post('http://meugateway.com/login')`.

### 5. Autenticação, Sessions e AuthGuards
Como lidamos com quem entra e quem não entra nas páginas privadas?
- **O Guardião (Route Guards):** Criamos um `auth.guard.ts`. No `app.routes.ts`, dizemos que a rota `/dashboard` tem um `canActivate: [authGuard]`. Antes de carregar a página, o Angular executa essa função. Se ela retornar `true`, o usuário entra. Se retornar uma `UrlTree`, ele é redirecionado para o login.
- **O Serviço de Estado (`AuthService`):** No Angular v16+, foi introduzido o **Signal** (a resposta do Angular ao `useState` do React ou `ref` do Vue). Nosso `auth.service.ts` possui:
  ```typescript
  isAuthenticated = signal<boolean>(false);
  ```
  Ele é um estado reativo global, acessível por injeção de dependência (`inject(AuthService)`) em qualquer parte do sistema.
- **Sessão:** Quando o usuário loga (com e-mail e senha), nós salvamos um "token" no `localStorage` do navegador e trocamos o estado do Signal para `true`. O Angular, ao inicializar (quando você atualiza a página F5), lê esse `localStorage` no construtor do `AuthService` para recuperar a sessão e não deslogar o usuário.

---

### 🎨 Sobre os Ícones (Phosphor Icons)
Você perguntou como eles funcionam:
No arquivo `src/index.html`, inserimos o script oficial da Phosphor via CDN:
```html
<script src="https://unpkg.com/@phosphor-icons/web"></script>
```
Com isso importado na raiz, você pode usar os ícones em qualquer lugar do HTML dos seus componentes, bastando criar uma tag `<i>` com as classes do ícone.
- Padrão: `<i class="ph ph-bell"></i>` (Sino)
- Preenchido: `<i class="ph-fill ph-bell"></i>`
- Você pode conferir todos os nomes no [site oficial do Phosphor Icons](https://phosphoricons.com/). O CSS controla o tamanho deles (usando `font-size`) e a cor (usando `color`), herdando a estilização como se fosse um texto!
