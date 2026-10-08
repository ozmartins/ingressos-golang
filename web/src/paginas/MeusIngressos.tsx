import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { listarMeusIngressos } from '../api/notificacao'
import type { Ingresso, StatusIngresso } from '../api/tipos'
import { useAuth } from '../auth/ContextoAuth'
import { CodigoQR } from '../componentes/CodigoQR'
import { Problema } from '../componentes/Problema'
import { Carregando, Cartao, EstadoVazio, Selecao, Selo } from '../componentes/ui'
import { formatarDataHora } from '../formato'

const tons: Record<StatusIngresso, 'bom' | 'neutro' | 'ruim'> = {
  VALIDO: 'bom',
  UTILIZADO: 'neutro',
  CANCELADO: 'ruim',
}

export function MeusIngressos() {
  const { token } = useAuth()
  const [status, setStatus] = useState<'' | StatusIngresso>('')

  const consulta = useQuery({
    queryKey: ['ingressos', status],
    queryFn: async () => listarMeusIngressos(await token(), status || undefined),
    // A emissão também é assíncrona: quem acabou de pagar chega aqui antes do
    // ingresso existir. Enquanto a lista estiver vazia, vale reler.
    refetchInterval: (c) => (c.state.data && c.state.data.length > 0 ? false : 3_000),
    staleTime: 0,
  })

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">Meus ingressos</h1>
          {/* O recorte é sempre pelo `sub` do token: não há como alcançar
              ingresso de terceiro por esta tela nem por parâmetro. */}
          <p className="text-sm text-slate-400">Apenas os ingressos da pessoa autenticada.</p>
        </div>
        <Selecao
          rotulo="Estado"
          className="w-44"
          value={status}
          onChange={(e) => setStatus(e.target.value as '' | StatusIngresso)}
        >
          <option value="">Todos</option>
          <option value="VALIDO">Válidos</option>
          <option value="UTILIZADO">Utilizados</option>
          <option value="CANCELADO">Cancelados</option>
        </Selecao>
      </header>

      {consulta.isPending && <Carregando />}
      {consulta.error && <Problema erro={consulta.error} />}

      {consulta.data?.length === 0 && (
        <EstadoVazio
          titulo="Nenhum ingresso ainda"
          descricao="Se o pagamento acabou de ser aprovado, a emissão chega em instantes."
        />
      )}

      <div className="grid gap-4 sm:grid-cols-2">
        {consulta.data?.map((i) => (
          <CartaoDeIngresso key={i.ingresso_id} ingresso={i} />
        ))}
      </div>
    </div>
  )
}

function CartaoDeIngresso({ ingresso }: { ingresso: Ingresso }) {
  return (
    <Cartao className="flex gap-4">
      <div className="shrink-0">
        <CodigoQR valor={ingresso.codigo_qr} tamanho={148} />
      </div>
      <div className="min-w-0 space-y-2">
        <Selo tom={tons[ingresso.status]}>{ingresso.status.toLowerCase()}</Selo>
        <p className="text-xs text-slate-400">
          Emitido em <span className="font-mono">{formatarDataHora(ingresso.criado_em)}</span>
        </p>
        <p className="text-xs text-slate-500">
          Reserva <span className="font-mono break-all">{ingresso.reserva_id}</span>
        </p>
        <p className="break-all font-mono text-[11px] leading-snug text-slate-400">{ingresso.codigo_qr}</p>
      </div>
    </Cartao>
  )
}
