DELETE FROM catalogo.sessoes;
DELETE FROM catalogo.salas;

INSERT INTO catalogo.cinemas
    (id, nome, cidade, estado, endereco)
VALUES
    ('6d5cf2b8-f92a-4d83-991c-87e8e69054f1', 'GNC Cinemas - Nações Shopping', 'Criciúma', 'SC', 'Av. Jorge Elias de Lucca, 765 - Nossa Senhora da Salete, Criciúma - SC, 88813-901'),
    ('81e54cc7-3329-4a8e-a0d5-f83a1e214ed8', 'Cine Uniplex Criciúma', 'Criciúma', 'SC', 'Av. Gabriel Zanette, 1455 - Próspera, Criciúma - SC, 88815-060'),
    ('f24b1d63-9274-462e-a0bf-781be50a76a4', 'Cine Della', 'Criciúma', 'SC', 'Praça Dr. Nereu Ramos, 364 - Centro, Criciúma - SC, 88801-505'),
    ('362acc87-d3a4-4c9e-9f13-d8f31ab20e0d', 'Cine Show Tubarão', 'Tubarão', 'SC', 'Av. Marcolino Martins Cabral, 2525 - Aeroporto, Tubarão - SC, 88705-003'),
    ('aa597de4-9908-4427-97e6-68401af5a33e', 'Arcoplex Cinemas - Center Shopping Araranguá', 'Araranguá', 'SC', 'Av. Sete de Setembro, 705 - Cidade Alta, Araranguá - SC, 88901-004');