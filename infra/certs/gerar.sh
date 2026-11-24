#!/bin/sh
# Gera a CA de desenvolvimento e os certificados do gRPC estoque -> catalogo
# (servidor: catalogo; cliente: estoque, para mTLS). Idempotente: se a CA já
# existe no volume, não faz nada. Roda no serviço `certs-grpc` do compose.
set -eu
DIR="${1:-/certs}"
cd "$DIR"
if [ -f ca.pem ] && [ -f catalogo.pem ] && [ -f estoque.pem ]; then
  echo "certificados já existem em $DIR"
  exit 0
fi

openssl req -x509 -newkey rsa:3072 -nodes -days 825 -subj "/CN=cinema-dev-ca" \
  -keyout ca-key.pem -out ca.pem

emitir() { # nome, SAN, extendedKeyUsage
  openssl req -newkey rsa:3072 -nodes -subj "/CN=$1" -keyout "$1-key.pem" -out "$1.csr"
  printf 'subjectAltName=%s\nextendedKeyUsage=%s\n' "$2" "$3" > "$1.ext"
  openssl x509 -req -in "$1.csr" -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
    -days 825 -extfile "$1.ext" -out "$1.pem"
  rm -f "$1.csr" "$1.ext"
}
emitir catalogo "DNS:catalogo,DNS:localhost,IP:127.0.0.1" serverAuth
emitir estoque "DNS:estoque" clientAuth

# Imagens distroless rodam como "nonroot" (uid 65532): precisam ler as chaves.
# Material só de desenvolvimento, dentro de um volume nomeado.
chmod 644 ./*.pem
echo "certificados gerados em $DIR"
