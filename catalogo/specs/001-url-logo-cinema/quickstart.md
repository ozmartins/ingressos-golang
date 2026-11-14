# Quickstart: validar o logo do cinema

Pré-requisitos: `cd ingressos-golang && docker compose up --build`; token Keycloak conforme `URLS.txt` (Swagger em `/docs`).

1. **Migração**: no `catalogo/`, `make migrate-up` (com `DATABASE_URL`); confirmar coluna `catalogo.cinemas.logo_url`. `make migrate-down` remove a coluna.
2. **Criar com logo**: `POST /api/v1/cinemas` com `logo_url` válida → 201 com o mesmo valor; `GET /api/v1/cinemas/{id}` devolve-o.
3. **Sem logo**: criar sem o campo → resposta com `"logo_url": null`.
4. **Editar**: `PUT` com nova URL → atualizada; `PUT` sem o campo → `null`.
5. **Inválidas**: `"javascript:alert(1)"`, `"ftp://x"`, `"nao e url"`, 2049 caracteres → 400.
6. **Legado**: cinemas anteriores listados com `logo_url: null`.
7. **Testes**: `make test` e `make test-integration`; `make lint`.
