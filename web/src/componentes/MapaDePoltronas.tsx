import type { Poltrona } from '../api/tipos'

// A identidade da poltrona no escopo da sessão é o `rotulo`; fileira e número
// vêm decompostos só para desenhar.
function agruparPorFileira(poltronas: Poltrona[]): [string, Poltrona[]][] {
  const mapa = new Map<string, Poltrona[]>()
  for (const p of poltronas) {
    const fileira = mapa.get(p.fileira) ?? []
    fileira.push(p)
    mapa.set(p.fileira, fileira)
  }
  return [...mapa.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([letra, assentos]) => [letra, assentos.sort((a, b) => a.numero - b.numero)] as [string, Poltrona[]])
}

const aparencia: Record<string, string> = {
  LIVRE: 'border-borda bg-painel hover:border-destaque',
  RESERVADA: 'border-amber-600/40 bg-amber-900/20 text-amber-600/60',
  OCUPADA: 'border-slate-700 bg-slate-800/60 text-slate-600',
  escolhida: 'border-destaque bg-destaque text-slate-950',
}

const marcaDoTipo: Record<string, string> = { PCD: '♿', NAMORADEIRA: '♥', NORMAL: '' }

export function MapaDePoltronas({
  poltronas,
  escolhidas,
  aoAlternar,
  limite,
}: {
  poltronas: Poltrona[]
  escolhidas: string[]
  aoAlternar: (rotulo: string) => void
  limite: number
}) {
  const fileiras = agruparPorFileira(poltronas)

  return (
    <div className="space-y-6">
      <div className="mx-auto h-1.5 w-2/3 rounded-full bg-slate-600" />
      <p className="text-center text-xs uppercase tracking-[0.3em] text-slate-500">tela</p>

      <div className="space-y-2 overflow-x-auto">
        {fileiras.map(([letra, assentos]) => (
          <div key={letra} className="flex items-center gap-2">
            <span className="w-6 shrink-0 text-right text-xs font-semibold text-slate-500">{letra}</span>
            <div className="flex flex-wrap gap-1.5">
              {assentos.map((p) => {
                const escolhida = escolhidas.includes(p.rotulo)
                const livre = p.status === 'LIVRE'
                const bloqueadaPeloLimite = !escolhida && escolhidas.length >= limite
                return (
                  <button
                    key={p.rotulo}
                    type="button"
                    disabled={!livre || bloqueadaPeloLimite}
                    onClick={() => aoAlternar(p.rotulo)}
                    aria-pressed={escolhida}
                    title={`${p.rotulo} · ${p.tipo} · ${p.status}`}
                    className={`h-9 w-9 rounded-md border text-[11px] font-semibold transition disabled:cursor-not-allowed ${
                      escolhida ? aparencia.escolhida : aparencia[p.status]
                    }`}
                  >
                    {marcaDoTipo[p.tipo] || p.numero}
                  </button>
                )
              })}
            </div>
          </div>
        ))}
      </div>

      <div className="flex flex-wrap gap-4 text-xs text-slate-400">
        <Legenda classe="border-borda bg-painel" texto="Livre" />
        <Legenda classe="border-destaque bg-destaque" texto="Escolhida" />
        <Legenda classe="border-amber-600/40 bg-amber-900/20" texto="Reservada" />
        <Legenda classe="border-slate-700 bg-slate-800/60" texto="Ocupada" />
        <span>♿ PCD · ♥ namoradeira</span>
      </div>
    </div>
  )
}

function Legenda({ classe, texto }: { classe: string; texto: string }) {
  return (
    <span className="flex items-center gap-1.5">
      <span className={`inline-block h-4 w-4 rounded border ${classe}`} />
      {texto}
    </span>
  )
}
