# Quickstart: validar a ajuda por controller

## Verificação automatizada
```
cd ingressos-golang/web
npm test
npm run build
```

## Roteiro manual (`npm run dev`, abrir a página)
1. `help`: a seção "Comandos da API do catálogo" mostra só `filmes`, `cinemas`, `salas`, `sessoes`, `health` e a dica; a saída inteira cabe em 20 linhas; as seções local e "Sessão" continuam.
2. `filmes help`: lista as 5 ações de filmes com sintaxe e descrição, só as de filmes.
3. Repetir com `cinemas help`, `salas help`, `sessoes help`.
4. `health help`: mostra a sintaxe e a descrição do `health`.
5. `FILMES   help`: mesmo resultado do passo 2.
6. `filmes help extra`: mesmo resultado do passo 2.
7. `filmes` e `filmes xpto`: lista as ações válidas e a dica `use: filmes help ...`.
8. `help filmes`: mostra o help geral resumido, sem erro.
9. Com a aba de rede aberta, repetir os passos 1–8: nenhuma requisição ao catálogo.
10. `login help` e `clear help`: comportamento anterior (sem mudança).
