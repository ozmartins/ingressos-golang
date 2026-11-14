# Implementation Plan: Comandos do Terminal para a API Servico-Catalogo

**Branch**: `003-catalogo-api-commands` | **Date**: 2026-11-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/003-catalogo-api-commands/spec.md`

## Summary

Estender o terminal retrô com 21 comandos, um por endpoint do Servico-Catalogo (`<controller> <ação> [args]` + `health`), mais `login`/`logout`. Os comandos são descritos por **uma tabela declarativa** (controller, ação, método, caminho, autenticação, filtros) interpretada por um único executor; um cliente HTTP fino sobre `fetch` (injetável, para testes) faz as chamadas e um formatador converte respostas e erros em linhas de saída. Como o catálogo **não envia cabeçalhos CORS** (verificado: `OPTIONS` devolve 405), o navegador acessa os serviços por **proxy do servidor de desenvolvimento do Vite**, sem alterar nenhum serviço Go. Nenhuma dependência nova.

## Technical Context

**Language/Version**: TypeScript estrito, React 19, Node.js 24

**Primary Dependencies**: nenhuma nova (usa `fetch`, `AbortController` e `URLSearchParams` nativos); Vitest já presente

**Storage**: N/A — token de acesso apenas em memória (estado do componente / módulo), nunca em `localStorage`

**Testing**: Vitest com `fetch` falso injetado para: tabela de comandos (21 únicos), parsing de argumentos, montagem de requisições, formatação de respostas/erros, login/logout, mascaramento do segredo; verificação manual contra os serviços reais via `quickstart.md`

**Target Platform**: Navegadores modernos; serviços do catálogo em `localhost:8082` e Keycloak em `localhost:8081` (desenvolvimento)

**Project Type**: web (frontend single-page, estende `web/`)

**Performance Goals**: Resultado de leitura visível em até 2 s em rede local; timeout de 10 s por requisição

**Constraints**: Sem alterações nos serviços Go; token e segredo nunca exibidos, registrados no histórico ou enviados ao console; conteúdo do serviço sempre como texto puro; ordem das respostas preservada (um comando por vez)

**Scale/Scope**: 21 comandos de API + 2 de sessão (`login`, `logout`), ~6 arquivos-fonte novos e seus testes

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Princípio | Conformidade |
|-----------|--------------|
| I. Simplicidade e YAGNI | PASS: sem bibliotecas HTTP/OpenAPI/CLI; tabela declarativa + um executor em vez de 21 funções; sem geração de código a partir do OpenAPI |
| II. Isolamento do Frontend | PASS: comunicação só por REST conforme o contrato OpenAPI publicado em `/openapi.yaml`, exigida pela especificação; nenhum código Go alterado |
| III. Tipagem Estrita | PASS: respostas JSON tratadas como `unknown` e estreitadas; sem `any` |
| IV. Verificação Obrigatória | PASS: testes automatizados da lógica + `npm run build` + quickstart contra os serviços reais |
| V. Preservação do Workspace | PASS: altera `src/terminal/`, `vite.config.ts` e a documentação; nada fora do escopo |
| Restrições: autenticação via realm Keycloak `cinema` | PASS: `login` usa o fluxo client credentials do realm `cinema`; segredo informado pela pessoa usuária, nunca embutido no código |

Pós-design: sem mudanças. **Resultado: PASS.**

## Project Structure

### Documentation (this feature)

```text
specs/003-catalogo-api-commands/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── comandos-terminal.md   # contrato de sintaxe dos 23 comandos
├── checklists/requirements.md
└── tasks.md                   # gerado por /speckit-tasks
```

### Source Code (repository root = `ingressos-golang/web/`)

```text
web/
├── vite.config.ts                    # + server.proxy (/catalogo → :8082, /identidade → :8081)
└── src/terminal/
    ├── comandos.ts                   # mantém os locais; exporta `AJUDA_LOCAL` e o tipo `Resultado` (+ `sessao?`)
    ├── despachar.ts                  # async: roteia API/login/logout, monta o `help` (locais + API) ou delega a executar()
    ├── despachar.test.ts
    ├── Terminal.tsx                  # usa despachar, aplica `resultado.sessao`, estado "ocupado", mascara segredo
    └── api/
        ├── tabela.ts                 # tabela declarativa dos 21 endpoints
        ├── tabela.test.ts
        ├── argumentos.ts             # parsing: chave=valor, id, corpo JSON
        ├── argumentos.test.ts
        ├── cliente.ts                # requisitar(): fetch, timeout, erros de rede
        ├── cliente.test.ts
        ├── formatar.ts               # resposta/erro/ajuda → Linha[]
        ├── formatar.test.ts
        ├── sessao.ts                 # login/logout, token em memória, mascarar()
        └── sessao.test.ts
```

**Structure Decision**: Subpasta `api/` para tudo que fala com o catálogo, mantendo a lógica pura e testável sem navegador (o `fetch` entra por parâmetro). `despachar.ts` é a única costura entre o componente React e os comandos: recebe a `Sessao` atual no contexto e devolve a nova em `Resultado.sessao` (login, logout e 401); o componente é o dono do estado. O `help` é montado em `despachar.ts`, unindo `AJUDA_LOCAL` (exportada por `comandos.ts`) à ajuda gerada da tabela; `comandos.ts` não conhece os comandos de API.

## Complexity Tracking

Sem violações.
