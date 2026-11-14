# Quickstart: validar "Cinemas com o filme em cartaz"

## Pré-requisitos
- Plataforma de pé (`cd ingressos-golang && docker compose up --build`) com o seed de exemplo do catálogo (filmes/salas/sessões em cinemas de mais de uma UF/cidade) e Keycloak.
- `cd ingressos-golang/front && npm install`

## Testes automatizados
```bash
npm test          # Vitest (domínio cinemasDoFilme, rotas, API, CartaoFilme, PaginaFilme, App)
npm run lint
npm run build
```

## Validação manual (`npm run dev`)
1. Aba **Filmes** → clicar no card de um filme em cartaz: abre `/filmes/{id}` com dados do filme e os cinemas que o exibem (e só eles).
2. Escolher uma **UF**: só cinemas da UF; lista de cidades restrita a ela. Escolher **cidade**: só da cidade. Trocar a UF: cidade incompatível some. **Limpar filtros** restaura tudo.
3. Combinação sem resultado → mensagem + "Limpar filtros".
4. Deslogado: "Comprar Ingressos" de um cinema → login → cai em `/cinemas/{cinema}/filmes/{filme}/comprar`. Logado: vai direto.
5. Na compra, "Voltar" → página do filme; "← Voltar aos filmes" → aba Filmes com filtros preservados.
6. Abrir `/filmes/{id}` direto (recarga/link) e `/filmes/inexistente` → "Filme não encontrado". Filme "em breve" → "não está em cartaz em nenhum cinema".
7. Reduzir a janela a 360 px: layout sem rolagem horizontal.

Contrato: [contracts/rota-filme.md](contracts/rota-filme.md) · Modelo: [data-model.md](data-model.md)
