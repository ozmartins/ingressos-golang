# Data Model: Roteamento do Catálogo via chi

Não há mudança de dados: nenhuma tabela, coluna, migração, contrato REST/gRPC/AMQP ou
entidade de domínio é alterada. As entidades da spec (Rota, Tabela de rotas) já existem no
código e permanecem como estão:

- **Rota** (`internal/adapter/http/router.go`): `Metodo`, `Caminho`, `Documentada`,
  `Protegida` e o `handler` privado. Sem campos novos.
- **Tabela de rotas** (`var rotas`, exposta por `Rotas()`): a mesma, com os mesmos caminhos
  (`{id}` é sintaxe comum ao ServeMux e ao chi). Continua sendo a base do teste de paridade
  com o OpenAPI (`docs_test.go`).
