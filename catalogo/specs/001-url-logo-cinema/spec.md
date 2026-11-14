# Feature Specification: URL do logo do cinema

**Feature Branch**: `001-url-logo-cinema` (trabalho direto na `master`, conforme convenção do workspace)

**Created**: 2026-11-14

**Status**: Draft

**Input**: User description: "crie uma coluna na entidade cinema para guardar a url da log do cinema."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Informar o logo ao cadastrar ou editar um cinema (Priority: P1)

Um administrador do catálogo cadastra um cinema (ou edita um existente) e informa, opcionalmente, o endereço (URL) da imagem do logo. O cinema passa a guardar essa informação junto com os demais dados.

**Why this priority**: Sem poder gravar a URL, nenhuma outra parte do sistema consegue usar o logo. É o núcleo da funcionalidade.

**Independent Test**: Cadastrar um cinema com uma URL de logo, consultá-lo e verificar que a mesma URL é devolvida; editar a URL e verificar a atualização.

**Acceptance Scenarios**:

1. **Given** um administrador autorizado, **When** cadastra um cinema informando uma URL de logo válida, **Then** o cinema é criado e a consulta devolve essa URL.
2. **Given** um cinema existente com logo, **When** o administrador altera a URL do logo, **Then** a consulta passa a devolver a nova URL.
3. **Given** um cinema existente com logo, **When** o administrador remove a URL (campo vazio), **Then** o cinema fica sem logo.
4. **Given** o administrador informa uma URL malformada ou acima do tamanho máximo, **When** tenta salvar, **Then** o cadastro é rejeitado com mensagem de validação clara e nada é alterado.

---

### User Story 2 - Consultar o logo nas listagens e no detalhe do cinema (Priority: P2)

Quem consome o catálogo (por exemplo, o frontend) recebe a URL do logo ao listar ou detalhar cinemas, podendo exibi-lo.

**Why this priority**: É o que dá valor visível à informação gravada, mas depende da Story 1.

**Independent Test**: Listar e detalhar cinemas e verificar que a URL do logo aparece quando existe e vem ausente/vazia quando não existe.

**Acceptance Scenarios**:

1. **Given** um cinema com logo, **When** é consultado individualmente ou em listagem, **Then** a resposta inclui a URL do logo.
2. **Given** um cinema sem logo, **When** é consultado, **Then** a resposta indica ausência de logo sem erro.

---

### User Story 3 - Cinemas já existentes continuam funcionando (Priority: P1)

Cinemas cadastrados antes da mudança permanecem intactos, sem logo, e continuam podendo ser consultados e editados normalmente.

**Why this priority**: Evita regressão em dados em produção.

**Independent Test**: Com cinemas pré-existentes, aplicar a mudança e verificar que todos continuam acessíveis, com logo vazio.

**Acceptance Scenarios**:

1. **Given** cinemas cadastrados antes da mudança, **When** a mudança é aplicada, **Then** todos permanecem com seus dados e sem logo.
2. **Given** um cliente que não envia o campo de logo, **When** cadastra ou edita um cinema, **Then** a operação funciona como antes.

### Edge Cases

- URL com espaços nas pontas: deve ser aparada antes de validar/gravar.
- URL vazia ou apenas espaços: tratada como "sem logo".
- URL com esquema diferente de http/https (ex.: `javascript:`): rejeitada.
- URL sem host (ex.: `https://`) ou com espaços internos: rejeitada.
- URL muito longa (acima do limite): rejeitada com erro de validação.
- Edição do cinema é substituição completa (como já ocorre com os demais campos): omitir o logo na edição remove o logo existente; quem quiser mantê-lo deve reenviá-lo.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: O cinema MUST poder guardar uma URL de logo, opcional.
- **FR-002**: O sistema MUST aceitar a URL de logo no cadastro e na edição de cinema.
- **FR-003**: O sistema MUST aparar espaços nas pontas e tratar valor vazio como ausência de logo.
- **FR-004**: O sistema MUST rejeitar URLs malformadas, com esquema diferente de http/https ou acima de 2048 caracteres, com erro de validação.
- **FR-005**: O sistema MUST devolver a URL do logo (quando houver) nas consultas individuais e nas listagens de cinemas.
- **FR-006**: Cinemas existentes MUST permanecer válidos, sem logo, após a mudança, e a mudança MUST ser reversível.
- **FR-007**: A documentação do contrato público da API de cinemas MUST refletir o novo campo.
- **FR-008**: O sistema MUST NOT armazenar nem processar o arquivo de imagem; apenas o endereço.

### Key Entities

- **Cinema**: unidade exibidora (nome, cidade, estado, endereço, situação ativo). Passa a ter também a **URL do logo**, opcional, que aponta para uma imagem hospedada externamente.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% dos cinemas cadastrados ou editados com URL válida devolvem exatamente essa URL nas consultas seguintes.
- **SC-002**: 100% dos cinemas anteriores à mudança continuam acessíveis e editáveis, sem perda de dados.
- **SC-003**: 100% das tentativas com URL inválida são rejeitadas com mensagem que indica o campo e o motivo.
- **SC-004**: Clientes existentes que não conhecem o novo campo continuam operando sem alterações.

## Assumptions

- "log" na descrição significa "logo" (imagem de marca do cinema).
- O campo é opcional; nem todo cinema terá logo.
- O sistema apenas guarda o endereço; a hospedagem e a disponibilidade da imagem são externas e não são verificadas.
- Limite de 2048 caracteres para a URL e apenas esquemas http/https são aceitos (padrões usuais).
- Mudança restrita ao serviço de catálogo; exibição do logo no frontend fica fora do escopo desta feature.
- Quem já pode cadastrar/editar cinemas continua sendo quem pode informar o logo.
