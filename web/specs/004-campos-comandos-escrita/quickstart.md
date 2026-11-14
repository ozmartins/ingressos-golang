# Quickstart: validar os comandos guiados por campos

## Pré-requisitos
- Plataforma no ar: `cd ingressos-golang && docker compose up --build` (catálogo e Keycloak).
- Frontend: `cd ingressos-golang/web && npm install && npm run dev` e abrir a URL exibida.

## Verificação automatizada
```
cd ingressos-golang/web
npm test          # Vitest: formulários, diálogo, despachante
npm run build     # tipos + build de produção (código de saída 0)
```

## Roteiro manual (autenticado via `login`)
1. `cinemas criar` → responder nome, cidade, estado, endereço, Enter em `ativo` → resumo → `s`. Esperado: cinema criado com id. Anotar o id.
2. `filmes criar` → informar `duracao_minutos` = `abc`: o terminal avisa e repete; depois valores válidos. Esperado: filme criado.
3. `salas criar` → informar o id do cinema, número, tipo de tela, duas fileiras (ex.: `A`/10 e `B`/12) → `s`. Esperado: sala com capacidade 22.
4. `sessoes criar` → ids de filme e sala, data/hora, idioma, preço `42.50`. Esperado: sessão criada; `preco_base` enviado como `"42.50"`.
5. `filmes atualizar <id>` → novos valores → confirmar. Esperado: filme atualizado.
6. `salas atualizar <id>` → só pede número, tipo de tela e ativo.
7. `filmes criar` e digitar `cancelar` no meio. Esperado: nenhuma requisição (aba Network vazia), prompt normal.
8. `filmes criar {"titulo":"x"}`. Esperado: erro orientando o novo uso; nada enviado.
9. `logout` seguido de `filmes criar`. Esperado: pede `login` antes de qualquer pergunta.
10. Após os passos, seta para cima no prompt: o histórico não contém as respostas dos diálogos.
11. Medir: o cadastro completo de um cinema (passo 1) leva menos de 1 minuto (SC-005).
12. `help`: exibe `criar` e `atualizar <id>` sem `<json>`.
