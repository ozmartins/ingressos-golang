# Data Model: Ingresso em PDF

Sem persistência nova. Alteração mínima no instantâneo e tipos derivados no cliente.

## ReservaPagamento (`dominio/reservaPagamento.ts`)

| Campo novo | Tipo | Regra |
|------------|------|-------|
| `classificacao` | `string` (opcional) | gravado em `PaginaCompra`; `lerReserva` aceita ausência (reserva antiga) mas rejeita tipo diferente de string |

## IngressoPdf (derivado, `dominio/ingressoPdf.ts`)

```ts
interface IngressoPdf {
  codigoPedido: string          // "ING-XXXXXXXX"
  nomeArquivo: string           // "ingresso-duna-15-11-1600.pdf"
  aviso: string                 // texto fixo de simulação (montado em montarIngressos)
  resumo: { ingressos: string; taxa: string; total: string; quantidade: number }  // já em "R$ 84,00"
  ingressos: {                  // um por assento
    assento: string             // "B4"
    filme: string
    classificacao?: string
    cinema: string
    sala: string                // "Sala 3"
    dia: string                 // "dom., 15/11"
    horario: string             // "16:00"
  }[]
}
```

**Regras**: um item por `rotulos` (ordem natural fileira/número); `codigoPedido` determinístico (ver research D3); valores via `formatarCentavos`; data/hora via `formatarDiaDe`/`formatarHora` (fuso local); nenhum campo de cartão existe no tipo.

## Estados do botão (em `PaginaPagamento`)

`ocioso → gerando → ocioso` (sucesso: download) | `gerando → erro → gerando` (tentar de novo).
