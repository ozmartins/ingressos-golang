-- Duas tabelas do Servico-Notificacao. DDL da ERS, mais as restrições CHECK que
-- trazem invariantes do domínio para dentro do banco (data-model.md §1 e §3).
--
-- A notificação é dona do seu próprio schema: nada dela vive em `public`. Cada
-- objeto é qualificado porque esta migração roda pelo CLI do golang-migrate,
-- cujo `search_path` é o padrão da conexão. A tabela de controle de versões
-- (`schema_migrations`) segue em `public`: ela é do ferramental de migração,
-- não do domínio.

CREATE SCHEMA IF NOT EXISTS notificacao;

CREATE TABLE notificacao.ingressos_emitidos (
    id           VARCHAR(36)  PRIMARY KEY,
    reserva_id   VARCHAR(36)  NOT NULL UNIQUE,
    usuario_id   VARCHAR(36)  NOT NULL,
    codigo_qr    VARCHAR(255) NOT NULL UNIQUE,
    status       VARCHAR(50)  NOT NULL DEFAULT 'VALIDO',
    utilizado_em TIMESTAMPTZ,
    criado_em    TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT status_valido CHECK (status IN ('VALIDO','UTILIZADO','CANCELADO')),
    -- FR-007 e FR-008 como igualdade nos dois sentidos: impede tanto a baixa
    -- sem carimbo quanto o carimbo sem baixa.
    CONSTRAINT utilizado_em_so_quando_utilizado CHECK
        ((status = 'UTILIZADO') = (utilizado_em IS NOT NULL))
);

-- Único índice além das chaves: serve à listagem por pessoa (research.md D8).
CREATE INDEX ingressos_por_pessoa ON notificacao.ingressos_emitidos (usuario_id, criado_em DESC);

CREATE TABLE notificacao.registros_notificacao (
    id          VARCHAR(36) PRIMARY KEY,
    ingresso_id VARCHAR(36) NOT NULL REFERENCES notificacao.ingressos_emitidos(id),
    usuario_id  VARCHAR(36) NOT NULL,
    canal       VARCHAR(50) NOT NULL DEFAULT 'EMAIL',
    status      VARCHAR(50) NOT NULL DEFAULT 'ENVIADO',
    detalhes    TEXT,
    enviado_em  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT canal_valido  CHECK (canal  IN ('EMAIL','PUSH','SMS')),
    CONSTRAINT status_valido CHECK (status IN ('ENVIADO','FALHA')),
    -- Registro de falha sem motivo não serve ao reenvio que justifica a tabela.
    CONSTRAINT detalhes_na_falha CHECK (status <> 'FALHA' OR detalhes IS NOT NULL)
);
