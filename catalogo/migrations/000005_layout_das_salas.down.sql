-- A capacidade volta a ser coluna, somada a partir do layout que está saindo:
-- o número sobrevive à volta, a planta não. Redesenhar as salas depois de um
-- `down` é trabalho manual, como era antes desta migração.
ALTER TABLE catalogo.salas ADD COLUMN capacidade_total INT;

UPDATE catalogo.salas
   SET capacidade_total = (
           SELECT COALESCE(SUM((fileira ->> 'assentos')::INT), 0)
             FROM jsonb_array_elements(layout) AS fileira);

ALTER TABLE catalogo.salas ALTER COLUMN capacidade_total SET NOT NULL;

ALTER TABLE catalogo.salas DROP COLUMN layout;
