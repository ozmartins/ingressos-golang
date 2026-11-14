# Quickstart: Aplicação React Vazia

**Pré-requisitos**: Node.js ≥ 20 (testado com 24) e npm.

Execute dentro de `ingressos-golang/web/`:

1. **Instalar**: `npm install` — conclui sem erros.
2. **Desenvolvimento**: `npm run dev` — abrir o endereço exibido (padrão `http://localhost:5173`). Esperado: página em branco, console do navegador sem erros ou avisos. Editar `src/App.tsx` e salvar atualiza a página sem reinstalar.
3. **Build de produção**: `npm run build` — código de saída 0, artefatos em `dist/`.
4. **Pré-visualização (opcional)**: `npm run preview` — serve `dist/` e exibe a mesma página vazia.
5. **Preservação**: `git status` não mostra remoções de `.claude/`, `.specify/`, `.gitignore`, `.dockerignore`; `node_modules/` e `dist/` ficam ignorados.
