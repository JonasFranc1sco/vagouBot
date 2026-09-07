# VagouBot

Scraper de vagas do LinkedIn que busca postings com base em palavras-chave,
aplica filtros de relevância e envia as vagas encontradas para um bot do Telegram.

## Pré-requisitos

- [Go](https://go.dev/dl/) instalado (a versão usada no projeto é a `1.26.5`,
  definida no `go.mod`).

## Como executar (passo a passo)

1. **Clone/abra o projeto**

   ```bash
   cd vagouBot
   ```

2. **Instale as dependências**

   ```bash
   go mod download
   go mod tidy
   ```

3. **(Opcional para usar telegram, mas recomendado) Configure o bot do Telegram e o chat**

   Crie/Edite o arquivo `.env` na raiz do projeto com:

   ```yaml
   TELEGRAM_CHAT_ID = SEU_CHAT_ID
   TELEGRAM_BOT_TOKEN = "SEU_BOT_TOKEN"
   ```

4. **Ajuste a busca e os filtros em `config.yaml`**

   - `linkedin.keywords`: termos de busca (ex.: `"python"`, `"junior"`).
   - `linkedin.location`: local (ex.: `remote`).
   - `filters.include_keywords` / `filters.exclude_keywords`: palavras que
     fazem a vaga passar ou ser descartada na filtragem.
   - `filters.min_description_length`: tamanho mínimo da descrição.

5. **Execute o scraper**

   ```bash
   go run ./cmd/scraper
   ```

   Durante a execução o programa:

   - Monta a URL de busca no LinkedIn e faz a requisição.
   - Faz o *parse* das vagas retornadas.
   - Remove vagas já vistas (controle de duplicados via `seen.json`).
   - Busca os detalhes (descrição) de cada vaga.
   - Aplica os filtros de relevância.
   - Imprime no terminal as 3 primeiras vagas encontradas.
   - Envia as vagas filtradas para o Telegram (se `bot_token` e `chat_id`
     estiverem configurados) ou uma mensagem de "Nenhuma vaga nova hoje".

> **Controle de duplicados:** as vagas enviadas são gravadas no arquivo
> `seen.json`, então execuções seguintes só consideram vagas novas.

## Estrutura do projeto

```
cmd/scraper/           Ponto de entrada do programa
internal/config/       Carregamento do config.yaml
internal/linkedin/     Cliente HTTP, parser e modelos do LinkedIn
internal/proxy/        Rotação de proxies
internal/rate/         Rate limiter
internal/filter/       Filtros de relevância + deduplicação (seen.json)
internal/telegram/     Envio de mensagens para o Telegram
config.yaml            Configurações (busca, filtros, telegram, proxy)
```
