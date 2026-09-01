-- Catálogo de exemplo para desenvolvimento e testes.
-- Inclui casos de borda: filme sem sinopse/imagem, sessões canceladas e
-- finalizadas, e volume suficiente para exercitar paginação.
--
-- As tabelas vivem no schema `catalogo`, então o arquivo aponta o `search_path`
-- para ele: assim `psql -f` funciona sem qualificar cada INSERT.
SET search_path TO catalogo;

INSERT INTO filmes (id, titulo, sinopse, duracao_minutos, classificacao_etaria, genero, imagem_url, status) VALUES
 ('c394c8b3-76a1-4328-b803-02f5923b7a15','Duna: Parte 2','Paul Atreides se une a Chani e aos Fremen...',166,'14 anos','Ficção Científica','https://cdn.cinema.com/posters/duna2.jpg','EM_CARTAZ'),
 ('a1b2c3d4-0000-4000-8000-000000000002','Aurora Boreal',NULL,100,'Livre','Drama',NULL,'BREVE'),
 ('a1b2c3d4-0000-4000-8000-000000000003','Zebra Selvagem','Documentário sobre a savana.',90,'Livre','Documentário','https://cdn.cinema.com/posters/zebra.jpg','EM_CARTAZ'),
 ('a1b2c3d4-0000-4000-8000-000000000004','Filme Retirado','Saiu de cartaz.',120,'16 anos','Ação',NULL,'FORA_DE_CARTAZ');

INSERT INTO cinemas (id, nome, cidade, estado, endereco) VALUES
 ('b1b2c3d4-0000-4000-8000-000000000001','CineMark - Shopping Centro','Florianópolis','SC','Rua Felipe Schmidt, 100'),
 ('b1b2c3d4-0000-4000-8000-000000000002','Arte Cine - Beiramar','Florianópolis','SC','Av. Beira-Mar Norte, 2000');

INSERT INTO salas (id, cinema_id, numero, tipo_tela, capacidade_total) VALUES
 ('d1b2c3d4-0000-4000-8000-000000000001','b1b2c3d4-0000-4000-8000-000000000001',1,'2D',120),
 ('d1b2c3d4-0000-4000-8000-000000000002','b1b2c3d4-0000-4000-8000-000000000001',3,'IMAX',300),
 ('d1b2c3d4-0000-4000-8000-000000000003','b1b2c3d4-0000-4000-8000-000000000002',1,'VIP',60);

INSERT INTO sessoes (id, filme_id, sala_id, data_hora_inicio, idioma, preco_base, status) VALUES
 ('e1b2c3d4-0000-4000-8000-000000000001','c394c8b3-76a1-4328-b803-02f5923b7a15','d1b2c3d4-0000-4000-8000-000000000002','2026-09-01T20:30:00Z','LEGENDADO',42.00,'AGENDADA'),
 ('e1b2c3d4-0000-4000-8000-000000000002','c394c8b3-76a1-4328-b803-02f5923b7a15','d1b2c3d4-0000-4000-8000-000000000001','2026-09-01T22:00:00Z','DUBLADO',32.50,'AGENDADA'),
 ('e1b2c3d4-0000-4000-8000-000000000003','a1b2c3d4-0000-4000-8000-000000000003','d1b2c3d4-0000-4000-8000-000000000003','2026-09-02T18:00:00Z','DUBLADO',55.00,'AGENDADA'),
 ('e1b2c3d4-0000-4000-8000-000000000004','a1b2c3d4-0000-4000-8000-000000000003','d1b2c3d4-0000-4000-8000-000000000001','2026-09-02T20:00:00Z','LEGENDADO',32.50,'AGENDADA'),
 ('e1b2c3d4-0000-4000-8000-000000000005','c394c8b3-76a1-4328-b803-02f5923b7a15','d1b2c3d4-0000-4000-8000-000000000002','2026-09-03T15:00:00Z','LEGENDADO',42.00,'CANCELADA'),
 ('e1b2c3d4-0000-4000-8000-000000000006','c394c8b3-76a1-4328-b803-02f5923b7a15','d1b2c3d4-0000-4000-8000-000000000001','2026-08-01T15:00:00Z','DUBLADO',32.50,'FINALIZADA'),
 ('e1b2c3d4-0000-4000-8000-000000000007','a1b2c3d4-0000-4000-8000-000000000003','d1b2c3d4-0000-4000-8000-000000000003','2026-09-04T19:00:00Z','DUBLADO',55.00,'EM_ANDAMENTO');
