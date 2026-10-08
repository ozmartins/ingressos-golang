proxy_http_version 1.1;
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
proxy_set_header Connection "";

# A reserva abre um gRPC para o estoque e o pagamento é assíncrono; nenhuma
# dessas chamadas é instantânea, mas nenhuma é longa o bastante para justificar
# o padrão de 60s.
proxy_connect_timeout 5s;
proxy_read_timeout 30s;
