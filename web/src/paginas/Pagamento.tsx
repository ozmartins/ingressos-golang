import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { consultarPagamento, escolherForma } from '../api/pagamento'
import { ErroDaApi } from '../api/cliente'
import type { FormaPagamento, StatusPagamento } from '../api/tipos'
import { useAuth } from '../auth/ContextoAuth'
import { ContagemRegressiva } from '../componentes/ContagemRegressiva'
import { Problema } from '../componentes/Problema'
import { Botao, Carregando, Cartao, Selo } from '../componentes/ui'
import { moedaDeNumero } from '../formato'

// A transação não nasce desta tela: ela nasce do consumo de `reserva.criada`
// pelo serviço de pagamento. Por isso o 404 dos primeiros segundos significa
// "ainda não chegou", e não "não existe".
const ESTADOS_FINAIS: StatusPagamento[] = ['PAGO', 'RECUSADO', 'CANCELADO']

const tomDoStatus: Record<StatusPagamento, 'bom' | 'ruim' | 'espera' | 'neutro'> = {
  AGUARDANDO_FORMA: 'espera',
  PROCESSANDO: 'espera',
  PENDENTE_VERIFICACAO: 'espera',
  PAGO: 'bom',
  RECUSADO: 'ruim',
  CANCELADO: 'neutro',
}

export function Pagamento() {
  const { reservaId = '' } = useParams()
  const { token } = useAuth()
  const cache = useQueryClient()
  const [forma, setForma] = useState<FormaPagamento>('PIX')

  const pagamento = useQuery({
    queryKey: ['pagamento', reservaId],
    queryFn: async () => consultarPagamento(await token(), reservaId),
    // Enquanto não há desfecho, a tela relê. O ritmo é curto porque a cobrança
    // acontece fora da requisição e pode terminar a qualquer instante.
    refetchInterval: (consulta) => {
      const atual = consulta.state.data
      if (atual && ESTADOS_FINAIS.includes(atual.status)) return false
      return 2_000
    },
    staleTime: 0,
    // O 404 aqui é espera, não ausência: vale insistir por alguns ciclos.
    retry: (tentativas, erro) => erro instanceof ErroDaApi && erro.status === 404 && tentativas < 10,
    retryDelay: 1_500,
  })

  const escolha = useMutation({
    mutationFn: async () => escolherForma(await token(), reservaId, forma),
    onSuccess: () => cache.invalidateQueries({ queryKey: ['pagamento', reservaId] }),
  })

  if (pagamento.isPending) {
    return <Carregando texto="Aguardando o serviço de pagamento registrar a reserva…" />
  }
  if (pagamento.error) {
    return (
      <div className="mx-auto max-w-lg space-y-4">
        <Problema erro={pagamento.error} />
        <Link to="/" className="block text-sm text-destaque hover:underline">
          Voltar para a grade
        </Link>
      </div>
    )
  }
  if (!pagamento.data) return null

  const p = pagamento.data
  const podeEscolher = p.status === 'AGUARDANDO_FORMA'
  const finalizado = ESTADOS_FINAIS.includes(p.status)

  return (
    <div className="mx-auto max-w-lg space-y-6">
      <header>
        <h1 className="text-2xl font-bold">Pagamento</h1>
        <p className="text-sm text-slate-400">
          Reserva <code className="text-slate-300">{p.reserva_id}</code>
        </p>
      </header>

      <Cartao className="space-y-4">
        <div className="flex items-center justify-between">
          <Selo tom={tomDoStatus[p.status]}>{p.status.replace(/_/g, ' ').toLowerCase()}</Selo>
          {!finalizado && (
            <span className="text-sm text-slate-400">
              expira em <ContagemRegressiva ate={p.expira_em} />
            </span>
          )}
        </div>

        <dl className="space-y-1 text-sm">
          <div className="flex justify-between">
            <dt className="text-slate-400">Valor total</dt>
            <dd className="font-mono text-base font-semibold">{moedaDeNumero(p.valor_total)}</dd>
          </div>
          {p.forma_pagamento && (
            <div className="flex justify-between">
              <dt className="text-slate-400">Forma</dt>
              <dd>{p.forma_pagamento === 'PIX' ? 'Pix' : 'Cartão de crédito'}</dd>
            </div>
          )}
          {p.motivo_falha && (
            <div className="flex justify-between">
              <dt className="text-slate-400">Motivo</dt>
              <dd className="text-rose-300">{p.motivo_falha}</dd>
            </div>
          )}
        </dl>
      </Cartao>

      {podeEscolher && (
        <Cartao className="space-y-4">
          <h2 className="font-bold">Como pagar</h2>
          <div className="grid grid-cols-2 gap-3">
            {(['PIX', 'CARTAO_CREDITO'] as FormaPagamento[]).map((f) => (
              <button
                key={f}
                type="button"
                onClick={() => setForma(f)}
                className={`rounded-lg border px-4 py-4 text-sm font-semibold transition ${
                  forma === f ? 'border-destaque bg-destaque/10 text-destaque' : 'border-borda hover:border-slate-500'
                }`}
              >
                {f === 'PIX' ? 'Pix' : 'Cartão de crédito'}
              </button>
            ))}
          </div>

          {escolha.error && <Problema erro={escolha.error} />}

          <Botao className="w-full" disabled={escolha.isPending} onClick={() => escolha.mutate()}>
            {escolha.isPending ? 'Enviando…' : 'Confirmar forma de pagamento'}
          </Botao>
          <p className="text-xs text-slate-500">
            A cobrança acontece fora desta requisição. O desfecho aparece aqui sozinho.
          </p>
        </Cartao>
      )}

      {p.status === 'PROCESSANDO' && (
        <p className="text-center text-sm text-slate-400">Cobrança em andamento. O resultado chega nesta tela.</p>
      )}

      {p.status === 'PAGO' && (
        <Cartao className="space-y-3 text-center">
          <p className="text-lg font-semibold text-emerald-300">Pagamento aprovado.</p>
          <p className="text-sm text-slate-400">
            O ingresso é emitido logo em seguida, também por mensageria.
          </p>
          <Link to="/meus-ingressos" className="inline-block">
            <Botao>Ver meus ingressos</Botao>
          </Link>
        </Cartao>
      )}

      {(p.status === 'RECUSADO' || p.status === 'CANCELADO') && (
        <Cartao className="space-y-3 text-center">
          <p className="font-semibold text-rose-300">
            {p.status === 'RECUSADO' ? 'A cobrança foi recusada.' : 'A reserva foi cancelada.'}
          </p>
          <p className="text-sm text-slate-400">As poltronas voltaram a ficar disponíveis.</p>
          <Link to="/" className="inline-block">
            <Botao variante="secundario">Escolher outra sessão</Botao>
          </Link>
        </Cartao>
      )}
    </div>
  )
}
