-- O catálogo passa a anunciar a sessão criada, e o anúncio precisa sobreviver ao
-- processo. A caixa de saída é gravada na mesma transação que insere a sessão:
-- ou as duas coisas acontecem, ou nenhuma. Publicar direto no broker perderia o
-- fato se o processo morresse entre a escrita e a publicação, e publicar dentro
-- da requisição colocaria a latência do broker no orçamento do cliente.
--
-- A tabela é a mesma do Servico-Estoque, de propósito: os dois resolvem o mesmo
-- problema, e duas formas diferentes só custariam a quem lê os dois.
CREATE TABLE catalogo.outbox_eventos (
    id            BIGSERIAL    PRIMARY KEY,
    message_id    VARCHAR(64)  NOT NULL UNIQUE,
    routing_key   VARCHAR(120) NOT NULL,
    payload       JSONB        NOT NULL,
    -- Contexto W3C capturado na requisição: o publicador roda fora dela, e sem
    -- isso o span do consumidor nasceria órfão.
    trace_context JSONB,
    criado_em     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    publicado_em  TIMESTAMPTZ,
    tentativas    INT          NOT NULL DEFAULT 0
);

-- O publicador só enxerga o que ainda não saiu, e a caixa cresce indefinidamente
-- com o que já saiu: sem o índice parcial, a varredura de cada tique passaria a
-- percorrer todo o histórico.
CREATE INDEX idx_outbox_pendentes ON catalogo.outbox_eventos (id) WHERE publicado_em IS NULL;
