import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { listarFilmes } from '../api/catalogo'
import type { StatusFilme } from '../api/tipos'
import { Paginacao } from '../componentes/Paginacao'
import { Problema } from '../componentes/Problema'
import { Carregando, EstadoVazio, Selecao, Selo } from '../componentes/ui'

export function Filmes() {
  const [status, setStatus] = useState<'' | StatusFilme>('')
  const [pagina, setPagina] = useState(1)

  const consulta = useQuery({
    queryKey: ['filmes', status, pagina],
    queryFn: () => listarFilmes({ status: status || undefined, page: pagina, page_size: 12 }),
  })

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">Filmes</h1>
          {/* Sem filtro, o serviço devolve só EM_CARTAZ e BREVE. */}
          <p className="text-sm text-slate-400">Sem filtro, aparecem apenas os em cartaz e os que estreiam em breve.</p>
        </div>
        <Selecao
          rotulo="Status"
          className="w-48"
          value={status}
          onChange={(e) => {
            setStatus(e.target.value as '' | StatusFilme)
            setPagina(1)
          }}
        >
          <option value="">Em cartaz e breve</option>
          <option value="EM_CARTAZ">Em cartaz</option>
          <option value="BREVE">Breve</option>
          <option value="FORA_DE_CARTAZ">Fora de cartaz</option>
        </Selecao>
      </header>

      {consulta.isPending && <Carregando />}
      {consulta.error && <Problema erro={consulta.error} />}
      {consulta.data?.itens.length === 0 && <EstadoVazio titulo="Nenhum filme cadastrado" />}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {consulta.data?.itens.map((f) => (
          <Link
            key={f.id}
            to={`/filmes/${f.id}`}
            className="overflow-hidden rounded-xl border border-borda bg-painel transition hover:border-destaque"
          >
            <div className="aspect-[2/3] bg-slate-800">
              {f.imagem_url ? (
                <img src={f.imagem_url} alt="" className="h-full w-full object-cover" />
              ) : (
                <div className="flex h-full items-center justify-center text-5xl text-slate-700">🎬</div>
              )}
            </div>
            <div className="space-y-2 p-4">
              <p className="font-semibold leading-tight">{f.titulo}</p>
              <p className="text-xs text-slate-400">
                {f.genero} · {f.duracao_minutos} min · {f.classificacao_etaria}
              </p>
              <Selo tom={f.status === 'EM_CARTAZ' ? 'bom' : f.status === 'BREVE' ? 'espera' : 'neutro'}>
                {f.status.replace(/_/g, ' ').toLowerCase()}
              </Selo>
            </div>
          </Link>
        ))}
      </div>

      {consulta.data && <Paginacao pagina={consulta.data.pagina} aoMudar={setPagina} />}
    </div>
  )
}
