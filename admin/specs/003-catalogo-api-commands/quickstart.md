# Quickstart: Comandos do Terminal para a API Servico-Catalogo

**Pré-requisitos**: Node.js 22.12+ e npm; plataforma no ar (`cd ingressos-golang && docker compose up`) com o catálogo em `http://localhost:8082` e o Keycloak em `http://localhost:8081`. Execute em `ingressos-golang/web/`.

1. `npm install`
2. `npm test` — todos os testes passam.
3. `npm run build` — código de saída 0.
4. `npm run dev` e abrir `http://localhost:5173` (o proxy do Vite encaminha `/catalogo` e `/identidade`).

## Verificação manual

| # | Comando | Esperado |
|---|---------|----------|
| 1 | `help` | Lista os comandos de API agrupados por controller, mais `login` e `logout` |
| 2 | `health` | `status: ok` e o estado do banco |
| 3 | `filmes listar page_size=1` | Um filme e a paginação (página, tamanho, total, tem_proxima) |
| 4 | `filmes listar status=EM_CARTAZ` | Apenas filmes em cartaz |
| 5 | `filmes obter <id do passo 3>` | Dados do filme |
| 6 | `filmes obter 00000000-0000-0000-0000-000000000000` | Erro 404 com título e detalhe |
| 7 | `cinemas listar`, `salas listar`, `sessoes listar` | Listas paginadas |
| 8 | `filmes criar {"titulo":"Teste"}` (sem login) | Orientação para usar `login`; nenhuma requisição enviada |
| 9 | `login <client_id> <client_secret>` (credenciais de desenvolvimento descritas em `securitySchemes` do `openapi.yaml`) | Confirmação; o segredo aparece como `****` no eco e **não** no histórico (↑) |
| 10 | `filmes criar {"titulo":"Teste","duracao_minutos":90,"classificacao_etaria":"Livre","genero":"Drama"}` | 201, filme criado e `Location` |
| 11 | `filmes criar {"titulo":""}` | Erro 400 com erros por campo |
| 12 | `filmes criar {titulo` | Erro de sintaxe JSON, sem requisição |
| 13 | `filmes atualizar <id> {...}` | Filme atualizado |
| 14 | `filmes remover <id>` | `Removido com sucesso` |
| 15 | `salas criar` repetindo o número de sala de um cinema | Erro 409 de conflito |
| 16 | `logout` e repetir o passo 8 | Volta a exigir login |
| 17 | Parar o catálogo e executar `filmes listar` | Mensagem de serviço indisponível; o terminal continua usável |
| 18 | `filmes` / `filmes xpto` / `filmes obter` | Ações válidas / sintaxe de uso |
