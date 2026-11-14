# Implementation Plan: Página de Terminal Retrô

**Branch**: `002-retro-terminal-page` | **Date**: 2026-11-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/002-retro-terminal-page/spec.md`

## Summary

Substituir a página vazia de `web/` por um terminal de monitor de fósforo: área de saída ocupando a tela e campo de comando fixo embaixo. Os comandos são simulados no navegador por uma função pura (`executar`) que recebe o texto digitado e devolve linhas de saída. A aparência vem de CSS puro (fonte monoespaçada, verde fósforo, brilho, linhas de varredura, cursor piscante, `prefers-reduced-motion`). Nenhuma dependência de runtime nova; Vitest entra como dependência de desenvolvimento para testar a lógica de comandos.

## Technical Context

**Language/Version**: TypeScript estrito, React 19, Node.js 24 (Vitest 5 exige Node ^22.12 || ^24 || ≥26)

**Primary Dependencies**: react, react-dom (existentes); dev: vitest 5.1 (novo; compatível com Vite ^6.4 || ^7 || ^8, conferido nos peerDependencies)

**Storage**: N/A (histórico apenas em memória, durante a sessão)

**Testing**: Vitest para a lógica pura (parser/comandos/histórico/limite de linhas); verificação visual e de teclado via `quickstart.md`

**Target Platform**: Navegadores modernos, desktop e celular (360–1920 px)

**Project Type**: web (frontend single-page, estende `web/`)

**Performance Goals**: Resultado visível em menos de 100 ms após Enter

**Constraints**: Texto sempre renderizado como texto puro (React escapa por padrão; proibido `dangerouslySetInnerHTML`); contraste ≥ 4,5:1; limite de linhas retidas (500); sem rede

**Scale/Scope**: 1 página, 6 comandos embutidos, 4 arquivos-fonte de lógica/UI (`comandos`, `historico`, `linhas`, `Terminal`) mais 3 de testes e 1 de estilos

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Princípio | Conformidade |
|-----------|--------------|
| I. Simplicidade e YAGNI | PASS: sem bibliotecas de UI/terminal (xterm.js rejeitado), sem roteamento; única dependência nova é dev (Vitest), justificada pelo princípio IV |
| II. Isolamento do Frontend | PASS: comandos simulados, sem chamadas aos serviços Go |
| III. Tipagem Estrita | PASS: TypeScript `strict`, sem `any` |
| IV. Verificação Obrigatória | PASS: há lógica de comportamento, logo testes automatizados (Vitest) + `npm run build` + quickstart |
| V. Preservação do Workspace | PASS: altera apenas `src/`, `package.json` e arquivos de configuração de teste |

Pós-design: sem mudanças. **Resultado: PASS.**

## Project Structure

### Documentation (this feature)

```text
specs/002-retro-terminal-page/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/requirements.md
└── tasks.md             # gerado por /speckit-tasks
```

`contracts/` não é gerado: não há interfaces externas. O contrato interno dos comandos está em `data-model.md`.

### Source Code (repository root = `ingressos-golang/web/`)

```text
web/
├── index.html                  # lang pt-BR, título atualizado
├── README.md                   # + comando npm test e descrição do terminal
├── package.json                # + script "test", + devDependency vitest
├── vite.config.ts              # + bloco test (ambiente node)
└── src/
    ├── main.tsx                # importa estilos
    ├── App.tsx                 # renderiza <Terminal />
    ├── estilos.css             # tema fósforo, CRT, cursor, reduced-motion
    └── terminal/
        ├── comandos.ts         # executar(entrada, contexto) → Linha[]; registro de comandos
        ├── comandos.test.ts
        ├── historico.ts        # navegação ↑/↓ e adição de entradas
        ├── historico.test.ts
        ├── linhas.ts           # tipo Linha, limite de retenção (adicionarLinhas)
        ├── linhas.test.ts
        └── Terminal.tsx        # componente: saída, campo, foco, rolagem, teclado
```

**Structure Decision**: Lógica pura separada do componente React (`comandos`, `historico`, `linhas`), o que permite testá-la com Vitest sem jsdom. `Terminal.tsx` apenas liga estado, teclado e renderização.

## Complexity Tracking

Sem violações.
