# O build acontece aqui e não sai daqui: o que vai para a imagem final é só o
# `dist/`, servido por nginx.
FROM node:22-alpine AS construcao
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install
COPY . .
RUN npm run build

FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY repasse.inc /etc/nginx/conf.d/repasse.inc
COPY --from=construcao /app/dist /usr/share/nginx/html
EXPOSE 80
