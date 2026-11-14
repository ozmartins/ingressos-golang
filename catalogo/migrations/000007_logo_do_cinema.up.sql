-- Logo do cinema: apenas o endereço (URL) da imagem, hospedada fora do sistema.
--
-- Coluna opcional, sem default: as linhas existentes ficam com NULL ("sem
-- logo") e, por ser nulável e sem default, o ALTER não reescreve a tabela.
-- O limite de 2048 caracteres espelha a validação do domínio.
ALTER TABLE catalogo.cinemas
    ADD COLUMN logo_url VARCHAR(2048);
