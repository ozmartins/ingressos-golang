# Contrato: comandos do terminal

Interface exposta à pessoa usuária. Sintaxe: `<controller> <ação> [argumentos]`. Nomes de controller/ação não diferenciam maiúsculas/minúsculas. Filtros são `chave=valor` separados por espaço; o corpo JSON é **tudo o que vem depois** do id (ou da ação, em `criar`).

| Comando | Endpoint | Auth | Argumentos |
|---------|----------|------|------------|
| `filmes listar [status=EM_CARTAZ\|BREVE\|FORA_DE_CARTAZ] [page=N] [page_size=N]` | `GET /api/v1/filmes` | — | filtros opcionais |
| `filmes criar <json>` | `POST /api/v1/filmes` | token | corpo `FilmeEntrada` |
| `filmes obter <id>` | `GET /api/v1/filmes/{id}` | — | id (UUID) |
| `filmes atualizar <id> <json>` | `PUT /api/v1/filmes/{id}` | token | id + corpo `FilmeEntrada` |
| `filmes remover <id>` | `DELETE /api/v1/filmes/{id}` | token | id |
| `cinemas listar [ativo=true\|false] [page=N] [page_size=N]` | `GET /api/v1/cinemas` | — | filtros opcionais |
| `cinemas criar <json>` | `POST /api/v1/cinemas` | token | corpo `CinemaEntrada` |
| `cinemas obter <id>` | `GET /api/v1/cinemas/{id}` | — | id |
| `cinemas atualizar <id> <json>` | `PUT /api/v1/cinemas/{id}` | token | id + corpo `CinemaEntrada` |
| `cinemas remover <id>` | `DELETE /api/v1/cinemas/{id}` | token | id |
| `salas listar [cinema_id=<uuid>] [ativo=true\|false] [page=N] [page_size=N]` | `GET /api/v1/salas` | — | filtros opcionais |
| `salas criar <json>` | `POST /api/v1/salas` | token | corpo `SalaEntrada` |
| `salas obter <id>` | `GET /api/v1/salas/{id}` | — | id |
| `salas atualizar <id> <json>` | `PUT /api/v1/salas/{id}` | token | id + corpo `SalaAtualizacao` |
| `salas remover <id>` | `DELETE /api/v1/salas/{id}` | token | id |
| `sessoes listar [filme_id=<uuid>] [cinema_id=<uuid>] [data=AAAA-MM-DD] [page=N] [page_size=N]` | `GET /api/v1/sessoes` | — | filtros opcionais |
| `sessoes criar <json>` | `POST /api/v1/sessoes` | token | corpo `SessaoEntrada` |
| `sessoes obter <id>` | `GET /api/v1/sessoes/{id}` | — | id |
| `sessoes atualizar <id> <json>` | `PUT /api/v1/sessoes/{id}` | token | id + corpo `SessaoAtualizacao` |
| `sessoes remover <id>` | `DELETE /api/v1/sessoes/{id}` | token | id |
| `health` | `GET /health` (raiz) | — | — |
| `login <client_id> <client_secret>` | token do realm `cinema` | — | credenciais |
| `logout` | — | — | — |

Os esquemas dos corpos (`FilmeEntrada`, `CinemaEntrada`, `SalaEntrada`, `SalaAtualizacao`, `SessaoEntrada`, `SessaoAtualizacao`) são os do OpenAPI do serviço; o terminal não os duplica, apenas repassa o JSON informado.

## Saída

- **2xx com corpo**: tabela de texto por recurso. Listagens: uma linha por registro, `Id` na primeira coluna, seguida de `Página X de Y · N registros no total` (e `Há mais resultados: use page=N.`); lista vazia: `Nenhum registro encontrado.`. Registro individual e `health`: tabela Campo/Valor. `201` mostra "Cadastrado com sucesso.", a tabela e `Localização: ...`; `200` de `atualizar` mostra "Atualizado com sucesso.". Nunca JSON bruto.
- **204**: `Removido com sucesso`.
- **Erro do serviço** (400, 401, 404, 409, 503 ou outro): linhas `erro` com `<status> <título>`, o detalhe e, se houver, `  campo: mensagem`.
- **Falha local**: uso incorreto → sintaxe esperada; JSON inválido → erro de sintaxe sem requisição; sem token em comando protegido → orientação de `login`; rede/tempo esgotado → mensagem de serviço indisponível.
