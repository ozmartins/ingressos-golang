# Feature Specification: Persistência do Pagamento via GORM

**Feature Branch**: _não aplicável — o trabalho é commitado direto na `master` (regra do mantenedor)_

**Created**: 2026-10-07

**Status**: Draft

**Input**: User description: "Modifique a aplicação "pagamento" de modo que ela passe a usar GORM pra persistir e buscar dados do BD."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Comportamento de negócio idêntico após a troca (Priority: P1)

O mantenedor troca o meio de persistência do Servico-Pagamento para o GORM, mantendo todo o comportamento observável: registrar a transação de uma reserva anunciada (uma única vez por reserva), registrar a forma de pagamento escolhida, reivindicar e liberar a cobrança, gravar o desfecho (pago, recusado, cancelado, pendente de verificação), marcar o resultado como anunciado, e varrer periodicamente as transações aguardando cobrança, as esperas vencidas e os anúncios pendentes. Quem consome o serviço (clientes REST e a mensageria de reserva e pagamento) não percebe diferença.

**Why this priority**: sem equivalência funcional a troca não tem valor; é o núcleo da entrega e o que torna a migração segura.

**Independent Test**: executar a suíte existente de testes do serviço (unitários e de integração) contra a nova persistência, sem alterar as asserções, e repetir o fluxo do quickstart da spec 001 (publicar reserva, escolher a forma, acompanhar o resultado) verificando os mesmos resultados de antes.

**Acceptance Scenarios**:

1. **Given** uma reserva anunciada sem transação, **When** o fato é consumido, **Then** exatamente uma transação é gravada aguardando a forma de pagamento, igual ao comportamento anterior.
2. **Given** o mesmo anúncio de reserva entregue duas vezes (ou em paralelo), **When** ambos são processados, **Then** existe uma única transação para a reserva e o segundo processamento recebe a transação já existente, sem erro e sem duplicidade.
3. **Given** uma transação pronta para cobrança, **When** dois processos tentam reivindicá-la ao mesmo tempo, **Then** exatamente um a reivindica e o outro é informado de que não conseguiu; se a cobrança for liberada, ela volta a poder ser reivindicada.
4. **Given** uma transação em cobrança, **When** o desfecho é gravado, **Then** estado, código do gateway, motivo da falha e instante de pagamento ficam registrados como antes, sem violar as regras de integridade do banco.
5. **Given** transações em estados diversos, **When** as rotinas periódicas buscam as aguardando cobrança, as esperas vencidas e os anúncios pendentes, **Then** recebem o mesmo conjunto, na mesma ordem e respeitando o mesmo limite, que a versão anterior devolvia, e uma lista sem itens — não um erro — quando não há nenhuma.
6. **Given** um identificador de reserva inexistente, **When** a transação é consultada, **Then** a resposta "não encontrada" é a mesma de antes.

---

### User Story 2 - Esquema de banco preservado e migrações continuam valendo (Priority: P2)

O banco `cinema`, o schema `pagamento` e as migrações versionadas existentes continuam sendo a fonte do esquema. Um ambiente já em operação passa a usar a nova persistência sem migração de dados e sem perda de informação; a criação do esquema continua sendo feita pelo container de migração, não pelo próprio serviço ao subir.

**Why this priority**: evita que a troca vire uma migração de dados arriscada e mantém o esquema compartilhado e versionado como hoje.

**Independent Test**: subir o serviço novo apontando para um banco criado pelas migrações atuais e já populado com transações da versão anterior; verificar que são lidas, varridas e atualizadas normalmente.

**Acceptance Scenarios**:

1. **Given** um banco populado pela versão anterior, **When** o serviço com a nova persistência inicia, **Then** nenhuma alteração de esquema ou de dados é necessária e os registros existentes são tratados normalmente.
2. **Given** um banco vazio, **When** `docker compose up` roda, **Then** as migrações existentes criam o esquema e o serviço sobe saudável.
3. **Given** uma regra de integridade do banco (estado inválido, forma incoerente com o estado, valor não positivo, reserva duplicada), **When** uma gravação a violaria, **Then** continua sendo recusada e o erro é tratado como antes.

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

- Entrega duplicada ou concorrente do mesmo anúncio de reserva: a unicidade por reserva garantida pelo banco continua sendo a barreira; nenhuma segunda linha é criada e o chamador recebe a transação original.
- Reivindicação concorrente da mesma cobrança: a atualização condicional continua atômica; apenas um processo vence, e a cobrança não é emitida duas vezes.
- Banco indisponível na inicialização ou durante a operação: o serviço reporta a indisponibilidade como antes (verificação de saúde e erro ao chamador); as rotinas periódicas registram a falha e tentam de novo na próxima volta.
- Campos opcionais nulos (forma de pagamento antes da escolha, código do gateway, motivo da falha, instante de pagamento): continuam gravados e lidos como ausentes, sem virar texto vazio ou data zero.
- Valor monetário: continua preservando exatamente duas casas decimais ao ir e voltar do banco, sem arredondamento de ponto flutuante.
- Instantes de tempo: criação, atualização, expiração e pagamento continuam preservando o mesmo instante (com fuso) ao ir e voltar do banco.
- Varreduras sem resultado: continuam devolvendo uma lista sem itens, sem erro (hoje o código devolve `nil`; quem consome apenas itera, então a diferença entre `nil` e vazia não é observável).
- Dados já existentes com valores aceitos pela versão anterior (inclusive transações antigas já finalizadas): continuam legíveis.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Toda leitura e escrita de dados do serviço de pagamento MUST passar a ser feita por meio do GORM, substituindo o acesso direto ao banco usado hoje.
- **FR-002**: O comportamento externo do serviço (contrato REST, mensagens consumidas e publicadas, códigos e mensagens de erro, ordenação, limites e filtros das varreduras) MUST permanecer idêntico ao anterior à troca.
- **FR-003**: O registro da transação MUST continuar idempotente por reserva: no máximo uma transação por reserva, mesmo sob entrega duplicada ou concorrente, com a mesma indicação ao chamador de que a transação é nova ou já existia.
- **FR-004**: A reivindicação e a liberação da cobrança MUST continuar atômicas e condicionais ao estado da transação, de modo que, sob concorrência, exatamente uma reivindicação tenha sucesso.
- **FR-005**: A gravação do desfecho, da forma escolhida e do anúncio do resultado MUST continuar condicionada ao estado esperado da transação, com a mesma indicação ao chamador quando a condição não se verifica.
- **FR-006**: O esquema do banco e suas migrações versionadas MUST permanecer a única fonte de criação e evolução do esquema; o serviço MUST NOT criar ou alterar tabelas por conta própria ao iniciar.
- **FR-007**: A conexão MUST continuar restrita ao schema `pagamento` do banco compartilhado e a tabela de controle de migrações MUST continuar fixada nesse schema.
- **FR-008**: O serviço MUST funcionar sobre dados já existentes, sem exigir migração de dados nem janela de manutenção.
- **FR-009**: A verificação de saúde MUST continuar refletindo a disponibilidade do banco.
- **FR-010**: O núcleo (domínio e casos de uso) MUST NOT importar o GORM nem o driver de banco; as portas de persistência consumidas pelos casos de uso MUST permanecer as mesmas.
- **FR-011**: O pool de conexões e o encerramento ordenado do serviço MUST continuar funcionando de forma equivalente.
- **FR-012**: A configuração do serviço (variável de conexão com o banco) MUST permanecer a mesma para quem opera o ambiente.

### Key Entities

- **Transação de pagamento**: cobrança de uma reserva de ingressos, ligada a uma reserva (única) e a uma pessoa, com valor total, forma de pagamento (ausente até a escolha), estado (aguardando forma, processando, pago, recusado, cancelado, pendente de verificação), código do gateway, motivo da falha, indicadores de cobrança emitida e de resultado anunciado, e instantes de expiração, pagamento, criação e atualização.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% dos testes automatizados existentes do serviço (unitários e de integração) passam sem enfraquecer nenhuma asserção.
- **SC-002**: Em teste de concorrência com N anúncios simultâneos da mesma reserva, exatamente uma transação existe ao final e zero erros chegam ao chamador; com N reivindicações simultâneas da mesma cobrança, exatamente uma é aceita.
- **SC-003**: O fluxo ponta a ponta do quickstart (reserva → escolha da forma → cobrança → resultado anunciado) produz resultados idênticos aos da versão anterior.
- **SC-004**: Um banco populado pela versão anterior é operado pelo serviço novo com zero scripts de migração de dados.
- **SC-005**: Nenhum pacote do núcleo depende da nova biblioteca de persistência nem do driver de banco (verificado automaticamente).
- **SC-006**: A latência típica de registro, consulta, reivindicação e varredura não piora de forma perceptível em relação à versão anterior (mesma ordem de grandeza em ambiente local).

## Assumptions

- A escolha do GORM é uma decisão do mantenedor, não está em discussão; o escopo é o serviço `pagamento` apenas (catalogo, estoque e notificacao não mudam). A mesma troca já tem especificação nos outros serviços (`notificacao/specs/002-persistencia-gorm`, `estoque/specs/002-persistencia-gorm`) e serve de precedente de abordagem.
- O GORM atuará sobre o mesmo PostgreSQL e o mesmo schema `pagamento`; as migrações SQL existentes continuam como fonte do esquema (sem auto-migração do GORM).
- Os contratos versionados (OpenAPI) e as mensagens AMQP não mudam, portanto não há nova versão de contrato.
- RabbitMQ, a segurança (JWT) e o gateway de pagamento simulado não são afetados.
- Se algum comportamento exigido (inserção que ignora conflito e devolve a linha criada, atualização condicional com contagem de linhas afetadas) não puder ser expresso por meios nativos do GORM, é aceitável usar o mecanismo de consulta bruta do próprio GORM dentro do adaptador, sem alterar o núcleo.
- Hoje o serviço não possui verificação automática da fronteira do núcleo; o plano a cria com uma regra do linter de dependências (`depguard`), como o `catalogo`, sem código novo de teste.
- O ambiente de testes de integração hoje obtém o pool de conexões do adaptador; ele será ajustado ao novo ponto de abertura da conexão, sem enfraquecer asserções.
- Divergências entre o código atual e a spec 001 são tratadas conforme o princípio de que o código é a fonte da verdade; qualquer divergência encontrada será levada ao mantenedor, não corrigida unilateralmente.
- O trabalho é commitado direto na `master`, sem branch de feature.
