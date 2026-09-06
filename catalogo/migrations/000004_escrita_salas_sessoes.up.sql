-- A escrita chega às salas e às sessões, e com ela as colunas que só a escrita
-- justifica: até agora as duas tabelas eram apenas lidas, e `criado_em` bastava.
--
-- A remoção de uma sala é lógica, como a de um cinema: `sessoes` referencia
-- `salas`, então apagar a linha desmontaria a grade já gravada. A sessão não
-- precisa de coluna nova: `status = 'CANCELADA'` já a tira da grade, pela mesma
-- lista de status visíveis que o serviço consulta.
ALTER TABLE catalogo.salas
    ADD COLUMN ativo BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN atualizado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE catalogo.sessoes
    ADD COLUMN atualizado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

-- O número identifica a sala para o espectador — é ele que aparece na grade —,
-- então dois "3" no mesmo cinema tornariam a grade ambígua.
--
-- O índice é PARCIAL de propósito: a restrição vale entre as salas ativas. Uma
-- sala desativada libera o número para a sala que a substitui, e o histórico
-- dela permanece.
--
-- Sem índice novo para a listagem: `idx_salas_cinema_numero_id` já serve à
-- ordenação, e o filtro por `ativo` seleciona quase todas as linhas — o mesmo
-- raciocínio registrado em 000002 para o `status` do filme.
CREATE UNIQUE INDEX idx_salas_cinema_numero_ativa
    ON catalogo.salas (cinema_id, numero) WHERE ativo;
