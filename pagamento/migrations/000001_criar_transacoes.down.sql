-- Derruba o schema inteiro: a tabela do pagamento não existe fora dele, então
-- não há o que preservar depois de removê-la.
DROP SCHEMA IF EXISTS pagamento CASCADE;
