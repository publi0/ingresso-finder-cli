# Ingresso Finder CLI

TUI para buscar cinemas e sessões do Ingresso.com direto no terminal, com histórico local e cache para acelerar as consultas.

<p align="center">
  <img src="https://img.shields.io/github/go-mod/go-version/publi0/ingresso-finder-cli?style=for-the-badge" alt="Go Version">
  <img src="https://img.shields.io/badge/homebrew-cask-orange?style=for-the-badge&logo=homebrew" alt="Homebrew">
  <a href="https://goreportcard.com/report/github.com/publi0/ingresso-finder-cli"><img src="https://goreportcard.com/badge/github.com/publi0/ingresso-finder-cli?style=for-the-badge" alt="Go Report Card"></a>
  <img src="https://img.shields.io/badge/License-GPLv3-blue.svg?style=for-the-badge" alt="License: GPL v3">
</p>

![Demo](assets/demo.gif)

## Visão Geral

- Fluxo guiado: cidade → cinema → filmes → sessões.
- Opção alternativa: buscar por filme em todos os cinemas visíveis.
- Busca incremental em todas as listas.
- **Painel lateral de metadados**: veja sinopses, duração, gêneros e classificação indicativa dos filmes.
- **Integração com IMDb (OMDb API)**: veja as notas de avaliação e diretores sem sair do terminal.
- Cache local inteligente para reduzir chamadas repetidas (Filmes, Sessões e Metadados do OMDb).
- Retry automático com backoff para erros transitórios da API.
- Preferências globais de visibilidade de cinemas (mostrar/ocultar).
- Ordenação por proximidade usando localização nativa do sistema (quando disponível), com fallback por IP.
- Mapa de assentos com interface gráfica colorida, indicando cadeiras ideais, acessibilidade e taxa de ocupação.

## Requisitos

- Go 1.26.1+

## Instalação

### Homebrew (Cask)

Agora você pode instalar o `ingresso-finder-cli` como um Cask diretamente do repositório:

```bash
brew install --cask publi0/ingresso-finder-cli/ingresso-finder-cli
```

### Binários Pré-compilados

Binários para Linux, macOS (Intel/Apple Silicon) e Windows estão disponíveis na página de [Releases](https://github.com/publi0/ingresso-finder-cli/releases).

### Build local

```bash
go build -o ingresso
```

## Uso

```bash
# via Homebrew ou binário no PATH
ingresso

# build local
./ingresso
```

## Exemplos de comandos

```bash
# define cidade inicial
INGRESSO_CITY="Sao Paulo" ingresso

# roda sem build (modo desenvolvimento)
INGRESSO_CITY="Rio de Janeiro" go run .
```

## Configuração

A ferramenta pode ser aprimorada através de variáveis de ambiente:

- `INGRESSO_CITY` define a cidade inicial e pula a tela de seleção.
- `INGRESSO_LOCATION_DEBUG=1` imprime no stderr o motivo de fallback de localização (quando a API nativa falha).
- `OMDB_API_KEY` chave da API gratuita do [OMDb](https://www.omdbapi.com/) para carregar notas do IMDb, diretores e gêneros dos filmes.

## Atalhos

- `q` ou `ctrl+c` para sair.
- `esc` para voltar.
- Digitar já filtra a lista atual.
- `ctrl+d` abre o seletor de data nas telas de cidades/cinemas/filmes/sessões.
- `ctrl+f` (na tela de cinemas) inicia o modo "filme em todos os cinemas visíveis".
- `ctrl+l` detecta sua localização usando API nativa do sistema (com fallback por IP), exibe a origem usada e ordena cinemas por proximidade.
- `ctrl+t` abre a tela de gestão de cinemas visíveis/ocultos.
- `enter` (na tela de gestão) alterna entre mostrar/ocultar um cinema.
- `x` (na tela de gestão) também alterna mostrar/ocultar um cinema.
- `enter` abre o checkout no navegador na tela de sessões.
- `tab` abre o mapa de assentos quando disponível.
- `n` alterna o modo de exibição de números no mapa de assentos.

## Desenvolvimento

O projeto utiliza **GoReleaser** para automação de builds e releases.

```bash
# Rodar testes
go test ./...

# Validar configuração do GoReleaser
goreleaser check

# Gerar release local (snapshot)
goreleaser release --snapshot --clean
```

## Licença

Este projeto está licenciado sob a [GPL v3](LICENSE).

## Contribuição

Contribuições são muito bem-vindas! Sinta-se à vontade para abrir issues ou enviar Pull Requests.
