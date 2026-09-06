ALTER TABLE estoque.reservas
    DROP CONSTRAINT IF EXISTS ck_reserva_pendente_tem_valor,
    DROP CONSTRAINT IF EXISTS ck_reserva_valor_positivo,
    DROP COLUMN IF EXISTS valor_total;
