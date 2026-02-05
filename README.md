# Ingresso Finder CLI

TUI para buscar cinemas e sessões do Ingresso.com direto no terminal, com histórico local e cache para acelerar as consultas.

![Demo](assets/demo.gif)

## Visão Geral

- Fluxo guiado: cidade → cinema → filmes → sessões.
- Busca incremental em todas as listas.
- Cache local para reduzir chamadas repetidas.
- Mapa de assentos quando o endpoint público está disponível.

## Requisitos

- Go 1.25+

## Instalação

### Homebrew (tap)

```bash
brew tap publi0/ingresso-finder-cli
brew install publi0/ingresso-finder-cli/ingresso-finder-cli
```

### Build local

```bash
go build -o ingresso
```

## Uso

```bash
./ingresso
```

## Exemplos de comandos

```bash
# executa direto do binário
./ingresso

# define cidade inicial
INGRESSO_CITY="Sao Paulo" ./ingresso

# roda sem build (modo desenvolvimento)
INGRESSO_CITY="Rio de Janeiro" go run .
```

## Configuração

- `INGRESSO_CITY` define a cidade inicial e pula a tela de seleção.

## Atalhos

- `q` ou `ctrl+c` para sair.
- `esc` para voltar.
- Digitar já filtra a lista atual.
- `ctrl+d` abre o seletor de data nas telas de cidades/cinemas/filmes/sessões.
- `enter` abre o checkout no navegador na tela de sessões.
- `tab` abre o mapa de assentos quando disponível.
- `n` alterna o modo de exibição de números no mapa de assentos.

## Desenvolvimento

```bash
go test ./...
```
