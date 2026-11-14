# Feature Specification: Página de detalhes do cinema (salas, sessões e filmes)

**Feature Branch**: `002-detalhe-cinema` (nenhuma branch dedicada; o trabalho segue direto na `master`)

**Created**: 2026-11-14

**Status**: Draft

**Input**: User description: "Quando o usuário clicar no card de um cinema, o usuário deve ser direcionado pra uma página que mostre os dados de sala+sessão+filme que estejam vinculados ao cinema clicado."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Abrir um cinema e ver sua programação (Priority: P1)

Na aba de cinemas da página inicial, o visitante clica no card de um cinema e é levado a uma página própria daquele cinema. A página mostra a identificação do cinema (nome, cidade/estado, endereço, logo) e a programação vinculada a ele: cada sessão apresentada junto com o filme que será exibido e a sala onde ocorrerá.

**Why this priority**: É o núcleo do pedido — transforma o card em porta de entrada para saber "o que está passando neste cinema". Sozinha já entrega valor completo (MVP).

**Independent Test**: Na aba de cinemas, clicar no card de um cinema que tenha sessões e verificar que a página aberta exibe somente as sessões desse cinema, cada uma com dados do filme e da sala.

**Acceptance Scenarios**:

1. **Given** a aba de cinemas exibida, **When** o visitante clica em qualquer área do card de um cinema, **Then** é levado à página desse cinema.
2. **Given** a página de um cinema com sessões cadastradas, **When** ela termina de carregar, **Then** são exibidas as sessões vinculadas a esse cinema, cada uma com data e horário, preço, a sala (nome/identificação) e o filme (título, gênero, duração, classificação etária e imagem quando houver).
3. **Given** que existem sessões de outros cinemas, **When** o visitante abre a página de um cinema, **Then** nenhuma sessão, sala ou filme exclusivo de outro cinema aparece.
4. **Given** a página de um cinema aberta, **When** o visitante usa o comando de voltar (da página ou do navegador), **Then** retorna à aba de cinemas da página inicial, com os filtros que havia aplicado ainda preservados.

---

### User Story 2 - Ver as salas do cinema (Priority: P2)

Além da programação, a página lista as salas do cinema (nome e características disponíveis), de modo que o visitante saiba a estrutura da unidade, inclusive salas que no momento não têm sessões.

**Why this priority**: Atende o termo "sala" do pedido de forma independente da programação, mas a página já é útil apenas com a User Story 1.

**Independent Test**: Abrir um cinema com várias salas, algumas sem sessões, e confirmar que todas as salas ativas aparecem e que as com sessões mostram as respectivas sessões.

**Acceptance Scenarios**:

1. **Given** um cinema com salas cadastradas, **When** o visitante abre sua página, **Then** todas as salas ativas desse cinema são listadas.
2. **Given** uma sala sem sessões, **When** a página é exibida, **Then** a sala aparece indicando que não há sessões programadas.

---

### User Story 3 - Organizar a programação por filme e por dia (Priority: P3)

A programação é apresentada agrupada por filme (cada filme com seus horários e salas) e as sessões são ordenadas cronologicamente. O visitante pode restringir a visualização a um dia específico.

**Why this priority**: Facilita a leitura quando há muitas sessões, mas não é indispensável para entregar o valor básico.

**Independent Test**: Abrir um cinema com vários filmes e dias de sessões, verificar o agrupamento por filme e a ordem de horários, e escolher um dia para ver só as sessões dele.

**Acceptance Scenarios**:

1. **Given** um cinema com sessões de vários filmes, **When** a página é exibida, **Then** cada filme aparece uma única vez com todas as suas sessões listadas em ordem de data e horário.
2. **Given** a programação exibida, **When** o visitante escolhe um dia, **Then** apenas as sessões daquele dia permanecem; ao limpar a escolha, voltam todas.

---

### Edge Cases

- Cinema sem nenhuma sessão de filme em cartaz: a página exibe os dados do cinema (e suas salas) e uma mensagem clara de que não há sessões programadas.
- Cinema sem salas cadastradas: a seção de salas informa que não há salas.
- Endereço da página aberto diretamente (link compartilhado, recarregar) para um cinema existente: a página abre normalmente, sem depender de ter passado pela aba de cinemas.
- Identificador de cinema inexistente ou cinema inativo: a página informa que o cinema não foi encontrado e oferece caminho de volta à lista de cinemas.
- Falha ao consultar os dados: mensagem de erro amigável com opção de tentar novamente, sem tela em branco.
- Carregamento em andamento: indicador de carregamento visível.
- Filme sem imagem ou cinema sem logo: usa o mesmo marcador visual padrão já adotado nos cards.
- Sessões já encerradas ou canceladas: não são apresentadas como opção de compra disponível (ver Assumptions).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: O card de cada cinema na aba de cinemas MUST ser clicável como um todo e MUST levar o visitante à página daquele cinema, podendo também ser acionado por teclado.
- **FR-002**: Cada cinema MUST ter um endereço de página próprio e estável, que possa ser compartilhado, favoritado e recarregado.
- **FR-003**: A página MUST exibir a identificação do cinema: nome, cidade, estado, endereço e logo (ou marcador padrão).
- **FR-004**: A página MUST exibir as sessões vinculadas ao cinema, e somente a ele, cada uma com data, horário e preço.
- **FR-013**: A página MUST exibir apenas filmes em cartaz: sessões de filmes que estejam "em breve" ou "fora de cartaz" não aparecem, e esses filmes não surgem na programação.
- **FR-005**: Cada sessão exibida MUST apresentar o filme correspondente (título, gênero, duração, classificação etária e imagem quando houver) e a sala onde ocorre.
- **FR-006**: A página MUST listar as salas ativas do cinema, inclusive as sem sessões.
- **FR-007**: A programação MUST ser agrupada por filme, com as sessões de cada filme em ordem cronológica.
- **FR-008**: O visitante MUST poder filtrar a programação por dia.
- **FR-009**: A página MUST oferecer forma explícita de voltar à lista de cinemas, e o voltar do navegador MUST retornar à aba de cinemas preservando os filtros aplicados.
- **FR-010**: A página MUST tratar e comunicar os estados de carregamento, lista vazia, cinema não encontrado e erro de consulta (com opção de tentar novamente).
- **FR-011**: A página MUST ser acessível a visitantes sem login, como a página inicial.
- **FR-012**: Os textos exibidos MUST estar em português do Brasil e preços/datas MUST seguir o formato brasileiro.

- **FR-014**: Cada filme da programação MUST exibir um botão "Comprar Ingressos", ponto de entrada do futuro fluxo de compra.

### Key Entities *(include if feature involves data)*

- **Cinema**: unidade exibidora (nome, cidade, estado, endereço, logo, ativo). É o ponto de partida; todas as demais informações da página são vinculadas a ele.
- **Sala**: espaço de exibição pertencente a exatamente um cinema (nome/identificação, características, ativa).
- **Sessão**: exibição de um filme em uma sala, em uma data e horário, com preço e situação; vincula sala, cinema e filme.
- **Filme**: obra exibida (título, gênero, duração, classificação etária, imagem); pode aparecer em várias sessões e em vários cinemas.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A partir da aba de cinemas, o visitante chega à programação de um cinema com um único clique.
- **SC-002**: Em 100% dos casos testados, todas as sessões exibidas pertencem ao cinema aberto e a filmes em cartaz, e nenhuma sessão elegível desse cinema deixa de aparecer.
- **SC-003**: Em até 3 segundos após abrir a página, em condições normais de rede, o visitante vê o conteúdo do cinema ou um indicador de carregamento seguido do resultado.
- **SC-004**: Um visitante identifica em menos de 30 segundos em que horários e salas um determinado filme passa naquele cinema.
- **SC-005**: Ao retornar da página do cinema, 100% das vezes a aba de cinemas reaparece com os filtros anteriormente aplicados.
- **SC-006**: Todos os estados excepcionais (sem sessões, sem salas, cinema inexistente, erro) mostram uma mensagem compreensível, sem tela em branco.

## Assumptions

- A página é pública e somente leitura; o fluxo de compra/reserva permanece fora de escopo. O botão "Comprar Ingressos" (FR-014) existe como ponto de entrada, mas ainda não leva a nenhum fluxo.
- Os dados de cinemas, salas, sessões e filmes já são fornecidos pelo catálogo e podem ser consultados filtrando por cinema; não é criada nenhuma nova regra de negócio no backend.
- Somente cinemas ativos têm página; apenas salas ativas são listadas.
- Sessões canceladas ou já encerradas não são exibidas como programação disponível; sessões futuras (e em andamento no dia) são exibidas.
- Somente filmes com situação "em cartaz" aparecem na página (FR-013); sessões de filmes em breve ou fora de cartaz ficam ocultas, e uma sala cujas únicas sessões sejam desses filmes é tratada como sem sessões.
- A feature reaproveita a identidade visual, os cartões e os estados de lista já existentes na página inicial (feature 001).
- O foco é uso em navegador de computador e celular, com layout responsivo.
