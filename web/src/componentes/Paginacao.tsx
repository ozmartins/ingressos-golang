import type { Paginacao as Pagina } from '../api/tipos'
import { Botao } from './ui'

export function Paginacao({ pagina, aoMudar }: { pagina: Pagina; aoMudar: (n: number) => void }) {
  if (pagina.pagina === 1 && !pagina.tem_proxima) return null
  return (
    <div className="flex items-center justify-between pt-4 text-sm text-slate-400">
      <span>
        Página {pagina.pagina} · {pagina.total} no total
      </span>
      <div className="flex gap-2">
        <Botao variante="secundario" disabled={pagina.pagina <= 1} onClick={() => aoMudar(pagina.pagina - 1)}>
          Anterior
        </Botao>
        <Botao variante="secundario" disabled={!pagina.tem_proxima} onClick={() => aoMudar(pagina.pagina + 1)}>
          Próxima
        </Botao>
      </div>
    </div>
  )
}
