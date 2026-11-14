# Feature Specification: Ajuda do Terminal por Controller

**Feature Branch**: `005-help-por-controller` (sem branch própria: o trabalho é commitado direto na `master`)

**Created**: 2026-11-14

**Status**: Draft

**Input**: User description: "Modifique o comando help. Seu eu digitar apenas help, quero ver, na seção \"Comandos da API do catálogo\" apenas o nome dos controllers. Quando eu digitar o nome de um controller + help, quero ver as opções disponíveis pro referido controller."

## Contexto

Hoje o `help` imprime, de uma vez, todos os comandos de API (21) com sintaxe e descrição. A saída ficou longa: a pessoa precisa rolar a tela para achar o que procura. Esta funcionalidade torna o `help` **resumido por padrão** e **detalhado sob demanda**: o `help` sozinho mostra só os nomes dos controllers na seção "Comandos da API do catálogo", e `<controller> help` mostra as opções (ações com sintaxe e descrição) daquele controller. O restante da ajuda (comandos locais e comandos de sessão) e todos os demais comandos não mudam.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Ver só os controllers no help geral (Priority: P1)

Como visitante do terminal, quero digitar `help` e ver, na seção "Comandos da API do catálogo", apenas o nome de cada controller, para ter uma visão geral curta e saber o que existe.

**Why this priority**: É o núcleo do pedido e resolve a saída longa.

**Independent Test**: Executar `help` e conferir que a seção da API lista somente os nomes dos controllers (sem sintaxe de ações nem descrições) e que a saída cabe em poucas linhas.

**Acceptance Scenarios**:

1. **Given** o terminal aberto, **When** o visitante executa `help`, **Then** a seção "Comandos da API do catálogo" lista apenas os nomes dos controllers (`filmes`, `cinemas`, `salas`, `sessoes`) e o comando `health`, um por linha, sem as ações nem as descrições de cada um.
2. **Given** a saída do `help`, **When** é lida, **Then** inclui uma dica de que `<controller> help` mostra as opções do controller (por exemplo, `filmes help`).
3. **Given** a saída do `help`, **When** é lida, **Then** as seções de comandos locais (`help`, `clear`) e de sessão (`login`, `logout`) continuam presentes como hoje.

---

### User Story 2 - Ver as opções de um controller (Priority: P2)

Como visitante, quero digitar o nome de um controller seguido de `help` (por exemplo, `filmes help`) e ver as opções disponíveis para aquele controller, para saber a sintaxe de cada ação sem consultar a documentação.

**Why this priority**: Sem isso, o help resumido da US1 esconderia a sintaxe dos comandos; é o complemento necessário.

**Independent Test**: Executar `salas help` e ver as cinco ações de salas, cada uma com sintaxe e descrição curta; repetir para os demais controllers.

**Acceptance Scenarios**:

1. **Given** o terminal aberto, **When** o visitante executa `filmes help`, **Then** a saída lista todas as ações de filmes (`listar`, `criar`, `obter`, `atualizar`, `remover`), cada uma com sua sintaxe de uso e descrição curta, exatamente como aparecem hoje no help geral.
2. **Given** qualquer um dos controllers (`filmes`, `cinemas`, `salas`, `sessoes`), **When** o visitante executa `<controller> help`, **Then** vê somente as opções daquele controller, sem as de outros.
3. **Given** `health help`, **When** executado, **Then** a saída mostra a sintaxe e a descrição do comando `health`, sem chamar o serviço.
4. **Given** a entrada em maiúsculas ou com espaços extras (por exemplo, `FILMES   help`), **When** executada, **Then** o resultado é o mesmo de `filmes help`.
5. **Given** `<controller> help` executado, **When** concluído, **Then** nenhuma requisição é enviada à API.

---

### User Story 3 - Orientação em entradas parecidas (Priority: P3)

Como visitante, quero que os caminhos de ajuda existentes continuem coerentes, para não me perder ao digitar um controller sem ação ou uma ação inexistente.

**Why this priority**: Conforto; o comportamento atual já funciona e só precisa apontar para o novo recurso.

**Independent Test**: Executar `filmes` e `filmes xpto` e ver a lista de ações válidas junto com a dica de `filmes help`.

**Acceptance Scenarios**:

1. **Given** um controller sem ação (`filmes`), **When** executado, **Then** a saída lista as ações válidas (como hoje) e indica que `filmes help` mostra os detalhes.
2. **Given** uma ação desconhecida (`filmes xpto`), **When** executada, **Then** a saída informa a ação desconhecida, lista as ações válidas e indica `filmes help`.
3. **Given** `help` seguido de qualquer texto (por exemplo, `help filmes`), **When** executado, **Then** o terminal mostra o help geral resumido, sem erro.

---

### Edge Cases

- `login help` e `logout help`: não são controllers; o comportamento atual desses comandos não muda (a ajuda deles continua na seção de sessão do `help` geral).
- `clear help` e comandos desconhecidos seguidos de `help`: continuam tratados como hoje ("comando não encontrado").
- `<controller> help` com argumentos adicionais (por exemplo, `filmes help extra`): mostra a ajuda do controller, ignorando o excedente.
- A saída de `<controller> help` deve ser quebrada em linhas e rolável como qualquer outra saída do terminal.
- A palavra `help` não pode ser confundida com uma ação do controller: `filmes help` nunca dispara uma requisição.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: O `help` sem argumentos MUST exibir, na seção "Comandos da API do catálogo", apenas os nomes dos controllers (`filmes`, `cinemas`, `salas`, `sessoes`) e do comando `health`, um por linha, sem sintaxe de ações nem descrições.
- **FR-002**: O `help` sem argumentos MUST incluir uma dica de que `<controller> help` mostra as opções do controller, com um exemplo.
- **FR-003**: O `help` sem argumentos MUST manter inalteradas as seções de comandos locais e de sessão (`login`, `logout`).
- **FR-004**: `<controller> help` MUST exibir todas as ações do controller, cada uma com sintaxe de uso e descrição curta, para os quatro controllers (`filmes`, `cinemas`, `salas`, `sessoes`).
- **FR-005**: `health help` MUST exibir a sintaxe e a descrição do `health`, sem chamar o serviço.
- **FR-006**: `<controller> help` MUST NOT enviar nenhuma requisição à API nem exigir autenticação.
- **FR-007**: O reconhecimento de `<controller> help` MUST ser insensível a maiúsculas e a espaços extras.
- **FR-008**: A mensagem para controller sem ação ou ação desconhecida MUST continuar listando as ações válidas e MUST indicar `<controller> help`.
- **FR-009**: `help` seguido de texto adicional MUST mostrar o help geral resumido, sem erro.
- **FR-010**: Todos os demais comandos (leitura, escrita guiada, `remover`, `health`, `login`, `logout`, `clear`) MUST manter o comportamento atual.

### Key Entities

- **Controller**: agrupamento de comandos da API do catálogo identificado por um nome digitável (`filmes`, `cinemas`, `salas`, `sessoes`); o grupo operacional é representado pelo comando `health`.
- **Opção de controller**: uma ação do controller com sintaxe de uso e descrição curta.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: O `help` sem argumentos cabe em, no máximo, 20 linhas de saída (hoje passa de 50).
- **SC-002**: Em 100% dos controllers, `<controller> help` lista todas as ações do controller com sintaxe e descrição, e nenhuma ação de outro controller.
- **SC-003**: Uma pessoa descobre a sintaxe de qualquer comando de API em no máximo 2 comandos de ajuda (`help` e depois `<controller> help`).
- **SC-004**: Nenhuma execução de `help` ou de `<controller> help` gera requisição à API (verificável pela aba de rede do navegador).

## Assumptions

- "Controllers" são os quatro grupos com ações (`filmes`, `cinemas`, `salas`, `sessoes`). O grupo "Operacional" tem um único comando, `health`, que é listado pelo próprio nome nessa seção, e `health help` mostra sua ajuda.
- O formato de cada ação em `<controller> help` é o mesmo que o `help` geral usa hoje (sintaxe e descrição em linhas separadas).
- `controller` sozinho (sem `help`) continua listando as ações válidas em uma linha, agora com a dica para `<controller> help`.
- Não faz parte do escopo a forma `help <controller>`; ela mostra apenas o help geral (FR-009).
- As seções "Comandos disponíveis" (locais) e "Sessão" do `help` não mudam.
