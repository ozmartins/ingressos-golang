# Feature Specification: Página de pagamento via cartão (simulada)

**Feature Branch**: `004-pagamento-cartao` (nenhuma branch dedicada; o trabalho segue direto na `master`)

**Created**: 2026-11-14

**Status**: Draft

**Input**: User description: "Crie uma página (fake, pois isso é so um trabalho de faculdade) para pagamento via cartão. Esta página deve ser chamada quando o usuário clicar no botão "Continuar"."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Ir ao pagamento ao continuar a compra (Priority: P1)

Na página de seleção de assentos, o visitante autenticado escolhe assentos e aciona "Continuar". Os assentos são reservados temporariamente (comportamento já existente) e, em vez de apenas mostrar a confirmação da reserva, o visitante é levado a uma página própria de pagamento com cartão. A página mostra o resumo do pedido (filme, cinema, sala, dia/horário, assentos, total) e o prazo da reserva.

**Why this priority**: É o gatilho pedido: sem a navegação a partir de "Continuar" a página de pagamento não é alcançável.

**Independent Test**: Logado, selecionar assentos livres, acionar "Continuar" e verificar que a página de pagamento abre com o resumo correspondente à seleção.

**Acceptance Scenarios**:

1. **Given** assentos selecionados e reserva criada com sucesso, **When** o visitante aciona "Continuar", **Then** é levado à página de pagamento com cartão.
2. **Given** a página de pagamento aberta, **When** ela termina de carregar, **Then** exibe filme, cinema, sala, dia/horário, assentos reservados, total em reais e até quando a reserva vale.
3. **Given** conflito de assentos ou falha na reserva, **When** o visitante aciona "Continuar", **Then** permanece na seleção de assentos com o aviso já existente e NÃO é levado ao pagamento.
4. **Given** a página de pagamento aberta, **When** o visitante usa o comando de voltar, **Then** retorna à seleção de assentos do mesmo filme e cinema.

---

### User Story 2 - Preencher os dados do cartão e "pagar" (Priority: P1)

Na página de pagamento o visitante informa número do cartão, nome impresso, validade e código de segurança, e aciona "Pagar". O formulário valida o formato dos campos. Como o pagamento é simulado, nenhuma cobrança real ocorre e nenhum dado do cartão é enviado ou guardado: após uma breve espera simulando o processamento, a página mostra a confirmação de pagamento aprovado.

**Why this priority**: Entrega o objetivo da página (pagar com cartão), ainda que de mentirinha.

**Independent Test**: Na página de pagamento, preencher dados válidos e pagar e ver a aprovação; deixar campos vazios/inválidos e ver as mensagens de erro, sem aprovação.

**Acceptance Scenarios**:

1. **Given** o formulário em branco, **When** o visitante aciona "Pagar", **Then** vê mensagens indicando cada campo obrigatório não preenchido e nada é processado.
2. **Given** validade já vencida ou em formato inválido, ou código de segurança com tamanho inválido, **When** tenta pagar, **Then** o campo correspondente exibe erro específico. O número do cartão NÃO é validado: qualquer número preenchido é aceito (sem tamanho mínimo nem dígito verificador).
3. **Given** dados válidos, **When** aciona "Pagar", **Then** o botão fica desabilitado com indicação de processamento por alguns segundos e, em seguida, a página mostra "Pagamento aprovado" com resumo dos assentos e total.
4. **Given** o número digitado, **When** o visitante digita, **Then** ele é exibido agrupado de 4 em 4 dígitos e a validade como MM/AA, aceitando apenas dígitos.
5. **Given** um cartão de teste documentado na própria página como "cartão de recusa" (ex.: terminado em 0000), **When** o visitante paga, **Then** a página mostra "Pagamento recusado" com opção de tentar outro cartão, sem perder os demais dados do pedido.
6. **Given** a página, **When** exibida, **Then** deixa claro que é uma simulação e que não se deve informar dados reais de cartão.

---

### User Story 3 - Tratar acesso indevido e reserva expirada (Priority: P2)

A página de pagamento só faz sentido com uma reserva em andamento do usuário autenticado. Acessos sem reserva, sem login ou com a reserva vencida são redirecionados de forma amigável.

**Why this priority**: Evita telas quebradas ao recarregar, abrir o endereço direto ou demorar além do prazo.

**Independent Test**: Abrir o endereço da página de pagamento sem reserva, deslogado e com prazo vencido, e verificar o tratamento de cada caso.

**Acceptance Scenarios**:

1. **Given** visitante não logado que abre o endereço do pagamento, **When** a página é requisitada, **Then** é direcionado ao login e, depois, volta ao endereço do pagamento: se a reserva ainda existir na mesma aba, a página abre normalmente; caso contrário, mostra a mensagem de que não há reserva (cenário 2).
2. **Given** logado sem reserva em andamento (endereço direto ou recarga sem dados), **When** abre a página, **Then** vê mensagem de que não há reserva e um caminho para a seleção de assentos ou para a página inicial.
3. **Given** a reserva com prazo vencido, **When** o visitante está na página ou tenta pagar, **Then** é avisado de que a reserva expirou, o pagamento é impedido e ele pode voltar a escolher assentos.
4. **Given** a página de pagamento com contagem do prazo, **When** o tempo restante diminui, **Then** o visitante o vê em minutos e segundos.

---

### Edge Cases

- Duplo clique em "Pagar": apenas um processamento ocorre.
- Recarregar a página durante o processamento ou após aprovado: não duplica a aprovação; sem reserva em memória cai no caso "sem reserva" (ver acima).
- Cartão com bandeira não reconhecida: aceito se o formato for válido (simulação).
- Campos colados com espaços ou hífens: normalizados.
- Telas estreitas (celular): formulário e resumo utilizáveis sem rolagem horizontal da página.
- Nome impresso com acentos e caracteres especiais: aceito.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A ação "Continuar" da seleção de assentos MUST, após a reserva ser criada com sucesso, levar o visitante à página de pagamento com cartão; em caso de conflito ou erro MUST manter o comportamento atual (permanecer na seleção com aviso).
- **FR-002**: A página de pagamento MUST ter endereço próprio, exigir usuário autenticado (reutilizando o fluxo de login com retorno) e depender de uma reserva em andamento criada pela etapa anterior.
- **FR-003**: A página MUST exibir o resumo do pedido: pôster, título, cinema, sala, dia/horário, assentos reservados, quantidade, taxa (se houver) e total em reais, e o prazo da reserva com contagem regressiva.
- **FR-004**: A página MUST oferecer formulário com número do cartão, nome impresso, validade (MM/AA) e código de segurança, com formatação automática enquanto se digita.
- **FR-005**: A página MUST validar os campos (obrigatoriedade de todos os campos, validade não vencida, tamanho do código); o número do cartão só precisa estar preenchido, sem validação de formato, tamanho ou dígito verificador e mostrar mensagens específicas por campo.
- **FR-006**: O pagamento MUST ser simulado: nenhuma cobrança real, nenhum dado de cartão enviado a serviços externos ou armazenado/registrado em log; a página MUST informar que é uma simulação.
- **FR-007**: Ao pagar com dados válidos a página MUST mostrar estado de processamento por breve intervalo e então a confirmação "Pagamento aprovado" com resumo; para o cartão de recusa de teste MUST mostrar "Pagamento recusado" com opção de tentar novamente.
- **FR-008**: O sistema MUST impedir o pagamento com a reserva expirada e MUST impedir envio duplo.
- **FR-009**: A página MUST oferecer comando de voltar à seleção de assentos e, após a aprovação, caminho para a página inicial.
- **FR-010**: A página MUST seguir o esquema de cores, tipografia e estilo das demais páginas.
- **FR-011**: A página MUST ser utilizável em celular e desktop e acessível por teclado e leitor de tela (rótulos e mensagens de erro associados aos campos).

### Key Entities *(include if feature involves data)*

- **Reserva em andamento**: dono (usuário que reservou), assentos reservados, sessão, valor total e prazo de validade, produzida ao continuar na seleção de assentos.
- **Dados de pagamento por cartão**: número, nome impresso, validade e código de segurança; existem apenas durante o preenchimento e nunca são persistidos nem enviados.
- **Resultado do pagamento simulado**: aprovado ou recusado, com o resumo do pedido.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% das reservas criadas com sucesso por "Continuar" levam à página de pagamento; 0% dos conflitos/erros levam a ela.
- **SC-002**: Um visitante com assentos escolhidos preenche o cartão e vê a aprovação em menos de 1 minuto na primeira tentativa, em pelo menos 90% dos testes de usabilidade.
- **SC-003**: 100% dos dados inválidos (campo vazio, validade vencida, código curto) são barrados com mensagem no campo correspondente antes de qualquer "aprovação".
- **SC-004**: Nenhum dado de cartão é transmitido pela rede nem armazenado após sair da página, verificável por inspeção do tráfego e do armazenamento do navegador.
- **SC-005**: A página é utilizável sem rolagem horizontal em telas de 360 px e todo o formulário é operável apenas pelo teclado.
- **SC-006**: Uma revisão visual não encontra cores fora da paleta das demais páginas.

## Assumptions

- Esta feature é um trabalho acadêmico: o pagamento é totalmente simulado no front-end e NÃO usa o serviço de pagamento da plataforma, nem publica ou consome eventos de pagamento. A confirmação de pagamento não altera o estado dos assentos além da reserva temporária já criada.
- A reserva temporária (comportamento existente de "Continuar") continua sendo criada antes de abrir o pagamento; o pagamento reaproveita seus dados e prazo, passados pela navegação (sem reconsultar o estoque).
- Somente cartão está disponível (sem Pix, boleto ou parcelamento); o valor é à vista.
- Cartão de recusa de teste: número terminado em 0000; demais números válidos são aprovados.
- Bandeira do cartão não é identificada nem exigida.
- Emissão do ingresso e notificação após a aprovação estão fora do escopo; a página apenas mostra a confirmação simulada.
- Login e sessão reutilizam o mecanismo de autenticação existente. A reserva guardada na aba pertence ao usuário que a criou: outra pessoa que logue na mesma aba não a vê (cai em "sem reserva").
- Terminologia: "assento" na interface.
