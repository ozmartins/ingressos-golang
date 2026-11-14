# Research: Comandos do Terminal para a API Servico-Catalogo

## Levantamento do contrato

- **Decision**: Tomar `http://localhost:8082/openapi.yaml` (Servico-Catalogo API 1.0.0) como fonte; são 5 controllers (Filmes, Cinemas, Salas, Sessões, Operacional) e 21 endpoints, listados no spec.
- **Rationale**: É o contrato publicado, embutido no serviço (`internal/adapter/http/openapi/openapi.yaml`).
- **Alternatives considered**: Gerar o cliente por código a partir do YAML (peso e dependências sem ganho para 21 chamadas).

## Como o navegador alcança os serviços (CORS)

- **Verificado**: `GET /api/v1/filmes` com `Origin` responde 200 **sem** `Access-Control-Allow-Origin`; `OPTIONS` em `/api/v1/filmes` responde 405. O Keycloak, por outro lado, responde ao preflight com `Access-Control-Allow-Origin: http://localhost:5173`.
- **Decision**: Proxy do servidor de desenvolvimento do Vite: `/catalogo/*` → `http://localhost:8082/*` e `/identidade/*` → `http://localhost:8081/*`, com `changeOrigin: true`.
- **Rationale**: Mesma origem para o navegador, sem CORS e sem alterar os serviços Go (constituição II). `changeOrigin` faz o Keycloak emitir o token com o emissor `http://localhost:8081/realms/cinema`, o mesmo que o catálogo espera, já que o `tokenUrl` documentado aponta para `localhost:8081`.
- **Alternatives considered**: Adicionar middleware CORS no catálogo (altera o serviço, fora do escopo); chamar o Keycloak direto do navegador (funciona só para o token, e o catálogo continuaria bloqueado).
- **Limite conhecido**: o proxy existe apenas no `npm run dev`/`preview`. Implantação em produção exigirá proxy reverso equivalente (fora de escopo; o frontend já foi removido do compose).

## Autenticação

- **Decision**: `login <client_id> <client_secret>` faz `POST` `application/x-www-form-urlencoded` (`grant_type=client_credentials`) ao endpoint de token do realm `cinema` e guarda `access_token` apenas em memória; `logout` o descarta; 401 do catálogo limpa o token e orienta a autenticar de novo.
- **Rationale**: É o fluxo descrito em `securitySchemes.clientCredentials`. O segredo nunca vai ao código, ao histórico, à saída nem ao console.
- **Alternatives considered**: Credenciais embutidas (inseguro); Authorization Code com usuário (a API só documenta client credentials); persistir token em `localStorage` (exposição desnecessária).
- **Observação**: o fluxo expõe o segredo ao navegador, aceitável apenas em desenvolvimento, como a própria documentação da API ressalva.

## Estrutura dos comandos

- **Decision**: `<controller> <ação> [argumentos]` com ações `listar`, `obter`, `criar`, `atualizar`, `remover`; `health`, `login`, `logout` isolados. Filtros como `chave=valor`; corpo JSON é o **restante da linha**, sem exigir escape.
- **Rationale**: Mapeamento 1:1 e previsível com os métodos HTTP; o JSON no restante da linha evita um parser de aspas.
- **Alternatives considered**: Uma função por comando (duplicação); flags estilo `--status` (parser mais complexo).

## Tabela declarativa

- **Decision**: Um array de definições `{ controller, acao, metodo, caminho, protegido, filtros?, id?, corpo? }` consumido por um único executor; o `help` e as mensagens de uso são gerados da mesma tabela.
- **Rationale**: Garante por teste "um comando por endpoint" (21 entradas únicas) e mantém `help` e execução sincronizados.

## Requisições, timeout e ordem

- **Decision**: `fetch` com `AbortController` e timeout de 10 s; erros de rede e de tempo esgotado viram linhas de erro. O componente fica em estado `ocupado` durante a chamada, ignorando novos Enter, o que preserva a ordem das respostas (FR-015).
- **Alternatives considered**: Fila de comandos (complexidade sem necessidade).

## Exibição

- **Decision**: Sucesso em texto amigável, sem JSON bruto: um formatador por recurso (filme, cinema, sala, sessão e sessão na grade, health) com rótulos em português, valores traduzidos (`EM_CARTAZ` → "Em cartaz"), data/hora via `toLocaleString('pt-BR')` e preço `R$ 42,00`; listas com blocos separados e resumo de paginação ("Página 1 de 3 · 5 registros no total", "Há mais resultados: use page=2."); formato desconhecido cai em linhas `chave: valor`. `201` mostra "Cadastrado com sucesso." e a localização; `204` mostra "Removido com sucesso."; erro `application/problem+json`: título, status, detalhe e `errors[]` como `campo: mensagem`.
- **Rationale**: A pessoa usuária lê o resultado sem interpretar JSON; sem dependências; tudo como texto puro (FR-013). Os ids continuam visíveis para encadear comandos.
- **Alternatives considered**: Mostrar o JSON (rejeitado a pedido); tabelas ASCII largas (quebram em telas pequenas).

## Segredo no eco e no histórico

- **Decision**: Uma função pura `mascarar(entrada)` troca o segundo argumento do `login` por `****` antes de gerar o eco e antes de registrar no histórico.
- **Rationale**: FR-008 / SC-005.

## Testes

- **Decision**: Vitest com `fetch` falso injetado nas funções puras; UI via quickstart manual contra os serviços reais.
- **Rationale**: Alinhado à feature 002 e à constituição IV, sem jsdom.
