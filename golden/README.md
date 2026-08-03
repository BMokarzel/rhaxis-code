# golden

Fixture de código-fonte real (NestJS + TypeScript) para testar a pipeline
completa do rhaxis-code: extração → persistência (Neo4j) → leitura (API).

Não é infra para rodar em produção — é matéria-prima estável para validar um
extrator contra um grafo esperado conhecido.

## Cenário

Dois serviços, com dependência cross-service `orders-api -> users-api`:

```
orders-api (porta 3001)          users-api (porta 3002)
  Postgres "orders"                Postgres "users"
  GET  /orders/:id                 GET  /users/:id
  POST /orders          ──HTTP──►  POST /notify (guarded por x-api-key)
```

`orders-api` e `users-api` reaproveitam deliberadamente os mesmos nomes de
serviço, endpoints e arquivos (`src/orders.controller.ts`,
`src/middleware/auth.ts`, `src/utils/forbidden.ts`, `src/utils/serialize.ts`)
já usados em `schema/fixtures/nestjs-orders.cypher`. Esse fixture Cypher pode
servir de gabarito aproximado do grafo esperado para o fluxo de `POST /orders`
assim que houver um extrator real — embora o código aqui seja mais completo
(adiciona try/catch, guard, switch e loop que o fixture Cypher não cobre).

## Cobertura de kinds/edges por trecho de código

| Código | Kind/Edge exercitado |
|---|---|
| `orders-api` `OrdersController.get` | `Endpoint` flat, `CallDB` (select), sem branch |
| `orders-api` `OrdersController.create` | `CallFunction` (auth inline) → `IfNode` → `then`: `TryNode` com `CallDB` (insert) + `CallHTTP` (cross-service) + `CallFunction` (serialize); `catch`: `LogNode`; `else`: `CallFunction` (forbidden) |
| `orders-api` `UsersClient` | `CallHTTP` não resolvido até o `rhaxis-link` rodar; `Config` (`USERS_API_URL`) |
| `orders-api` `OrdersService.findById` | `NotFoundException` → `ErrorType`/`THROWS` |
| `users-api` `UsersController.get` | `Endpoint` flat, `CallDB` (select) |
| `users-api` `UsersController.notify` | `Middleware` (`ApiKeyGuard`) + edge `PROTECTS`; alvo do `CallHTTP` do orders-api |
| `users-api` `ApiKeyGuard` | `Config` (`INTERNAL_API_KEY`), `UnauthorizedException` → `ErrorType` |
| `users-api` `UsersService.notify` | `SwitchNode` (tipo de notificação) → `TryNode` → `LoopNode` (canais) → `CallFunction` (`dispatchToChannel`) |
| `users-api` `dispatchToChannel` | `SwitchNode` aninhado + `LogNode` (`nest-logger`) |
| `Order`/`User` entities (TypeORM) | `Struct` + `FieldSlot` |

## Rodando localmente (opcional)

Cada serviço é um projeto NestJS independente (`package.json` +
`tsconfig.json` próprios). Não fazem parte do `docker-compose.yml` da raiz —
são fixture de leitura, não infra do rhaxis-code em si.

```
cd golden/orders-api && cp .env.example .env && npm install && npm run build
cd golden/users-api  && cp .env.example .env && npm install && npm run build
```

Ambos esperam um Postgres em `DATABASE_URL` (não provisionado aqui).
