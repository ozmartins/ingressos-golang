# Data Model: Comandos de Escrita Guiados por Campos

Estruturas em memória (TypeScript); nada é persistido.

## CampoDef (união discriminada por `tipo`)

| Atributo | Descrição |
|----------|-----------|
| `chave` | nome da propriedade no corpo JSON (ex.: `duracao_minutos`) |
| `rotulo` | texto exibido na pergunta |
| `obrigatorio` | `true` ⇒ resposta em branco é rejeitada; `false` ⇒ branco omite o campo |
| `padrao?` | valor padrão informado na pergunta (apenas exibição; o serviço aplica) |
| `tipo` | `texto` \| `inteiro` \| `logico` \| `opcao` \| `uuid` \| `decimal` \| `datahora` \| `fileiras` |
| `maxTamanho?` | para `texto` |
| `minimo?` | para `inteiro` |
| `opcoes?` | para `opcao` (valores canônicos do enum) |

`fileiras` é composto: cada item tem `fileira` (texto, `^[A-Z]{1,5}$`, obrigatório), `assentos` (inteiro ≥ 1, obrigatório) e `tipo` (opção `NORMAL|PCD|NAMORADEIRA`, padrão `NORMAL`).

## Formulário

`FORMULARIOS[controller][acao]` → lista ordenada de `CampoDef` (ver [contracts/formularios.md](contracts/formularios.md)).

## Dialogo (estado)

| Atributo | Descrição |
|----------|-----------|
| `definicao` | `Definicao` do comando (de `tabela.ts`) |
| `id?` | identificador, nas atualizações |
| `campos` | formulário ativo |
| `indice` | campo atual |
| `respostas` | valores já convertidos, por `chave` |
| `fileiras` | fileiras já aceitas (sub-diálogo) |
| `etapa` | `campo` → (`fileira-letra` → `fileira-assentos` → `fileira-tipo` → `fileira-mais`) → `confirmacao` |

### Transições
- `responder(texto)` em `campo`: converte; inválida ⇒ mesma etapa + mensagem; válida ⇒ próximo campo (ou `confirmacao` ao fim).
- Em `fileiras`: percorre o sub-diálogo; "mais" `s` volta a `fileira-letra`, `n` avança (exige ≥ 1 fileira).
- `confirmacao`: `s` ⇒ resultado `enviar` com o corpo montado; `n`/`cancelar` ⇒ `cancelado`.
- `cancelar` em qualquer etapa ⇒ `cancelado`.

### Saída de cada passo
`{ dialogo | null, linhas: Linha[], enviar?: { corpo } }`.
