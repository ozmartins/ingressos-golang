import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode, SelectHTMLAttributes } from 'react'

export function Cartao({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <div className={`rounded-xl border border-borda bg-painel p-5 ${className}`}>{children}</div>
}

type VarianteBotao = 'primario' | 'secundario' | 'perigo'

const variantes: Record<VarianteBotao, string> = {
  primario: 'bg-destaque text-slate-950 hover:brightness-110',
  secundario: 'border border-borda bg-transparent text-slate-200 hover:bg-painel',
  perigo: 'border border-rose-500/50 text-rose-300 hover:bg-rose-500/10',
}

export function Botao({
  variante = 'primario',
  className = '',
  ...resto
}: ButtonHTMLAttributes<HTMLButtonElement> & { variante?: VarianteBotao }) {
  return (
    <button
      {...resto}
      className={`rounded-lg px-4 py-2 text-sm font-semibold transition disabled:cursor-not-allowed disabled:opacity-40 ${variantes[variante]} ${className}`}
    />
  )
}

const classeCampo =
  'w-full rounded-lg border border-borda bg-fundo px-3 py-2 text-sm outline-none focus:border-destaque'

export function Campo({
  rotulo,
  dica,
  className = '',
  ...resto
}: InputHTMLAttributes<HTMLInputElement> & { rotulo: string; dica?: string }) {
  return (
    <label className={`block ${className}`}>
      <span className="mb-1 block text-xs font-medium text-slate-400">{rotulo}</span>
      <input {...resto} className={classeCampo} />
      {dica && <span className="mt-1 block text-xs text-slate-500">{dica}</span>}
    </label>
  )
}

export function Selecao({
  rotulo,
  children,
  className = '',
  ...resto
}: SelectHTMLAttributes<HTMLSelectElement> & { rotulo: string; children: ReactNode }) {
  return (
    <label className={`block ${className}`}>
      <span className="mb-1 block text-xs font-medium text-slate-400">{rotulo}</span>
      <select {...resto} className={classeCampo}>
        {children}
      </select>
    </label>
  )
}

export function Selo({ children, tom = 'neutro' }: { children: ReactNode; tom?: 'neutro' | 'bom' | 'ruim' | 'espera' }) {
  const tons = {
    neutro: 'bg-slate-700/60 text-slate-200',
    bom: 'bg-emerald-500/20 text-emerald-300',
    ruim: 'bg-rose-500/20 text-rose-300',
    espera: 'bg-amber-500/20 text-amber-300',
  }
  return <span className={`rounded-full px-2.5 py-1 text-xs font-semibold ${tons[tom]}`}>{children}</span>
}

export function EstadoVazio({ titulo, descricao }: { titulo: string; descricao?: string }) {
  return (
    <div className="rounded-xl border border-dashed border-borda p-10 text-center">
      <p className="font-medium text-slate-300">{titulo}</p>
      {descricao && <p className="mt-1 text-sm text-slate-500">{descricao}</p>}
    </div>
  )
}

export function Carregando({ texto = 'Carregando…' }: { texto?: string }) {
  return <p className="py-8 text-center text-sm text-slate-500">{texto}</p>
}
