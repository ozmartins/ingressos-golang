-- A remoção de um cinema é lógica, como a de um filme: `salas` referencia
-- `cinemas`, e `sessoes` referencia `salas`, então apagar a linha desmontaria a
-- grade já gravada. Desativar tira o cinema da listagem e preserva o histórico.
--
-- `atualizado_em` chega junto porque a escrita passa a existir: até agora a
-- tabela só era lida, e `criado_em` bastava.
--
-- Sem índice novo: `idx_cinemas_nome_id` já serve à ordenação, e o filtro por
-- `ativo` seleciona quase todas as linhas — o mesmo raciocínio registrado em
-- 000002 para o `status` do filme.
ALTER TABLE catalogo.cinemas
    ADD COLUMN ativo BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN atualizado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;
