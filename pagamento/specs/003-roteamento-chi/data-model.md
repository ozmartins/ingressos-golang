# Data Model: Roteamento do Pagamento via chi

Sem mudança de modelo de dados, esquema ou migrações.

A única "entidade" da feature é a **tabela de rotas** (método + caminho → tratamento), que continua definida num único lugar, `API.Rotas()`:

| Método | Caminho | Tratamento | Autenticação |
|--------|---------|------------|--------------|
| GET | `/api/v1/pagamentos/reserva/{reserva_id}` | `consultar` | sim (no handler) |
| POST | `/api/v1/pagamentos/reserva/{reserva_id}` | `escolherForma` | sim (no handler) |
| GET | `/api/v1/health/live` | `vivo` | não |
| GET | `/api/v1/health/ready` | `pronto` | não |
| GET | `/openapi.yaml` | `openapi.HandlerEspecificacao` | não |
| GET | `/docs`, `/docs/` | `openapi.HandlerUI` | não |

A autenticação continua dentro de cada handler (`a.Auth.Identificar`); não é movida para middleware (Constituição I).
