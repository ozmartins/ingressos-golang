# Feature Specification: Comandos do Terminal para a API Servico-Catalogo

**Feature Branch**: `003-catalogo-api-commands`

**Created**: 2026-11-14

**Status**: Draft

**Input**: User description: "Leia o documento http://localhost:8082/openapi.yaml e identifique os controller e endpoints de Servico-Catalogo API. Crie, no terminal um comando pra cada endpoint."

## Levantamento da API (fonte: `http://localhost:8082/openapi.yaml`, Servico-Catalogo API 1.0.0, prefixo `/api/v1`)

A especificação define 5 grupos de endpoints (controllers) e 21 endpoints:

| Controller | Método e caminho | Autenticação | Comando do terminal |
|------------|------------------|--------------|---------------------|
| Filmes | `GET /filmes` (filtros `status`, `page`, `page_size`) | Pública | `filmes listar` |
| Filmes | `POST /filmes` | Exige token | `filmes criar` |
| Filmes | `GET /filmes/{id}` | Pública | `filmes obter` |
| Filmes | `PUT /filmes/{id}` | Exige token | `filmes atualizar` |
| Filmes | `DELETE /filmes/{id}` | Exige token | `filmes remover` |
| Cinemas | `GET /cinemas` (filtros `ativo`, `page`, `page_size`) | Pública | `cinemas listar` |
| Cinemas | `POST /cinemas` | Exige token | `cinemas criar` |
| Cinemas | `GET /cinemas/{id}` | Pública | `cinemas obter` |
| Cinemas | `PUT /cinemas/{id}` | Exige token | `cinemas atualizar` |
| Cinemas | `DELETE /cinemas/{id}` | Exige token | `cinemas remover` |
| Salas | `GET /salas` (filtros `cinema_id`, `ativo`, `page`, `page_size`) | Pública | `salas listar` |
| Salas | `POST /salas` | Exige token | `salas criar` |
| Salas | `GET /salas/{id}` | Pública | `salas obter` |
| Salas | `PUT /salas/{id}` | Exige token | `salas atualizar` |
| Salas | `DELETE /salas/{id}` | Exige token | `salas remover` |
| Sessões | `GET /sessoes` (filtros `filme_id`, `cinema_id`, `data`, `page`, `page_size`) | Pública | `sessoes listar` |
| Sessões | `POST /sessoes` | Exige token | `sessoes criar` |
| Sessões | `GET /sessoes/{id}` | Pública | `sessoes obter` |
| Sessões | `PUT /sessoes/{id}` | Exige token | `sessoes atualizar` |
| Sessões | `DELETE /sessoes/{id}` | Exige token | `sessoes remover` |
| Operacional | `GET /health` (na raiz, fora do prefixo `/api/v1`) | Pública | `health` |

Os endpoints protegidos usam credenciais de máquina (fluxo *client credentials*) com token obtido do provedor de identidade da plataforma. Erros seguem o formato de problema padronizado (título, status, detalhe e lista de erros por campo).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Consultar o catálogo pelo terminal (Priority: P1)

Como pessoa desenvolvedora, quero digitar comandos no terminal retrô para listar e obter filmes, cinemas, salas e sessões, e verificar a saúde do serviço, para explorar o catálogo sem sair da página.

**Why this priority**: Os 11 endpoints de leitura são públicos e entregam valor imediato sem nenhuma credencial.

**Independent Test**: Executar `filmes listar` e ver a página de filmes na saída; copiar um id e executar `filmes obter <id>`; executar `health` e ver o estado do serviço.

**Acceptance Scenarios**:

1. **Given** o serviço de catálogo ativo, **When** o visitante executa `filmes listar`, **Then** a saída mostra os filmes retornados e os dados de paginação (página, tamanho, total, se há próxima).
2. **Given** o comando `filmes listar status=EM_CARTAZ page=2 page_size=5`, **When** é executado, **Then** os filtros e a paginação informados são aplicados à consulta.
3. **Given** um id existente, **When** o visitante executa `cinemas obter <id>`, **Then** a saída exibe os dados do cinema.
4. **Given** um id inexistente, **When** o visitante executa `salas obter <id>`, **Then** a saída mostra um erro "não encontrado" com o título e o detalhe devolvidos pelo serviço.
5. **Given** o serviço ativo, **When** o visitante executa `health`, **Then** a saída informa `ok` e o estado das dependências.
6. **Given** o serviço fora do ar, **When** qualquer comando de API é executado, **Then** a saída mostra uma mensagem clara de serviço indisponível, sem travar o terminal.

---

### User Story 2 - Autenticar para operações de escrita (Priority: P2)

Como pessoa desenvolvedora, quero me autenticar pelo terminal, para poder executar os comandos que criam, atualizam e removem registros.

**Why this priority**: Pré-requisito de todos os comandos de escrita (US3), mas a leitura funciona sem ele.

**Independent Test**: Executar um comando de escrita sem autenticação e ver a orientação para autenticar; autenticar e ver a confirmação; executar `logout` e ver que a sessão foi encerrada.

**Acceptance Scenarios**:

1. **Given** nenhuma autenticação, **When** o visitante executa um comando de escrita, **Then** a saída informa que é preciso autenticar e indica o comando `login`.
2. **Given** credenciais válidas, **When** o visitante executa `login <client_id> <client_secret>`, **Then** a saída confirma a autenticação e os comandos de escrita passam a funcionar.
3. **Given** credenciais inválidas, **When** executa `login`, **Then** a saída mostra falha de autenticação, sem revelar o segredo.
4. **Given** uma sessão autenticada, **When** executa `logout`, **Then** o token é descartado e os comandos de escrita voltam a exigir autenticação.
5. **Given** um token expirado ou rejeitado (401), **When** um comando de escrita é executado, **Then** a saída indica que a sessão expirou e orienta a autenticar de novo.

---

### User Story 3 - Cadastrar, atualizar e remover registros (Priority: P3)

Como pessoa desenvolvedora autenticada, quero criar, atualizar e remover filmes, cinemas, salas e sessões pelo terminal, para administrar o catálogo.

**Why this priority**: Completa a cobertura dos 21 endpoints; depende de US2.

**Independent Test**: Autenticado, executar `filmes criar {...}` e ver o filme criado (com identificador); `filmes atualizar <id> {...}` e ver o resultado; `filmes remover <id>` e ver a confirmação.

**Acceptance Scenarios**:

1. **Given** uma sessão autenticada, **When** o visitante executa `filmes criar {"titulo":"...","duracao_minutos":120,"classificacao_etaria":"12","genero":"Drama"}`, **Then** a saída mostra o filme criado e o caminho informado pelo serviço para o novo registro.
2. **Given** um corpo com campos faltando ou inválidos, **When** o comando é executado, **Then** a saída lista os erros por campo devolvidos pelo serviço.
3. **Given** um registro existente, **When** o visitante executa `salas atualizar <id> {...}`, **Then** a saída mostra o registro atualizado.
4. **Given** um conflito (por exemplo, número de sala repetido ou sala ocupada no horário), **When** o comando é executado, **Then** a saída exibe o erro de conflito com o detalhe do serviço.
5. **Given** um registro existente, **When** o visitante executa `sessoes remover <id>`, **Then** a saída confirma a remoção.
6. **Given** um corpo que não é JSON válido, **When** o comando é executado, **Then** a saída mostra erro de sintaxe e nenhuma requisição é enviada.

---

### User Story 4 - Descobrir os comandos de API (Priority: P4)

Como visitante, quero que o `help` liste todos os comandos de API com sua sintaxe e exemplos, para usá-los sem consultar a documentação externa.

**Why this priority**: Conforto de uso; os comandos funcionam sem isso.

**Independent Test**: Executar `help` e ver os 21 comandos de API, mais `login` e `logout`, agrupados por controller; executar um comando sem os argumentos obrigatórios e ver a sintaxe de uso.

**Acceptance Scenarios**:

1. **Given** o terminal aberto, **When** o visitante executa `help`, **Then** os comandos aparecem agrupados por controller (Filmes, Cinemas, Salas, Sessões, Operacional) com descrição curta.
2. **Given** um comando sem argumento obrigatório (por exemplo `filmes obter`), **When** executado, **Then** a saída mostra a mensagem de uso com a sintaxe esperada.
3. **Given** um controller sem ação ou com ação desconhecida (por exemplo `filmes` ou `filmes xpto`), **When** executado, **Then** a saída lista as ações válidas daquele controller.

---

### Edge Cases

- Resposta sem conteúdo (remoções): a saída confirma o sucesso em vez de ficar em branco.
- Resposta muito longa (listas, plantas de sala): a saída é quebrada em linhas e a rolagem continua funcionando.
- Corpo JSON com espaços e aspas: tudo o que vem após o id (ou após o nome da ação, nas criações) é lido como o corpo, sem exigir escape.
- Id que não é um UUID válido: o terminal avisa antes de enviar ou exibe o erro de entrada inválida devolvido pelo serviço.
- Parâmetro de filtro desconhecido ou valor inválido (`status=XYZ`): exibe o erro de entrada inválida devolvido pelo serviço.
- Segredo digitado no `login`: não pode aparecer na saída, no eco do comando nem no histórico navegável.
- Requisição lenta ou sem resposta: o terminal indica "aguardando" e encerra com mensagem de tempo esgotado.
- Comandos de API não diferenciam maiúsculas/minúsculas no controller e na ação.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: O terminal MUST oferecer exatamente um comando para cada um dos 21 endpoints da Servico-Catalogo API, conforme a tabela do levantamento.
- **FR-002**: Os comandos de listagem MUST aceitar os filtros e a paginação do respectivo endpoint no formato `chave=valor` (por exemplo `status=EM_CARTAZ`, `cinema_id=<uuid>`, `data=2026-11-20`, `page=2`, `page_size=10`).
- **FR-003**: Os comandos `obter` e `remover` MUST receber o identificador do registro como argumento; `atualizar` MUST receber o identificador e o corpo; `criar` MUST receber o corpo.
- **FR-004**: O corpo de `criar` e `atualizar` MUST ser informado como JSON na própria linha de comando, e JSON inválido MUST ser rejeitado localmente, sem enviar requisição.
- **FR-005**: O comando `health` MUST consultar a sonda de saúde na raiz do serviço e exibir o estado e o das dependências.
- **FR-006**: Os comandos que correspondem a endpoints protegidos MUST exigir autenticação prévia e, sem ela, orientar o uso de `login` sem enviar a requisição.
- **FR-007**: O terminal MUST oferecer `login <client_id> <client_secret>` e `logout`; o token MUST ficar apenas em memória, durante a sessão da página, e nunca ser exibido.
- **FR-008**: O segredo digitado no `login` MUST ser ocultado (mascarado) no eco do comando e no histórico.
- **FR-009**: Respostas de sucesso MUST ser exibidas em **texto amigável para pessoas**, sem mostrar o JSON bruto: campos com rótulos em português, valores traduzidos (por exemplo "Em cartaz", "Dublado"), datas e preços no formato brasileiro, cada registro em um bloco legível, listas com resumo de paginação (página X de Y, total e como pedir a próxima) e, nas criações, o caminho do novo registro.
- **FR-010**: Respostas de erro MUST exibir título, status, detalhe e a lista de erros por campo, quando houver; os estados 400, 401, 404, 409 e 503 MUST ter mensagens compreensíveis.
- **FR-011**: Falhas de rede, serviço indisponível e tempo esgotado MUST resultar em mensagem clara, sem interromper o terminal.
- **FR-012**: O `help` MUST listar todos os comandos de API agrupados por controller, com sintaxe e descrição; comandos com argumentos faltando MUST exibir a sintaxe de uso.
- **FR-013**: Todo conteúdo vindo do serviço MUST ser exibido como texto puro, nunca interpretado como marcação.
- **FR-014**: A saída MUST manter o limite de linhas retidas e a rolagem automática já existentes.
- **FR-015**: Enquanto uma requisição está em andamento, o terminal MUST indicar a espera e não permitir que a resposta apareça fora de ordem em relação aos comandos digitados.

### Key Entities

- **Filme**: título, sinopse, duração em minutos, classificação etária, gênero, imagem, situação (em cartaz, em breve, fora de cartaz).
- **Cinema**: nome, cidade, estado, endereço, indicador de ativo.
- **Sala**: pertence a um cinema; número, tipo de tela, fileiras (planta com letra, quantidade e tipo de assento), capacidade total calculada, indicador de ativa.
- **Sessão**: filme, sala, data e hora de início, idioma, preço base, situação; a grade de sessões traz também título do filme, nome do cinema, número da sala e tipo de tela.
- **Página de resultados**: itens e dados de paginação (página, tamanho, total, tem próxima).
- **Problema**: título, status, detalhe e lista de erros por campo.
- **Credencial de sessão**: token de acesso mantido em memória.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% dos 21 endpoints do catálogo podem ser acionados por um comando do terminal, sem duplicidade (um comando por endpoint).
- **SC-002**: Um visitante consulta um filme específico (listar e obter) em menos de 1 minuto, usando apenas o `help`.
- **SC-003**: Em 100% das respostas de erro do serviço, a saída mostra o título e o detalhe do problema.
- **SC-004**: O resultado de um comando de leitura aparece em até 2 segundos em condições normais de rede local.
- **SC-005**: O segredo digitado no `login` aparece em 0 lugares (saída, histórico e console).
- **SC-006**: Executar a sequência autenticar → criar → obter → atualizar → remover para um filme conclui sem erros de uso em uma única sessão.
- **SC-007**: Nenhuma resposta do serviço quebra o layout ou é interpretada como marcação.

## Assumptions

- O Servico-Catalogo está acessível em `http://localhost:8082` durante o desenvolvimento, e o provedor de identidade da plataforma em `http://localhost:8081`; ambos são reais (não simulados), diferentemente dos comandos da feature 002.
- A comunicação do navegador com esses serviços em desenvolvimento será resolvida na fase de planejamento (por exemplo, sem exigir alteração nos serviços Go).
- O `login` usa o fluxo de credenciais de máquina descrito na especificação da API; as credenciais de desenvolvimento são informadas pela pessoa usuária e nunca embutidas no código.
- Fora de escopo: os demais serviços (`estoque`, `pagamento`, `notificacao`), cadastro de usuários, renovação automática de token e persistência do token entre recargas.
- O formato dos comandos (`<controller> <ação> [argumentos]`) e os nomes em português seguem o idioma da API e do projeto.
- Dos comandos locais da feature 002, permanecem apenas `help` e `clear`; `echo`, `date`, `whoami` e `about` foram removidos a pedido.
