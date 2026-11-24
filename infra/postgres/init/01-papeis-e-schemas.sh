#!/bin/sh
# Cria um papel e um schema por serviço. As senhas vêm do ambiente do container
# (docker-compose.yml, a partir do .env); nada de senha fica neste arquivo.
# Só roda quando o volume do Postgres é criado do zero.
set -eu

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
  -v pw_catalogo="$CATALOGO_DB_PASSWORD" \
  -v pw_estoque="$ESTOQUE_DB_PASSWORD" \
  -v pw_notificacao="$NOTIFICACAO_DB_PASSWORD" \
  -v pw_pagamento="$PAGAMENTO_DB_PASSWORD" <<'SQL'
CREATE ROLE catalogo    LOGIN PASSWORD :'pw_catalogo';
CREATE ROLE estoque     LOGIN PASSWORD :'pw_estoque';
CREATE ROLE notificacao LOGIN PASSWORD :'pw_notificacao';
CREATE ROLE pagamento   LOGIN PASSWORD :'pw_pagamento';

CREATE SCHEMA catalogo    AUTHORIZATION catalogo;
CREATE SCHEMA estoque     AUTHORIZATION estoque;
CREATE SCHEMA notificacao AUTHORIZATION notificacao;
CREATE SCHEMA pagamento   AUTHORIZATION pagamento;

GRANT CREATE ON DATABASE cinema TO catalogo, estoque, notificacao, pagamento;

REVOKE CREATE ON SCHEMA public FROM PUBLIC;
SQL
