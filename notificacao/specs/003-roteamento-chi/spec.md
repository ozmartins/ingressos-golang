# Feature Specification: Roteamento HTTP da Notificação via chi

**Feature Branch**: _não aplicável — o trabalho é commitado direto na `master` (regra do mantenedor)_

**Created**: 2026-10-07

**Status**: Draft

**Input**: User description: "Modifique "notificacao" de modo que ele passe a usar chi como mecanismo de rota"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Contrato HTTP idêntico após a troca do roteador (Priority: P1)

O mantenedor troca o mecanismo que associa método + caminho aos tratadores HTTP do Servico-Notificacao para o roteador **chi**, em substituição ao roteador padrão da biblioteca-padrão hoje usado. Para quem consome o serviço (portal web, portaria, demais serviços, sondas de saúde) nada muda: as mesmas rotas existem, com os mesmos métodos, status, corpos, cabeçalhos e regras de autenticação.

**Why this priority**: sem equivalência de contrato a troca não tem valor e quebraria consumidores; é o núcleo da entrega.

**Independent Test**: executar a suíte existente de testes HTTP do serviço sem alterar as asserções e repetir o fluxo do quickstart (listar "meus ingressos", validar um ingresso, abrir `/docs`, consultar `/health/*`), obtendo os mesmos resultados de antes.

**Acceptance Scenarios**:

1. **Given** um cliente autenticado com JWT, **When** faz `GET /api/v1/ingressos/meus-ingressos` (com ou sem filtro de situação), **Then** recebe a mesma resposta de antes.
2. **Given** a portaria com chave de API válida, **When** faz `POST /api/v1/ingressos/validar`, **Then** o ingresso é validado com o mesmo resultado e os mesmos códigos de erro de antes.
3. **Given** uma requisição sem credenciais ou com credenciais inválidas a uma rota protegida, **When** é processada, **Then** é rejeitada com o mesmo status e o mesmo formato de problema de antes.
4. **Given** o serviço em execução, **When** `GET /health/live`, `GET /health/ready`, `GET /openapi.yaml`, `GET /docs` e `GET /docs/` são chamados (sem autenticação), **Then** respondem como antes.

---

### User Story 2 - Respostas para rotas inexistentes e métodos não permitidos preservadas (Priority: P2)

Caminhos que não existem e métodos não suportados em caminhos existentes (por exemplo `GET` em `/api/v1/ingressos/validar`) continuam produzindo respostas equivalentes às de hoje (404 e 405, com o cabeçalho de métodos permitidos quando já presente), sem expor detalhes internos.

**Why this priority**: trocar de roteador costuma alterar silenciosamente esses casos de borda; protege consumidores e testes que dependem deles.

**Independent Test**: requisitar um caminho inexistente e um método incorreto em uma rota existente, comparando status e cabeçalhos com o comportamento anterior capturado antes da troca.

**Acceptance Scenarios**:

1. **Given** um caminho que não corresponde a nenhuma rota, **When** é requisitado, **Then** a resposta é 404 equivalente à anterior.
2. **Given** uma rota existente chamada com método não suportado, **When** é requisitada, **Then** a resposta é 405 equivalente à anterior, informando os métodos permitidos.

---

### Edge Cases

- Barra final em caminhos (`/docs` vs `/docs/`): ambos continuam atendidos como hoje; demais rotas não ganham novos apelidos com barra final.
- Rotas públicas (saúde, especificação, documentação) não podem passar a exigir autenticação, nem rotas protegidas deixar de exigi-la.
- Falha de dependência na verificação de prontidão continua refletida em `/health/ready` como antes.
- A interface expõe `Rotas()` como entrada única; os testes existentes que a usam continuam funcionando.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: O serviço MUST usar o roteador chi como único mecanismo de roteamento HTTP.
- **FR-002**: O serviço MUST expor exatamente o mesmo conjunto de rotas, métodos e caminhos de antes: listagem de ingressos da pessoa, validação de ingresso, saúde (vivo/pronto), especificação do contrato e documentação interativa.
- **FR-003**: Status, corpos, cabeçalhos e formato de erro de cada rota MUST permanecer idênticos aos da versão anterior.
- **FR-004**: As regras de autenticação e autorização por rota (JWT, chave de API da portaria, rotas públicas) MUST permanecer inalteradas.
- **FR-005**: Respostas a caminhos inexistentes (404) e métodos não permitidos (405) MUST permanecer equivalentes às anteriores.
- **FR-006**: O contrato publicado (especificação OpenAPI) MUST permanecer inalterado e coerente com as rotas realmente servidas.
- **FR-007**: A troca MUST ser coberta por testes automatizados que comprovem a equivalência, incluindo rotas protegidas, públicas, 404 e 405.
- **FR-008**: A mudança MUST ficar restrita ao serviço `notificacao`; lógica de negócio, persistência, mensageria e outros serviços não são alterados.

### Key Entities

Não há mudança de dados. Entidades existentes (ingresso, aviso) permanecem como estão.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% da suíte de testes existente do serviço passa sem alteração de asserções após a troca.
- **SC-002**: 100% das rotas documentadas no contrato respondem com os mesmos status e corpos que antes, verificado por teste automatizado.
- **SC-003**: Nenhum consumidor (portal, portaria, demais serviços, sondas de saúde) precisa de qualquer ajuste para continuar funcionando.
- **SC-004**: Casos de 404 e 405 apresentam comportamento equivalente ao anterior em 100% dos cenários testados.

## Assumptions

- O roteador a adotar é o chi, conforme pedido explícito do mantenedor; a versão exata é definida no planejamento.
- Trata-se de refatoração de infraestrutura: nenhuma rota, campo ou comportamento novo é introduzido.
- Middlewares existentes (autenticação, log) são preservados; adoção de middlewares adicionais do chi está fora do escopo.
- Por regra do mantenedor, o trabalho é commitado direto na `master`, sem branch de feature.
- Os demais serviços (`catalogo`, `estoque`, `pagamento`) estão fora do escopo.
