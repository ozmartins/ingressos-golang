CREATE ROLE catalogo    LOGIN PASSWORD 'catalogo';
CREATE ROLE estoque     LOGIN PASSWORD 'estoque';
CREATE ROLE notificacao LOGIN PASSWORD 'notificacao';
CREATE ROLE pagamento   LOGIN PASSWORD 'pagamento';

CREATE SCHEMA catalogo    AUTHORIZATION catalogo;
CREATE SCHEMA estoque     AUTHORIZATION estoque;
CREATE SCHEMA notificacao AUTHORIZATION notificacao;
CREATE SCHEMA pagamento   AUTHORIZATION pagamento;

GRANT CREATE ON DATABASE cinema TO catalogo, estoque, notificacao, pagamento;

REVOKE CREATE ON SCHEMA public FROM PUBLIC;
