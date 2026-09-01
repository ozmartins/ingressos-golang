-- Derruba o schema inteiro: as tabelas do catálogo não existem fora dele, então
-- não há o que preservar depois de removê-las.
DROP SCHEMA IF EXISTS catalogo CASCADE;
