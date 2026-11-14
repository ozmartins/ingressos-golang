import { useId, useRef, useState, type FormEvent } from 'react'
import { formatarCvv, formatarNumero, formatarValidade, validarCartao, type DadosCartao, type ErrosCartao } from '../dominio/cartao'

interface Props {
  /** Chamado só com dados válidos. */
  onPagar: (dados: DadosCartao) => void
  desabilitado: boolean
  processando: boolean
}

const CAMPOS: (keyof DadosCartao)[] = ['numero', 'nome', 'validade', 'cvv']

/** Formulário de cartão (simulado): máscaras, validação por campo e erros acessíveis. */
export function FormularioCartao({ onPagar, desabilitado, processando }: Props) {
  const id = useId()
  const [dados, setDados] = useState<DadosCartao>({ numero: '', nome: '', validade: '', cvv: '' })
  const [erros, setErros] = useState<ErrosCartao>({})
  const form = useRef<HTMLFormElement>(null)

  function alterar(campo: keyof DadosCartao, valor: string) {
    setDados((d) => ({ ...d, [campo]: valor }))
    if (erros[campo]) setErros((e) => ({ ...e, [campo]: undefined }))
  }

  function enviar(evento: FormEvent) {
    evento.preventDefault()
    if (desabilitado || processando) return
    const e = validarCartao(dados)
    setErros(e)
    const primeiro = CAMPOS.find((c) => e[c])
    if (primeiro) {
      form.current?.querySelector<HTMLInputElement>(`[name="${primeiro}"]`)?.focus()
      return
    }
    onPagar(dados)
  }

  const campo = (nome: keyof DadosCartao, rotulo: string, extra: React.InputHTMLAttributes<HTMLInputElement>, formatar: (v: string) => string = (v) => v) => (
    <div className="campo">
      <label htmlFor={`${id}-${nome}`}>{rotulo}</label>
      <input
        id={`${id}-${nome}`}
        name={nome}
        value={dados[nome]}
        onChange={(e) => alterar(nome, formatar(e.target.value))}
        disabled={desabilitado || processando}
        aria-invalid={erros[nome] ? true : undefined}
        aria-describedby={erros[nome] ? `${id}-${nome}-erro` : undefined}
        {...extra}
      />
      {erros[nome] ? (
        <p className="campo-erro" id={`${id}-${nome}-erro`}>
          {erros[nome]}
        </p>
      ) : null}
    </div>
  )

  return (
    <form ref={form} className="formulario-cartao" onSubmit={enviar} noValidate aria-labelledby={`${id}-titulo`}>
      <h2 id={`${id}-titulo`}>Pagamento com cartão</h2>
      <p className="aviso-simulacao">
        Simulação acadêmica: nenhuma cobrança é feita. Não informe dados reais de cartão. Para testar uma recusa, use um número terminado em 0000.
      </p>
      {campo('numero', 'Número do cartão', { inputMode: 'numeric', autoComplete: 'cc-number', placeholder: '0000 0000 0000 0000' }, formatarNumero)}
      {campo('nome', 'Nome impresso no cartão', { autoComplete: 'cc-name' })}
      <div className="campos-linha">
        {campo('validade', 'Validade (MM/AA)', { inputMode: 'numeric', autoComplete: 'cc-exp', placeholder: 'MM/AA' }, formatarValidade)}
        {campo('cvv', 'Código de segurança', { inputMode: 'numeric', autoComplete: 'cc-csc', placeholder: '123' }, formatarCvv)}
      </div>
      <button type="submit" className="botao-comprar" disabled={desabilitado || processando}>
        {processando ? 'Processando…' : 'Pagar'}
      </button>
    </form>
  )
}
