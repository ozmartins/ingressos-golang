#!/usr/bin/env python3
"""Dados de exemplo: filmes, salas e sessões, criados pela API do catálogo.

    python3 scripts/seed_salas_sessoes.py

Rode depois de carregar os cinemas (scripts/seed_cinemas.sql), com a plataforma
no ar. Tudo passa pela API (POST /filmes, /salas, /sessoes), e não por SQL,
porque criar uma sessão grava o fato `sessao.criada` na caixa de saída do
catálogo, e é esse fato que faz o estoque criar as poltronas da sessão.

Cria:
  * 6 filmes de exemplo (duração, classificação e gênero são valores
    plausíveis, não os dados reais dos filmes);
  * 2 salas por cinema: sala 1 (2D) e sala 2 (3D), com 5 fileiras de 8
    poltronas (A-D NORMAL, E NAMORADEIRA);
  * sessões nos próximos 3 dias (a partir de amanhã), às 16:00 e 20:00 no
    horário de Brasília, em cada sala, com filme e idioma variando.

Pode ser repetido: filmes e salas já existentes são reaproveitados e sessões
que colidem com uma já existente (HTTP 409) são puladas, então uma nova
execução só acrescenta os dias que ainda não existem.

Variáveis de ambiente (todas opcionais):
  CATALOGO_URL   http://localhost:8082
  KEYCLOAK_URL   http://localhost:8081
  CLIENT_SECRET  segredo-de-desenvolvimento   (client cinema-m2m)
"""
import json
import os
import sys
import threading
import urllib.error
import urllib.parse
import urllib.request
import zlib
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, time, timedelta
from zoneinfo import ZoneInfo

CATALOGO = os.environ.get("CATALOGO_URL", "http://localhost:8082").rstrip("/") + "/api/v1"
KEYCLOAK = os.environ.get("KEYCLOAK_URL", "http://localhost:8081").rstrip("/")
SECRET = os.environ.get("CLIENT_SECRET", "segredo-de-desenvolvimento")
DIAS, HORARIOS, TRABALHADORES = 3, (time(16, 0), time(20, 0)), 8
FUSO = ZoneInfo("America/Sao_Paulo")

FILMES = [
    ("Apenas uma Ilusão", "Direção: Olivier Nakache, Eric Toledano. Elenco: Louis Garrel, Camille Cottin, Pierre Lottin.", 105, "12 anos", "Comédia", "https://br.web.img3.acsta.net/img/e7/e8/e7e8fc374509dda807ab0ca4d9b1dc6b.jpg"),
    ("Boiúna: Terror no Amazonas", "Direção: Mike P. Nelson. Elenco: Kiana Madeira, Jessica Rothe, Logan Marshall-Green.", 98, "16 anos", "Terror", "https://br.web.img3.acsta.net/img/75/e1/75e115ee346c422fb295d4446c48bc3d.jpg"),
    ("Cara-de-Barro", "Direção: James Watkins. Elenco: Tom Rhys Harries, Naomi Ackie, Eddie Marsan.", 112, "14 anos", "Suspense", "https://br.web.img3.acsta.net/img/fd/5e/fd5e5ae279ca29e388c1b9138d8667bf.png"),
    ("Godzilla Minus Zero", "Direção: Takashi Yamazaki. Elenco: Ryûnosuke Kamiki, Minami Hamabe, Yamada Yuki.", 125, "14 anos", "Ação", "https://br.web.img3.acsta.net/img/62/c4/62c45d6c1236346b47b5b2e9e124363e.png"),
    ("Lago dos Ossos", "Direção: Mercedes Bryce Morgan. Elenco: Maddie Hasson, Alex Roe, Marco Pigossi.", 101, "16 anos", "Terror", "https://br.web.img3.acsta.net/img/25/5e/255ed6ffbb56c8220f6161d20d962b21.jpg"),
    ("O Bosque Selvagem", "Direção: Travis Knight. Elenco: Peyton Elizabeth Lee, Jacob Tremblay, Mahershala Ali.", 94, "Livre", "Animação", "https://br.web.img3.acsta.net/img/9b/83/9b836af29d0d1dd86234d9b18a5b76a3.jpg"),
]
FILEIRAS = [{"fileira": f, "assentos": 8, "tipo": "NORMAL"} for f in "ABCD"] + [
    {"fileira": "E", "assentos": 8, "tipo": "NAMORADEIRA"}]

_token, _trava = None, threading.Lock()


def token(renovar=False):
    global _token
    with _trava:
        if _token is None or renovar:
            corpo = urllib.parse.urlencode({
                "grant_type": "client_credentials",
                "client_id": "cinema-m2m", "client_secret": SECRET}).encode()
            url = f"{KEYCLOAK}/realms/cinema/protocol/openid-connect/token"
            with urllib.request.urlopen(urllib.request.Request(url, corpo)) as r:
                _token = json.load(r)["access_token"]
        return _token


def chamar(metodo, caminho, corpo=None, tentativa=0):
    req = urllib.request.Request(
        CATALOGO + caminho, method=metodo,
        data=None if corpo is None else json.dumps(corpo).encode(),
        headers={"Content-Type": "application/json",
                 "Authorization": f"Bearer {token()}"})
    try:
        with urllib.request.urlopen(req) as r:
            return r.status, json.load(r)
    except urllib.error.HTTPError as e:
        if e.code == 401 and tentativa == 0:
            token(renovar=True)
            return chamar(metodo, caminho, corpo, 1)
        return e.code, e.read().decode(errors="replace")


def listar(caminho):
    itens, pagina = [], 1
    while True:
        sep = "&" if "?" in caminho else "?"
        status, dados = chamar("GET", f"{caminho}{sep}page={pagina}&page_size=100")
        if status != 200:
            sys.exit(f"GET {caminho}: HTTP {status} {dados}")
        itens += dados["itens"]
        if len(dados["itens"]) < 100:
            return itens
        pagina += 1


def exigir(status, dados, o_que):
    if status not in (200, 201):
        sys.exit(f"{o_que}: HTTP {status} {dados}")
    return dados


def garantir_filmes():
    existentes = {f["titulo"]: f["id"] for f in listar("/filmes")}
    ids = []
    for titulo, sinopse, minutos, classificacao, genero, imagem in FILMES:
        if titulo not in existentes:
            f = exigir(*chamar("POST", "/filmes", {
                "titulo": titulo, "sinopse": sinopse, "duracao_minutos": minutos,
                "classificacao_etaria": classificacao, "genero": genero,
                "imagem_url": imagem, "status": "EM_CARTAZ"}), f"filme {titulo}")
            existentes[titulo] = f["id"]
        ids.append(existentes[titulo])
    return ids


def garantir_salas():
    existentes = {(s["cinema_id"], s["numero"]): s["id"] for s in listar("/salas")}
    pendentes = [(c["id"], n) for c in listar("/cinemas") for n in (1, 2)
                 if (c["id"], n) not in existentes]

    def criar(par):
        cinema_id, numero = par
        s = exigir(*chamar("POST", "/salas", {
            "cinema_id": cinema_id, "numero": numero,
            "tipo_tela": "2D" if numero == 1 else "3D", "fileiras": FILEIRAS}),
            f"sala {numero} do cinema {cinema_id}")
        return par, s["id"]

    with ThreadPoolExecutor(TRABALHADORES) as pool:
        existentes.update(pool.map(criar, pendentes))
    print(f"salas: {len(pendentes)} criadas, {len(existentes) - len(pendentes)} já existiam")
    return existentes


def main():
    filmes = garantir_filmes()
    salas = garantir_salas()

    amanha = datetime.now(FUSO).date() + timedelta(days=1)
    tarefas = []
    for (cinema_id, numero), sala_id in sorted(salas.items()):
        for d in range(DIAS):
            for hora in HORARIOS:
                inicio = datetime.combine(amanha + timedelta(days=d), hora, FUSO)
                h = zlib.crc32(f"{sala_id}{inicio.isoformat()}".encode())
                tarefas.append({
                    "filme_id": filmes[h % len(filmes)], "sala_id": sala_id,
                    "data_hora_inicio": inicio.isoformat(),
                    "idioma": "DUBLADO" if h % 2 == 0 else "LEGENDADO",
                    "preco_base": "36.00" if numero == 2 else "28.00"})

    def criar(corpo):
        return chamar("POST", "/sessoes", corpo)[0]

    with ThreadPoolExecutor(TRABALHADORES) as pool:
        codigos = list(pool.map(criar, tarefas))
    criadas, puladas = codigos.count(201), codigos.count(409)
    outras = [c for c in codigos if c not in (201, 409)]
    print(f"sessões: {criadas} criadas, {puladas} puladas (já existiam), {len(outras)} com erro")
    if outras:
        sys.exit(f"códigos inesperados: {sorted(set(outras))}")


if __name__ == "__main__":
    main()
