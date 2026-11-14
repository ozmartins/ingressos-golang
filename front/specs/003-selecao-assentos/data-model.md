# Data Model (cliente): Página de seleção de assentos

Tudo é lido de serviços existentes; o front mantém apenas estado de tela.

## Consumidos (tipos em `src/api/tipos.ts`)
- **SessaoNaGrade** (existente): `id, filme_id, filme_titulo, cinema_id, cinema_nome, sala_numero, tipo_tela, data_hora_inicio (ISO UTC), idioma, preco_base ("42.00")`.
- **Filme / Cinema** (existentes).
- **Poltrona** (novo): `rotulo` ("B4", id no escopo da sessão), `fileira` (letra), `numero` (int), `tipo` ∈ {NORMAL, PCD, NAMORADEIRA}, `status` ∈ {LIVRE, RESERVADA, OCUPADA}.
- **MapaPoltronas** (novo): `sessao_id`, `poltronas: Poltrona[]`.
- **RespostaBloqueio** (novo): `reserva_id`, `expira_em` (ISO), `mensagem`.
- **Problem** (RFC 9457): `type` (último segmento = categoria), `title`, `status`, `detail?`, `metadata?` (`limite` no 400 de quantidade).

## Derivados (puros, `dominio/`)
- **SessaoDoFilme**: `SessaoNaGrade` + `inicio: Date`, `dia`; lista ordenada, só futuras, do filme e cinema da rota. Padrão = primeira.
- **FileiraMapa**: `{ letra, poltronas: Poltrona[] ordenadas por numero }`; fileiras ordenadas pela letra (a referência exibe de cima para baixo da última à primeira; tela embaixo — a ordem de exibição é decidida na UI: fileira A mais próxima da tela).
- **SituacaoAssento**: `livre | selecionado | ocupado | indisponivel` (+ `tipo`).
- **Pedido**: `{ rotulos: string[] (ordem de seleção), precoCentavos, taxaCentavos=0, itens, totalCentavos }`.

## Regras / validações
- `rotulos.length ≤ LIMITE (10)`; só poltronas LIVRE entram; sem repetição.
- Trocar de sessão zera a seleção.
- Após 409: seleção ← seleção ∩ poltronas LIVRE do mapa recarregado.
- Total = `precoCentavos × itens + taxaCentavos`; 0 itens → total 0 e "Continuar" desabilitado.

## Estados da página
`verificando-login → carregando → pronto | vazio (sem sessões) | nao-encontrado | erro` ; submissão: `ocioso → reservando → reservado | conflito | erro`.

## Transições de reserva
`ocioso --Continuar--> reservando --201--> reservado(expira_em)`; `--409--> conflito(mapa recarregado, seleção reduzida) --> ocioso`; `--rede/5xx--> erro --tentar de novo--> reservando`.
