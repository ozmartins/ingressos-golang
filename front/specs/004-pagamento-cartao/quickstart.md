# Quickstart: validar a página de pagamento (simulada)

## Pré-requisitos

Plataforma no ar (`cd ingressos-golang && docker compose up --build`), com filmes/sessões semeados; usuário cadastrado no Keycloak.

```
cd ingressos-golang/front
npm install
npm run dev
```

## Cenários

1. **Fluxo feliz**: logado, abrir um cinema → "Comprar Ingressos" → escolher 2 assentos → "Continuar". Esperado: abre `/cinemas/.../pagamento` com resumo (assentos e total) e contagem regressiva.
2. **Aprovação**: cartão `4111 1111 1111 1111`, nome qualquer, validade futura (ex.: `12/40`), CVV `123` → "Pagar". Esperado: "Processando…" ~2 s e "Pagamento aprovado".
3. **Recusa**: qualquer número terminado em `0000` (ex.: `1234 0000`), demais campos válidos → "Pagar". Esperado: "Pagamento recusado" e "Tentar outro cartão".
4. **Validação**: enviar em branco; validade vencida; CVV curto. Esperado: erro por campo, sem processamento.
5. **Conflito na reserva**: reservar assento já tomado. Esperado: continua na seleção com o aviso; não abre o pagamento.
6. **Acesso direto/recarga**: abrir o endereço do pagamento sem reserva → "Nenhuma reserva em andamento". Deslogado → login.
7. **Expiração**: aguardar o prazo (10 min) ou alterar `expiraEm` no `sessionStorage` → "Reserva expirada" e pagamento impedido.
8. **Privacidade**: no DevTools, aba Network não mostra requisições ao pagar; Application → Storage não contém dados de cartão.

## Automatizados

```
npm test
npm run lint
npm run build
```
