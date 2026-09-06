-- Carga dos filmes descritos em filmes_em_cartaz.json, filmes_em_breve.json e
-- filmes_fora_cartaz.json. Script avulso: não faz parte das migrations, rode
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
-- conjunto de 10 filmes, com os mesmos ids, a cada execução.
SET search_path TO catalogo;

BEGIN;

-- Ordem importa: primeiro as filhas, depois os filmes.
DELETE FROM sessoes;
DELETE FROM filmes;

INSERT INTO filmes (id, titulo, sinopse, duracao_minutos, classificacao_etaria, genero, imagem_url, status) VALUES
-- EM_CARTAZ (filmes_em_cartaz.json)
  ('84e1d004-6db0-53f9-9d4a-166964ff1b42', 'O Sorveteiro', 'Uma pacata cidade litorânea mergulha no caos quando um misterioso vendedor de sorvetes distribui doces amaldiçoados que transformam crianças em criaturas violentas.', 86, '18', 'Terror', 'https://placehold.co/500x750?text=O+Sorveteiro', 'EM_CARTAZ'),
  ('8b65b766-b508-5912-87a4-fc2bf47234c2', 'Idiotas', 'Um trabalho aparentemente simples rapidamente se transforma em uma viagem completamente fora de controle para dois homens encarregados de uma missão incomum.', 99, '18', 'Comédia', 'https://placehold.co/500x750?text=Idiotas', 'EM_CARTAZ'),
  ('5c0c96f1-b89d-5222-b449-9e248fb817e7', 'Pressão', 'Nas horas que antecedem o Dia D, um meteorologista precisa fornecer uma previsão capaz de determinar o destino de uma das maiores operações militares da Segunda Guerra Mundial.', 100, '14', 'Guerra, Histórico, Suspense', 'https://placehold.co/500x750?text=Pressao', 'EM_CARTAZ'),
-- BREVE (filmes_em_breve.json)
  ('859407cb-fc15-5186-8089-6d1a8b36795d', 'Código: Vingança', 'Após testemunhar o assassinato de seu chefe bilionário e ser acusado injustamente pelo crime, Cole Reed entra em uma jornada de vingança e descobre uma perigosa conspiração internacional.', 95, '16', 'Ação, Thriller', 'https://placehold.co/500x750?text=Codigo+Vinganca', 'BREVE'),
  ('5de239bb-116b-5383-a0fe-566f83326ecf', 'Da Magia à Sedução: Feitiço de Amor', 'Sally e Gillian Owens precisam confrontar novamente a antiga maldição de sua família quando forças sombrias ameaçam destruir definitivamente o clã.', 130, '12', 'Fantasia, Romance', 'https://placehold.co/500x750?text=Da+Magia+a+Seducao', 'BREVE'),
  ('618ff32a-657c-5cd8-82c0-96249773b63b', 'Coração Selvagem', 'Depois de sofrer um acidente de avião em uma região isolada do Alasca, um ex-soldado das Forças Especiais e seu cão precisam lutar para sobreviver e encontrar o caminho de volta para casa.', 101, '14', 'Aventura, Ação', 'https://placehold.co/500x750?text=Coracao+Selvagem', 'BREVE'),
-- FORA_DE_CARTAZ (filmes_fora_cartaz.json)
  ('364494ab-87e2-51ba-a15e-bcff2d81bb82', 'Mestres do Universo', 'Adam recupera a espada que o conecta ao planeta Eternia e retorna ao seu mundo natal para impedir que Esqueleto domine o reino.', 140, '14', 'Ação, Aventura, Fantasia, Ficção Científica', 'https://placehold.co/500x750?text=Mestres+do+Universo', 'FORA_DE_CARTAZ'),
  ('d9f25619-23c7-53eb-ae3d-a79f431b615d', 'Todo Mundo em Pânico', 'Mais de duas décadas depois, Cindy, Brenda, Shorty e Ray acabam novamente envolvidos em uma confusão com assassinos, monstros e fenômenos sobrenaturais.', 96, '18', 'Comédia, Terror', 'https://placehold.co/500x750?text=Todo+Mundo+em+Panico', 'FORA_DE_CARTAZ'),
  ('1eae7a62-eb3e-5d25-a80c-4bfd27c2660e', 'Minions & Monstros', 'Depois de libertarem acidentalmente criaturas monstruosas durante sua passagem por Hollywood, os Minions precisam se unir para impedir que os monstros destruam o mundo.', 90, '10', 'Animação, Aventura, Comédia, Família', 'https://placehold.co/500x750?text=Minions+e+Monstros', 'FORA_DE_CARTAZ'),
  ('f7143722-6e72-533d-be6f-090f419d84b4', 'Supergirl', 'Kara Zor-El enfrenta os traumas da destruição de Krypton enquanto se envolve na busca de uma jovem alienígena por vingança contra o mercenário responsável pela morte de seu pai.', 108, '14', 'Ação, Aventura, Ficção Científica', 'https://placehold.co/500x750?text=Supergirl', 'FORA_DE_CARTAZ');

COMMIT;
