# Contrato consumido: serviço `estoque`

Base no navegador: `/estoque` (proxy → `http://localhost:8085`). Cabeçalho obrigatório: `Authorization: Bearer <access_token>` (Keycloak realm `cinema`, aud `cinema-app`). Fonte: `estoque/internal/adapter/http/openapi/openapi.yaml`.

## GET `/api/v1/sessoes/{sessao_id}/poltronas`
- 200 `{ sessao_id, poltronas: [{ rotulo, fileira, numero, tipo: NORMAL|PCD|NAMORADEIRA, status: LIVRE|RESERVADA|OCUPADA }] }`
- 401 token ausente/expirado → refazer login com retorno
- 404 sessão desconhecida no estoque → "sessão indisponível"
- 503/500 → erro com tentar novamente

## POST `/api/v1/sessoes/{sessao_id}/bloqueios`
- Corpo: `{ "poltronas_ids": ["B4","B5"] }` (1–10, sem repetição)
- 201 `{ reserva_id, expira_em, mensagem }`
- 400 formato/limite (`metadata.limite`) · 401 · 404 `sessao-nao-encontrada` · 409 poltrona RESERVADA/OCUPADA · 422 `sessao-nao-reservavel`/sem matriz/rótulo inexistente · 502 · 503
- Erros em `application/problem+json`; o cliente decide pelo **status** e pelo último segmento de `type`, nunca por `title`/`detail`.

## Consumido do `catalogo` (já existente, sem mudança)
`GET /api/v1/cinemas/{id}`, `GET /api/v1/filmes`, `GET /api/v1/sessoes?cinema_id=` (via `src/api/catalogo.ts`).
