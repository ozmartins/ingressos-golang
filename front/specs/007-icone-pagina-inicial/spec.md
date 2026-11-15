# Feature Specification: Ícone para retornar à página inicial em todas as páginas

**Feature Branch**: `007-icone-pagina-inicial` (nenhuma branch dedicada; o trabalho segue direto na `master`)

**Created**: 2026-11-14

**Status**: Draft

**Input**: User description: "adicione uma ícone em todas as páginas que permita ao usuário retornar à pagina inicial"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Voltar à página inicial com um clique, de qualquer página (Priority: P1)

Em qualquer página do site (detalhe de cinema, detalhe de filme, seleção de assentos, pagamento e confirmação), o visitante vê no mesmo lugar um ícone de "casa" que o leva de volta à página inicial (lista de filmes e cinemas). Hoje cada página só oferece um "Voltar" para a etapa anterior; não há um atalho único para recomeçar do início.

**Why this priority**: É o núcleo do pedido: um caminho previsível e curto para o início, sem depender de repetir "Voltar" várias vezes. Sozinho já entrega o valor completo (MVP).

**Independent Test**: Abrir cada página do site, acionar o ícone e conferir que a página inicial é exibida.

**Acceptance Scenarios**:

1. **Given** qualquer página do site que não seja a inicial, **When** o visitante aciona o ícone, **Then** a página inicial é exibida sem recarregar o site por inteiro.
2. **Given** qualquer página do site, **When** ela é aberta, **Then** o ícone está visível sem rolar a página e na mesma posição em todas elas.
3. **Given** o visitante na página inicial, **When** ele olha o cabeçalho, **Then** o ícone também está presente (mantém a posição consistente) e acioná-lo mantém o visitante na página inicial, sem recarregar e sem criar nova entrada no histórico.
4. **Given** o visitante que voltou ao início pelo ícone, **When** usa o botão "voltar" do navegador, **Then** retorna à página de onde veio.
5. **Given** o visitante em uma página qualquer, **When** aciona o ícone com Ctrl/Cmd ou botão do meio, **Then** a página inicial abre em nova aba, como num link comum.

---

### User Story 2 - Reconhecer e usar o ícone com facilidade e acessibilidade (Priority: P2)

O ícone é reconhecível como "início" (casa), tem área de toque confortável em celular, pode ser acionado pelo teclado, tem foco visível e um nome acessível ("Página inicial") para leitores de tela e dica ao passar o mouse.

**Why this priority**: Garante que o atalho seja de fato descoberto e utilizável por todos, mas a US1 já entrega a navegação.

**Independent Test**: Navegar só com o teclado até o ícone e acioná-lo; verificar com leitor de tela o nome anunciado; testar em tela estreita.

**Acceptance Scenarios**:

1. **Given** o ícone, **When** o visitante passa o mouse ou o foca pelo teclado, **Then** vê uma indicação clara de "Página inicial" e o foco fica visível.
2. **Given** um leitor de tela, **When** o ícone é alcançado, **Then** ele é anunciado como link "Página inicial".
3. **Given** uma tela estreita (celular), **When** a página é exibida, **Then** o ícone continua visível, com área de toque de pelo menos 44×44 px, sem sobrepor a barra de conta nem o conteúdo.
4. **Given** os temas claro e escuro do site, **When** o ícone é exibido, **Then** tem contraste legível em ambos.

---

### Edge Cases

- Páginas em estado de carregamento, erro ou "não encontrado" (cinema/filme inexistente, falha de rede): o ícone continua visível, pois é a saída mais simples.
- Visitante sem sessão ou logado: o ícone aparece igual nos dois casos, convivendo com os botões de conta.
- Página de pagamento com reserva em andamento: acionar o ícone sai da página sem concluir o pagamento (mesmo comportamento do "Voltar" atual); a reserva segue o seu tempo de expiração normal.
- Voltar ao início preserva o estado da página inicial já aberta (aba, filtros, busca), como acontece hoje com o "Voltar".
- Endereço desconhecido (rota inexistente) já exibe a página inicial; acionar o ícone ali leva ao endereço raiz do site (a URL passa a ser a da página inicial), exibindo a mesma página inicial sem perder o estado dela.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Todas as páginas do site MUST exibir um ícone, reconhecível como "início", que leva o visitante à página inicial.
- **FR-002**: O ícone MUST estar na mesma posição e com a mesma aparência em todas as páginas, visível ao abrir a página sem rolagem, inclusive nos estados de carregamento, erro e "não encontrado".
- **FR-003**: Acionar o ícone MUST exibir a página inicial, e o botão "voltar" do navegador MUST levar à página anterior.
- **FR-004**: O ícone MUST se comportar como link comum: abrir em nova aba com Ctrl/Cmd/botão do meio e ter destino copiável.
- **FR-005**: O ícone MUST ter nome acessível "Página inicial", ser alcançável e acionável pelo teclado e ter foco visível.
- **FR-006**: O ícone MUST ter área de toque de ao menos 44×44 px e não sobrepor a barra de conta nem o conteúdo, em telas estreitas e largas.
- **FR-007**: O ícone MUST ter contraste legível nos temas claro e escuro.
- **FR-008**: A adição do ícone MUST NOT alterar os "Voltar" já existentes nem o fluxo de compra e pagamento.
- **FR-009**: Qualquer página criada no futuro MUST herdar o ícone sem trabalho adicional por página.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Em 100% das páginas do site, o ícone de início está visível e leva à página inicial com um único clique.
- **SC-002**: A partir da página mais profunda do fluxo (pagamento), o visitante chega à página inicial em 1 ação, em vez das 3 ou mais de "Voltar" necessárias hoje.
- **SC-003**: 90% dos visitantes de teste encontram o caminho de volta ao início em até 5 segundos, em qualquer página.
- **SC-004**: O ícone é acionável só com teclado e anunciado corretamente por leitor de tela em todas as páginas.

## Assumptions

- "Todas as páginas" significa as páginas atuais do site do visitante: inicial, detalhe de cinema, detalhe de filme, compra (assentos), pagamento e confirmação. Telas externas de login/cadastro (identidade) estão fora do escopo.
- O ícone fica no cabeçalho do site, no canto oposto aos botões de conta (início à esquerda), em um único lugar compartilhado por todas as páginas.
- O ícone é apenas gráfico (casa), sem texto visível; o texto "Página inicial" aparece como dica e nome acessível.
- Na própria página inicial o ícone permanece visível por consistência e não recarrega nem descarta o estado da página.
- A página inicial é o endereço raiz do site, com a lista de filmes e cinemas.
- Não há mudança no conteúdo, nos dados ou nos serviços de back-end.
