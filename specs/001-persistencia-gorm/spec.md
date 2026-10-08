# Feature Specification: Persistência da plataforma via GORM

**Feature Branch**: `master` (sem branch de feature — o workspace commita direto na master)

**Created**: 2026-10-07

**Status**: Draft

**Input**: User description: "Modifica a aplicação Ingressos de modo que ela passe a usar GORM como mecanismo de persistência."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Todos os serviços persistem por GORM sem mudar o comportamento (Priority: P1)

O mantenedor da plataforma Ingressos quer que o acesso ao banco de dados nos quatro serviços (`catalogo`, `estoque`, `pagamento`, `notificacao`) passe a ser feito por GORM, em lugar do acesso direto ao PostgreSQL que existe hoje. Para quem usa a plataforma — cliente que reserva e paga, operador que cadastra filmes e sessões — nada pode mudar: as mesmas operações, os mesmos resultados, os mesmos erros.

**Why this priority**: é o objetivo da feature. Sem a troca completa, a plataforma teria dois mecanismos de persistência convivendo, o que é pior que qualquer um dos dois.

**Independent Test**: subir a plataforma inteira, percorrer o fluxo ponta a ponta (cadastrar sessão → reservar poltronas → pagar → receber ingresso) e rodar as suítes de teste existentes de cada serviço; todas passam sem alteração de expectativa, e nenhum serviço mantém acesso ao banco fora do GORM.

**Acceptance Scenarios**:

1. **Given** a plataforma no estado atual com suítes unitárias e de integração verdes, **When** a persistência é migrada para GORM nos quatro serviços, **Then** as mesmas suítes continuam verdes sem que nenhuma expectativa de comportamento seja relaxada.
2. **Given** um cliente que reserva poltronas de uma sessão e paga, **When** o pagamento é aprovado, **Then** a reserva é confirmada e o ingresso é emitido exatamente como antes.
3. **Given** a base de código migrada, **When** se inspeciona o acesso a dados de qualquer serviço, **Then** todo acesso ao banco passa pelo GORM e nenhum serviço mantém um segundo caminho de acesso direto ao banco.

---

### User Story 2 - Garantias de exclusividade e consistência são preservadas (Priority: P1)

A plataforma nunca pode vender a mesma poltrona duas vezes, nunca pode perder um evento publicado após gravar a reserva, e nunca pode processar duas vezes a mesma mensagem. Essas garantias hoje dependem de travas de linha, ordem determinística de aquisição, transações que englobam reserva e caixa de saída, e guardas de estado. Elas devem sobreviver intactas à troca de mecanismo.

**Why this priority**: é o risco real da migração. Um ORM que esconde a trava ou a fronteira da transação pode reintroduzir venda duplicada sem que nenhum teste funcional simples perceba.

**Independent Test**: executar a suíte de concorrência e invariantes do `estoque` (largada com muitas reservas simultâneas sobre as mesmas poltronas) e os testes da caixa de saída e de idempotência; o resultado é zero poltronas vendidas em duplicidade, zero eventos perdidos, zero mensagens processadas em duplicidade.

**Acceptance Scenarios**:

1. **Given** muitos clientes disputando as mesmas poltronas ao mesmo tempo, **When** todos tentam reservar, **Then** cada poltrona é concedida a no máximo um cliente e os demais recebem a mesma resposta de indisponibilidade de hoje, sem espera indefinida.
2. **Given** uma reserva gravada, **When** a transação conclui, **Then** o evento correspondente está registrado na caixa de saída na mesma transação; se a transação falha, nenhum dos dois existe.
3. **Given** uma mensagem entregue duas vezes a um consumidor, **When** ambas as entregas são processadas, **Then** o efeito acontece uma única vez.
4. **Given** a perda do índice de expiração em memória, **When** a expiração roda, **Then** reservas vencidas continuam sendo liberadas e nenhuma poltrona é vendida em duplicidade.

---

### User Story 3 - Implantação e dados existentes seguem funcionando (Priority: P2)

Quem sobe a plataforma (`docker compose up --build`) ou já tem dados gravados num banco existente precisa que a troca não exija recriar o banco nem reaprender o procedimento de subida. Os esquemas por serviço (`catalogo`, `estoque`, `pagamento`, `notificacao`) no banco `cinema` continuam sendo o formato dos dados.

**Why this priority**: sem isso a migração seria uma troca de plataforma com perda de dados, não uma troca de mecanismo. É menos arriscada que a US2, por isso P2.

**Independent Test**: restaurar um banco populado pela versão anterior, subir a versão migrada, e confirmar que todas as consultas retornam os dados existentes e que novas gravações convivem com os antigos.

**Acceptance Scenarios**:

1. **Given** um banco populado pela versão atual, **When** a versão migrada sobe contra ele, **Then** todos os registros existentes são lidos corretamente e novas operações funcionam sem passo manual de conversão.
2. **Given** um ambiente novo e vazio, **When** `docker compose up --build` é executado, **Then** os esquemas são criados e a plataforma fica saudável nos mesmos endpoints de saúde de hoje.
3. **Given** a versão migrada em operação, **When** se compara o desempenho das operações de leitura paginada e da reserva com o das metas já medidas pelos testes de desempenho existentes, **Then** as metas continuam atendidas.

---

### Edge Cases

- O que acontece quando duas transações disputam a mesma linha e a trava imediata falha? O erro de indisponibilidade deve continuar chegando ao chamador como hoje, e não como falha genérica de infraestrutura.
- O que acontece com operações que dependem de comportamento específico do PostgreSQL (trava sem espera, ordenação determinística de linhas, valores nulos, tipos de data/hora com fuso)? O resultado observável deve ser idêntico ao atual.
- O que acontece quando o banco fica indisponível no meio de uma transação? A transação é desfeita por inteiro, como hoje, e o erro é reportado nas mesmas categorias já declaradas nos contratos.
- O que acontece com a leitura paginada de listas grandes? Ordem, tamanho de página e cursores/limites permanecem os mesmos.
- O que acontece com os testes de arquitetura que proíbem o núcleo (domínio e casos de uso) de importar infraestrutura? A biblioteca de persistência é infraestrutura e não pode vazar para o núcleo.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Os quatro serviços (`catalogo`, `estoque`, `pagamento`, `notificacao`) MUST fazer todo acesso ao banco de dados relacional por meio do GORM.
- **FR-002**: Nenhum serviço MAY manter, após a migração, um segundo caminho de acesso direto ao banco para operações de negócio.
- **FR-003**: O comportamento observável por clientes, operadores e demais serviços — contratos REST, gRPC e de eventos, códigos e categorias de erro, regras de negócio — MUST permanecer idêntico ao atual.
- **FR-004**: A exclusividade de poltronas MUST continuar garantida pelo banco, com travas de linha sem espera, ordem determinística de aquisição, e reserva e linha da caixa de saída gravadas na mesma transação.
- **FR-005**: A caixa de saída transacional e a idempotência dos consumidores (registro de mensagens processadas e guardas de estado) MUST continuar com as mesmas garantias.
- **FR-006**: O formato dos dados em cada esquema por serviço do banco `cinema` MUST permanecer compatível com bancos já populados pela versão atual, sem conversão manual.
- **FR-007**: A criação e a evolução do esquema do banco MUST continuar funcionando no fluxo de subida atual da plataforma (`docker compose up --build` e os alvos `make migrate-up` / `make migrate-down`), de modo que um ambiente novo fique pronto sem passos extras.
- **FR-008**: O núcleo (domínio e casos de uso) de cada serviço MUST NOT depender do GORM nem de qualquer tipo dele; a dependência fica restrita à camada de adaptadores e à fiação, e o teste de arquitetura existente MUST continuar verde.
- **FR-009**: Os erros de persistência MUST continuar sendo traduzidos nas mesmas categorias que o núcleo e os contratos já conhecem (não encontrado, conflito, indisponível, etc.), sem que o tipo de erro do GORM chegue ao núcleo.
- **FR-010**: As suítes de teste existentes de cada serviço (unitárias, de contrato, de arquitetura e de integração) MUST ser mantidas e passar, sem enfraquecimento de expectativas; ajustes se limitam ao que muda por causa da troca de mecanismo.
- **FR-011**: A dependência direta do driver de acesso anterior MUST ser removida dos módulos que deixarem de usá-la, exceto onde ele continuar sendo exigido como parte do próprio GORM.
- **FR-012**: A documentação do workspace que descreve a persistência (CLAUDE.md, README, constituições e specs de serviço afetadas) MUST ser atualizada para refletir o novo mecanismo.

### Key Entities

- **Esquema por serviço**: espaço de nomes no banco `cinema` que pertence a um serviço; o conjunto de tabelas e o significado dos dados não mudam.
- **Reserva e poltrona (estoque)**: entidades cuja concessão exclusiva é a garantia central da plataforma.
- **Caixa de saída**: registro de eventos a publicar, gravado na mesma transação da mudança que os originou.
- **Mensagem processada**: registro que torna os consumidores idempotentes.
- **Filme, cinema, sala, sessão (catálogo)**, **transação de pagamento (pagamento)**, **ingresso e aviso (notificação)**: entidades de negócio persistidas, inalteradas em significado.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% das suítes de teste existentes (unitárias, contrato, arquitetura e integração) dos quatro serviços passam após a migração, sem relaxar nenhuma expectativa de comportamento.
- **SC-002**: Em teste de concorrência com muitas reservas simultâneas sobre as mesmas poltronas, o número de poltronas concedidas em duplicidade é 0.
- **SC-003**: Em 100% das operações que gravam um fato de negócio e o evento correspondente, ambos existem juntos ou nenhum existe, inclusive sob falha induzida no meio da transação.
- **SC-004**: Um banco populado pela versão anterior é lido e escrito pela versão migrada sem nenhum passo manual de conversão de dados.
- **SC-005**: Um ambiente novo fica saudável com um único `docker compose up --build`, como hoje.
- **SC-006**: Os tempos medidos pelos testes de desempenho existentes (leitura paginada, reserva, largada) permanecem dentro das metas já definidas para cada um.
- **SC-007**: Em 100% dos pontos de acesso a dados dos quatro serviços, o acesso é feito pelo GORM, verificável por inspeção do código.

## Assumptions

- "A aplicação Ingressos" significa a plataforma inteira: os quatro serviços Go. O frontend `web/` não acessa banco e fica fora do escopo.
- GORM é exigência do mantenedor, não escolha a justificar: a feature é a troca de mecanismo de persistência e não uma mudança de funcionalidade. Nenhum requisito de negócio novo é introduzido.
- O banco continua sendo o PostgreSQL único `cinema` com um esquema por serviço; Redis e RabbitMQ não mudam.
- Os contratos versionados (OpenAPI, proto, eventos) não mudam, portanto não há sincronização de contratos a fazer.
- O núcleo hexagonal permanece como está: a troca acontece nos adaptadores de persistência e na fiação de cada serviço.
- Comportamentos que o GORM não expressa de forma nativa (por exemplo, travas sem espera) continuam atendidos dentro do mecanismo escolhido, sem reintroduzir acesso direto ao banco como segundo caminho.
- Como é uma troca de infraestrutura, o princípio II da constituição não exige teste novo de domínio; exige que os testes existentes de domínio e de API continuem verdes, e a suíte de integração é o que prova a equivalência.
- Ponto de decisão para a fase de esclarecimento: quem passa a ser dono da criação e da evolução do esquema do banco (manter os scripts de migração versionados atuais ou delegar ao mecanismo novo). O padrão assumido, na ausência de resposta, é manter os scripts versionados atuais, por preservarem a compatibilidade com dados existentes (FR-006).
