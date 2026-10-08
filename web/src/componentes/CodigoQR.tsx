import { useEffect, useRef } from 'react'
import QRCode from 'qrcode'

export function CodigoQR({ valor, tamanho = 180 }: { valor: string; tamanho?: number }) {
  const canvas = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    if (!canvas.current) return
    void QRCode.toCanvas(canvas.current, valor, {
      width: tamanho,
      margin: 1,
      color: { dark: '#0f172a', light: '#ffffff' },
    })
  }, [valor, tamanho])

  return <canvas ref={canvas} className="rounded-lg bg-white p-2" aria-label="Código do ingresso" />
}
