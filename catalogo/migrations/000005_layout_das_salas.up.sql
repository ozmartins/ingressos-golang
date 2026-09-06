-- A sala passa a declarar sua planta. Até aqui ela guardava só um inteiro,
-- `capacidade_total`, digitado à mão e que nada conferia; agora guarda as
-- fileiras, e a capacidade é a soma delas — um número só, sem como divergir.
--
-- O layout é uma coluna JSONB, e não uma tabela filha. Ele é sempre lido e
-- escrito inteiro junto da sala, e nunca consultado por si: não há consulta que
-- a normalização sirva, e a tabela custaria transação na escrita e um segundo
-- SELECT na listagem paginada.
ALTER TABLE catalogo.salas ADD COLUMN layout JSONB;

-- As salas já gravadas não têm planta que ninguém tenha desenhado, e o layout é
-- obrigatório. A tradução mais honesta é uma fileira só, `A`, com a capacidade
-- inteira — preserva o número exato e não inventa geometria.
--
-- Uma fileira `A` de 180 lugares não é uma planta de verdade: cada sala
-- existente precisa ser redesenhada por `PUT /api/v1/salas/{id}` antes de o
-- layout valer alguma coisa para quem for materializar as poltronas.
UPDATE catalogo.salas
   SET layout = jsonb_build_array(
           jsonb_build_object('fileira', 'A', 'assentos', capacidade_total, 'tipo', 'NORMAL'));

ALTER TABLE catalogo.salas ALTER COLUMN layout SET NOT NULL;

ALTER TABLE catalogo.salas DROP COLUMN capacidade_total;
