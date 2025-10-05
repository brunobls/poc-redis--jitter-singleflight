# POC Redis - Jitter + Singleflight

Projeto de prova de conceito que demonstra uso de cache Redis com jitter no TTL e deduplicação de requisições usando `singleflight` em Go.

## Estrutura

- `docker-compose.yml` - define serviços `redis` e `app` para facilitar execução local.
- `app/` - código fonte Go e `Dockerfile` para a aplicação.

## Pré-requisitos

- Docker.
- (Opcional) Go 1.20+ para executar localmente sem container.

## Como executar

1) Usando Docker Compose (recomendado)

   Levanta o Redis e a aplicação em modo de desenvolvimento:

   ```bash
   docker compose up --build
   ```

   Para rodar em segundo plano:

   ```bash
   docker compose up --build -d
   ```

   Logs da aplicação podem ser vistos com:

   ```bash
   docker compose logs -f app
   ```

2) Executando localmente (Go)

   - Inicie um Redis local (por exemplo via Docker):

     ```bash
     docker run -p 6379:6379 --name redis-cache -d redis:7
     ```

   - Em seguida, rode a aplicação:

     ```bash
     cd app
     go run main.go
     ```

   A aplicação busca `REDIS_HOST` e `REDIS_PORT` das variáveis de ambiente (padrão: `localhost:6379`).

## O que o código faz

A aplicação simula múltiplas goroutines pedindo o saldo de um usuário.

Fluxo detalhado:

- Tenta pegar do cache Redis.
  - Se existir → retorna imediatamente.

- Se der cache miss, usa `singleflight.Group` (do pacote `golang.org/x/sync/singleflight`)
  - Garante que apenas uma goroutine busque o valor no banco.

- Aplica jittered TTL ao salvar no cache:

  ```go
  baseTTL := 60 * time.Second
  jitter := time.Duration(rand.Int63n(int64(baseTTL/5))) // ±20%
  // ttl := baseTTL + jitter  (o código também pode alternar o sinal aleatoriamente)
  ```

  Isso evita que várias chaves expirem exatamente juntas.

- Salva o resultado no cache com o TTL aleatório.

- As demais requisições concorrentes esperam o resultado da primeira, sem bater no banco.

🧪 Saída típica

Quando você rodar esse código, verá algo assim:

```
Consultando saldo no banco para user 123...
[Goroutine 0] Saldo = 1234.56
[Goroutine 3] Saldo = 1234.56
[Goroutine 1] Saldo = 1234.56
[Goroutine 2] Saldo = 1234.56
...
```

Note que a mensagem “Consultando saldo no banco...” aparece apenas uma vez,
mesmo com 10 requisições simultâneas → ✅ o request coalescing funcionou.

## Variáveis de ambiente

- `REDIS_HOST` (padrão `localhost`)
- `REDIS_PORT` (padrão `6379`)

## Arquivos principais

- `docker-compose.yml` - orquestra Redis e app.
- `app/main.go` - lógica principal (cache, singleflight, jitter).
- `app/Dockerfile` - imagem da aplicação.

## Observações

Este repo é uma prova de conceito. Em produção, considere:

- lidar com falhas de conexão e retries de forma robusta;
- monitorar métricas de cache hit/miss;
- políticas de invalidação e segurança (auth do Redis, TLS, etc.).

---

Licença: MIT (por padrão para POCs — adapte conforme necessário).