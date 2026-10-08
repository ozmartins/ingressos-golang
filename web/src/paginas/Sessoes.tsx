import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { listarCinemas, listarFilmes, listarSessoes } from '../api/catalogo'
import type { SessaoNaGrade } from '../api/tipos'
import { formatarDataHora, moedaDeTexto } from '../formato'
import { Paginacao } from '../componentes/Paginacao'
import { Problema } from '../componentes/Problema'
import { Botao, Carregando, Cartao, EstadoVazio, Selecao, Selo } from '../componentes/ui'

// A grade é pública: dá para percorrer o catálogo sem entrar. O login só é
// cobrado na hora de escolher poltrona.
export function Sessoes() {
  const [filtros, setFiltros] = useState({ filme_id: '', cinema_id: '', data: '' })
  const [pagina, setPagina] = useState(1)

  const filmes = useQuery({ queryKey: ['filmes', 'todos'], queryFn: () => listarFilmes({ page_size: 100 }) })
  const cinemas = useQuery({ queryKey: ['cinemas', 'todos'], queryFn: () => listarCinemas({ page_size: 100 }) })

  const grade = useQuery({
    queryKey: ['sessoes', filtros, pagina],
    queryFn: () => listarSessoes({ ...filtros, page: pagina, page_size: 20 }),
  })

  function mudarFiltro(chave: keyof typeof filtros, valor: string) {
    setFiltros((f) => ({ ...f, [chave]: valor }))
    setPagina(1)
  }

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-bold">Grade de sessões</h1>
        <p className="text-sm text-slate-400">Somente sessões agendadas ou em andamento.</p>
      </header>

      <Cartao className="grid gap-4 sm:grid-cols-4">
        <Selecao rotulo="Filme" value={filtros.filme_id} onChange={(e) => mudarFiltro('filme_id', e.target.value)}>
          <option value="">Todos</option>
          {filmes.data?.itens.map((f) => (
            <option key={f.id} value={f.id}>
              {f.titulo}
            </option>
          ))}
        </Selecao>
        <Selecao rotulo="Cinema" value={filtros.cinema_id} onChange={(e) => mudarFiltro('cinema_id', e.target.value)}>
          <option value="">Todos</option>
          {cinemas.data?.itens.map((c) => (
            <option key={c.id} value={c.id}>
              {c.nome}
            </option>
          ))}
        </Selecao>
        <label className="block">
          <span className="mb-1 block text-xs font-medium text-slate-400">Data</span>
          <input
            type="date"
            value={filtros.data}
            onChange={(e) => mudarFiltro('data', e.target.value)}
            className="w-full rounded-lg border border-borda bg-fundo px-3 py-2 text-sm outline-none focus:border-destaque"
          />
        </label>
        <div className="flex items-end">
          <Botao
            variante="secundario"
            className="w-full"
            onClick={() => {
              setFiltros({ filme_id: '', cinema_id: '', data: '' })
              setPagina(1)
            }}
          >
            Limpar filtros
          </Botao>
        </div>
      </Cartao>

      {grade.isPending && <Carregando />}
      {grade.error && <Problema erro={grade.error} />}

      {grade.data && grade.data.itens.length === 0 && (
        <EstadoVazio
          titulo="Nenhuma sessão na grade"
          descricao="Cadastre filme, cinema, sala e sessão em Operação para ver algo aqui."
        />
      )}

      <div className="grid gap-3">
        {grade.data?.itens.map((s) => (
          <LinhaDaGrade key={s.id} sessao={s} />
        ))}
      </div>

      {grade.data && <Paginacao pagina={grade.data.pagina} aoMudar={setPagina} />}
    </div>
  )
}

function LinhaDaGrade({ sessao }: { sessao: SessaoNaGrade }) {
  return (
    <Link
      to={`/sessoes/${sessao.id}`}
      className="flex flex-wrap items-center gap-4 rounded-xl border border-borda bg-painel p-4 transition hover:border-destaque"
    >
      <div className="min-w-48 flex-1">
        <p className="font-semibold">{sessao.filme_titulo}</p>
        <p className="text-sm text-slate-400">
          {sessao.cinema_nome} · sala {sessao.sala_numero}
        </p>
      </div>
      <div className="flex items-center gap-2">
        <Selo>{sessao.tipo_tela}</Selo>
        <Selo>{sessao.idioma}</Selo>
      </div>
      <div className="text-right">
        <p className="font-mono text-sm">{formatarDataHora(sessao.data_hora_inicio)}</p>
        <p className="text-sm font-semibold text-destaque">{moedaDeTexto(sessao.preco_base)}</p>
      </div>
    </Link>
  )
}
