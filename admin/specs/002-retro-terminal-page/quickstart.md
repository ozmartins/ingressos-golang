# Quickstart: Página de Terminal Retrô

**Pré-requisitos**: Node.js 20+ e npm. Execute em `ingressos-golang/web/`.

1. `npm install`
2. **Testes automatizados**: `npm test` — todos passam (comandos, histórico, limite de linhas).
3. **Build**: `npm run build` — código de saída 0.
4. **Execução**: `npm run dev` e abrir `http://localhost:5173`.

## Verificação manual

| # | Passo | Esperado |
|---|-------|----------|
| 1 | Abrir a página | Fundo escuro, texto verde monoespaçado, boas-vindas sugerindo `help`, campo embaixo com foco e cursor piscante |
| 2 | Digitar `help` + Enter | Comando e lista de comandos aparecem; campo limpo |
| 3 | Digitar `echo <b>oi</b>` | Exibe literalmente `<b>oi</b>` |
| 4 | Digitar `xyz` | `comando não encontrado: xyz` |
| 5 | Apenas Enter | Nenhum erro |
| 6 | Executar `date` várias vezes | Saída rola até a linha mais recente |
| 7 | Seta ↑ / ↓ | Percorre comandos anteriores no campo |
| 8 | `clear` | Saída esvaziada |
| 9 | Clicar fora do campo | Foco volta ao campo |
| 10 | Ativar "reduzir movimento" no sistema e recarregar | Cursor/varredura sem animação |
| 11 | Redimensionar para 360 px | Campo visível e utilizável, sem rolagem horizontal |
