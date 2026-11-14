-- Carga dos filmes em cartaz lidos das páginas salvas do AdoroCinema
-- (~/Downloads/cinemas). Script avulso: não faz parte das migrations, rode
-- quando quiser, quantas vezes quiser.
--
--   psql "$DATABASE_URL" -f scripts/seed_filmes.sql
--
-- ATENÇÃO: o script ZERA o catálogo antes de inserir. Ele apaga TODOS os
-- filmes já existentes e, junto, as sessões que dependem deles (a FK
-- sessoes.filme_id impede apagar filme com sessão pendurada). Tudo roda numa
-- transação: ou apaga e insere, ou não faz nada.
--
-- Os IDs são UUIDv5 derivados do título, então o resultado é sempre o mesmo
-- conjunto de 13 filmes, com os mesmos ids, a cada execução.
--
-- As páginas só informam título, direção e elenco (a sinopse guarda os dois
-- últimos). Duração, classificação e gênero são obrigatórios no catálogo e não
-- constam da fonte: ficam com valores padrão (90 min, 'Não informada', 'Não
-- informado') até serem corrigidos pela API (PUT /api/v1/filmes/{id}).
--
-- `imagem_url` aponta para os pôsteres em imagens-cinemas/filmes/, servidos pelo
-- catálogo em GET /imagens/filmes/<arquivo> (porta padrão do compose: 8082).
SET search_path TO catalogo;

BEGIN;

-- Ordem importa: primeiro as filhas, depois os filmes.
DELETE FROM sessoes;
DELETE FROM filmes;

INSERT INTO filmes (id, titulo, sinopse, duracao_minutos, classificacao_etaria, genero, imagem_url, status) VALUES
  ('32dee210-f9f3-5091-84e9-6ae1f44be96b', 'A Virada: Ignição da Alma', 'Direção: Alessandro Barros. Elenco: Mario Bregieira, Pérola Faria, Gui Tavares.', 90, 'Não informada', 'Não informado', 'http://localhost:8082/imagens/filmes/a-virada-ignicao-da-alma.jpg', 'EM_CARTAZ'),
  ('2dcebe56-e4ea-5778-aa85-6c3b93d944ce', 'Apenas uma Ilusão', 'Direção: Olivier Nakache, Eric Toledano. Elenco: Louis Garrel, Camille Cottin, Pierre Lottin.', 90, 'Não informada', 'Não informado', 'http://localhost:8082/imagens/filmes/apenas-uma-ilusao.jpg', 'EM_CARTAZ'),
  ('4f2966d9-aa4b-50bc-b96e-a1d2aec74f0e', 'Boiúna: Terror no Amazonas', 'Direção: Mike P. Nelson. Elenco: Kiana Madeira, Jessica Rothe, Logan Marshall-Green.', 90, 'Não informada', 'Não informado', 'http://localhost:8082/imagens/filmes/boiuna-terror-no-amazonas.jpg', 'EM_CARTAZ'),
  ('63af20a3-f387-5de1-95c4-3ccffc6027d9', 'Cara-de-Barro', 'Direção: James Watkins. Elenco: Tom Rhys Harries, Naomi Ackie, Eddie Marsan.', 90, 'Não informada', 'Não informado', 'http://localhost:8082/imagens/filmes/cara-de-barro.png', 'EM_CARTAZ'),
  ('cb6ba173-8c6d-5abb-a460-3aec00641dbb', 'Contos Paralelos', 'Direção: Asghar Farhadi. Elenco: Isabelle Huppert, Virginie Efira, Pierre Niney.', 90, 'Não informada', 'Não informado', 'http://localhost:8082/imagens/filmes/contos-paralelos.jpg', 'EM_CARTAZ'),
  ('eebe2b41-9a3e-5162-83fc-589b13580970', 'Deixe-me Viver', 'Direção: Walther Neto. Elenco: Mônica Carvalho, Cat Dantas, Humberto Martins.', 90, 'Não informada', 'Não informado', 'http://localhost:8082/imagens/filmes/deixe-me-viver.jpg', 'EM_CARTAZ'),
  ('a873c49f-5665-5dc7-be9e-a00addfc0929', 'Ebenezer e os Fantasmas do Natal', 'Direção: Ti West. Elenco: Johnny Depp, Rupert Grint, Daisy Ridley.', 90, 'Não informada', 'Não informado', 'http://localhost:8082/imagens/filmes/ebenezer-e-os-fantasmas-do-natal.jpg', 'EM_CARTAZ'),
  ('5febf49d-4bac-5fa3-a86f-9847324fcd83', 'Godzilla Minus Zero', 'Direção: Takashi Yamazaki. Elenco: Ryûnosuke Kamiki, Minami Hamabe, Yamada Yuki.', 90, 'Não informada', 'Não informado', 'http://localhost:8082/imagens/filmes/godzilla-minus-zero.png', 'EM_CARTAZ'),
  ('d13195d3-6ca9-5bdf-9e66-2111916a85fd', 'Lago dos Ossos', 'Direção: Mercedes Bryce Morgan. Elenco: Maddie Hasson, Alex Roe, Marco Pigossi.', 90, 'Não informada', 'Não informado', 'http://localhost:8082/imagens/filmes/lago-dos-ossos.jpg', 'EM_CARTAZ'),
  ('c0fc662d-e8f7-5a89-8fdd-a239940c8d2f', 'O Bosque Selvagem', 'Direção: Travis Knight. Elenco: Peyton Elizabeth Lee, Jacob Tremblay, Mahershala Ali.', 90, 'Não informada', 'Não informado', 'http://localhost:8082/imagens/filmes/o-bosque-selvagem.jpg', 'EM_CARTAZ'),
  ('62cd1cbd-f755-5a76-8a67-dce8c4c261d6', 'Pinóquio: Maldição de Madeira', 'Direção: Rhys Frake-Waterfield. Elenco: Robert Englund, Richard Brake, Jude Evan Lloyd.', 90, 'Não informada', 'Não informado', 'http://localhost:8082/imagens/filmes/pinoquio-maldicao-de-madeira.jpg', 'EM_CARTAZ'),
  ('54ff2ab3-6423-5aa6-860d-af95dda1bf3a', 'Só Não Posso Dizer o Nome', 'Direção: Helvécio Ratton. Elenco: Lara Martinelli, Bruno Matos, Laura Del Picchia.', 90, 'Não informada', 'Não informado', 'http://localhost:8082/imagens/filmes/so-nao-posso-dizer-o-nome.jpg', 'EM_CARTAZ'),
  ('9714c41b-5084-5dde-b418-dca42d0ed0df', 'Verity', 'Direção: Michael Showalter. Elenco: Anne Hathaway, Dakota Johnson, Josh Hartnett.', 90, 'Não informada', 'Não informado', 'http://localhost:8082/imagens/filmes/verity.png', 'EM_CARTAZ');

COMMIT;
