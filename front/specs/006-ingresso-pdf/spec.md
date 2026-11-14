# Feature Specification: Baixar o ingresso em PDF após o pagamento aprovado

**Feature Branch**: `006-ingresso-pdf` (nenhuma branch dedicada; o trabalho segue direto na `master`)

**Created**: 2026-11-14

**Status**: Draft

**Input**: User description: "A página "Pagamento aprovado" deve permitir baixa um PDF com os dados do ingresso."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Baixar o ingresso em PDF ao ver "Pagamento aprovado" (Priority: P1)

Depois de pagar, o visitante vê a confirmação "Pagamento aprovado". Nessa confirmação há um botão para baixar o ingresso. Ao acioná-lo, o navegador baixa um arquivo PDF com os dados do ingresso: filme, cinema, sala, dia e horário da sessão, assentos, valores (ingressos, taxa e total) e um código de identificação do pedido.

**Why this priority**: É o núcleo do pedido — o visitante sai da compra com algo concreto para apresentar. Sozinho já entrega valor completo (MVP).

**Independent Test**: Concluir um pagamento aprovado (simulado), acionar o botão de baixar e abrir o PDF obtido, conferindo que cada dado do ingresso corresponde ao pedido que acabou de ser pago.

**Acceptance Scenarios**:

1. **Given** a confirmação "Pagamento aprovado" exibida, **When** o visitante aciona o botão de baixar o ingresso, **Then** o navegador baixa um arquivo PDF sem sair da página nem perder a confirmação.
2. **Given** o PDF baixado, **When** é aberto, **Then** mostra filme (título e classificação), cinema, sala, dia e horário da sessão, os assentos escolhidos, o valor dos ingressos, a taxa, o total e o código do pedido.
3. **Given** um pedido com vários assentos, **When** o PDF é aberto, **Then** há um ingresso para cada assento, cada um identificando o seu assento.
4. **Given** a confirmação exibida, **When** o visitante baixa o PDF mais de uma vez, **Then** cada download gera o mesmo conteúdo (mesmo código do pedido) sem criar outro pedido.
5. **Given** o PDF baixado, **When** o nome do arquivo é observado, **Then** ele identifica o filme e a sessão de forma legível (por exemplo, "ingresso-duna-15-11-1600.pdf").

---

### User Story 2 - Reconhecer o PDF como ingresso e entendê-lo (Priority: P2)

O PDF é legível e organizado como um ingresso: cabeçalho com o nome da plataforma, destaque para filme, data/horário, cinema e sala, assentos em evidência e o código do pedido bem visível. Como o pagamento é simulado, o documento deixa claro que se trata de uma simulação, sem validade real de entrada.

**Why this priority**: Melhora a qualidade do documento e evita confusão sobre sua validade, mas a US1 já entrega o download com os dados.

**Independent Test**: Abrir o PDF e verificar que um leitor não técnico identifica em poucos segundos filme, dia, horário, cinema, sala e assento, e vê o aviso de simulação.

**Acceptance Scenarios**:

1. **Given** o PDF aberto, **When** o visitante o observa, **Then** identifica filme, dia/horário, cinema, sala e assentos sem precisar interpretar códigos.
2. **Given** o PDF aberto, **When** o visitante procura informação de validade, **Then** encontra um aviso de que o ingresso é uma simulação, sem valor para entrada.
3. **Given** títulos, nomes ou assentos longos, **When** o PDF é gerado, **Then** o texto não é cortado nem sobrepõe outros elementos.
4. **Given** caracteres acentuados e especiais (ç, ã, é, ª), **When** o PDF é aberto, **Then** todos aparecem corretamente.

---

### Edge Cases

- Falha ao gerar o arquivo: a confirmação continua visível, uma mensagem curta avisa que não foi possível gerar o PDF e o botão permite tentar de novo.
- Recarregar a página após o pagamento aprovado: o pedido já foi concluído e não está mais disponível na página, então o botão deixa de existir; o visitante precisa baixar o PDF antes de sair (ver Assumptions).
- Clique repetido e rápido no botão: gera um único download por vez (botão indica o andamento).
- Pagamento recusado, reserva expirada ou ausência de reserva: nenhum botão de baixar ingresso é exibido, pois não há ingresso.
- O PDF não inclui o pôster do filme (com ou sem pôster cadastrado); o layout não deixa espaço vazio quebrado.
- Navegador que bloqueie downloads: o botão continua disponível para nova tentativa e a confirmação permanece.
- Nenhum dado de cartão (número, validade, código de segurança, nome impresso) aparece no PDF.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A confirmação "Pagamento aprovado" MUST oferecer um botão "Baixar ingresso (PDF)", visível e acionável por teclado.
- **FR-002**: O botão MUST aparecer somente quando o pagamento foi aprovado; nos estados de formulário, processando, recusado, reserva expirada e sem reserva MUST NOT aparecer.
- **FR-003**: Ao acionar o botão, o navegador MUST baixar um arquivo PDF, sem abandonar a página e sem alterar a confirmação exibida.
- **FR-004**: O PDF MUST conter: filme (título e classificação etária), cinema, sala, dia e horário da sessão, assentos, valor dos ingressos, taxa, total e um código de identificação do pedido.
- **FR-005**: Para um pedido com mais de um assento, o PDF MUST apresentar um ingresso por assento, cada um com o seu assento e os dados da sessão; os valores do pedido (ingressos, taxa, total) MUST constar uma vez, no resumo do pedido.
- **FR-006**: O código de identificação do pedido MUST ser o mesmo em todos os downloads do mesmo pedido.
- **FR-007**: O PDF MUST exibir um aviso claro de que o pagamento é simulado e o documento não vale como ingresso real.
- **FR-008**: O PDF MUST NOT conter dado de cartão (número, nome impresso, validade ou código de segurança).
- **FR-009**: Valores MUST ser apresentados em reais no formato brasileiro e data/horário no fuso local do visitante; acentos e caracteres especiais MUST ser exibidos corretamente.
- **FR-010**: O nome do arquivo MUST identificar o filme e a sessão, sem caracteres inválidos para nome de arquivo.
- **FR-011**: Em caso de falha na geração, a página MUST informar o erro de forma amigável, manter a confirmação visível e permitir nova tentativa.
- **FR-012**: O PDF MUST ser gerado sem depender de nova consulta a serviços externos; usa apenas os dados do pedido já presentes na página.

### Key Entities *(include if feature involves data)*

- **Pedido aprovado**: resumo do pedido pago — filme, cinema, sala, início da sessão, assentos, valor dos ingressos, taxa, total.
- **Ingresso**: um por assento do pedido — assento, filme, cinema, sala e sessão, com o código do pedido.
- **Código do pedido**: identificador curto, legível, estável para o mesmo pedido.
- **Documento PDF**: arquivo com os ingressos do pedido e o resumo de valores.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Da confirmação "Pagamento aprovado" ao arquivo baixado, o visitante precisa de 1 clique.
- **SC-002**: O download termina em até 3 segundos em um pedido de até 10 assentos, em um computador comum.
- **SC-003**: 100% dos campos do FR-004 coincidem com os do pedido pago, conferidos em pedidos com 1, 2 e 10 assentos.
- **SC-004**: 100% dos PDFs gerados exibem o aviso de simulação e nenhum contém dado de cartão.
- **SC-005**: Em uma revisão com 5 pessoas, todas identificam filme, dia, horário, cinema, sala e assento no PDF em até 10 segundos.
- **SC-006**: Nenhum cenário de falha (geração, bloqueio de download) deixa a página sem a confirmação ou sem orientação ao visitante.

## Assumptions

- O PDF é gerado no próprio navegador a partir dos dados do pedido já presentes na página de pagamento (filme, cinema, sala, início, assentos, valores); não há novo serviço nem consulta extra.
- O pagamento é simulado (feature 004); portanto o PDF é um comprovante simulado, com aviso explícito, e não um ingresso com validade real para entrada.
- O código do pedido é gerado localmente de forma determinística a partir dos dados do pedido (sessão, assentos e prazo da reserva), pois o serviço de notificação não emite ingresso real neste fluxo simulado; não há QR code nem código de barras neste escopo.
- Um único arquivo PDF contém um ingresso por assento (uma página por ingresso) e o resumo de valores; não há envio por e-mail, impressão especial nem "adicionar à carteira digital".
- A reserva é removida ao aprovar o pagamento (feature 004); por isso o botão só existe na tela de aprovação, enquanto a página não for recarregada ou deixada. Reabrir ingressos antigos (histórico de compras) está fora do escopo.
- O nome do comprador não consta no PDF, pois a página não dispõe dele de forma confiável; a identificação do pedido é feita pelo código.
- O visual do PDF segue a identidade das demais páginas (nome "Ingressos", cores neutras), com texto legível em impressão em preto e branco, tamanho A4 retrato.
