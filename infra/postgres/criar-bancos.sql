-- Bootstrap do PostgreSQL da máquina hospedeira, uma vez por instalação.
--
-- Os serviços não sobem um PostgreSQL próprio: cada um se conecta ao Postgres
-- já instalado no computador, mas continua dono do seu banco — usuário, base e
-- senha com o mesmo nome do serviço. Bancos distintos (e não apenas schemas
-- distintos) mantêm um `public.schema_migrations` por serviço; num banco
-- compartilhado as quatro ferramentas de migração disputariam a mesma tabela de
-- controle.
--
-- Rodar como superusuário:
--   psql -U postgres -f infra/postgres/criar-bancos.sql

CREATE ROLE catalogo    LOGIN PASSWORD 'catalogo';
CREATE ROLE estoque     LOGIN PASSWORD 'estoque';
CREATE ROLE notificacao LOGIN PASSWORD 'notificacao';
CREATE ROLE pagamento   LOGIN PASSWORD 'pagamento';

CREATE DATABASE catalogo    OWNER catalogo;
CREATE DATABASE estoque     OWNER estoque;
CREATE DATABASE notificacao OWNER notificacao;
CREATE DATABASE pagamento   OWNER pagamento;
