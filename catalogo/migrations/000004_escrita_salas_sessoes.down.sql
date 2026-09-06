DROP INDEX IF EXISTS catalogo.idx_salas_cinema_numero_ativa;

ALTER TABLE catalogo.sessoes
    DROP COLUMN IF EXISTS atualizado_em;

ALTER TABLE catalogo.salas
    DROP COLUMN IF EXISTS atualizado_em,
    DROP COLUMN IF EXISTS ativo;
