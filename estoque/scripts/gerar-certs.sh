#!/usr/bin/env bash
# Gera CA de DESENVOLVIMENTO e os pares servidor/cliente usados pelo mTLS do
# canal gRPC. Nada aqui serve para produção: em produção o material vem do
# ambiente (constituição, princípio II; FR-040).
set -euo pipefail

DIR="${1:-certs}"
mkdir -p "$DIR"
cd "$DIR"

if [ -f ca.pem ] && [ -f servidor.pem ] && [ -f cliente.pem ]; then
  echo "certificados já existem em $DIR — nada a fazer"
  exit 0
fi

# CA
openssl req -x509 -newkey rsa:2048 -nodes -keyout ca-key.pem -out ca.pem \
  -days 365 -subj "/CN=estoque-dev-ca" 2>/dev/null

gerar_par() {
  local nome="$1" cn="$2"
  openssl req -newkey rsa:2048 -nodes -keyout "${nome}-key.pem" -out "${nome}.csr" \
    -subj "/CN=${cn}" 2>/dev/null
  openssl x509 -req -in "${nome}.csr" -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
    -out "${nome}.pem" -days 365 \
    -extfile <(printf "subjectAltName=DNS:localhost,DNS:estoque,IP:127.0.0.1\nextendedKeyUsage=serverAuth,clientAuth\n") 2>/dev/null
  rm -f "${nome}.csr"
}

gerar_par servidor estoque
gerar_par cliente servico-catalogo

# Estes arquivos são montados dentro do contêiner, que roda como usuário não-root
# (uid 65532 na imagem distroless) e não é dono deles. Sem modo legível por
# outros, o serviço não sobe. Isto vale APENAS para o material descartável de
# desenvolvimento: em produção a chave vem do ambiente, com dono e modo próprios
# (constituição, princípio II).
chmod 0644 ./*.pem

echo "certificados de desenvolvimento gerados em $DIR"
echo "aviso: chaves com modo 0644 — material descartável, apenas para desenvolvimento"
