# Research: Página de Terminal Retrô

## Execução de comandos

- **Decision**: Comandos simulados no cliente, via função pura que mapeia entrada → linhas de saída.
- **Rationale**: Navegador não executa shell; backend real seria risco de segurança e viola o isolamento do frontend (constituição II). Função pura é trivialmente testável.
- **Alternatives considered**: Endpoint de execução real (inseguro, fora de escopo); WebAssembly com shell (excessivo).

## Biblioteca de terminal

- **Decision**: Implementar em React + CSS, sem xterm.js.
- **Rationale**: O requisito é um terminal estilizado com comandos próprios, não emulação VT100. xterm.js adicionaria peso e complexidade (YAGNI).
- **Alternatives considered**: xterm.js (emulação completa, desnecessária).

## Efeito de monitor de fósforo

- **Decision**: CSS puro: fundo quase preto, texto verde fósforo (`#33ff66` sobre `#050805`), `text-shadow` para brilho, sobreposição `repeating-linear-gradient` para linhas de varredura (`pointer-events: none`), cursor em bloco piscante por `@keyframes`, `@media (prefers-reduced-motion: reduce)` desativa animações.
- **Rationale**: Sem dependências, acessível, contraste muito acima de 4,5:1 (verde claro sobre quase preto).
- **Alternatives considered**: Canvas/WebGL com distorção CRT (custo e legibilidade).

## Fonte

- **Decision**: Pilha monoespaçada do sistema (`ui-monospace, "Cascadia Mono", Consolas, "Courier New", monospace`).
- **Rationale**: Sem carregar fontes externas.
- **Alternatives considered**: Web font retrô (requisição externa, privacidade).

## Foco, rolagem e teclado

- **Decision**: `input` controlado com `autoFocus`; clique no contêiner devolve o foco via ref; `useEffect` rola a área de saída ao final a cada nova linha; ↑/↓ percorrem o histórico; Enter executa.
- **Rationale**: Atende FR-002, FR-006, FR-007 com APIs nativas.

## Segurança de renderização

- **Decision**: Renderizar linhas como nós de texto do React; proibir `dangerouslySetInnerHTML`.
- **Rationale**: FR-010; React escapa automaticamente.

## Limite de retenção

- **Decision**: Máximo de 500 linhas; excedente descarta as mais antigas.
- **Rationale**: FR-012, mantém responsividade.

## Testes

- **Decision**: Vitest (ambiente node) para `comandos`, `historico` e `linhas`.
- **Rationale**: Integra ao Vite já usado; a constituição exige testes quando há comportamento. Compatibilidade verificada: Vitest 5.1.0 declara peer `vite ^6.4.0 || ^7.0.0 || ^8.0.0` (projeto usa Vite 8) e exige Node `^22.12.0 || ^24.0.0 || >=26.0.0` (instalado: 24.22.3).
- **Alternatives considered**: Jest (configuração extra); Testing Library/jsdom (UI coberta por quickstart, evitando dependências extras).
