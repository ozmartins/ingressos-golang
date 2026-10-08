# Data Model: Roteamento do Estoque via chi

Sem mudança. Nenhuma tabela, migração, entidade de domínio, evento ou contrato é
tocado. As únicas "entidades" da spec — **Rota** e **Superfície HTTP** — são estruturas
do adaptador:

| Superfície | Porta (env) | Rotas |
|------------|-------------|-------|
| API REST | `HTTPAddr` | `POST /api/v1/sessoes/{sessao_id}/bloqueios` (auth), `GET /api/v1/sessoes/{sessao_id}/poltronas` (auth), `GET /openapi.yaml`, `GET /docs`, `GET /docs/*` |
| Administração | `AdminAddr` | `GET /health/live`, `GET /health/ready` |
