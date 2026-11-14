import type { jsPDF } from 'jspdf'
import type { IngressoPdf } from '../dominio/ingressoPdf'

const LARGURA = 210
const ALTURA = 297
const MARGEM = 20
const UTIL = LARGURA - 2 * MARGEM

/** Quebra o texto na largura útil; reduz a fonte (do maior para o menor tamanho) até caber em `maxLinhas`. */
function ajustar(doc: jsPDF, texto: string, tamanhos: number[], maxLinhas: number): { linhas: string[]; tamanho: number } {
  let linhas: string[] = []
  for (const tamanho of tamanhos) {
    doc.setFontSize(tamanho)
    linhas = doc.splitTextToSize(texto, UTIL) as string[]
    if (linhas.length <= maxLinhas) return { linhas, tamanho }
  }
  const tamanho = tamanhos[tamanhos.length - 1]
  return { linhas: linhas.slice(0, maxLinhas), tamanho }
}

function cabecalho(doc: jsPDF, titulo: string) {
  doc.setFillColor(30, 30, 40)
  doc.rect(0, 0, LARGURA, 28, 'F')
  doc.setTextColor(255, 255, 255)
  doc.setFont('helvetica', 'bold')
  doc.setFontSize(20)
  doc.text('INGRESSOS', MARGEM, 18)
  doc.setFontSize(11)
  doc.text(titulo, LARGURA - MARGEM, 18, { align: 'right' })
  doc.setTextColor(20, 20, 20)
}

function rodape(doc: jsPDF, aviso: string, codigo: string, pagina: number, total: number) {
  doc.setDrawColor(120, 120, 120)
  doc.line(MARGEM, ALTURA - 32, LARGURA - MARGEM, ALTURA - 32)
  doc.setFont('helvetica', 'bold')
  doc.setFontSize(9)
  doc.setTextColor(20, 20, 20)
  doc.text(doc.splitTextToSize(aviso, UTIL) as string[], MARGEM, ALTURA - 24)
  doc.setFont('helvetica', 'normal')
  doc.text(`Pedido ${codigo}`, MARGEM, ALTURA - 14)
  doc.text(`Página ${pagina} de ${total}`, LARGURA - MARGEM, ALTURA - 14, { align: 'right' })
}

function campo(doc: jsPDF, rotulo: string, valor: string, y: number): number {
  doc.setFont('helvetica', 'normal')
  doc.setFontSize(10)
  doc.setTextColor(90, 90, 90)
  doc.text(rotulo.toUpperCase(), MARGEM, y)
  doc.setFont('helvetica', 'bold')
  doc.setFontSize(14)
  doc.setTextColor(20, 20, 20)
  const linhas = doc.splitTextToSize(valor, UTIL) as string[]
  doc.text(linhas, MARGEM, y + 7)
  return y + 7 + linhas.length * 6 + 8
}

/** Monta o documento: uma página por ingresso (assento) e uma página própria de resumo (N+1 páginas). */
export async function gerarDocumento(ingresso: IngressoPdf): Promise<jsPDF> {
  const { jsPDF: Documento } = await import('jspdf')
  const doc = new Documento({ unit: 'mm', format: 'a4', orientation: 'portrait' })
  doc.setProperties({ title: `Ingresso ${ingresso.codigoPedido}`, subject: 'Ingresso (simulação)', creator: 'Ingressos' })
  const total = ingresso.ingressos.length + 1

  ingresso.ingressos.forEach((item, indice) => {
    if (indice > 0) doc.addPage()
    cabecalho(doc, 'Ingresso')

    const { linhas, tamanho } = ajustar(doc, item.filme, [24, 20, 16, 13], 3)
    doc.setFont('helvetica', 'bold')
    doc.setFontSize(tamanho)
    doc.setTextColor(20, 20, 20)
    doc.text(linhas, MARGEM, 44)
    let y = 44 + linhas.length * (tamanho * 0.45) + 4
    if (item.classificacao) {
      doc.setFont('helvetica', 'normal')
      doc.setFontSize(11)
      doc.text(`Classificação: ${item.classificacao}`, MARGEM, y)
      y += 10
    }
    y += 4
    y = campo(doc, 'Cinema', item.cinema, y)
    y = campo(doc, 'Sala', item.sala, y)
    y = campo(doc, 'Dia', item.dia, y)
    y = campo(doc, 'Horário', item.horario, y)

    doc.setDrawColor(30, 30, 40)
    doc.setLineWidth(0.8)
    doc.roundedRect(MARGEM, y + 2, UTIL, 46, 3, 3, 'S')
    doc.setFont('helvetica', 'normal')
    doc.setFontSize(11)
    doc.setTextColor(90, 90, 90)
    doc.text('ASSENTO', LARGURA / 2, y + 14, { align: 'center' })
    doc.setFont('helvetica', 'bold')
    doc.setFontSize(54)
    doc.setTextColor(20, 20, 20)
    doc.text(item.assento, LARGURA / 2, y + 38, { align: 'center' })

    rodape(doc, ingresso.aviso, ingresso.codigoPedido, indice + 1, total)
  })

  doc.addPage()
  cabecalho(doc, 'Resumo do pedido')
  let y = 48
  y = campo(doc, 'Código do pedido', ingresso.codigoPedido, y)
  y = campo(doc, 'Ingressos', `${ingresso.resumo.quantidade} × (${ingresso.resumo.ingressos})`, y)
  y = campo(doc, 'Taxa', ingresso.resumo.taxa, y)
  doc.setLineWidth(0.4)
  doc.line(MARGEM, y - 2, LARGURA - MARGEM, y - 2)
  campo(doc, 'Total', ingresso.resumo.total, y + 4)
  rodape(doc, ingresso.aviso, ingresso.codigoPedido, total, total)
  return doc
}

/** Gera o PDF no navegador (sem rede) e dispara o download com o nome do arquivo do ingresso. */
export async function baixarIngressoPdf(ingresso: IngressoPdf): Promise<void> {
  const doc = await gerarDocumento(ingresso)
  const url = URL.createObjectURL(doc.output('blob'))
  const ancora = document.createElement('a')
  ancora.href = url
  ancora.download = ingresso.nomeArquivo
  ancora.style.display = 'none'
  document.body.appendChild(ancora)
  try {
    ancora.click()
  } finally {
    ancora.remove()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
  }
}
