-- A reserva passa a carregar o valor a cobrar. Ele não é decidido aqui: quem
-- tem autoridade sobre o preço é o Servico-Catalogo, dono do cadastro da
-- sessão. Este serviço guarda o valor e o repassa no fato `reserva.criada`,
-- para quem cobra — sem ele, o Servico-Pagamento não consegue cobrar nada, e
-- era exatamente aí que o fluxo morria.
ALTER TABLE estoque.reservas ADD COLUMN valor_total DECIMAL(10, 2);

-- A coluna é anulável de propósito, e não por falta de rigor: as reservas
-- gravadas antes desta migração não têm valor porque o valor não existia, e
-- inventar um número para elas seria pior que admitir a ausência.
--
-- O rigor vem da invariante abaixo, não da nulidade: toda reserva que ainda
-- pode virar cobrança — isto é, `PENDENTE` — tem valor. As terminais não
-- podem, e por isso podem não ter.
ALTER TABLE estoque.reservas
    ADD CONSTRAINT ck_reserva_valor_positivo
        CHECK (valor_total IS NULL OR valor_total > 0),
    ADD CONSTRAINT ck_reserva_pendente_tem_valor
        CHECK (status <> 'PENDENTE' OR valor_total IS NOT NULL);
