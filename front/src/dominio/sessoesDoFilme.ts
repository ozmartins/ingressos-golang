import type { SessaoNaGrade } from '../api/tipos'

/** Sessões futuras do filme (a grade já vem de um único cinema), em ordem cronológica. */
export function sessoesDoFilme(sessoes: SessaoNaGrade[], filmeId: string, agora: Date = new Date()): SessaoNaGrade[] {
  return sessoes
    .filter((s) => s.filme_id === filmeId && new Date(s.data_hora_inicio).getTime() > agora.getTime())
    .sort((a, b) => new Date(a.data_hora_inicio).getTime() - new Date(b.data_hora_inicio).getTime())
}

/**
 * Sessão a exibir: a pedida na URL, se estiver entre as disponíveis; senão a
 * primeira (id ausente, inexistente, passado ou de outro filme → primeira futura).
 */
export function sessaoPadrao(lista: SessaoNaGrade[], sessaoId?: string | null): SessaoNaGrade | undefined {
  return lista.find((s) => s.id === sessaoId) ?? lista[0]
}
