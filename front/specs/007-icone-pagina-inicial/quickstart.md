# Quickstart: validar "Ícone de página inicial"

## Pré-requisitos
- `cd ingressos-golang/front && npm install`

## Testes automatizados
```bash
npm test          # Vitest (LinkInicio, CabecalhoSite/App, BarraConta)
npm run lint
npm run build
```

## Validação manual (`npm run dev`)
1. Abrir `/`: o ícone de casa aparece à esquerda do cabeçalho; clicar não muda nada nem cria entrada no histórico.
2. Abrir um cinema, um filme, a compra de assentos e o pagamento: o ícone está no mesmo lugar em todas; clicar leva a `/` sem recarregar, preservando aba/filtros da página inicial.
3. Após voltar ao início pelo ícone, o botão "voltar" do navegador retorna à página anterior.
4. Ctrl/Cmd+clique (ou botão do meio): `/` abre em nova aba.
5. Teclado: Tab até o ícone mostra o foco; Enter navega. Leitor de tela anuncia "Página inicial, link". Passar o mouse mostra a dica.
6. Abrir `/cinemas/inexistente` (erro/não encontrado) e derrubar a rede (estado de erro): o ícone continua visível.
7. Reduzir a janela a ~360 px e alternar tema claro/escuro: ícone visível, sem sobrepor Login/Criar Conta, contraste legível, alvo ≥ 44×44 px.
8. Logado e deslogado: o ícone aparece igual.

Contrato: [contracts/link-inicio.md](contracts/link-inicio.md) · Modelo: [data-model.md](data-model.md)
