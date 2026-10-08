import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { listarCinemas } from '../api/catalogo'
import { Paginacao } from '../componentes/Paginacao'
import { Problema } from '../componentes/Problema'
import { Carregando, Cartao, EstadoVazio, Selo } from '../componentes/ui'

export function Cinemas() {
  const [pagina, setPagina] = useState(1)
  const consulta = useQuery({
    queryKey: ['cinemas', pagina],
    queryFn: () => listarCinemas({ page: pagina, page_size: 12 }),
  })

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Cinemas</h1>

      {consulta.isPending && <Carregando />}
      {consulta.error && <Problema erro={consulta.error} />}
      {consulta.data?.itens.length === 0 && <EstadoVazio titulo="Nenhum cinema cadastrado" />}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {consulta.data?.itens.map((c) => (
          <Cartao key={c.id} className="space-y-2">
            <div className="flex items-start justify-between gap-2">
              <p className="font-semibold">{c.nome}</p>
              {!c.ativo && <Selo tom="neutro">inativo</Selo>}
            </div>
            <p className="text-sm text-slate-400">
              {c.cidade} / {c.estado}
            </p>
            <p className="text-sm text-slate-500">{c.endereco}</p>
            <Link to={`/?cinema=${c.id}`} className="inline-block pt-1 text-sm text-destaque hover:underline">
              Ver na grade
            </Link>
          </Cartao>
        ))}
      </div>

      {consulta.data && <Paginacao pagina={consulta.data.pagina} aoMudar={setPagina} />}
    </div>
  )
}
