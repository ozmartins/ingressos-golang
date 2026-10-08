# Feature Specification: Persistência da Notificação via GORM

**Feature Branch**: _não aplicável — o trabalho é commitado direto na `master` (regra do mantenedor)_

**Created**: 2026-10-07

**Status**: Draft

**Input**: User description: "modifique o servico "notificacao" de modo que ele passe a usar GORM como mecanimos pra persistir e recuperar dados do BD."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Comportamento de negócio idêntico após a troca (Priority: P1)

O mantenedor troca o meio de persistência do Servico-Notificacao para o GORM, mantendo todo o comportamento observável: emitir o ingresso digital a partir de um pagamento confirmado (uma única vez por reserva), registrar o aviso enviado ao cliente, dar baixa (validar) um ingresso, consultar um ingresso e listar os ingressos de uma pessoa com filtro por situação. Quem consome o serviço (clientes REST, e a mensageria de pagamento) não percebe diferença.

**Why this priority**: sem equivalência funcional a troca não tem valor; é o núcleo da entrega e o que torna a migração segura.

**Independent Test**: executar a suíte existente de testes do serviço (unitários e de integração) contra a nova persistência, sem alterar as asserções, e repetir o fluxo do quickstart (publicar pagamento, consultar e validar o ingresso) verificando os mesmos resultados de antes.

**Acceptance Scenarios**:

1. **Given** um pagamento confirmado de uma reserva sem ingresso, **When** o fato é consumido, **Then** exatamente um ingresso válido é gravado e o aviso correspondente é registrado, igual ao comportamento anterior.
2. **Given** a mesma mensagem de pagamento entregue duas vezes (ou em paralelo), **When** ambas são processadas, **Then** existe um único ingresso para a reserva e o segundo processamento devolve o ingresso já existente, sem erro e sem duplicidade.
3. **Given** um ingresso válido, **When** duas validações chegam ao mesmo tempo, **Then** exatamente uma dá baixa e a outra é informada de que ele já foi utilizado.
4. **Given** uma pessoa com ingressos em situações diferentes, **When** ela lista seus ingressos (com ou sem filtro de situação), **Then** recebe o mesmo conjunto, na mesma ordem (mais recentes primeiro), que a versão anterior devolvia, e uma lista vazia — não um erro — quando não há nenhum.
5. **Given** um identificador inexistente, **When** o ingresso é consultado ou validado, **Then** a resposta "não encontrado" é a mesma de antes.

---

### User Story 2 - Esquema de banco preservado e migrações continuam valendo (Priority: P2)

O banco `cinema`, o schema `notificacao` e a migração versionada existente continuam sendo a fonte do esquema. Um ambiente já em operação passa a usar a nova persistência sem migração de dados e sem perda de informação; a criação do esquema continua sendo feita pelo container de migração, não pelo próprio serviço ao subir.

**Why this priority**: evita que a troca vire uma migração de dados arriscada e mantém o esquema compartilhado e versionado como hoje.

**Independent Test**: subir o serviço novo apontando para um banco criado pela migração atual e já populado com ingressos e avisos da versão anterior; verificar que são lidos, listados e validados normalmente.

**Acceptance Scenarios**:

1. **Given** um banco populado pela versão anterior, **When** o serviço com a nova persistência inicia, **Then** nenhuma alteração de esquema ou de dados é necessária e os registros existentes são tratados normalmente.
2. **Given** um banco vazio, **When** `docker compose up` roda, **Then** a migração existente cria o esquema e o serviço sobe saudável.
3. **Given** uma regra de integridade do banco (situação inválida, aviso de falha sem motivo, ingresso duplicado para a mesma reserva), **When** uma gravação a violaria, **Then** continua sendo recusada e o erro é tratado como antes.

---

### User Story 3 - Fronteira entre núcleo e infraestrutura mantida (Priority: P3)

Quem mantém o código continua encontrando o acesso a dados isolado em adaptadores. O domínio e os casos de uso seguem sem conhecer o GORM nem o driver de banco, e continuam testáveis sem banco.

**Why this priority**: preserva a regra de dependências apontando para dentro da arquitetura hexagonal dos serviços; é uma qualidade da troca, não um comportamento novo.

**Independent Test**: verificar que nenhum pacote do núcleo (domínio e casos de uso) importa a biblioteca nova nem o driver de banco, e rodar os testes unitários sem banco.

**Acceptance Scenarios**:

1. **Given** a troca concluída, **When** os imports do núcleo são inspecionados, **Then** não há dependência de GORM nem de driver de banco.
2. **Given** o núcleo do serviço, **When** os testes unitários rodam, **Then** continuam executando sem banco, sem rede e sem servidor.

---

### Edge Cases

- Entrega duplicada ou concorrente do mesmo pagamento: a unicidade por reserva garantida pelo banco continua sendo a barreira; nenhuma segunda linha é criada e o chamador recebe o ingresso original.
- Baixa concorrente do mesmo ingresso: a atualização condicional ("só se ainda válido") continua atômica; apenas uma vence.
- Banco indisponível na inicialização ou durante a operação: o serviço reporta a indisponibilidade como antes (verificação de saúde e erro ao chamador).
- Campos opcionais nulos (data de utilização de ingresso ainda válido, detalhes de aviso bem-sucedido): continuam gravados e lidos como ausentes, sem virar texto vazio ou data zero.
- Instantes de tempo: criação e utilização continuam preservando o mesmo instante (com fuso) ao ir e voltar do banco.
- Lista de ingressos vazia: continua devolvendo coleção vazia, não nula.
- Dados já existentes com valores aceitos pela versão anterior: continuam legíveis.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Toda leitura e escrita de dados do serviço de notificação MUST passar a ser feita por meio do GORM, substituindo o acesso direto ao banco usado hoje.
- **FR-002**: O comportamento externo do serviço (contrato REST, mensagens consumidas, códigos e mensagens de erro, ordenação e filtros da listagem) MUST permanecer idêntico ao anterior à troca.
- **FR-003**: A emissão do ingresso MUST continuar idempotente por reserva: no máximo um ingresso por reserva, mesmo sob entrega duplicada ou concorrente, com a mesma indicação ao chamador de que o ingresso é novo ou já existia.
- **FR-004**: A baixa do ingresso MUST continuar atômica e condicional à situação "válido", de modo que, sob concorrência, exatamente uma baixa tenha sucesso e a data de utilização seja gravada junto.
- **FR-005**: O registro de aviso MUST continuar gravando canal, situação, detalhes opcionais e instante de envio, vinculado ao ingresso.
- **FR-006**: O esquema do banco e sua migração versionada MUST permanecer a única fonte de criação e evolução do esquema; o serviço MUST NOT criar ou alterar tabelas por conta própria ao iniciar.
- **FR-007**: A conexão MUST continuar restrita ao schema `notificacao` do banco compartilhado e a tabela de controle de migrações MUST continuar fixada nesse schema.
- **FR-008**: O serviço MUST funcionar sobre dados já existentes, sem exigir migração de dados nem janela de manutenção.
- **FR-009**: A verificação de saúde MUST continuar refletindo a disponibilidade do banco.
- **FR-010**: O núcleo (domínio e casos de uso) MUST NOT importar o GORM nem o driver de banco; as portas de persistência consumidas pelos casos de uso MUST permanecer as mesmas.
- **FR-011**: O pool de conexões e o encerramento ordenado do serviço MUST continuar funcionando de forma equivalente.
- **FR-012**: A configuração do serviço (variável de conexão com o banco) MUST permanecer a mesma para quem opera o ambiente.

### Key Entities

- **Ingresso emitido**: ingresso digital ligado a uma reserva e a uma pessoa, com código único, situação (válido, utilizado, cancelado) e instantes de criação e de utilização.
- **Registro de notificação (aviso)**: histórico de cada tentativa de avisar a pessoa, com canal, resultado (enviado ou falha), detalhes e instante, vinculado a um ingresso.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% dos testes automatizados existentes do serviço (unitários e de integração) passam sem enfraquecer nenhuma asserção.
- **SC-002**: Em teste de concorrência com N entregas simultâneas do mesmo pagamento, exatamente um ingresso existe ao final e zero erros chegam ao chamador; com N baixas simultâneas do mesmo ingresso, exatamente uma é aceita.
- **SC-003**: O fluxo ponta a ponta do quickstart (pagamento → ingresso → consulta → validação) produz resultados idênticos aos da versão anterior.
- **SC-004**: Um banco populado pela versão anterior é operado pelo serviço novo com zero scripts de migração de dados.
- **SC-005**: Nenhum pacote do núcleo depende da nova biblioteca de persistência nem do driver de banco (verificado automaticamente).
- **SC-006**: A latência típica de emissão, consulta e listagem não piora de forma perceptível em relação à versão anterior (mesma ordem de grandeza em ambiente local).

## Assumptions

- A escolha do GORM é uma decisão do mantenedor, não está em discussão; o escopo é o serviço `notificacao` apenas (catalogo, estoque e pagamento não mudam). A mesma troca no `estoque` está em especificação separada (`estoque/specs/002-persistencia-gorm`).
- O GORM atuará sobre o mesmo PostgreSQL e o mesmo schema `notificacao`; a migração SQL existente continua como fonte do esquema (sem auto-migração do GORM).
- Os contratos versionados (OpenAPI) e as mensagens AMQP não mudam, portanto não há nova versão de contrato.
- RabbitMQ, a segurança (JWT e chave de API) e o envio efetivo do aviso (adaptador de notificador) não são afetados.
- Se algum comportamento exigido (inserção que ignora conflito e devolve a linha criada, atualização condicional com contagem de linhas afetadas) não puder ser expresso por meios nativos do GORM, é aceitável usar o mecanismo de consulta bruta do próprio GORM dentro do adaptador, sem alterar o núcleo.
- Hoje o serviço não possui verificação automática da fronteira do núcleo; o plano a cria com uma regra do linter de dependências (`depguard`), como o `catalogo`, sem código novo de teste.
- Divergências entre o código atual e a spec 001 são tratadas conforme o princípio de que o código é a fonte da verdade; qualquer divergência encontrada será levada ao mantenedor, não corrigida unilateralmente.
- O trabalho é commitado direto na `master`, sem branch de feature.
