import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { buscarFilme, buscarSala, buscarSessao, reservarPoltronas } from '../api/catalogo'
import { consultarMapa } from '../api/estoque'
import { ErroDaApi } from '../api/cliente'
import { useAuth } from '../auth/ContextoAuth'
import { MapaDePoltronas } from '../componentes/MapaDePoltronas'
import { Problema } from '../componentes/Problema'
import { Botao, Carregando, Cartao, Selo } from '../componentes/ui'
import { formatarDataHora, moedaDeTexto } from '../formato'

// Espelha `POLTRONAS_MAX_POR_BLOQUEIO` do serviço de estoque. Vale como aviso
// na interface; quem decide de verdade continua sendo o serviço, que responde
// 400 com o limite no `detail`.
const MAXIMO_POR_RESERVA = 10

export function Sessao() {
  const { id = '' } = useParams()
  const { token } = useAuth()
  const navegar = useNavigate()
  const cache = useQueryClient()
  const [escolhidas, setEscolhidas] = useState<string[]>([])

  const sessao = useQuery({ queryKey: ['sessao', id], queryFn: () => buscarSessao(id) })
  const filme = useQuery({
    queryKey: ['filme', sessao.data?.filme_id],
    queryFn: () => buscarFilme(sessao.data!.filme_id),
    enabled: !!sessao.data,
  })
  const sala = useQuery({
    queryKey: ['sala', sessao.data?.sala_id],
    queryFn: () => buscarSala(sessao.data!.sala_id),
    enabled: !!sessao.data,
  })

  const mapa = useQuery({
    queryKey: ['mapa', id],
    queryFn: async () => consultarMapa(await token(), id),
    // O mapa muda por baixo, conforme outras pessoas reservam. Vale reler de
    // tempos em tempos em vez de mostrar um retrato parado.
    refetchInterval: 15_000,
    staleTime: 0,
  })

  const reserva = useMutation({
    mutationFn: async () => reservarPoltronas(await token(), id, escolhidas),
    onSuccess: (confirmada) => {
      navegar(`/pagamento/${confirmada.reserva_id}`, { state: { expiraEm: confirmada.expira_em } })
    },
    onError: () => {
      // A recusa mais comum é poltrona já tomada; reler o mapa mostra a quem
      // olha o que mudou, em vez de deixar a escolha velha na tela.
      void cache.invalidateQueries({ queryKey: ['mapa', id] })
      setEscolhidas([])
    },
  })

  function alternar(rotulo: string) {
    setEscolhidas((atuais) =>
      atuais.includes(rotulo) ? atuais.filter((r) => r !== rotulo) : [...atuais, rotulo],
    )
  }

  if (sessao.isPending) return <Carregando />
  if (sessao.error) return <Problema erro={sessao.error} />
  if (!sessao.data) return null

  const s = sessao.data
  const semMatriz = mapa.error instanceof ErroDaApi && mapa.error.status === 422

  return (
    <div className="space-y-6">
      <Link to="/" className="text-sm text-slate-400 hover:text-slate-100">
        ← Voltar para a grade
      </Link>

      <header className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">{filme.data?.titulo ?? 'Sessão'}</h1>
          <p className="text-sm text-slate-400">
            {formatarDataHora(s.data_hora_inicio)}
            {sala.data && ` · sala ${sala.data.numero} · ${sala.data.tipo_tela}`}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Selo>{s.idioma}</Selo>
          <Selo tom={s.status === 'AGENDADA' ? 'bom' : 'neutro'}>{s.status.toLowerCase()}</Selo>
        </div>
      </header>

      <div className="grid gap-6 lg:grid-cols-[1fr_300px]">
        <Cartao>
          {mapa.isPending && <Carregando texto="Carregando o mapa de poltronas…" />}
          {semMatriz ? (
            <div className="space-y-2 py-6 text-center">
              <p className="font-medium text-amber-300">O mapa desta sessão ainda não foi provisionado.</p>
              <p className="text-sm text-slate-400">
                O estoque monta a matriz ao consumir o anúncio <code>sessao.criada</code>. Costuma levar segundos.
              </p>
              <Botao variante="secundario" onClick={() => mapa.refetch()}>
                Tentar de novo
              </Botao>
            </div>
          ) : (
            mapa.error && <Problema erro={mapa.error} />
          )}
          {mapa.data && (
            <MapaDePoltronas
              poltronas={mapa.data.poltronas}
              escolhidas={escolhidas}
              aoAlternar={alternar}
              limite={MAXIMO_POR_RESERVA}
            />
          )}
        </Cartao>

        <Cartao className="h-fit space-y-4">
          <h2 className="font-bold">Sua escolha</h2>

          {escolhidas.length === 0 ? (
            <p className="text-sm text-slate-500">Nenhuma poltrona escolhida.</p>
          ) : (
            <div className="flex flex-wrap gap-1.5">
              {escolhidas.map((r) => (
                <Selo key={r}>{r}</Selo>
              ))}
            </div>
          )}

          <dl className="space-y-1 border-t border-borda pt-4 text-sm">
            <div className="flex justify-between">
              <dt className="text-slate-400">Preço por poltrona</dt>
              <dd className="font-mono">{moedaDeTexto(s.preco_base)}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-slate-400">Poltronas</dt>
              <dd className="font-mono">{escolhidas.length}</dd>
            </div>
          </dl>

          <p className="text-xs text-slate-500">
            O valor final é calculado pelo catálogo, que é quem tem autoridade sobre o preço. Máximo de{' '}
            {MAXIMO_POR_RESERVA} poltronas por reserva.
          </p>

          {reserva.error && <Problema erro={reserva.error} />}

          <Botao
            className="w-full"
            disabled={escolhidas.length === 0 || reserva.isPending}
            onClick={() => reserva.mutate()}
          >
            {reserva.isPending ? 'Reservando…' : 'Reservar'}
          </Botao>
        </Cartao>
      </div>
    </div>
  )
}
