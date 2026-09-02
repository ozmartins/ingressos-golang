-- Bootstrap do banco `cinema`, executado uma única vez: o entrypoint da imagem
-- do PostgreSQL roda os arquivos deste diretório apenas quando o volume de
-- dados está vazio. Para reexecutá-lo, derrube o volume
-- (`docker compose down -v`) e suba de novo.
--
-- Os quatro serviços dividem uma instância e um banco, mas não se misturam:
-- cada um tem seu papel de login e é dono do schema homônimo, onde ficam suas
-- tabelas e também o `schema_migrations` da sua ferramenta de migração — que
-- em `public` seria disputado pelos quatro.

CREATE ROLE catalogo    LOGIN PASSWORD 'catalogo';
CREATE ROLE estoque     LOGIN PASSWORD 'estoque';
CREATE ROLE notificacao LOGIN PASSWORD 'notificacao';
CREATE ROLE pagamento   LOGIN PASSWORD 'pagamento';

-- Os schemas nascem aqui, e não nas migrações, porque o golang-migrate cria o
-- `schema_migrations` no primeiro schema do `search_path` ANTES de aplicar a
-- primeira migração — que é justamente quem faria o `CREATE SCHEMA`. As
-- migrações mantêm o seu `CREATE SCHEMA IF NOT EXISTS`, inofensivo aqui e
-- necessário quando rodam fora do compose.
CREATE SCHEMA catalogo    AUTHORIZATION catalogo;
CREATE SCHEMA estoque     AUTHORIZATION estoque;
CREATE SCHEMA notificacao AUTHORIZATION notificacao;
CREATE SCHEMA pagamento   AUTHORIZATION pagamento;

-- As migrações abrem com `CREATE SCHEMA IF NOT EXISTS <serviço>`, e o
-- PostgreSQL verifica o privilégio CREATE no banco antes de avaliar o
-- `IF NOT EXISTS` — sem este GRANT elas falhariam mesmo com o schema já criado
-- acima. O privilégio não é concedido a PUBLIC por padrão.
GRANT CREATE ON DATABASE cinema TO catalogo, estoque, notificacao, pagamento;

-- Ninguém escreve em `public`: o banco é compartilhado, e um objeto sem dono
-- claro ali viraria acoplamento acidental entre serviços.
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
