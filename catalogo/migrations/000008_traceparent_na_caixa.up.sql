-- Trace distribuído através da caixa de saída. O fato é gravado na transação da
-- requisição e publicado depois, por outro processo de execução; sem guardar o
-- contexto do trace junto dele, o trace morreria aqui. O valor é o header W3C
-- `traceparent` da requisição de origem ('' quando não havia trace).
ALTER TABLE catalogo.outbox_eventos ADD COLUMN traceparent VARCHAR(64) NOT NULL DEFAULT '';
