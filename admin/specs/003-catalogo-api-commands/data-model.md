# Data Model: Comandos do Terminal para a API Servico-Catalogo

Estado apenas em memória, durante a sessão da página. Os recursos de domínio (Filme, Cinema, Sala, Sessão) pertencem ao serviço e **não** são modelados no frontend: respostas são tratadas como `unknown` e exibidas como JSON.

## DefinicaoComando (tabela declarativa)

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `controller` | `'filmes' \| 'cinemas' \| 'salas' \| 'sessoes' \| 'operacional'` | Grupo do endpoint (`operacional` só tem `health`) |
| `acao` | `'listar' \| 'obter' \| 'criar' \| 'atualizar' \| 'remover' \| 'health'` | Verbo do comando |
| `metodo` | `'GET' \| 'POST' \| 'PUT' \| 'DELETE'` | Método HTTP |
| `caminho` | `string` | Caminho com `{id}` quando houver (ex.: `/filmes/{id}`); `/health` na raiz |
| `raiz` | `boolean` | `true` apenas para `health` (fora do prefixo `/api/v1`) |
| `protegido` | `boolean` | Exige token |
| `filtros` | `string[]` | Chaves aceitas em `listar` (inclui `page`, `page_size`) |
| `usaId` / `usaCorpo` | `boolean` | Argumentos obrigatórios |
| `uso` / `descricao` | `string` | Texto para `help` e mensagens de uso |

Invariante (testado): 21 definições, pares `(controller, acao)` únicos e pares `(metodo, caminho)` únicos.

## ArgumentosComando (resultado do parsing)

`{ id?: string; query: Record<string,string>; corpo?: unknown }` ou um erro de uso/sintaxe.

## RequisicaoApi

`{ metodo; url; query; corpo?; token? }` — montada a partir da definição e dos argumentos.

## RespostaApi

`{ status: number; corpo: unknown; location?: string }`; falha de rede/tempo esgotado é representada como erro tipado (`rede` | `timeout`).

## Sessao

`{ token: string | null }` — preenchido por `login`, esvaziado por `logout` e por resposta 401.

## Contexto de despacho

`{ agora: Date; fetchFn: typeof fetch; sessao: Sessao; base: { catalogo: string; identidade: string } }` — injetado para permitir testes sem rede. `despachar` nunca altera `sessao` no lugar: devolve a nova em `Resultado.sessao`.

## Linha (existente)

Reutiliza `Linha` da feature 002 (`sistema | entrada | saida | erro`).

## Resultado (estendido)

`{ linhas: Linha[]; limpar: boolean; sessao?: Sessao }` — definido em `src/terminal/comandos.ts`. `sessao` só vem preenchido quando a sessão muda (`login` bem-sucedido, `logout`, resposta 401); ausente significa "sem alteração", e o componente mantém o estado atual.
