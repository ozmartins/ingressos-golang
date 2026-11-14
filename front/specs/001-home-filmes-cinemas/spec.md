# Feature Specification: Página inicial com abas de Filmes e Cinemas

**Feature Branch**: `001-home-filmes-cinemas` (nenhuma branch dedicada; o trabalho segue direto na `master`)

**Created**: 2026-11-14

**Status**: Draft

**Input**: User description: "Crie uma página inicial que contenha duas abas: Cinemas e Filmes. A primeira deve listar os filmes e a segunda os cinemas. Ambas as abas devem possuir filtros para facilitar a localização de um cinema ou filme respectivamente."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Explorar e filtrar filmes (Priority: P1)

Ao abrir o site, o visitante cai na página inicial já posicionado na primeira aba, que lista os filmes disponíveis. Ele pode restringir a lista por título, gênero, classificação etária e situação (em cartaz, em breve) para encontrar rapidamente o filme que procura.

**Why this priority**: É a primeira aba e a porta de entrada natural da compra de ingressos: quem vai ao cinema geralmente parte do filme. Sozinha já entrega uma página inicial útil (MVP).

**Independent Test**: Abrir a página inicial, confirmar que a lista de filmes aparece sem nenhuma ação adicional, aplicar cada filtro e verificar que apenas os filmes correspondentes permanecem visíveis.

**Acceptance Scenarios**:

1. **Given** que existem filmes cadastrados, **When** o visitante abre a página inicial, **Then** a aba de filmes está selecionada e exibe os filmes, cada um com título, gênero, duração, classificação etária, situação e imagem (quando houver).
2. **Given** a lista de filmes exibida, **When** o visitante digita parte de um título no filtro de busca, **Then** apenas filmes cujo título contenha o texto digitado (sem diferenciar maiúsculas/minúsculas nem acentos) permanecem na lista.
3. **Given** a lista de filmes exibida, **When** o visitante escolhe um gênero, uma classificação etária ou uma situação, **Then** a lista mostra somente filmes que satisfaçam essa escolha.
4. **Given** mais de um filtro aplicado, **When** a lista é atualizada, **Then** somente filmes que atendam a todos os filtros simultaneamente são exibidos.
5. **Given** filtros aplicados, **When** o visitante aciona a opção de limpar filtros, **Then** a lista completa volta a ser exibida.

---

### User Story 2 - Explorar e filtrar cinemas (Priority: P2)

O visitante troca para a segunda aba e vê a lista de cinemas. Pode restringi-la por nome, estado e cidade para encontrar a unidade mais próxima de onde está.

**Why this priority**: Complementa a descoberta de filmes: depois de escolher o que assistir, o visitante precisa saber onde. Depende apenas da estrutura de abas já existir.

**Independent Test**: Selecionar a aba de cinemas, verificar que a lista aparece e aplicar cada filtro conferindo o resultado.

**Acceptance Scenarios**:

1. **Given** que existem cinemas cadastrados, **When** o visitante seleciona a aba de cinemas, **Then** são exibidos os cinemas com nome, cidade, estado e endereço.
2. **Given** a lista de cinemas exibida, **When** o visitante digita parte do nome no filtro de busca, **Then** apenas cinemas cujo nome contenha o texto (sem diferenciar maiúsculas/minúsculas nem acentos) permanecem.
3. **Given** a lista de cinemas exibida, **When** o visitante escolhe um estado, **Then** somente cinemas daquele estado aparecem e o filtro de cidade passa a oferecer apenas cidades desse estado.
4. **Given** um estado escolhido, **When** o visitante escolhe uma cidade, **Then** somente cinemas daquela cidade aparecem.
5. **Given** filtros aplicados, **When** o visitante aciona a opção de limpar filtros, **Then** a lista completa de cinemas volta a ser exibida.

---

### User Story 3 - Alternar entre abas preservando o contexto (Priority: P3)

O visitante alterna livremente entre as duas abas sem perder o que já filtrou: ao voltar a uma aba, os filtros e o texto digitados anteriormente continuam aplicados. A aba ativa também é refletida no endereço da página, permitindo compartilhar ou recarregar a página mantendo a mesma aba.

**Why this priority**: Melhora a experiência, mas as duas abas já são úteis sem isso.

**Independent Test**: Aplicar filtros na aba de filmes, ir para cinemas, voltar e confirmar que os filtros de filmes continuam; recarregar a página com a aba de cinemas ativa e confirmar que ela reabre na mesma aba.

**Acceptance Scenarios**:

1. **Given** filtros aplicados na aba de filmes, **When** o visitante vai à aba de cinemas e retorna, **Then** os filtros de filmes continuam aplicados.
2. **Given** que o filtro de uma aba não afeta a outra, **When** o visitante filtra cinemas, **Then** a lista de filmes permanece inalterada.
3. **Given** a aba de cinemas ativa, **When** a página é recarregada ou o endereço é aberto em outro navegador, **Then** a aba de cinemas é exibida como ativa.
4. **Given** o uso apenas de teclado, **When** o visitante navega pelas abas, **Then** consegue alternar entre elas e perceber qual está ativa.

---

### Edge Cases

- Nenhum item atende aos filtros: exibir mensagem clara de "nenhum resultado" com atalho para limpar os filtros, sem deixar a área em branco.
- Não há filmes ou cinemas cadastrados: exibir estado vazio explicativo, distinto do estado de "sem resultados para o filtro".
- Falha ao carregar a lista (serviço indisponível): exibir mensagem de erro com opção de tentar novamente, sem impedir o uso da outra aba.
- Carregamento demorado: indicar que a lista está carregando.
- Filmes sem imagem ou sem sinopse: exibir um substituto visual neutro e omitir o campo ausente.
- Texto de busca com acentos, maiúsculas ou espaços extras nas pontas: deve encontrar o item normalmente.
- Cinemas inativos: não aparecem na lista pública.
- Filmes fora de cartaz: não aparecem por padrão (ver Assumptions).
- Quantidade grande de itens: a lista deve continuar navegável (ver SC-004).
- Telas pequenas (celular): abas e filtros permanecem utilizáveis.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A página inicial MUST apresentar duas abas, rotuladas "Filmes" e "Cinemas", nessa ordem, com a aba de filmes selecionada ao abrir.
- **FR-002**: A aba de filmes MUST listar os filmes disponíveis, mostrando para cada um título, gênero, duração, classificação etária, situação e imagem quando existir.
- **FR-003**: A aba de cinemas MUST listar os cinemas ativos, mostrando para cada um nome, cidade, estado e endereço.
- **FR-004**: A aba de filmes MUST oferecer filtros por texto (título), gênero, classificação etária e situação (em cartaz, em breve).
- **FR-005**: A aba de cinemas MUST oferecer filtros por texto (nome), estado e cidade.
- **FR-006**: As opções dos filtros de gênero, classificação etária, estado e cidade MUST ser derivadas dos dados realmente existentes, de modo que nunca se ofereça uma opção que resultaria em lista vazia por inexistência de dados.
- **FR-007**: O filtro de cidade MUST ser restrito às cidades do estado escolhido, quando houver um estado escolhido.
- **FR-008**: A busca por texto MUST ignorar diferença de maiúsculas/minúsculas e de acentuação.
- **FR-009**: Filtros combinados MUST ser aplicados em conjunto (E lógico), e a lista MUST atualizar imediatamente a cada alteração, sem recarregar a página.
- **FR-010**: Cada aba MUST oferecer uma ação de limpar filtros que restaure a lista completa.
- **FR-011**: Cada aba MUST exibir a quantidade de itens exibidos após a filtragem.
- **FR-012**: O sistema MUST exibir estados distintos para carregamento, lista vazia, nenhum resultado para os filtros e falha de carregamento, este último com opção de tentar novamente.
- **FR-013**: Os filtros de uma aba MUST ser independentes dos da outra e MUST ser preservados ao alternar entre abas durante a mesma visita.
- **FR-014**: A aba ativa MUST ser refletida no endereço da página, de modo que recarregar ou compartilhar o endereço reabra a mesma aba.
- **FR-015**: As abas e os filtros MUST ser operáveis por teclado, com indicação visual e programática da aba ativa.
- **FR-016**: A página MUST ser utilizável em telas de celular, tablet e desktop.
- **FR-017**: A página inicial MUST ser acessível sem exigir login do visitante.
- **FR-018**: A lista de filmes MUST ocultar por padrão os filmes fora de cartaz e a lista de cinemas MUST ocultar os cinemas inativos.

### Key Entities *(include if feature involves data)*

- **Filme**: obra exibida nos cinemas. Atributos relevantes: título, sinopse (opcional), duração em minutos, classificação etária, gênero, imagem (opcional) e situação (em cartaz, em breve, fora de cartaz).
- **Cinema**: unidade onde as sessões acontecem. Atributos relevantes: nome, cidade, estado (sigla), endereço e indicação de ativo.
- **Filtro**: conjunto de critérios escolhidos pelo visitante em uma aba; existe apenas durante a visita e não é armazenado no servidor.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Um visitante vê a lista de filmes em até 3 segundos após abrir a página inicial em conexão comum, sem nenhuma interação adicional.
- **SC-002**: Um visitante localiza um filme ou cinema específico, usando busca ou filtros, em menos de 15 segundos em pelo menos 90% das tentativas de teste de usabilidade.
- **SC-003**: Após qualquer alteração de filtro, a lista reflete o novo resultado em menos de 1 segundo.
- **SC-004**: Com até 500 filmes e 200 cinemas cadastrados, a página continua responsiva, atendendo aos mesmos tempos de SC-001 e SC-003.
- **SC-005**: 100% das combinações de filtros resultam em lista correta, mensagem de "nenhum resultado" ou lista vazia explicada, nunca em tela em branco ou erro.
- **SC-006**: Todas as funções da página (trocar de aba, filtrar, limpar filtros) podem ser concluídas usando apenas o teclado.
- **SC-007**: A página permanece utilizável, sem rolagem horizontal, em larguras de tela a partir de 360 px.

## Assumptions

- Os dados de filmes e cinemas já são fornecidos pelo serviço de catálogo existente por meio de consultas públicas, e esta feature apenas os consome; cadastro e edição de filmes/cinemas estão fora do escopo (pertencem à área administrativa).
- Os rótulos e a ordem das abas seguem o conteúdo descrito no pedido: a primeira aba lista filmes ("Filmes") e a segunda lista cinemas ("Cinemas"), apesar de o pedido citar "Cinemas e Filmes" na ordem inversa.
- A filtragem ocorre sobre o conjunto de dados já carregado, o que é adequado ao volume esperado (SC-004); a paginação do servidor não é exposta ao visitante.
- Filmes "em breve" são exibidos e filtráveis; filmes "fora de cartaz" e cinemas inativos ficam ocultos, sem opção de exibi-los nesta versão.
- Ao clicar em um filme ou cinema, a navegação para páginas de detalhes, grade de sessões ou compra de ingressos está fora do escopo desta feature.
- Os filtros valem apenas para a visita corrente e não são persistidos entre visitas.
- O idioma da interface é o português do Brasil.
- Esta página substitui o conteúdo atual (modelo vazio) da aplicação web voltada ao público, que hoje não tem nenhuma tela.
