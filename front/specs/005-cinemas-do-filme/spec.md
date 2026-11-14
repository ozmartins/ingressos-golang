# Feature Specification: Cinemas com o filme em cartaz (a partir do card de filme)

**Feature Branch**: `005-cinemas-do-filme` (nenhuma branch dedicada; o trabalho segue direto na `master`)

**Created**: 2026-11-14

**Status**: Draft

**Input**: User description: "Quando o usuário clicar em um card da aba filmes, ele deve ser direcionado pra uma listagem de cinemas que estejam com o filmes em cartaz. A nova página deve permitir o filtro por UF e cidade. Cada item desta listagem de cinemes deve possuir o botão "Comprar Ingressos" que deve possuir o mesmo comportamento do outro botão "Comprar Ingressos"."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Abrir um filme e ver os cinemas que o exibem (Priority: P1)

Na aba de filmes da página inicial, o visitante clica no card de um filme e é levado a uma página própria daquele filme. A página mostra a identificação do filme (pôster, título, gênero, duração, classificação etária) e a listagem dos cinemas que estão com esse filme em cartaz, isto é, cinemas com ao menos uma sessão futura daquele filme. Cada cinema aparece com nome, cidade/UF, endereço e logo.

**Why this priority**: É o núcleo do pedido — transforma o card de filme em porta de entrada para "onde posso ver este filme". Sozinha já entrega valor completo (MVP).

**Independent Test**: Na aba de filmes, clicar no card de um filme que tenha sessões em mais de um cinema e verificar que a página aberta lista exatamente os cinemas com sessões desse filme, e nenhum outro.

**Acceptance Scenarios**:

1. **Given** a aba de filmes exibida, **When** o visitante clica em qualquer área do card de um filme, **Then** é levado à página desse filme.
2. **Given** a página de um filme com sessões em vários cinemas, **When** ela termina de carregar, **Then** são listados os cinemas ativos que têm sessões futuras desse filme, cada um com nome, cidade/UF, endereço e logo (ou marcador padrão).
3. **Given** cinemas que não exibem o filme, **When** a página do filme é aberta, **Then** esses cinemas não aparecem na listagem.
4. **Given** a página do filme aberta, **When** o visitante usa o comando de voltar (da página ou do navegador), **Then** retorna à aba de filmes da página inicial, com os filtros que havia aplicado ainda preservados.

---

### User Story 2 - Filtrar os cinemas por UF e cidade (Priority: P1)

Na página do filme, o visitante pode restringir a listagem escolhendo uma UF e, depois, uma cidade. As cidades oferecidas dependem da UF escolhida e apenas UFs e cidades que de fato têm cinema com o filme em cartaz são apresentadas. Há um comando para limpar os filtros.

**Why this priority**: Requisito explícito do pedido; sem ele a listagem fica impraticável quando o filme está em muitas cidades.

**Independent Test**: Abrir um filme em cartaz em cinemas de UFs e cidades distintas, escolher uma UF e confirmar que só restam cinemas dela; escolher uma cidade e confirmar que só restam cinemas da cidade; limpar e ver todos novamente.

**Acceptance Scenarios**:

1. **Given** a listagem exibida, **When** o visitante escolhe uma UF, **Then** só permanecem os cinemas daquela UF e a lista de cidades passa a conter apenas cidades dessa UF.
2. **Given** uma UF escolhida, **When** o visitante escolhe uma cidade, **Then** só permanecem os cinemas daquela cidade.
3. **Given** uma cidade escolhida, **When** o visitante troca a UF por outra que não contém essa cidade, **Then** a cidade é descartada e a listagem reflete apenas a nova UF.
4. **Given** filtros aplicados, **When** o visitante aciona "Limpar filtros", **Then** todos os cinemas com o filme em cartaz voltam a ser exibidos.
5. **Given** filtros que não correspondem a nenhum cinema, **When** a listagem é atualizada, **Then** uma mensagem informa que nenhum cinema atende ao filtro, com opção de limpá-lo.
6. **Given** os filtros aplicados, **When** a contagem de cinemas é exibida, **Then** ela reflete a quantidade de cinemas visíveis.

---

### User Story 3 - Comprar ingressos a partir de um cinema da listagem (Priority: P1)

Cada cinema da listagem tem um botão "Comprar Ingressos" que se comporta exatamente como o botão "Comprar Ingressos" já existente na página do cinema: se o visitante não estiver logado, é enviado ao login e, concluído o login, é levado à página de compra do filme naquele cinema; se já estiver logado, é levado diretamente à página de compra (seleção de assentos) do filme naquele cinema.

**Why this priority**: Requisito explícito e é o que fecha o fluxo de descoberta até a compra.

**Independent Test**: Deslogado, clicar em "Comprar Ingressos" num cinema da listagem, concluir o login e verificar que o destino é a compra do filme escolhido naquele cinema; repetir logado e verificar o acesso direto.

**Acceptance Scenarios**:

1. **Given** um visitante logado e a listagem exibida, **When** clica em "Comprar Ingressos" de um cinema, **Then** é levado à página de compra do filme daquele cinema, sem passar pelo login.
2. **Given** um visitante não logado, **When** clica em "Comprar Ingressos" de um cinema, **Then** é direcionado ao login e, concluído com sucesso, chega à página de compra do filme naquele cinema.
3. **Given** um visitante já logado que abriu a compra a partir da listagem (sem passar pelo login), **When** usa o comando de voltar, **Then** retorna à página do filme, e não à página inicial. Se a compra foi aberta após o login (que recarrega a aplicação) ou por link direto, o comando de voltar segue o comportamento já existente da página de compra (volta ao cinema).

---

### Edge Cases

- Filme sem nenhum cinema com sessão futura: a página exibe os dados do filme e uma mensagem clara de que não há cinemas com o filme em cartaz.
- Filme "em breve" ou "fora de cartaz": a página abre e informa que o filme não está em cartaz em nenhum cinema (sem cinemas listados), sem tela em branco.
- Endereço da página aberto diretamente (link compartilhado, recarregar) para um filme existente: abre normalmente, sem depender de ter passado pela aba de filmes.
- Identificador de filme inexistente: informa que o filme não foi encontrado e oferece caminho de volta à lista de filmes.
- Falha ao consultar os dados: mensagem de erro amigável com opção de tentar novamente.
- Carregamento em andamento: indicador de carregamento visível.
- Filme sem pôster ou cinema sem logo: usa o mesmo marcador visual padrão já adotado nos cards.
- Cinema inativo ou apenas com sessões já encerradas/canceladas: não aparece na listagem.
- Sessões de um cinema para esse filme todas esgotadas: o cinema continua listado (o impedimento de compra é tratado na página de compra).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: O card de cada filme na aba de filmes MUST ser clicável como um todo e MUST levar o visitante à página daquele filme, podendo também ser acionado por teclado.
- **FR-002**: Cada filme MUST ter um endereço de página próprio e estável, que possa ser compartilhado, favoritado e recarregado.
- **FR-003**: A página MUST exibir a identificação do filme: pôster (ou marcador padrão), título, gênero, duração e classificação etária.
- **FR-004**: A página MUST listar somente cinemas ativos que tenham ao menos uma sessão futura do filme (filme em cartaz no cinema), e nenhum outro.
- **FR-005**: Cada item da listagem MUST exibir nome, cidade, UF, endereço e logo (ou marcador padrão) do cinema.
- **FR-006**: A página MUST oferecer filtro por UF e filtro por cidade, ambos com opção "todas"; as opções MUST ser derivadas apenas dos cinemas que exibem o filme.
- **FR-007**: As cidades oferecidas MUST ser restritas à UF escolhida; ao trocar a UF, uma cidade incompatível MUST ser descartada.
- **FR-008**: A página MUST mostrar a contagem de cinemas visíveis, um comando "Limpar filtros" (desabilitado sem filtros ativos) e uma mensagem quando nenhum cinema atender ao filtro.
- **FR-009**: Cada item da listagem MUST ter o botão "Comprar Ingressos" com o mesmo comportamento do botão homônimo da página do cinema, incluindo a exigência de login e o retorno à compra após autenticar.
- **FR-010**: Ao acionar "Comprar Ingressos" de um cinema, a compra MUST ser aberta para o filme da página e para aquele cinema.
- **FR-011**: A página MUST tratar os estados de carregamento, erro (com "tentar novamente"), filme inexistente e filme sem cinemas em cartaz, sem tela em branco.
- **FR-012**: Ao voltar da página do filme, a aba de filmes da página inicial MUST reaparecer com aba e filtros preservados; ao voltar da compra aberta a partir da listagem, o visitante MUST retornar à página do filme.
- **FR-013**: A página MUST seguir o mesmo esquema de cores e os mesmos padrões visuais das demais páginas e ser utilizável em telas pequenas.

### Key Entities *(include if feature involves data)*

- **Filme**: título, gênero, duração, classificação etária, pôster e situação (em cartaz, em breve, fora de cartaz).
- **Cinema**: nome, cidade, UF, endereço, logo e indicador de atividade.
- **Sessão**: liga um filme a uma sala de um cinema em data/horário futuros; é ela que define "filme em cartaz no cinema".
- **Cinema com o filme em cartaz**: visão derivada — cinema ativo com ao menos uma sessão futura do filme escolhido.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A partir da aba de filmes, o visitante chega à listagem de cinemas do filme com um único clique.
- **SC-002**: 100% dos cinemas listados têm ao menos uma sessão futura do filme e 100% dos cinemas ativos com tal sessão são listados.
- **SC-003**: O visitante localiza um cinema de sua cidade aplicando UF e cidade em até 3 interações (escolher UF, escolher cidade, ler o resultado).
- **SC-004**: Do card do filme até a página de compra, o visitante logado precisa de no máximo 2 cliques (card do filme, "Comprar Ingressos").
- **SC-005**: 100% dos acessos a "Comprar Ingressos" por visitante deslogado passam pelo login e terminam na compra do filme e cinema escolhidos.
- **SC-006**: Em nenhum cenário (carregando, erro, vazio, inexistente) a página fica em branco ou sem orientação ao visitante.

## Assumptions

- "Filme em cartaz num cinema" significa que há ao menos uma sessão do filme naquele cinema com início futuro (sessões já iniciadas não contam); só cinemas ativos entram.
- "UF" corresponde ao campo de estado já usado no filtro da aba de cinemas, que será exibido com o rótulo "UF" nesta página.
- O comportamento do botão "Comprar Ingressos" de referência é o da página do cinema (exigir login e abrir a compra do filme naquele cinema), já entregue nas funcionalidades 003 e 004; esta feature o reutiliza sem alterá-lo.
- Filtros por UF e cidade são sobre os cinemas já carregados do filme, sem nova consulta; não há busca por texto neste escopo.
- Os dados vêm do catálogo existente (filmes, cinemas e sessões); não se prevê novo cadastro nem mudança de contrato nos serviços.
- Não há paginação visual da listagem; usa-se a mesma grade dos cards de cinema.
