-- Carrega o mapa completo da sessão (FR-029).
CREATE INDEX idx_poltronas_sessao ON poltronas (sessao_id);

-- Índice parcial que sustenta a varredura de expiração sem varrer reservas
-- já finalizadas (D4).
CREATE INDEX idx_reservas_expiracao ON reservas (expira_em) WHERE status = 'PENDENTE';

CREATE INDEX idx_reservas_sessao_usuario ON reservas (sessao_id, usuario_id);

-- "Qual reserva prende esta poltrona" sem varredura — usado na liberação.
CREATE INDEX idx_reserva_poltronas_poltrona ON reserva_poltronas (poltrona_id);

CREATE INDEX idx_outbox_pendentes ON outbox_eventos (id) WHERE publicado_em IS NULL;

CREATE INDEX idx_mensagens_processadas_limpeza ON mensagens_processadas (processado_em);
