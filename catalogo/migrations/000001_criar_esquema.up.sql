-- O catálogo é dono do seu próprio schema: nada dele vive em `public`.
--
-- Objetos aqui são sempre qualificados porque esta migração roda pelo CLI do
-- golang-migrate, cujo `search_path` é o padrão da conexão. A tabela de
-- controle de versões (`schema_migrations`) segue em `public`: ela é do
-- ferramental de migração, não do domínio.
CREATE SCHEMA IF NOT EXISTS catalogo;

CREATE TABLE catalogo.filmes (
    id VARCHAR(36) PRIMARY KEY,
    titulo VARCHAR(255) NOT NULL,
    sinopse TEXT,
    duracao_minutos INT NOT NULL,
    classificacao_etaria VARCHAR(50) NOT NULL,
    genero VARCHAR(100) NOT NULL,
    imagem_url VARCHAR(500),
    status VARCHAR(50) NOT NULL DEFAULT 'EM_CARTAZ',
    criado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    atualizado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE catalogo.cinemas (
    id VARCHAR(36) PRIMARY KEY,
    nome VARCHAR(255) NOT NULL,
    cidade VARCHAR(100) NOT NULL,
    estado VARCHAR(2) NOT NULL,
    endereco TEXT NOT NULL,
    criado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE catalogo.salas (
    id VARCHAR(36) PRIMARY KEY,
    cinema_id VARCHAR(36) NOT NULL REFERENCES catalogo.cinemas(id),
    numero INT NOT NULL,
    tipo_tela VARCHAR(50) NOT NULL,
    capacidade_total INT NOT NULL,
    criado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE catalogo.sessoes (
    id VARCHAR(36) PRIMARY KEY,
    filme_id VARCHAR(36) NOT NULL REFERENCES catalogo.filmes(id),
    sala_id VARCHAR(36) NOT NULL REFERENCES catalogo.salas(id),
    data_hora_inicio TIMESTAMP WITH TIME ZONE NOT NULL,
    idioma VARCHAR(50) NOT NULL,
    preco_base DECIMAL(10, 2) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'AGENDADA',
    criado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
