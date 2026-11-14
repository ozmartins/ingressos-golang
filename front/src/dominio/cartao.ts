/** Dados digitados no formulário de cartão. Existem só em memória (estado do componente). */
export interface DadosCartao {
  numero: string
  nome: string
  validade: string
  cvv: string
}

export type ErrosCartao = Partial<Record<keyof DadosCartao, string>>

const soDigitos = (v: string) => v.replace(/\D/g, '')

/** "4111111111111111" → "4111 1111 1111 1111" (aceita espaços/hífens, máx. 19 dígitos). */
export function formatarNumero(v: string): string {
  return (soDigitos(v).slice(0, 19).match(/.{1,4}/g) ?? []).join(' ')
}

/** "1230" → "12/30"; "1" → "1". */
export function formatarValidade(v: string): string {
  const d = soDigitos(v).slice(0, 4)
  return d.length > 2 ? `${d.slice(0, 2)}/${d.slice(2)}` : d
}

/** CVV: só dígitos, até 4. */
export function formatarCvv(v: string): string {
  return soDigitos(v).slice(0, 4)
}

/** Valida cada campo; devolve apenas os que têm erro. A validade vale até o fim do mês. */
export function validarCartao(dados: DadosCartao, agora: Date = new Date()): ErrosCartao {
  const erros: ErrosCartao = {}

  const numero = soDigitos(dados.numero)
  // Simulação: qualquer número é aceito, desde que preenchido (sem tamanho mínimo nem dígito verificador).
  if (numero === '') erros.numero = 'Informe o número do cartão.'

  if (dados.nome.trim().length < 2) erros.nome = 'Informe o nome impresso no cartão.'

  const m = /^(\d{2})\/(\d{2})$/.exec(dados.validade.trim())
  if (dados.validade.trim() === '') erros.validade = 'Informe a validade.'
  else if (!m || Number(m[1]) < 1 || Number(m[1]) > 12) erros.validade = 'Validade inválida. Use MM/AA.'
  else {
    const fim = new Date(2000 + Number(m[2]), Number(m[1]), 1) // 1º dia do mês seguinte
    if (fim.getTime() <= agora.getTime()) erros.validade = 'Cartão vencido.'
  }

  if (dados.cvv === '') erros.cvv = 'Informe o código de segurança.'
  else if (!/^\d{3,4}$/.test(dados.cvv)) erros.cvv = 'Código de segurança inválido.'

  return erros
}

/** Cartão de teste que simula recusa: número terminado em 0000. */
export function cartaoRecusado(numero: string): boolean {
  return soDigitos(numero).endsWith('0000')
}
