-- Derruba o schema inteiro: as tabelas da notificação não existem fora dele,
-- então não há o que preservar depois de removê-las.
DROP SCHEMA IF EXISTS notificacao CASCADE;
