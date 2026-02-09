# Ingresso Finder CLI

TUI para buscar cinemas e sessões do Ingresso.com direto no terminal, com histórico local e cache para acelerar as consultas.

![Demo](assets/demo.gif)

## Visão Geral

- Fluxo guiado: cidade → cinema → filmes → sessões.
- Opção alternativa: buscar por filme em todos os cinemas visíveis.
- Busca incremental em todas as listas.
- Cache local para reduzir chamadas repetidas.
- Retry automático com backoff para erros transitórios da API.
- Preferências globais de visibilidade de cinemas (mostrar/ocultar).
- Ordenação por proximidade usando localização nativa do sistema (quando disponível), com fallback por IP.
- Mapa de assentos quando o endpoint público está disponível.

## Requisitos

- Go 1.25+

## Instalação

### Homebrew (tap)

```bash
brew tap publi0/ingresso-finder-cli https://github.com/publi0/ingresso-finder-cli
brew install publi0/ingresso-finder-cli/ingresso-finder-cli
```

### Build local

```bash
go build -o ingresso
```

## Uso

```bash
# via Homebrew
ingresso

# build local
./ingresso
```

## Exemplos de comandos

```bash
# executa direto do binário
ingresso

# define cidade inicial
INGRESSO_CITY="Sao Paulo" ingresso

# roda sem build (modo desenvolvimento)
INGRESSO_CITY="Rio de Janeiro" go run .
```

## Configuração

- `INGRESSO_CITY` define a cidade inicial e pula a tela de seleção.
- `INGRESSO_LOCATION_DEBUG=1` imprime no stderr o motivo de fallback de localização (quando a API nativa falha).

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
- Em erros de "sem sessões", `enter` tenta automaticamente o próximo dia e `ctrl+d` abre o seletor para escolher qualquer data.

## Desenvolvimento

```bash
go test ./...
```

## Automacao da formula Homebrew

A formula do tap e atualizada automaticamente via GitHub Actions a cada push na `main`.
