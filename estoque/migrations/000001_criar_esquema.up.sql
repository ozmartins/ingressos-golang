-- Esquema do Servico-Estoque. Fonte: data-model.md.
-- As três primeiras tabelas vêm da ERS; as duas últimas sustentam requisitos
-- que a ERS enuncia mas não modela (entrega confiável do fato e idempotência).

CREATE TABLE poltronas (
    id            VARCHAR(36)  PRIMARY KEY,       -- UUID v5 de sessao_id|fileira|numero
    sessao_id     VARCHAR(36)  NOT NULL,
    fileira       VARCHAR(5)   NOT NULL,
    numero        INT          NOT NULL,
    rotulo        VARCHAR(10)  NOT NULL,          -- fileira || numero, identidade de negócio
    tipo          VARCHAR(50)  NOT NULL DEFAULT 'NORMAL',
    status        VARCHAR(50)  NOT NULL DEFAULT 'LIVRE',
    criado_em     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uk_sessao_poltrona UNIQUE (sessao_id, fileira, numero),
    CONSTRAINT uk_sessao_rotulo   UNIQUE (sessao_id, rotulo),
    CONSTRAINT ck_poltrona_status CHECK (status IN ('LIVRE', 'RESERVADA', 'OCUPADA')),
    CONSTRAINT ck_poltrona_tipo   CHECK (tipo IN ('NORMAL', 'PCD', 'NAMORADEIRA')),
    CONSTRAINT ck_poltrona_numero CHECK (numero > 0)
);

CREATE TABLE reservas (
    id             VARCHAR(36) PRIMARY KEY,       -- UUID v4
    sessao_id      VARCHAR(36) NOT NULL,
    usuario_id     VARCHAR(36) NOT NULL,
    expira_em      TIMESTAMPTZ NOT NULL,
    status         VARCHAR(50) NOT NULL DEFAULT 'PENDENTE',
    criado_em      TIMESTAMPTZ NOT NULL DEFAULT now(),
    finalizado_em  TIMESTAMPTZ,
    CONSTRAINT ck_reserva_status CHECK (status IN ('PENDENTE', 'CONFIRMADA', 'EXPIRADA', 'CANCELADA')),
    CONSTRAINT ck_reserva_prazo  CHECK (expira_em > criado_em),
    -- Uma reserva pendente nunca tem instante de finalização, e uma finalizada
    -- sempre tem. Invariante de FR-011 expressa de forma declarativa.
    CONSTRAINT ck_reserva_finalizacao CHECK ((status = 'PENDENTE') = (finalizado_em IS NULL))
);

CREATE TABLE reserva_poltronas (
    reserva_id  VARCHAR(36) NOT NULL REFERENCES reservas(id),
    poltrona_id VARCHAR(36) NOT NULL REFERENCES poltronas(id),
    PRIMARY KEY (reserva_id, poltrona_id)
);

-- Caixa de saída: o fato é gravado na mesma transação que o produziu e
-- publicado depois, fora do caminho da requisição (FR-018, SC-005).
CREATE TABLE outbox_eventos (
    id            BIGSERIAL   PRIMARY KEY,
    message_id    VARCHAR(64) NOT NULL UNIQUE,
    routing_key   VARCHAR(120) NOT NULL,
    payload       JSONB       NOT NULL,
    -- Contexto W3C capturado na concessão: o publicador roda fora da requisição
    -- e sem isso o span nasceria órfão (FR-044, SC-009).
    trace_context JSONB,
    criado_em     TIMESTAMPTZ NOT NULL DEFAULT now(),
    publicado_em  TIMESTAMPTZ,
    tentativas    INT         NOT NULL DEFAULT 0
);

-- Idempotência do consumo (FR-021, FR-034).
CREATE TABLE mensagens_processadas (
    fila          VARCHAR(120) NOT NULL,
    message_id    VARCHAR(64)  NOT NULL,
    processado_em TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (fila, message_id)
);
