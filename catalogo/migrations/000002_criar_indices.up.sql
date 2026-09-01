-- Índices de ORDENAÇÃO: começam pela chave do ORDER BY.
--
-- São eles que tornam a página proporcional ao seu tamanho, e não ao acervo.
-- O filtro por situação seleciona quase todas as linhas (praticamente todo filme
-- está em cartaz ou em breve), então um índice que comece por `status` não ajuda
-- a ordenar: o planejador o descarta e varre a tabela. Começando pela chave de
-- ordenação, o banco percorre o índice em ordem, aplica o filtro em cada linha e
-- para nas 20 primeiras que passarem.
CREATE INDEX idx_filmes_titulo_id ON catalogo.filmes (titulo, id);
CREATE INDEX idx_cinemas_nome_id ON catalogo.cinemas (nome, id);
CREATE INDEX idx_salas_cinema_numero_id ON catalogo.salas (cinema_id, numero, id);
CREATE INDEX idx_sessoes_inicio_id ON catalogo.sessoes (data_hora_inicio, id);

-- Índices de FILTRO: servem às consultas seletivas e à contagem do total.
--
-- Quando o filtro é seletivo (uma sessão de um filme, as salas de um cinema), é
-- ele que evita a varredura. Na contagem, permite varredura só do índice, que lê
-- muito menos páginas que a tabela.
CREATE INDEX idx_filmes_status ON catalogo.filmes (status);
CREATE INDEX idx_sessoes_status_inicio ON catalogo.sessoes (status, data_hora_inicio);
CREATE INDEX idx_sessoes_filme_inicio ON catalogo.sessoes (filme_id, data_hora_inicio);
CREATE INDEX idx_sessoes_sala ON catalogo.sessoes (sala_id);
