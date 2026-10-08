# Feature Specification: Persistência do Estoque via GORM

**Feature Branch**: _não aplicável — o trabalho é commitado direto na `master` (regra do mantenedor)_

**Created**: 2026-10-07

**Status**: Draft

**Input**: User description: "Modifique o serviço "estoque" de modo que ele passe a usar GORM como meio de persistência."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Comportamento de negócio idêntico após a troca (Priority: P1)

O mantenedor troca o meio de persistência do Servico-Estoque para o GORM, mantendo todo o comportamento observável: bloquear poltronas de forma exclusiva, confirmar e cancelar reservas, expirar reservas não pagas, provisionar e cancelar sessões e consultar o mapa de poltronas. Quem consome o serviço (catalogo via gRPC, pagamento e notificacao via mensagens) não percebe diferença.

**Why this priority**: sem equivalência funcional a troca não tem valor; é o núcleo da entrega e o que torna a migração segura.

**Independent Test**: executar a suíte existente de testes do serviço (unitários e de integração) contra a nova persistência, sem alterar as asserções, e rodar o fluxo de quickstart (publicar sessão, reservar, publicar pagamento) verificando os mesmos resultados de antes.

**Acceptance Scenarios**:

1. **Given** uma sessão provisionada com poltronas livres, **When** uma reserva é solicitada para um conjunto de poltronas, **Then** todas ficam bloqueadas, a reserva e o fato `reserva.criada` são gravados juntos, e a resposta é igual à anterior à troca.
2. **Given** duas solicitações concorrentes pelas mesmas poltronas, **When** ambas executam ao mesmo tempo, **Then** exatamente uma tem sucesso e a outra recebe a mesma recusa de indisponibilidade de antes; nunca há venda dupla.
3. **Given** uma reserva pendente, **When** chega `pagamento.sucesso` ou `pagamento.falhou` (inclusive repetido), **Then** a reserva é confirmada ou liberada uma única vez, com os mesmos efeitos de antes.
4. **Given** uma reserva não paga, **When** passam 10 minutos, **Then** ela expira e as poltronas voltam a ficar livres.

---

### User Story 2 - Esquema de banco preservado e migrações continuam valendo (Priority: P2)

O banco `cinema`, o schema `estoque` e as migrações versionadas existentes continuam sendo a fonte do esquema. Um ambiente já em operação passa a usar a nova persistência sem migração de dados e sem perda de informação; a criação do esquema continua sendo feita pelos containers de migração, não pelo próprio serviço ao subir.

**Why this priority**: evita que a troca vire uma migração de dados arriscada e mantém o esquema compartilhado e versionado como hoje.

**Independent Test**: subir o serviço novo apontando para um banco criado pelas migrações atuais e já populado com dados da versão anterior; verificar que reservas, poltronas e fatos pendentes existentes são lidos e processados normalmente.

**Acceptance Scenarios**:

1. **Given** um banco populado pela versão anterior, **When** o serviço com a nova persistência inicia, **Then** nenhuma alteração de esquema ou de dados é necessária e os registros existentes são tratados normalmente.
2. **Given** um banco vazio, **When** `docker compose up` roda, **Then** as migrações existentes criam o esquema e o serviço sobe saudável.

---

### User Story 3 - Fronteira entre núcleo e infraestrutura mantida (Priority: P3)

Quem mantém o código continua encontrando o acesso a dados isolado em adaptadores. O domínio e os casos de uso seguem sem conhecer o GORM, e o teste de arquitetura continua passando.

**Why this priority**: preserva a regra de dependências apontando para dentro da constituição do serviço; é uma qualidade da troca, não um comportamento novo.

**Independent Test**: rodar o teste de arquitetura e verificar que nenhum pacote do núcleo importa a biblioteca nova nem o driver de banco.

**Acceptance Scenarios**:

1. **Given** a troca concluída, **When** o teste de arquitetura roda, **Then** ele passa sem alterar suas regras.
2. **Given** o núcleo do serviço, **When** os testes unitários rodam, **Then** continuam executando sem banco, sem rede e sem servidor.

---

### Edge Cases

- Duas transações disputando as mesmas poltronas: a trava deve continuar não-bloqueante e em ordem determinística, de forma que a perdedora falhe de imediato em vez de esperar ou causar impasse.
- Falha no meio da transação (ex.: erro ao gravar o fato de saída): reserva, vínculos, bloqueio e fato devem ser desfeitos juntos; nada fica pela metade. Isto MUST ter teste automatizado com banco real (ver tarefa de rollback em tasks.md).
- Banco indisponível na inicialização ou durante a operação: o serviço reporta a indisponibilidade como antes (verificação de saúde e erro ao chamador).
- Mensagem repetida ou fora de ordem: continua descartada de forma idempotente, sem efeito duplicado.
- Reservas expiradas encontradas em lote após o serviço ficar fora do ar: continuam sendo liberadas sem duplicidade.
- Dados já existentes com valores nulos ou formatos aceitos pela versão anterior: continuam legíveis.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Toda leitura e escrita de dados do serviço de estoque MUST passar a ser feita por meio do GORM, substituindo o acesso direto ao banco usado hoje.
- **FR-002**: O comportamento externo do serviço (contrato gRPC, contrato REST, mensagens consumidas e publicadas, códigos e mensagens de erro) MUST permanecer idêntico ao anterior à troca.
- **FR-003**: O bloqueio de poltronas MUST continuar exclusivo e protegido contra venda dupla sob concorrência, com travas de linha não-bloqueantes e ordem determinística de aquisição, no mesmo modelo de exclusão de hoje.
- **FR-004**: A gravação da reserva, o bloqueio das poltronas e o fato de saída (`reserva.criada`) MUST continuar ocorrendo na mesma transação atômica.
- **FR-005**: Os consumidores de mensagens MUST continuar idempotentes (registro de mensagens processadas e guardas de estado), com os mesmos resultados diante de repetição.
- **FR-006**: O esquema do banco e suas migrações versionadas MUST permanecer a única fonte de criação e evolução do esquema; o serviço MUST NOT criar ou alterar tabelas por conta própria ao iniciar.
- **FR-007**: A conexão MUST continuar restrita ao schema `estoque` do banco compartilhado e a tabela de controle de migrações MUST continuar fixada nesse schema.
- **FR-008**: O serviço MUST funcionar sobre dados já existentes, sem exigir migração de dados nem janela de manutenção.
- **FR-009**: A verificação de saúde MUST continuar refletindo a disponibilidade do banco.
- **FR-010**: O núcleo (domínio e casos de uso) MUST NOT importar o GORM nem o driver de banco; o teste de arquitetura existente MUST continuar passando.
- **FR-011**: O pool de conexões, a expiração por ociosidade e o encerramento ordenado do serviço MUST continuar funcionando de forma equivalente.
- **FR-012**: A configuração do serviço (variável de conexão com o banco) MUST permanecer a mesma para quem opera o ambiente.

### Key Entities

- **Poltrona**: assento de uma sessão, com estado (livre, bloqueada, vendida) e vínculo opcional a uma reserva.
- **Reserva**: bloqueio temporário de poltronas, com estado, valor e prazo de expiração.
- **Fato de saída (outbox)**: mensagem a ser publicada, gravada na mesma transação da reserva.
- **Mensagem processada**: registro que garante idempotência no consumo.
- **Sessão provisionada**: conjunto de poltronas criado a partir de `sessao.criada`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% dos testes automatizados existentes do serviço (unitários e de integração) passam sem enfraquecer nenhuma asserção.
- **SC-002**: Em teste de concorrência com N solicitações simultâneas pelas mesmas poltronas, exatamente uma é aceita e zero poltronas ficam vendidas em duplicidade.
- **SC-003**: O fluxo ponta a ponta do quickstart (sessão → reserva → pagamento → confirmação/expiração) produz resultados idênticos aos da versão anterior.
- **SC-004**: Um banco populado pela versão anterior é operado pelo serviço novo com zero scripts de migração de dados.
- **SC-005**: Nenhum pacote do núcleo depende da nova biblioteca de persistência (verificado automaticamente pelo teste de arquitetura).
- **SC-006**: O p99 do tempo de resposta de uma solicitação de reserva continua ≤ 100 ms e não fica mais de 20% acima do p99 medido na versão anterior, no mesmo ambiente e com a mesma carga.

## Assumptions

- A escolha do GORM é uma decisão do mantenedor, não está em discussão; o escopo é o serviço `estoque` apenas (catalogo, pagamento e notificacao não mudam).
- O GORM atuará sobre o mesmo PostgreSQL e o mesmo schema `estoque`; as migrações SQL existentes continuam como fonte do esquema (sem auto-migração do GORM).
- Os contratos versionados (OpenAPI e proto) e as mensagens AMQP não mudam, portanto não há nova versão de contrato.
- Redis (índice de expiração), RabbitMQ e a segurança (JWT, mTLS) não são afetados.
- Se algum recurso de travamento exigido (trava não-bloqueante, ordem determinística) não puder ser expresso por meios nativos do GORM, é aceitável usar o mecanismo de consulta bruta do próprio GORM dentro do adaptador, sem alterar o núcleo.
- Divergências entre o código atual e a spec 001 são tratadas conforme o princípio de que o código é a fonte da verdade; qualquer divergência encontrada será levada ao mantenedor, não corrigida unilateralmente.
- O trabalho é commitado direto na `master`, sem branch de feature.
