# Contrato: campo `logo_url` em Cinema

Fonte de verdade: `internal/adapter/http/openapi/openapi.yaml`.

## Schema `Cinema` (resposta de GET /cinemas, GET /cinemas/{id}, POST, PUT)
- `logo_url`: `string | null`, `format: uri`, `maxLength: 2048`, `nullable: true`. O servidor sempre emite a chave (`null` quando não há logo). No schema, **fica fora de `required`**, para não quebrar clientes e mocks que não conhecem o campo.

## Schema `CinemaEntrada` (corpo de POST /cinemas e PUT /cinemas/{id})
- `logo_url`: `string | null`, opcional, `maxLength: 2048`. Aparado; `""`/`null`/ausente = sem logo.
- Inválida (não http/https, malformada, > 2048): `400` Problem Details com erro de validação citando `logo_url`.
- PUT é substituição completa: ausente remove o logo existente.

## Exemplos
```json
POST /api/v1/cinemas
{"nome":"Cine X","cidade":"Florianópolis","estado":"SC","endereco":"Rua X, 100","logo_url":"https://cdn.exemplo.com/cinex.png"}
→ 201 {"id":"...","nome":"Cine X",...,"ativo":true,"logo_url":"https://cdn.exemplo.com/cinex.png"}
```
