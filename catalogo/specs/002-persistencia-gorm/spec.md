# Feature Specification: Persistência do Catálogo via GORM

**Feature Branch**: _não aplicável — o trabalho é commitado direto na `master` (regra do mantenedor)_

**Created**: 2026-10-07

**Status**: Draft

**Input**: User description: "Modifique o serviço catalogo de modo que ele passe a usar GORM pra persistir e recuperar dados do BD."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Comportamento de negócio idêntico após a troca (Priority: P1)

O mantenedor troca o meio de persistência do Servico-Catalogo para o GORM, mantendo todo o comportamento observável: consultar filmes, cinemas, salas e sessões (com paginação e filtros), cadastrar e alterar salas e sessões, congelar o layout de uma sala depois de cadastrada, iniciar uma reserva e anunciar os fatos do ciclo de vida de uma sessão pela caixa de saída. Quem consome o serviço (clientes REST, o frontend, o estoque via gRPC e os consumidores das mensagens) não percebe diferença.

**Why this priority**: sem equivalência funcional a troca não tem valor; é o núcleo da entrega e o que torna a migração segura.

**Independent Test**: executar a suíte existente de testes do serviço (unitários e de integração) contra a nova persistência, sem alterar as asserções, e percorrer o fluxo de quickstart (cadastrar sala e sessão, listar, reservar) verificando os mesmos resultados de antes.

**Acceptance Scenarios**:

1. **Given** filmes, cinemas, salas e sessões já cadastrados, **When** um cliente consulta cada listagem ou detalhe (com filtros e paginação), **Then** recebe os mesmos itens, na mesma ordem e com a mesma paginação de antes da troca.
2. **Given** uma sala válida, **When** ela é cadastrada ou uma sessão é criada, **Then** o registro e o fato correspondente na caixa de saída são gravados juntos, e a resposta é igual à anterior.
3. **Given** uma falha ao gravar o fato de saída, **When** a escrita de sala ou sessão é tentada, **Then** nada fica gravado pela metade.
4. **Given** fatos pendentes na caixa de saída, **When** o publicador os drena (inclusive com mais de uma réplica), **Then** cada fato é reenviado até ser aceito e nenhuma réplica o processa em duplicidade simultânea.
5. **Given** uma sala já cadastrada, **When** se tenta alterar seu layout de poltronas, **Then** a recusa é a mesma de antes.

---

### User Story 2 - Esquema de banco preservado e migrações continuam valendo (Priority: P2)

O banco `cinema`, o schema `catalogo` e as migrações versionadas existentes continuam sendo a fonte do esquema. Um ambiente já em operação passa a usar a nova persistência sem migração de dados e sem perda de informação; a criação do esquema continua sendo feita pelo container `migrate-catalogo`, não pelo próprio serviço ao subir.

**Why this priority**: evita que a troca vire uma migração de dados arriscada e mantém o esquema compartilhado e versionado como hoje.

**Independent Test**: subir o serviço novo apontando para um banco criado pelas migrações atuais e já populado com dados da versão anterior; verificar que filmes, salas, sessões e fatos pendentes existentes são lidos e processados normalmente.

**Acceptance Scenarios**:

1. **Given** um banco populado pela versão anterior, **When** o serviço com a nova persistência inicia, **Then** nenhuma alteração de esquema ou de dados é necessária e os registros existentes são tratados normalmente.
2. **Given** um banco vazio, **When** `docker compose up` roda, **Then** as migrações existentes criam o esquema e o serviço sobe saudável.

---

### User Story 3 - Fronteira entre núcleo e infraestrutura mantida (Priority: P3)

Quem mantém o código continua encontrando o acesso a dados isolado em adaptadores. O domínio e os casos de uso seguem sem conhecer o GORM, e a verificação automática de dependências do núcleo (o linter da esteira) continua passando.

**Why this priority**: preserva a regra de dependências apontando para dentro da constituição do serviço; é uma qualidade da troca, não um comportamento novo.

**Independent Test**: rodar a verificação de dependências (`make lint`) e verificar que nenhum pacote do núcleo importa a biblioteca nova nem o driver de banco.

**Acceptance Scenarios**:

1. **Given** a troca concluída, **When** a verificação de dependências roda, **Then** ela passa, e a biblioteca nova passa a constar entre as proibidas ao núcleo.
2. **Given** o núcleo do serviço, **When** os testes unitários rodam, **Then** continuam executando sem banco, sem rede e sem servidor.

---

### Edge Cases

- Falha no meio de uma escrita que grava registro e fato de saída: ambos devem ser desfeitos juntos; nada fica pela metade.
- Duas réplicas drenando a mesma caixa de saída: o mesmo fato não pode ser tomado por ambas ao mesmo tempo, e uma réplica não pode ficar esperando a outra.
- Fato republicado com o mesmo identificador de mensagem: continua sendo gravado uma única vez.
- Banco indisponível na inicialização ou durante a operação: o serviço reporta a indisponibilidade como antes (verificação de saúde e erro ao chamador), sem expor detalhe interno na resposta.
- Listagens paginadas sobre conjuntos vazios, no limite da última página ou com filtros sem resultado: continuam devolvendo o mesmo formato de antes.
- Dados já existentes com valores nulos ou formatos aceitos pela versão anterior (inclusive tipos de coluna especiais e o layout das salas): continuam legíveis e graváveis.
- Os planos de consulta e os índices existentes continuam sendo usados pelas consultas principais; a troca não pode degradar consultas que hoje usam índice para varredura completa.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Toda leitura e escrita de dados do serviço de catálogo MUST passar a ser feita por meio do GORM, substituindo o acesso direto ao banco usado hoje.
- **FR-002**: O comportamento externo do serviço (contrato REST, contrato gRPC consumido/oferecido, mensagens publicadas, categorias e códigos de erro) MUST permanecer idêntico ao anterior à troca.
- **FR-003**: A gravação de sala ou sessão e a do fato correspondente na caixa de saída MUST continuar ocorrendo na mesma transação atômica.
- **FR-004**: A drenagem da caixa de saída MUST continuar segura para múltiplas réplicas simultâneas (cada fato tomado por uma só réplica por vez, sem bloqueio entre elas) e a gravação do fato MUST continuar idempotente pelo identificador de mensagem.
- **FR-005**: As listagens MUST manter a mesma ordenação, filtros e paginação de hoje.
- **FR-006**: O esquema do banco e suas migrações versionadas MUST permanecer a única fonte de criação e evolução do esquema; o serviço MUST NOT criar ou alterar tabelas por conta própria ao iniciar.
- **FR-007**: A conexão MUST continuar restrita ao schema `catalogo` do banco compartilhado e a tabela de controle de migrações MUST continuar fixada nesse schema.
- **FR-008**: O serviço MUST funcionar sobre dados já existentes, sem exigir migração de dados nem janela de manutenção.
- **FR-009**: A verificação de saúde MUST continuar refletindo a disponibilidade do banco.
- **FR-010**: O núcleo (domínio e casos de uso) MUST NOT importar o GORM nem o driver de banco; a verificação automática de dependências do núcleo MUST continuar passando e MUST passar a proibir também a biblioteca nova.
- **FR-011**: O pool de conexões, o tempo de vida das conexões e o encerramento ordenado do serviço MUST continuar funcionando de forma equivalente.
- **FR-012**: A configuração do serviço (variável de conexão com o banco) MUST permanecer a mesma para quem opera o ambiente.
- **FR-013**: O tratamento de erros do banco MUST permanecer o de hoje — o adaptador embrulha o erro com contexto e devolve `NaoEncontrado` onde já devolvia — e nenhuma mensagem de driver, consulta ou endereço MUST chegar à resposta. As categorias de erro do contrato não mudam.

### Key Entities

- **Filme**: obra em cartaz, com título, sinopse, duração, classificação etária, gênero, imagem e status.
- **Cinema**: unidade exibidora, com estado de ativação.
- **Sala**: ambiente de exibição de um cinema, com layout de poltronas que fica congelado depois de cadastrada.
- **Sessão**: exibição de um filme em uma sala em um horário, com seu ciclo de vida.
- **Fato de saída (outbox)**: mensagem a ser publicada, gravada na mesma transação da escrita que a originou.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% dos testes automatizados existentes do serviço (unitários e de integração) passam sem enfraquecer nenhuma asserção.
- **SC-002**: Em teste com múltiplas réplicas drenando a mesma caixa de saída, cada fato pendente é tomado por exatamente uma réplica por vez e nenhum fato é perdido.
- **SC-003**: O fluxo ponta a ponta do quickstart (cadastrar sala e sessão, listar, reservar) produz resultados idênticos aos da versão anterior.
- **SC-004**: Um banco populado pela versão anterior é operado pelo serviço novo com zero scripts de migração de dados.
- **SC-005**: Nenhum pacote do núcleo depende da nova biblioteca de persistência (verificado automaticamente pela verificação de dependências da esteira).
- **SC-006**: A latência típica das consultas de listagem e de reserva não piora de forma perceptível em relação à versão anterior (mesma ordem de grandeza em ambiente local), e o teste de planos de consulta existente continua passando.

## Assumptions

- A escolha do GORM é uma decisão do mantenedor, não está em discussão (embora a pesquisa da spec 001 a tenha descartado em favor do pgx, o pedido atual a supera); o escopo é o serviço `catalogo` apenas (estoque, pagamento e notificacao não mudam).
- O GORM atuará sobre o mesmo PostgreSQL e o mesmo schema `catalogo`; as migrações SQL existentes continuam como fonte do esquema (sem auto-migração do GORM).
- Os contratos versionados (OpenAPI e proto) e as mensagens AMQP não mudam, portanto não há nova versão de contrato.
- RabbitMQ, a segurança (JWT) e a chamada gRPC ao estoque não são afetados.
- Se algum recurso exigido (por exemplo `FOR UPDATE SKIP LOCKED`, `ON CONFLICT DO NOTHING` ou consultas de listagem específicas) não puder ser expresso por meios nativos do GORM, é aceitável usar cláusulas do próprio GORM ou seu mecanismo de consulta bruta dentro do adaptador, sem alterar o núcleo.
- Divergências entre o código atual e a spec 001 são tratadas conforme o princípio de que o código é a fonte da verdade; qualquer divergência encontrada será levada ao mantenedor, não corrigida unilateralmente.
- O trabalho é commitado direto na `master`, sem branch de feature.
