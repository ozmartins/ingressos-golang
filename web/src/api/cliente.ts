// Os quatro serviços erram em dois dialetos: catálogo, estoque e notificação
// falam RFC 9457 (`application/problem+json`), e o pagamento tem forma própria
// (`{ codigo, mensagem }`). Aqui os dois viram o mesmo `ErroDaApi`, para que
// nenhuma tela precise saber com quem está falando.

export interface CampoInvalido {
  campo: string
  mensagem: string
}

export class ErroDaApi extends Error {
  readonly status: number
  /** Categoria estável do erro — o último segmento do `type` da RFC 9457, ou
   *  o `codigo` do pagamento. É por ela que as telas decidem o que dizer. */
  readonly categoria: string
  readonly detalhe?: string
  readonly campos?: CampoInvalido[]

  constructor(status: number, categoria: string, mensagem: string, detalhe?: string, campos?: CampoInvalido[]) {
    super(mensagem)
    this.name = 'ErroDaApi'
    this.status = status
    this.categoria = categoria
    this.detalhe = detalhe
    this.campos = campos
  }
}

type Servico = 'catalogo' | 'estoque' | 'pagamento' | 'notificacao'

interface Opcoes {
  metodo?: 'GET' | 'POST' | 'PUT' | 'DELETE'
  corpo?: unknown
  /** Token de acesso; ausente significa chamada pública. */
  token?: string | null
  /** Chave da portaria, que substitui o token na única rota que a exige. */
  chaveDeApi?: string
  consulta?: Record<string, string | number | undefined | null>
}

function montarCaminho(servico: Servico, rota: string, consulta?: Opcoes['consulta']): string {
  const base = `/api/${servico}${rota}`
  if (!consulta) return base
  const params = new URLSearchParams()
  for (const [chave, valor] of Object.entries(consulta)) {
    if (valor !== undefined && valor !== null && valor !== '') params.set(chave, String(valor))
  }
  const texto = params.toString()
  return texto ? `${base}?${texto}` : base
}

function categoriaDoTipo(tipo: unknown): string {
  if (typeof tipo !== 'string' || tipo === '') return 'desconhecido'
  const partes = tipo.split('/')
  return partes[partes.length - 1] || 'desconhecido'
}

async function erroDaResposta(resposta: Response): Promise<ErroDaApi> {
  let corpo: Record<string, unknown> = {}
  try {
    corpo = (await resposta.json()) as Record<string, unknown>
  } catch {
    return new ErroDaApi(resposta.status, 'desconhecido', `Falha ${resposta.status} sem corpo legível.`)
  }

  // Dialeto do pagamento.
  if (typeof corpo.codigo === 'string') {
    return new ErroDaApi(resposta.status, corpo.codigo, String(corpo.mensagem ?? 'Falha no pagamento.'))
  }

  const campos = Array.isArray(corpo.errors) ? (corpo.errors as CampoInvalido[]) : undefined
  return new ErroDaApi(
    resposta.status,
    categoriaDoTipo(corpo.type),
    String(corpo.title ?? `Falha ${resposta.status}`),
    typeof corpo.detail === 'string' ? corpo.detail : undefined,
    campos,
  )
}

export async function chamar<T>(servico: Servico, rota: string, opcoes: Opcoes = {}): Promise<T> {
  const cabecalhos: Record<string, string> = { Accept: 'application/json, application/problem+json' }
  if (opcoes.token) cabecalhos.Authorization = `Bearer ${opcoes.token}`
  if (opcoes.chaveDeApi) cabecalhos['X-API-Key'] = opcoes.chaveDeApi
  if (opcoes.corpo !== undefined) cabecalhos['Content-Type'] = 'application/json'

  let resposta: Response
  try {
    resposta = await fetch(montarCaminho(servico, rota, opcoes.consulta), {
      method: opcoes.metodo ?? 'GET',
      headers: cabecalhos,
      body: opcoes.corpo !== undefined ? JSON.stringify(opcoes.corpo) : undefined,
    })
  } catch {
    // Rede indisponível ou proxy fora do ar: é a mesma classe de falha do 503
    // que os serviços devolvem, e a tela trata igual.
    throw new ErroDaApi(0, 'sem-conexao', 'Não foi possível falar com o serviço.')
  }

  if (!resposta.ok) throw await erroDaResposta(resposta)

  if (resposta.status === 204 || resposta.headers.get('Content-Length') === '0') {
    return undefined as T
  }
  const texto = await resposta.text()
  return (texto ? JSON.parse(texto) : undefined) as T
}
