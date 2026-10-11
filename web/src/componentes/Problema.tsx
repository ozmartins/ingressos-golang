import { ErroDaApi } from '../api/cliente'

/** Mensagens por categoria, que é a parte estável do contrato. `title` e
 *  `detail` são texto humano e podem mudar de redação, então servem de
 *  complemento, não de identidade do erro. */
const porCategoria: Record<string, string> = {
  'poltronas-indisponiveis': 'Alguma das poltronas escolhidas acabou de ser levada. Escolha outras.',
  'solicitacao-invalida': 'O pedido de reserva é inválido. Revise as poltronas escolhidas.',
  'limite-poltronas-excedido': 'Você escolheu mais poltronas do que o permitido por reserva.',
  'poltrona-inexistente': 'Uma das poltronas escolhidas não existe nesta sessão.',
  'sessao-nao-provisionada': 'A sessão ainda não teve o mapa de poltronas provisionado. Tente em instantes.',
  'sessao-nao-reservavel': 'Esta sessão não aceita reserva.',
  'sessao-nao-encontrada': 'Sessão não encontrada.',
  'catalogo-indisponivel': 'O catálogo não respondeu. Vale tentar de novo.',
  'dependencia-indisponivel': 'O serviço de estoque está indisponível. Vale tentar de novo.',
  'resposta-invalida-do-parceiro': 'O catálogo respondeu de forma inesperada. Repetir não resolve.',
  'nao-autenticado': 'Sua sessão expirou. Entre novamente.',
  'sem-conexao': 'Sem conexão com o serviço.',
  'reserva-expirada': 'O prazo da reserva venceu e as poltronas foram liberadas.',
  'forma-pagamento-ja-escolhida': 'A forma de pagamento já havia sido escolhida para esta reserva.',
}

export function mensagemDe(erro: unknown): string {
  if (!(erro instanceof ErroDaApi)) {
    return erro instanceof Error ? erro.message : 'Falha inesperada.'
  }
  return porCategoria[erro.categoria] ?? erro.detalhe ?? erro.message
}

export function Problema({ erro }: { erro: unknown }) {
  if (!erro) return null
  const campos = erro instanceof ErroDaApi ? erro.campos : undefined
  return (
    <div role="alert" className="rounded-lg border border-rose-500/40 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
      <p>{mensagemDe(erro)}</p>
      {campos && campos.length > 0 && (
        <ul className="mt-2 list-inside list-disc text-xs text-rose-300/80">
          {campos.map((c) => (
            <li key={c.campo}>
              <span className="font-mono">{c.campo}</span>: {c.mensagem}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
