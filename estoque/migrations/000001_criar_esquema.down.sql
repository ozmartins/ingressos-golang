-- Derruba o schema inteiro: as tabelas do estoque não existem fora dele, então
-- não há o que preservar depois de removê-las.
DROP SCHEMA IF EXISTS estoque CASCADE;
