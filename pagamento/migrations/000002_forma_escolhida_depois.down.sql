-- A volta é destrutiva por natureza: a coluna volta a ser obrigatória, e as
-- linhas sem forma escolhida não têm o que gravar nela. Elas são canceladas — e
-- ficam com PIX, não porque alguém escolheu, mas porque o esquema antigo não
-- comporta a ausência. É perda de informação, inevitável na volta.
UPDATE pagamento.transacoes_pagamento
   SET status = 'CANCELADO', motivo_falha = 'RESERVA_EXPIRADA', forma_pagamento = 'PIX'
 WHERE status = 'AGUARDANDO_FORMA' OR forma_pagamento IS NULL;

ALTER TABLE pagamento.transacoes_pagamento
    DROP CONSTRAINT forma_coerente_com_estado,
    DROP CONSTRAINT forma_valida,
    DROP CONSTRAINT status_valido,
    ADD CONSTRAINT status_valido CHECK (status IN
        ('PROCESSANDO','PAGO','RECUSADO','CANCELADO','PENDENTE_VERIFICACAO')),
    ADD CONSTRAINT forma_valida CHECK (forma_pagamento IN ('PIX','CARTAO_CREDITO'));

ALTER TABLE pagamento.transacoes_pagamento DROP COLUMN IF EXISTS expira_em;
ALTER TABLE pagamento.transacoes_pagamento ALTER COLUMN forma_pagamento SET NOT NULL;
