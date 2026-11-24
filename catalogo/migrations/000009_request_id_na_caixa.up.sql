-- Request id (correlação de logs) através da caixa de saída. Como o traceparent,
-- ele é gravado junto do fato na requisição de origem para o publicador, que
-- roda depois e fora dela, devolvê-lo nos headers AMQP. '' quando não havia.
ALTER TABLE catalogo.outbox_eventos ADD COLUMN request_id VARCHAR(64) NOT NULL DEFAULT '';
