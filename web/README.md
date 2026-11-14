# web

Terminal retrô (monitor de fósforo) da plataforma de ingressos, feito com Vite + React + TypeScript.

Os comandos do catálogo chamam a Servico-Catalogo API de verdade e mostram o resultado em
tabelas de texto (não em JSON); tabelas mais largas que a tela rolam na horizontal. Localmente existem apenas `help` e `clear`. Digite `help` para ver os controllers (`filmes`, `cinemas`, `salas`, `sessoes`, `health`) e `<controller> help` (ex.: `filmes help`) para ver as opções de cada um.

| Comando | O que faz |
|---------|-----------|
| `filmes`, `cinemas`, `salas`, `sessoes` + `listar` / `obter <id>` | Consultas públicas |
| `... criar` / `atualizar <id>` | Escrita guiada (exige `login`): o terminal pergunta os campos, mostra um resumo e pede confirmação; a aplicação monta o JSON. `cancelar` ou Esc aborta |
| `... remover <id>` | Remoção (exige `login`) |
| `health` | Estado do serviço |
| `login` / `logout` | Autenticação pela página de login do Keycloak (Authorization Code + PKCE; token só em memória) |
| `login <client_id> <client_secret>` | Autenticação por client credentials, sem sair do terminal |

Pré-requisitos: Node.js 22.12+ e npm; para os comandos de API, a plataforma no ar
(`cd ingressos-golang && docker compose up`, catálogo em `localhost:8082` e Keycloak em `localhost:8081`).

```
npm install        # instala as dependências
npm run dev        # servidor de desenvolvimento (http://localhost:5173)
npm test           # testes da lógica de comandos, formatação, histórico e linhas
npm run build      # checagem de tipos + build de produção em dist/
npm run preview    # serve o build de dist/
```

O catálogo não envia cabeçalhos CORS, então `npm run dev` e `npm run preview` encaminham `/catalogo` para
`localhost:8082` e `/identidade` para `localhost:8081` (veja `vite.config.ts`). Uma implantação em produção
precisará de um proxy reverso equivalente.

## Login pela página do Keycloak

`login` (sem argumentos) leva o navegador à tela de login nativa do realm `cinema` (client `cinema-app`, Authorization
Code + PKCE). Após autenticar, o Keycloak devolve o usuário a `/` e o front troca o `code` por um bearer token, mantido
só em memória (um recarregamento da página exige novo `login`).

O Keycloak do `docker-compose.yml` usa `KC_HOSTNAME=http://keycloak:8081`, então o navegador precisa resolver `keycloak`:
adicione `127.0.0.1 keycloak` ao `/etc/hosts`. Para outro endereço, defina `VITE_KEYCLOAK_URL` (ex.: `http://localhost:8081`,
caso o `KC_HOSTNAME` seja ajustado em conformidade). Usuário de teste do realm: `ufsc`.
