-- A cobrança roda depois da requisição que originou a reserva (varredura
-- periódica), então o request id precisa ficar gravado para chegar aos logs e
-- aos headers de pagamento.sucesso/falhou. Anulável: linhas antigas não têm.
ALTER TABLE pagamento.transacoes_pagamento ADD COLUMN request_id VARCHAR(64);
