# web

Terminal retrô (monitor de fósforo) da plataforma de ingressos, feito com Vite + React + TypeScript.

Os comandos do catálogo chamam a Servico-Catalogo API de verdade e mostram o resultado em
tabelas de texto (não em JSON); tabelas mais largas que a tela rolam na horizontal. Localmente existem apenas `help` e `clear`. Digite `help` para ver tudo.

| Comando | O que faz |
|---------|-----------|
| `filmes`, `cinemas`, `salas`, `sessoes` + `listar` / `obter <id>` | Consultas públicas |
| `... criar <json>` / `atualizar <id> <json>` / `remover <id>` | Escrita (exige `login`) |
| `health` | Estado do serviço |
| `login <client_id> <client_secret>` / `logout` | Autenticação (token só em memória) |

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
