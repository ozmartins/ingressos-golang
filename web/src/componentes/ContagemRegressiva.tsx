import { useEffect, useState } from 'react'

/** Mostra quanto falta para um instante. A reserva e o pagamento têm prazo, e
 *  quem está comprando precisa ver isso correndo. */
export function ContagemRegressiva({ ate, aoVencer }: { ate: string; aoVencer?: () => void }) {
  const alvo = new Date(ate).getTime()
  const [restante, setRestante] = useState(() => alvo - Date.now())

  useEffect(() => {
    const id = setInterval(() => setRestante(alvo - Date.now()), 1000)
    return () => clearInterval(id)
  }, [alvo])

  useEffect(() => {
    if (restante <= 0) aoVencer?.()
  }, [restante <= 0])

  if (restante <= 0) return <span className="font-mono text-rose-300">expirada</span>

  const total = Math.floor(restante / 1000)
  const minutos = String(Math.floor(total / 60)).padStart(2, '0')
  const segundos = String(total % 60).padStart(2, '0')
  return (
    <span className={`font-mono ${restante < 60_000 ? 'text-rose-300' : 'text-amber-300'}`}>
      {minutos}:{segundos}
    </span>
  )
}
