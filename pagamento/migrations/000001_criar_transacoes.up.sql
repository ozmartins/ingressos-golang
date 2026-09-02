-- Tabela única do Servico-Pagamento. DDL da ERS, mais resultado_anunciado,
-- pago_em e o estado PENDENTE_VERIFICACAO — justificados em data-model.md §1.
--
-- O pagamento é dono do seu próprio schema: nada dele vive em `public`. A
-- tabela é qualificada porque esta migração roda pelo CLI do golang-migrate,
-- cujo `search_path` vem da conexão. A tabela de controle de versões
-- (`schema_migrations`) fica no mesmo schema `pagamento`: o banco é
-- compartilhado pelos quatro serviços, e em `public` os quatro disputariam
-- uma só tabela de controle.

CREATE SCHEMA IF NOT EXISTS pagamento;

CREATE TABLE pagamento.transacoes_pagamento (
    id                       VARCHAR(36) PRIMARY KEY,
    reserva_id               VARCHAR(36) NOT NULL UNIQUE,
    usuario_id               VARCHAR(36) NOT NULL,
    valor_total              DECIMAL(10, 2) NOT NULL,
    forma_pagamento          VARCHAR(50) NOT NULL,
    status                   VARCHAR(50) NOT NULL DEFAULT 'PROCESSANDO',
    codigo_transacao_gateway VARCHAR(255),
    motivo_falha             TEXT,
    cobranca_emitida         BOOLEAN NOT NULL DEFAULT FALSE,
    resultado_anunciado      BOOLEAN NOT NULL DEFAULT FALSE,
    pago_em                  TIMESTAMPTZ,
    criado_em                TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em            TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT status_valido CHECK (status IN
        ('PROCESSANDO','PAGO','RECUSADO','CANCELADO','PENDENTE_VERIFICACAO')),
    CONSTRAINT forma_valida CHECK (forma_pagamento IN ('PIX','CARTAO_CREDITO')),
    CONSTRAINT valor_positivo CHECK (valor_total > 0),
    CONSTRAINT pago_em_so_quando_pago CHECK ((status = 'PAGO') = (pago_em IS NOT NULL)),
    CONSTRAINT anuncio_so_apos_estado_final CHECK
        (NOT resultado_anunciado OR status <> 'PROCESSANDO'),
    CONSTRAINT verificacao_nunca_anunciada CHECK
        (NOT (status = 'PENDENTE_VERIFICACAO' AND resultado_anunciado))
);
