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
