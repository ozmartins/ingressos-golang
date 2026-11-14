import { useId } from 'react'
import type { SessaoNaGrade } from '../api/tipos'
import { formatarDiaDe, formatarHora, ROTULO_IDIOMA } from '../dominio/formatar'

interface Props {
  sessao: SessaoNaGrade
  /** Todas as sessões futuras do filme neste cinema (inclui a atual). */
  sessoes: SessaoNaGrade[]
  onTrocar: (sessaoId: string) => void
  /** Bloqueia a troca (reserva em andamento ou feita). */
  desabilitado?: boolean
}

function rotuloOpcao(s: SessaoNaGrade): string {
  const inicio = new Date(s.data_hora_inicio)
  return `${formatarDiaDe(inicio)} · ${formatarHora(inicio)} · Sala ${s.sala_numero} · ${s.tipo_tela} · ${ROTULO_IDIOMA[s.idioma]}`
}

/** Etapa atual, dados da sessão (cinema, sala, dia, hora) e troca de sessão. */
export function CabecalhoSessao({ sessao, sessoes, onTrocar, desabilitado = false }: Props) {
  const id = useId()
  const inicio = new Date(sessao.data_hora_inicio)
  return (
    <header className="sessao-cabecalho">
      <p className="sessao-etapa">Etapa 1 de 1 · Selecione seus assentos</p>
      <ul className="sessao-dados" aria-label="Dados da sessão">
        <li>{sessao.cinema_nome} - Sala {sessao.sala_numero}</li>
        <li>
          <time dateTime={inicio.toISOString()}>
            {formatarDiaDe(inicio)} · <strong>{formatarHora(inicio)}</strong>
          </time>
        </li>
      </ul>
      {sessoes.length > 1 ? (
        <div className="campo">
          <label htmlFor={id}>Sessão</label>
          <select id={id} value={sessao.id} disabled={desabilitado} onChange={(e) => onTrocar(e.target.value)}>
            {sessoes.map((s) => (
              <option key={s.id} value={s.id}>
                {rotuloOpcao(s)}
              </option>
            ))}
          </select>
        </div>
      ) : null}
    </header>
  )
}
