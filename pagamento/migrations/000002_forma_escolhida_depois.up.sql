-- A forma de pagamento deixa de vir no anúncio da reserva e passa a ser
-- escolhida por quem paga, depois de a reserva existir. Reservar uma poltrona e
-- decidir como pagar são decisões distintas, e o fato `reserva.criada` amarrava
-- as duas: como o estoque nunca soube a forma de pagamento — ninguém no sistema
-- a pedia —, todo anúncio vindo dele era inválido e ia para a fila morta.
--
-- Com a separação, a transação nasce sabendo quanto cobrar e esperando saber
-- como.
ALTER TABLE pagamento.transacoes_pagamento
    ALTER COLUMN forma_pagamento DROP NOT NULL;

-- O prazo da reserva era usado e descartado no consumo. Agora precisa ficar: a
-- escolha da forma acontece depois, e sem o prazo não há como saber se ainda dá
-- tempo de cobrar.
--
-- Entra anulável e é preenchido nas linhas existentes com o instante de criação
-- — todas elas já são terminais, e um prazo no passado não muda o desfecho de
-- quem já foi cobrado. Só então vira obrigatório.
ALTER TABLE pagamento.transacoes_pagamento ADD COLUMN expira_em TIMESTAMPTZ;
UPDATE pagamento.transacoes_pagamento SET expira_em = criado_em WHERE expira_em IS NULL;
ALTER TABLE pagamento.transacoes_pagamento ALTER COLUMN expira_em SET NOT NULL;

ALTER TABLE pagamento.transacoes_pagamento
    DROP CONSTRAINT status_valido,
    DROP CONSTRAINT forma_valida,
    ADD CONSTRAINT status_valido CHECK (status IN
        ('AGUARDANDO_FORMA','PROCESSANDO','PAGO','RECUSADO','CANCELADO','PENDENTE_VERIFICACAO')),
    ADD CONSTRAINT forma_valida CHECK
        (forma_pagamento IS NULL OR forma_pagamento IN ('PIX','CARTAO_CREDITO')),
    -- Quem espera a escolha não tem forma; quem foi cobrado tem. O cancelamento
    -- é o único que aceita as duas coisas, porque pode acontecer antes da
    -- escolha (reserva vencida sem ninguém pagar) ou depois dela.
    --
    -- Escrito como CASE, e não como igualdade, porque a regra tem três ramos, e
    -- forçá-la em dois obrigaria a inventar uma forma de pagamento para quem
    -- nunca escolheu nenhuma.
    ADD CONSTRAINT forma_coerente_com_estado CHECK (
        CASE status
            WHEN 'AGUARDANDO_FORMA' THEN forma_pagamento IS NULL
            WHEN 'CANCELADO'        THEN TRUE
            ELSE forma_pagamento IS NOT NULL
        END);
