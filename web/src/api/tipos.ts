// Escritos a partir dos quatro `openapi.yaml` dos serviços. Os nomes seguem os
// dos contratos, em português, para que a comparação com a especificação seja
// direta.

// ---------------------------------------------------------------- catálogo

export type StatusFilme = 'EM_CARTAZ' | 'BREVE' | 'FORA_DE_CARTAZ'
export type TipoTela = '2D' | '3D' | 'IMAX' | 'VIP'
export type TipoAssento = 'NORMAL' | 'PCD' | 'NAMORADEIRA'
export type Idioma = 'DUBLADO' | 'LEGENDADO'
export type StatusSessao = 'AGENDADA' | 'EM_ANDAMENTO' | 'FINALIZADA' | 'CANCELADA'

export interface Paginacao {
  pagina: number
  tamanho: number
  total: number
  tem_proxima: boolean
}

export interface Pagina<T> {
  itens: T[]
  pagina: Paginacao
}

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

export type FilmeEntrada = Omit<Filme, 'id' | 'status'> & { status?: StatusFilme }

export interface Cinema {
  id: string
  nome: string
  cidade: string
  estado: string
  endereco: string
  ativo: boolean
}

export type CinemaEntrada = Omit<Cinema, 'id' | 'ativo'> & { ativo?: boolean }

export interface Fileira {
  fileira: string
  assentos: number
  tipo?: TipoAssento
}

export interface Sala {
  id: string
  cinema_id: string
  numero: number
  tipo_tela: TipoTela
  fileiras: Fileira[]
  capacidade_total: number
  ativo: boolean
}

export interface SalaEntrada {
  cinema_id: string
  numero: number
  tipo_tela: TipoTela
  fileiras: Fileira[]
  ativo?: boolean
}

export interface Sessao {
  id: string
  filme_id: string
  sala_id: string
  data_hora_inicio: string
  idioma: Idioma
  // Decimal exato em texto. Nunca converter para número antes de exibir: é
  // este o valor que vira cobrança.
  preco_base: string
  status: StatusSessao
}

/** O PUT da sala não redesenha a planta: `fileiras` é imutável depois do
 *  cadastro, e `cinema_id`, se vier, precisa repetir o cinema atual. */
export interface SalaAtualizacao {
  cinema_id?: string
  numero: number
  tipo_tela: TipoTela
  ativo?: boolean
}

export interface SessaoEntrada {
  filme_id: string
  sala_id: string
  data_hora_inicio: string
  idioma: Idioma
  preco_base: string
  status?: StatusSessao
}

/** O PUT da sessão não a move de sala: `sala_id`, se vier, precisa repetir a
 *  sala atual. */
export interface SessaoAtualizacao {
  filme_id: string
  sala_id?: string
  data_hora_inicio: string
  idioma: Idioma
  preco_base: string
  status?: StatusSessao
}

export interface SessaoNaGrade {
  id: string
  filme_id: string
  filme_titulo: string
  cinema_id: string
  cinema_nome: string
  sala_numero: number
  tipo_tela: TipoTela
  data_hora_inicio: string
  idioma: Idioma
  preco_base: string
}

export interface SolicitacaoReserva {
  poltronas_ids: string[]
}

export interface ReservaConfirmada {
  reserva_id: string
  expira_em: string
}

// ----------------------------------------------------------------- estoque

export type StatusPoltrona = 'LIVRE' | 'RESERVADA' | 'OCUPADA'

export interface Poltrona {
  rotulo: string
  fileira: string
  numero: number
  tipo: TipoAssento
  status: StatusPoltrona
}

export interface RespostaMapa {
  sessao_id: string
  poltronas: Poltrona[]
}

// --------------------------------------------------------------- pagamento

export type FormaPagamento = 'PIX' | 'CARTAO_CREDITO'

export type StatusPagamento =
  | 'AGUARDANDO_FORMA'
  | 'PROCESSANDO'
  | 'PAGO'
  | 'RECUSADO'
  | 'CANCELADO'
  | 'PENDENTE_VERIFICACAO'

export interface Pagamento {
  transacao_id: string
  reserva_id: string
  status: StatusPagamento
  valor_total: number
  forma_pagamento?: FormaPagamento
  motivo_falha?: string
  codigo_transacao_externa?: string
  expira_em: string
  criado_em: string
}

// ------------------------------------------------------------- notificação

export type StatusIngresso = 'VALIDO' | 'UTILIZADO' | 'CANCELADO'

export interface Ingresso {
  ingresso_id: string
  reserva_id: string
  codigo_qr: string
  status: StatusIngresso
  criado_em: string
}

export interface EntradaAutorizada {
  valido: true
  mensagem: string
  ingresso_id: string
  utilizado_em: string
}

export interface EntradaRecusada {
  valido: false
  mensagem: string
}

export type ResultadoValidacao = EntradaAutorizada | EntradaRecusada
