# Feature Specification: Página de seleção de assentos ("Comprar Ingressos")

**Feature Branch**: `003-selecao-assentos` (nenhuma branch dedicada; o trabalho segue direto na `master`)

**Created**: 2026-11-14

**Status**: Draft

**Input**: User description: "Modifique o botão "Comprar Ingressos" de modo que ele abra uma nova página. Use a [Image #1] como referência pra nova página (tela de seleção de assentos do ingresso.com: cabeçalho com cinema/sala, data e horário; mapa de assentos por fileira com disponíveis, ocupados, indisponíveis e acessíveis; indicação de "TELA"; painel "Resumo do pedido" com pôster, filme, cinema, sala, data/horário, itens, taxa e total; indicador de etapas "Selecione seus assentos"). A nova página não precisa ser idêntica à referência e deve usar o mesmo esquema de cores das demais páginas da aplicação. Se o usuário ainda não estiver logado, direcioná-lo à página de login antes de enviá-lo à nova página."

## Clarifications

### Session 2026-11-14

- Q: "Continuar" deve reservar de fato os assentos ou só selecionar? → A: Selecionar e reservar (reserva temporária no estoque, com confirmação e prazo); pagamento fica fora do escopo.
- Q: Como tratar poltronas do tipo namoradeira? → A: Seleção individual, com ícone distinto, sem regra de par.
- Q: Atualização da disponibilidade com a página aberta? → A: Apenas ao carregar/trocar de sessão e após conflito ao continuar; sem atualização periódica.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Abrir a página de compra estando logado (Priority: P1)

Na página de um cinema, o visitante já autenticado clica em "Comprar Ingressos" no card de um filme e é levado a uma página própria de compra, aberta na seleção de assentos de uma sessão daquele filme naquele cinema. A página mostra o cabeçalho da sessão (cinema, sala, dia e horário), o mapa de assentos da sala e o resumo do pedido.

**Why this priority**: É o núcleo do pedido — o botão hoje não faz nada. Sozinha, a abertura da página com o mapa e o resumo já entrega o valor mínimo.

**Independent Test**: Logado, clicar em "Comprar Ingressos" de um filme com sessões e verificar que abre a página de compra mostrando cinema, sala, dia/horário, o mapa de assentos e o resumo do pedido do filme certo.

**Acceptance Scenarios**:

1. **Given** um visitante logado na página de um cinema, **When** clica em "Comprar Ingressos" de um filme, **Then** é levado à página de compra daquele filme naquele cinema, sem passar pela tela de login.
2. **Given** a página de compra aberta, **When** ela termina de carregar, **Then** exibe cinema, sala, dia e horário da sessão, o mapa de assentos com a indicação da tela e o resumo do pedido com pôster, título, classificação/formato, cinema, sala, dia e horário.
3. **Given** um filme com várias sessões no cinema, **When** a página abre, **Then** vem selecionada a próxima sessão futura do filme e o visitante pode trocar para outra sessão do mesmo filme e cinema (dia/horário/sala) sem sair da página; o mapa e o resumo acompanham a troca.
4. **Given** a página de compra aberta, **When** o visitante usa o comando de voltar, **Then** retorna à página do cinema.

---

### User Story 2 - Entrar antes de comprar quando não está logado (Priority: P1)

O visitante não autenticado que clica em "Comprar Ingressos" é primeiro enviado ao login. Concluído o login, ele é levado automaticamente à página de compra do filme em que havia clicado, e não à página inicial.

**Why this priority**: Requisito explícito e marcado como importante; sem ele a página de compra ficaria acessível a visitantes anônimos.

**Independent Test**: Deslogado, clicar em "Comprar Ingressos", concluir o login e verificar que o destino final é a página de compra daquele filme/cinema.

**Acceptance Scenarios**:

1. **Given** um visitante não logado, **When** clica em "Comprar Ingressos", **Then** é direcionado à página de login antes de qualquer acesso à página de compra.
2. **Given** o login concluído com sucesso, **When** o visitante volta à aplicação, **Then** chega à página de compra do filme que havia escolhido.
3. **Given** um visitante não logado que abre diretamente o endereço da página de compra (link, favorito, recarga), **When** a página é requisitada, **Then** também é direcionado ao login e, depois, à própria página de compra.
4. **Given** um visitante que desiste do login, **When** retorna sem autenticar, **Then** permanece sem acesso à página de compra e pode voltar à navegação pública.

---

### User Story 3 - Escolher assentos e ver o resumo do pedido (Priority: P2)

No mapa, organizado por fileiras identificadas por letra, o visitante clica nos assentos livres para selecioná-los (e clica de novo para desmarcar). Assentos ocupados ou indisponíveis aparecem distintos e não podem ser escolhidos; assentos acessíveis aparecem sinalizados. O resumo do pedido mostra a quantidade de itens, os assentos escolhidos, a taxa e o total em reais, atualizados a cada alteração. Uma legenda explica cada tipo de assento.

**Why this priority**: É o objetivo da referência, mas a página já tem valor (US1) apenas mostrando o mapa; a interação completa vem em seguida.

**Independent Test**: Na página de compra, selecionar e desmarcar assentos livres, tentar clicar em um ocupado e verificar o mapa, a legenda e os totais do resumo em cada passo.

**Acceptance Scenarios**:

1. **Given** o mapa exibido, **When** o visitante clica em um assento livre, **Then** ele aparece como selecionado e o resumo mostra 1 item, o assento e o total atualizado (preço da sessão × quantidade, mais taxa quando houver).
2. **Given** um assento selecionado, **When** o visitante clica nele novamente, **Then** é desmarcado e o resumo volta ao valor anterior.
3. **Given** assentos ocupados ou indisponíveis, **When** o visitante tenta selecioná-los, **Then** nada acontece e eles permanecem visualmente distintos dos livres.
4. **Given** o limite de assentos por pedido atingido, **When** tenta marcar mais um, **Then** é informado do limite e a seleção não muda.
5. **Given** nenhum assento selecionado, **When** a página é exibida, **Then** o resumo mostra 0 itens e total R$ 0,00 e a ação de continuar está desabilitada.
6. **Given** a opção "exibir números dos assentos" ativada, **When** o mapa é exibido, **Then** cada assento mostra seu número; desativada, apenas a fileira é rotulada.
7. **Given** o mapa exibido, **When** ele é navegado por teclado ou leitor de tela, **Then** cada assento tem nome acessível (fileira, número, situação) e pode ser marcado/desmarcado pelo teclado.

---

### User Story 4 - Confirmar a escolha e reservar os assentos (Priority: P3)

Com ao menos um assento selecionado, o visitante aciona "Continuar" e o sistema reserva temporariamente os assentos escolhidos, informando o prazo da reserva. Se algum assento tiver sido tomado por outra pessoa nesse meio-tempo, o visitante é avisado, o mapa é atualizado e a seleção é ajustada.

**Why this priority**: Fecha o ciclo da seleção com a reserva já suportada pela plataforma; o pagamento e as demais etapas ficam fora deste escopo.

**Independent Test**: Selecionar assentos livres e continuar; verificar a confirmação com prazo. Repetir com um assento tomado por outro usuário e verificar o aviso e a atualização do mapa.

**Acceptance Scenarios**:

1. **Given** assentos selecionados, **When** o visitante aciona "Continuar", **Then** a reserva é criada e a página mostra confirmação com os assentos reservados e até quando a reserva vale.
2. **Given** um assento escolhido que foi reservado por outra pessoa, **When** o visitante aciona "Continuar", **Then** vê aviso claro de conflito, o mapa é atualizado e os demais assentos escolhidos continuam selecionados.
3. **Given** falha de comunicação ou sessão de login expirada, **When** o visitante aciona "Continuar", **Then** vê mensagem amigável (e, se for expiração do login, é levado a entrar de novo retornando à mesma página), sem perder a seleção.

---

### Edge Cases

- Filme sem sessões futuras no cinema: a página informa que não há sessões disponíveis e oferece voltar ao cinema.
- Endereço da página com cinema, filme ou sessão inexistente/inativo/encerrado: mensagem de "não encontrado" com caminho de volta.
- Sessão cancelada ou já iniciada/encerrada: assentos não podem ser selecionados e a página explica o motivo.
- Falha ao carregar o mapa ou os dados da sessão: mensagem de erro com opção de tentar novamente; carregamento mostra indicador visível.
- Sala sem mapa de assentos cadastrado (ou mapa vazio): mensagem informando que a seleção não está disponível para a sessão.
- Estado dos assentos mudou enquanto a página estava aberta: o mapa é atualizado ao recarregar/ao tentar continuar, sem permitir dupla venda.
- Filme sem pôster: usa o marcador visual padrão já adotado nas demais páginas.
- Telas estreitas (celular): o mapa continua utilizável (rolagem/zoom) e o resumo permanece acessível sem rolagem horizontal da página.
- Login com retorno: parâmetros de retorno inválidos levam à página inicial, nunca a erro.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: O botão "Comprar Ingressos" MUST abrir uma página própria de compra, com endereço estável (compartilhável, favoritável e recarregável) que identifique cinema, filme e sessão selecionada.
- **FR-002**: Se o visitante não estiver autenticado, o sistema MUST direcioná-lo ao login antes de exibir a página de compra, e MUST levá-lo de volta à página de compra pretendida após o login bem-sucedido. Isso vale também para acesso direto ao endereço da página.
- **FR-003**: Se o visitante já estiver autenticado, o sistema MUST abri-la diretamente, sem passar pelo login.
- **FR-004**: A página MUST exibir cabeçalho com cinema, sala, dia e horário da sessão, e um indicador da etapa atual ("Selecione seus assentos").
- **FR-005**: Quando o filme tiver várias sessões no cinema, a página MUST abrir na próxima sessão futura e MUST permitir trocar de sessão sem sair dela.
- **FR-006**: A página MUST exibir o mapa de assentos da sala, agrupado por fileiras identificadas por letra, com a indicação da posição da tela.
- **FR-007**: Cada assento MUST ser exibido em uma destas situações, distinguíveis visualmente e por texto acessível: livre, selecionado e ocupado/reservado (assento indisponível = sessão que não aceita reserva ou situação desconhecida, caso em que o mapa inteiro fica desabilitado com explicação); e por tipo: comum, acessível (PCD) e namoradeira (selecionada individualmente, sem regra de par). Uma legenda MUST explicar cada uma.
- **FR-008**: O visitante MUST poder selecionar e desmarcar assentos livres; assentos ocupados ou indisponíveis MUST NOT ser selecionáveis.
- **FR-009**: O sistema MUST respeitar o limite máximo de assentos por pedido definido pela plataforma e MUST informar o visitante ao atingi-lo.
- **FR-010**: A página MUST exibir o resumo do pedido com pôster, título, classificação etária, formato/idioma, cinema, sala, dia e horário, quantidade de itens, assentos escolhidos, taxa (quando houver) e total em reais, atualizados a cada seleção.
- **FR-011**: A página MUST oferecer a opção de exibir/ocultar os números dos assentos no mapa.
- **FR-012**: A ação "Continuar" MUST ficar desabilitada sem assentos selecionados e, ao ser acionada, MUST reservar temporariamente os assentos escolhidos e informar o prazo da reserva.
- **FR-018**: O mapa MUST ser carregado ao abrir a página e ao trocar de sessão, e recarregado após um conflito ao continuar; não há atualização periódica automática.
- **FR-013**: O sistema MUST tratar conflito de assentos (já tomados por outra pessoa) com aviso claro, atualização do mapa e preservação dos demais assentos escolhidos; MUST NOT permitir que o mesmo assento seja reservado para duas pessoas.
- **FR-014**: A página MUST tratar com mensagens amigáveis os estados de carregamento, erro (com tentar novamente), sessão/filme/cinema inexistente, sessão encerrada e ausência de sessões.
- **FR-015**: A página MUST seguir o mesmo esquema de cores, tipografia e estilo visual das demais páginas da aplicação (tema atual), sem copiar a identidade visual da referência.
- **FR-016**: A página MUST oferecer comando de voltar à página do cinema de origem.
- **FR-017**: A página MUST ser utilizável em telas de celular e desktop e acessível por teclado e leitor de tela.

### Key Entities *(include if feature involves data)*

- **Sessão**: exibição de um filme em uma sala, com dia, horário, preço, formato/idioma e situação (ativa, encerrada, cancelada).
- **Mapa de assentos**: conjunto de assentos da sala da sessão, organizados em fileiras (letra) e posições (número), com a situação de cada um na sessão.
- **Assento**: posição identificada por fileira e número, com tipo (comum, acessível, namoradeira) e situação (livre, ocupado/reservado, indisponível).
- **Seleção/Pedido em andamento**: assentos escolhidos pelo visitante nesta sessão, com quantidade, taxa e total.
- **Reserva temporária**: bloqueio dos assentos escolhidos, com prazo de validade, criado ao continuar.
- **Visitante/Usuário autenticado**: quem pode acessar a página de compra e fazer a reserva.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% dos cliques em "Comprar Ingressos" resultam em abertura da página de compra (logado) ou do login seguido da página de compra (não logado), sem cair na página inicial nem em tela em branco.
- **SC-002**: Um visitante logado chega da página do cinema à seleção de assentos de um filme em até 2 interações e a página fica utilizável em até 3 segundos em condições normais de rede.
- **SC-003**: Um visitante consegue escolher 2 assentos e chegar à confirmação da reserva em menos de 1 minuto, na primeira tentativa, em pelo menos 90% dos testes de usabilidade.
- **SC-004**: Em 100% das tentativas, nenhum assento ocupado ou indisponível pode ser selecionado e nenhum assento é reservado para duas pessoas.
- **SC-005**: Os totais do resumo do pedido estão corretos em 100% das combinações testadas de seleção/desmarcação.
- **SC-006**: A página é utilizável sem rolagem horizontal da página em telas de 360 px de largura e todas as ações principais são alcançáveis apenas pelo teclado.
- **SC-007**: Uma revisão visual não encontra cores fora da paleta das demais páginas da aplicação.

## Assumptions

- Terminologia: "assento" na interface e na documentação; "poltrona" apenas nos nomes técnicos da API do estoque.
- Se o endereço trouxer uma sessão inexistente, já passada ou de outro filme/cinema, a página usa a próxima sessão futura do filme.
- Fileiras exibidas com a letra A mais próxima da tela (embaixo) e a última no topo, como na referência.
- O botão fica no card de cada filme da página do cinema; como o botão é por filme e não por sessão, a página abre na próxima sessão futura do filme naquele cinema, com troca de sessão dentro da página.
- A referência (ingresso.com) serve de inspiração de layout (cabeçalho da sessão, mapa, resumo lateral, etapas); a página não precisa ser idêntica e não inclui recursos como zoom por controle deslizante, favoritos, tipos de ingresso/meia-entrada, lanches ou apelo promocional.
- O escopo cobre a etapa "Selecione seus assentos" e a reserva temporária; as etapas seguintes (ingressos/descontos, pagamento, confirmação) estão fora do escopo e serão tratadas em outras features.
- Login e sessão do usuário reutilizam o mecanismo de autenticação já existente na aplicação (botões de entrar/sair e retorno após login).
- A situação dos assentos e a reserva temporária usam as capacidades já existentes na plataforma (consulta de poltronas da sessão e bloqueio com prazo de expiração); a taxa de serviço é exibida apenas se a plataforma a informar, caso contrário aparece como R$ 0,00.
- O preço por assento é o preço da sessão; não há diferenciação de preço por tipo de assento.
- O limite de assentos por pedido é o já definido pela plataforma.
- Reuso do tema visual atual (cores, tipografia, componentes) das páginas inicial e de cinema.
