#!/usr/bin/env bash
# Gera um JWT HS256 assinado com a chave de teste, para o roteiro de quickstart.md.
# NÃO serve para nenhum ambiente real: o serviço em produção valida por JWKS
# contra o Keycloak, e não aceita esta chave.
set -euo pipefail

SUB="${1:-}"
if [[ -z "$SUB" ]]; then
  echo "uso: $0 <uuid-do-usuario>" >&2
  exit 1
fi

SEGREDO="${JWT_SEGREDO_TESTE:-chave-de-teste}"
ISS="${JWT_ISSUER:-https://keycloak.teste/realms/cinema}"
AUD="${JWT_AUDIENCE:-conta-cinema}"
EXP=$(( $(date +%s) + 3600 ))

b64() { openssl base64 -e -A | tr '+/' '-_' | tr -d '='; }

CAB=$(printf '{"alg":"HS256","typ":"JWT"}' | b64)
COR=$(printf '{"sub":"%s","iss":"%s","aud":"%s","exp":%s}' "$SUB" "$ISS" "$AUD" "$EXP" | b64)
ASS=$(printf '%s.%s' "$CAB" "$COR" | openssl dgst -sha256 -hmac "$SEGREDO" -binary | b64)

printf '%s.%s.%s\n' "$CAB" "$COR" "$ASS"
