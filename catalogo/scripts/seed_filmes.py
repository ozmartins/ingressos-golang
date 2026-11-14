#!/usr/bin/env python3
"""Importa filmes das listagens "Melhores filmes de todos os tempos" do AdoroCinema.

Lê os HTMLs salvos em uma pasta (padrão ~/Downloads/filmes), extrai de cada
cartão o id, título, duração, gêneros, direção e elenco, e consulta a página de
cada filme (JSON-LD) para obter a sinopse completa e a URL do pôster. Depois
cria os filmes pela API do catálogo (POST /filmes), com status EM_CARTAZ.

A classificação etária NÃO existe nas páginas: é estimada a partir do gênero
(ver `classificar`) e deve ser tratada como valor de exemplo.

Pode ser repetido: filmes cujo título já existe são pulados.

Uso: python3 seed_filmes.py [PASTA_COM_HTMLS]

Variáveis de ambiente (todas opcionais):
  CATALOGO_URL   http://localhost:8082
  KEYCLOAK_URL   http://localhost:8081
  CLIENT_SECRET  segredo-de-desenvolvimento   (client cinema-m2m)
"""
import glob
import html
import json
import os
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

CATALOGO = os.environ.get("CATALOGO_URL", "http://localhost:8082").rstrip("/") + "/api/v1"
KEYCLOAK = os.environ.get("KEYCLOAK_URL", "http://localhost:8081").rstrip("/")
SECRET = os.environ.get("CLIENT_SECRET", "segredo-de-desenvolvimento")
PASTA = os.path.expanduser("~/Downloads/filmes")
UA = {"User-Agent": "Mozilla/5.0 (X11; Linux x86_64)"}

_token = None


def token(renovar=False):
    global _token
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
        status, dados = chamar("GET", f"{caminho}?page={pagina}&page_size=100")
        if status != 200:
            sys.exit(f"GET {caminho}: HTTP {status} {dados}")
        itens += dados["itens"]
        if len(dados["itens"]) < 100:
            return itens
        pagina += 1


def classificar(generos):
    g = set(generos)
    if g & {"Terror", "Guerra"}:
        return "16 anos"
    if g & {"Animação", "Família"} and not g & {"Terror", "Policial", "Suspense"}:
        return "Livre"
    if g & {"Policial", "Suspense", "Drama", "Ação", "Histórico", "Biopic"}:
        return "14 anos"
    return "12 anos"


def ler_cartoes(pasta):
    filmes = {}
    for arq in sorted(glob.glob(os.path.join(pasta, "*.html"))):
        h = open(arq, encoding="utf8").read()
        for c in h.split('class="card entity-card entity-card-list cf">')[1:]:
            c = c.split('class="card entity-card')[0]
            m = re.search(r'meta-title-link" href="[^"]*filme-(\d+)/">([^<]*)', c)
            if not m:
                continue
            d = re.search(r'meta-body-info">\s*(?:(\d+)h)?\s*(?:(\d+)min)?', c)
            minutos = int(d[1] or 0) * 60 + int(d[2] or 0) if d else 0
            direcao = re.search(r'meta-body-direction.*?personalidade-\d+/">([^<]*)', c, re.S)
            elenco = re.search(r'meta-body-actor(.*?)</div>', c, re.S)
            filmes[m[1]] = {
                "id": m[1], "titulo": html.unescape(m[2]).strip(),
                "minutos": minutos,
                "generos": [html.unescape(g) for g in re.findall(r'genero-\d+/">([^<]*)', c)],
                "direcao": direcao and html.unescape(direcao[1]).strip(),
                "elenco": [html.unescape(a).strip() for a in
                           re.findall(r'personalidade-\d+/">([^<]*)', elenco[1])][:3] if elenco else [],
            }
    return list(filmes.values())


def detalhes(id_):
    req = urllib.request.Request(f"https://www.adorocinema.com/filmes/filme-{id_}/", headers=UA)
    h = urllib.request.urlopen(req, timeout=30).read().decode("utf8")
    m = re.search(r'<script type="application/ld\+json">(.*?)</script>', h, re.S)
    d = json.loads(m.group(1)) if m else {}
    imagem = (d.get("image") or {}).get("url") if isinstance(d.get("image"), dict) else None
    return (d.get("description") or "").strip(), imagem


def main():
    pasta = sys.argv[1] if len(sys.argv) > 1 else PASTA
    filmes = ler_cartoes(pasta)
    if not filmes:
        sys.exit(f"nenhum filme encontrado em {pasta}")
    existentes = {f["titulo"] for f in listar("/filmes")}
    criados = pulados = 0
    for i, f in enumerate(filmes, 1):
        if f["titulo"] in existentes:
            pulados += 1
            continue
        sinopse, imagem = detalhes(f["id"])
        time.sleep(1)  # educação com o site de origem
        partes = [sinopse]
        if f["direcao"]:
            partes.append(f"Direção: {f['direcao']}.")
        if f["elenco"]:
            partes.append(f"Elenco: {', '.join(f['elenco'])}.")
        corpo = {
            "titulo": f["titulo"], "sinopse": " ".join(p for p in partes if p),
            "duracao_minutos": max(f["minutos"], 1),
            "classificacao_etaria": classificar(f["generos"]),
            "genero": ", ".join(f["generos"])[:100] or "Drama",
            "status": "EM_CARTAZ"}
        if imagem:
            corpo["imagem_url"] = imagem[:500]
        status, dados = chamar("POST", "/filmes", corpo)
        if status not in (200, 201):
            sys.exit(f"filme {f['titulo']}: HTTP {status} {dados}")
        criados += 1
        print(f"[{i}/{len(filmes)}] {f['titulo']}", flush=True)
    print(f"filmes: {criados} criados, {pulados} pulados (já existiam)")


if __name__ == "__main__":
    main()
