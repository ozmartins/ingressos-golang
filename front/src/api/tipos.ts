export type StatusFilme = 'EM_CARTAZ' | 'BREVE' | 'FORA_DE_CARTAZ'

export interface Filme {
  id: string
  titulo: string
  sinopse?: string
  duracao_minutos: number
  classificacao_etaria: string
  genero: string
  imagem_url?: string
  status: StatusFilme
}

export interface Cinema {
  id: string
  nome: string
  cidade: string
  estado: string
  endereco: string
  /** URL remota do logo; `null` quando o cinema não tem logo. */
  logo_url?: string | null
  ativo: boolean
}

export interface Paginacao {
  pagina: number
  tamanho: number
  total: number
  tem_proxima: boolean
}

export interface PaginaApi<T> {
  itens: T[]
  pagina: Paginacao
}

export type TipoTela = '2D' | '3D' | 'IMAX' | 'VIP'

export type Idioma = 'DUBLADO' | 'LEGENDADO'

export interface Sala {
  id: string
  cinema_id: string
  numero: number
  tipo_tela: TipoTela
  capacidade_total: number
  ativo: boolean
}

/** Sessão da grade pública, já com filme, cinema e sala resolvidos. */
export interface SessaoNaGrade {
  id: string
  filme_id: string
  filme_titulo: string
  cinema_id: string
  cinema_nome: string
  sala_numero: number
  tipo_tela: TipoTela
  /** ISO 8601 (UTC). */
  data_hora_inicio: string
  idioma: Idioma
  /** Decimal exato em texto, p. ex. "42.00". */
  preco_base: string
}

export type TipoPoltrona = 'NORMAL' | 'PCD' | 'NAMORADEIRA'

export type StatusPoltrona = 'LIVRE' | 'RESERVADA' | 'OCUPADA'

/** Poltrona (assento) da sessão, como o estoque a devolve. */
export interface Poltrona {
  /** Identidade no escopo da sessão, p. ex. "B4". */
  rotulo: string
  fileira: string
  numero: number
  tipo: TipoPoltrona
  status: StatusPoltrona
}

export interface MapaPoltronas {
  sessao_id: string
  poltronas: Poltrona[]
}

export interface RespostaBloqueio {
  reserva_id: string
  /** ISO 8601: instante em que a reserva pendente expira. */
  expira_em: string
  mensagem: string
}

/** Erro no formato RFC 9457 devolvido pelo estoque. */
export interface Problem {
  type: string
  title: string
  status: number
  detail?: string
  metadata?: Record<string, string>
}
