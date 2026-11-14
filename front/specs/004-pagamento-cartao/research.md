# Research: Página de pagamento via cartão (simulada)

## Como passar os dados da reserva à página de pagamento

- **Decision**: gravar um instantâneo (JSON) em `sessionStorage` na chave `pagamento:reserva` antes de `navegar(caminhoPagamento(...))`; a página lê, valida (formato, cinema/filme da rota, `expira_em`) e, se inválido, trata como "sem reserva".
- **Rationale**: `navegar` já usa `pushState` sem estado; `sessionStorage` sobrevive a recarga e ao retorno do login (restrição do spec: recarregar não deve quebrar), é local à aba e já é usado pelo app (`sessao`, seleção de assentos). O instantâneo contém só dados do pedido, nunca dados de cartão.
- **Alternatives**: `history.state` (exigiria mudar `navegar`; some em nova aba); parâmetros na URL (expõe dados e permite forjar valores); reconsultar o estoque (não há endpoint de leitura da reserva e a feature é simulada).

## Validação dos campos do cartão

- **Decision**: número sem validação (qualquer valor preenchido, por pedido do usuário); validade `MM/AA` com mês 01–12 e não vencida (vale até o fim do mês); CVV 3–4 dígitos; nome com ≥ 2 caracteres não brancos. Entrada normalizada (remove espaços/hífens). Máscara do número em grupos de 4 e validade `MM/AA`.
- **Rationale**: regras padrão de formulário de cartão, sem dependência; Número livre para facilitar a demonstração.
- **Alternatives**: biblioteca de validação (dependência desnecessária); identificar bandeira (fora do escopo).

## Simulação do pagamento

- **Decision**: `setTimeout` (~2 s, injetável) → aprovado; número terminado em `0000` → recusado. Estado `ocioso | processando | aprovado | recusado`; botão desabilitado em `processando` (impede duplo envio). Verifica expiração ao enviar e ao concluir.
- **Rationale**: demonstra os dois caminhos sem rede; atraso injetável mantém testes rápidos.
- **Alternatives**: chamar o serviço `pagamento` (contradiz o pedido de página "fake" e exigiria backend/eventos).

## Prazo da reserva

- **Decision**: contagem regressiva por `setInterval` de 1 s a partir de `expira_em`; ao zerar, estado "expirada" e formulário desabilitado.
- **Rationale**: simples, atende SC/FR-008. O relógio do cliente é suficiente em simulação.

## Privacidade

- Dados do cartão só em estado React; sem `localStorage`/`sessionStorage`, sem `console.*`, sem `fetch`; campos com `autocomplete` apropriado e `inputMode="numeric"`.
