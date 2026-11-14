# Feature Specification: Página de Terminal Retrô

**Feature Branch**: `002-retro-terminal-page`

**Created**: 2026-11-14

**Status**: Draft

**Input**: User description: "Crie uma página que lembre um terminal antigo (daqueles usados em monitores de fósforo). Ele deve possuur um campo na inferir da página pra ditação de comandos e o restante da página deve mostra o resultado da execução do comando."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Digitar um comando e ver o resultado (Priority: P1)

Como visitante, quero ver uma página que se pareça com um terminal antigo de monitor de fósforo, com um campo de entrada de comandos fixo na parte inferior e, acima dele, uma área ocupando o restante da tela que mostra o resultado de cada comando executado.

**Why this priority**: É o núcleo da funcionalidade: entrada de comando embaixo, saída no restante da página.

**Independent Test**: Abrir a página, digitar um comando reconhecido no campo inferior e pressionar Enter; o comando digitado e o seu resultado aparecem na área de saída.

**Acceptance Scenarios**:

1. **Given** a página aberta, **When** ela termina de carregar, **Then** o campo de comandos está na parte inferior, já com foco, e a área de saída ocupa o restante da página.
2. **Given** o campo de comandos com foco, **When** o visitante digita um comando conhecido e pressiona Enter, **Then** o comando digitado e seu resultado são exibidos na área de saída e o campo é limpo.
3. **Given** um comando desconhecido, **When** o visitante o executa, **Then** a saída exibe uma mensagem de "comando não encontrado" indicando o comando digitado.
4. **Given** uma saída maior que a área visível, **When** novas linhas são adicionadas, **Then** a área rola automaticamente para mostrar a linha mais recente.

---

### User Story 2 - Aparência de monitor de fósforo (Priority: P2)

Como visitante, quero que a página evoque um monitor de fósforo (texto monocromático verde ou âmbar sobre fundo escuro, fonte monoespaçada, cursor piscante e efeito sutil de brilho/linhas de varredura), para que a experiência remeta a um terminal antigo.

**Why this priority**: Define a identidade visual pedida, mas o terminal funciona sem ela.

**Independent Test**: Abrir a página e verificar visualmente fonte monoespaçada, cor monocromática sobre fundo escuro, cursor piscante e o efeito de brilho/varredura sem prejuízo da leitura.

**Acceptance Scenarios**:

1. **Given** a página aberta, **When** o visitante a observa, **Then** todo o texto é monoespaçado e monocromático sobre fundo escuro, com um cursor piscante no campo de comandos.
2. **Given** um visitante com preferência de movimento reduzido, **When** a página abre, **Then** animações (piscar, varredura) ficam desativadas ou atenuadas.

---

### User Story 3 - Histórico e comandos de utilidade (Priority: P3)

Como visitante, quero recuperar comandos anteriores com as setas e limpar a tela, para usar o terminal de forma confortável.

**Why this priority**: Conforto de uso, não essencial ao fluxo principal.

**Independent Test**: Executar dois comandos, pressionar seta para cima e ver o último comando reaparecer no campo; executar o comando de limpar e ver a saída esvaziada.

**Acceptance Scenarios**:

1. **Given** comandos já executados, **When** o visitante pressiona seta para cima/baixo no campo, **Then** os comandos anteriores são percorridos no campo.
2. **Given** saída acumulada, **When** o visitante executa `clear`, **Then** a área de saída é esvaziada.

---

### Edge Cases

- Entrada vazia (apenas Enter): nenhuma saída de erro; apenas uma nova linha de prompt.
- Linhas de saída muito longas: quebram em múltiplas linhas sem rolagem horizontal.
- Grande volume de saída acumulada: a página permanece responsiva (limite máximo de linhas retidas).
- Telas pequenas (celular): o campo permanece visível e utilizável, com teclado aberto.
- Texto com caracteres especiais ou marcação (por exemplo `<b>`): é exibido literalmente, sem ser interpretado.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A página MUST ocupar toda a tela, com um campo de entrada de comandos fixo na parte inferior e a área de saída ocupando todo o espaço restante.
- **FR-002**: O campo de comandos MUST receber foco ao abrir a página e MUST recuperar o foco quando o visitante clicar em qualquer ponto da página.
- **FR-003**: Ao pressionar Enter, o sistema MUST exibir na área de saída o comando digitado (com prompt) seguido do seu resultado, e limpar o campo.
- **FR-004**: O sistema MUST oferecer um conjunto de comandos embutidos simulados, executados inteiramente na página, incluindo pelo menos: `help` (lista comandos), `clear`, `echo <texto>`, `date`, `whoami` e `about`.
- **FR-005**: Comandos desconhecidos MUST resultar em mensagem "comando não encontrado" com o nome digitado.
- **FR-006**: A área de saída MUST rolar automaticamente para a linha mais recente.
- **FR-007**: O sistema MUST manter histórico de comandos navegável com as setas para cima e para baixo.
- **FR-008**: A aparência MUST ser a de monitor de fósforo: fonte monoespaçada, texto monocromático sobre fundo escuro, cursor piscante e efeito sutil de brilho/linhas de varredura.
- **FR-009**: Animações e efeitos MUST ser reduzidos ou desativados quando o visitante preferir movimento reduzido.
- **FR-010**: O texto digitado e o exibido MUST ser tratados como texto puro, nunca interpretado como marcação.
- **FR-011**: A página MUST exibir uma mensagem de boas-vindas com a sugestão de digitar `help`.
- **FR-012**: O sistema MUST limitar a quantidade de linhas retidas na saída para manter a responsividade.

### Key Entities

- **Comando**: texto digitado pelo visitante; possui nome e argumentos.
- **Linha de saída**: entrada exibida na área de saída (eco do comando, resultado, erro ou mensagem do sistema).
- **Histórico**: sequência de comandos executados na sessão, usada para navegação com as setas.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Um visitante novo executa um comando com sucesso em menos de 30 segundos após abrir a página, sem instruções externas.
- **SC-002**: O resultado de qualquer comando embutido aparece em menos de 100 ms após o Enter.
- **SC-003**: Em 100% dos comandos executados, o comando digitado e seu resultado ficam visíveis na área de saída (rolagem automática).
- **SC-004**: Em uma avaliação com 5 pessoas, ao menos 4 identificam a página como "lembrando um terminal antigo" à primeira vista.
- **SC-005**: A página é utilizável (campo visível e digitável) em telas de 360 px a 1920 px de largura.
- **SC-006**: O texto mantém contraste suficiente para leitura (razão de contraste de ao menos 4,5:1) apesar dos efeitos visuais.

## Assumptions

- "Executar comandos" significa um conjunto fixo de comandos simulados no navegador; **não há execução de comandos reais no sistema operacional nem comunicação com os serviços de backend**.
- A página substitui a página vazia atual da aplicação em `web/` (rota única, sem navegação).
- Idioma da interface e das mensagens: português do Brasil.
- Cor padrão: verde fósforo; a escolha exata da tonalidade fica para o planejamento.
- O histórico vale apenas durante a sessão da página (não é persistido entre recargas).
- Fora de escopo: autenticação, multiusuário, sistema de arquivos simulado, execução de scripts e som.
