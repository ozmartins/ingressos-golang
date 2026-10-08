import { Link, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { buscarFilme, listarSessoes } from '../api/catalogo'
import { formatarDataHora, moedaDeTexto } from '../formato'
import { Problema } from '../componentes/Problema'
import { Carregando, Cartao, EstadoVazio, Selo } from '../componentes/ui'

export function FilmeDetalhe() {
  const { id = '' } = useParams()

  const filme = useQuery({ queryKey: ['filme', id], queryFn: () => buscarFilme(id) })
  const sessoes = useQuery({
    queryKey: ['sessoes', { filme_id: id }],
    queryFn: () => listarSessoes({ filme_id: id, page_size: 50 }),
  })

  if (filme.isPending) return <Carregando />
  if (filme.error) return <Problema erro={filme.error} />
  if (!filme.data) return null

  const f = filme.data
  return (
    <div className="space-y-8">
      <div className="grid gap-6 md:grid-cols-[220px_1fr]">
        <div className="aspect-[2/3] overflow-hidden rounded-xl bg-slate-800">
          {f.imagem_url ? (
            <img src={f.imagem_url} alt="" className="h-full w-full object-cover" />
          ) : (
            <div className="flex h-full items-center justify-center text-6xl text-slate-700">🎬</div>
          )}
        </div>
        <div className="space-y-3">
          <h1 className="text-3xl font-bold">{f.titulo}</h1>
          <div className="flex flex-wrap gap-2">
            <Selo>{f.genero}</Selo>
            <Selo>{f.duracao_minutos} min</Selo>
            <Selo>{f.classificacao_etaria}</Selo>
            <Selo tom={f.status === 'EM_CARTAZ' ? 'bom' : 'neutro'}>{f.status.replace(/_/g, ' ').toLowerCase()}</Selo>
          </div>
          {f.sinopse && <p className="max-w-prose leading-relaxed text-slate-300">{f.sinopse}</p>}
        </div>
      </div>

      <section className="space-y-3">
        <h2 className="text-lg font-bold">Sessões</h2>
        {sessoes.isPending && <Carregando />}
        {sessoes.error && <Problema erro={sessoes.error} />}
        {sessoes.data?.itens.length === 0 && <EstadoVazio titulo="Nenhuma sessão agendada para este filme" />}
        <div className="grid gap-3 sm:grid-cols-2">
          {sessoes.data?.itens.map((s) => (
            <Link key={s.id} to={`/sessoes/${s.id}`}>
              <Cartao className="transition hover:border-destaque">
                <p className="font-mono text-sm">{formatarDataHora(s.data_hora_inicio)}</p>
                <p className="text-sm text-slate-400">
                  {s.cinema_nome} · sala {s.sala_numero} · {s.tipo_tela} · {s.idioma}
                </p>
                <p className="mt-2 font-semibold text-destaque">{moedaDeTexto(s.preco_base)}</p>
              </Cartao>
            </Link>
          ))}
        </div>
      </section>
    </div>
  )
}
