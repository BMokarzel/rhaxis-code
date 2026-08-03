# Plano de implementação — arestas explícitas por Endpoint (Tela 2)

**Objetivo:** aplicar à Tela 2 (`ListEndpoints`) a mesma filosofia da Tela 1 (Abordagem B do `PLANO-SERVICEMAP-EDGES.md`) — devolver, além dos nós de Endpoint, **todas as relações** que cada endpoint mantém com outros elementos do ecossistema:

- Handler (Function/Method)
- Middlewares aplicados
- Erros tratados
- Databases tocados (via flow)
- Brokers publicados/consumidos (via flow)
- Serviços chamados (cross-service, via flow)
- Configs lidas (via flow)

Abordagem escolhida: **materializar edges agregadas por Endpoint** durante o linking (mesma estratégia já usada para `DEPENDS_ON` no nível Service), e ler direto na query da Tela 2.

---

## 1. Diagnóstico

### 1.1. Estado atual

`infra/neo4j/queries.go:cypherListEndpoints`:
```cypher
MATCH (s:Service {urn: $serviceURN})
OPTIONAL MATCH (s)-[:EXPOSES]->(e:Endpoint)
RETURN s AS service, collect(e) AS endpoints
```

Devolve **apenas nós Endpoint** com metadados de rota. Sem nenhuma relação. Para saber "o que este endpoint faz", o cliente precisa cair na Tela 3 (fluxo detalhado) — não há visão intermediária.

`domain.EndpointList`:
```go
type EndpointList struct {
    Service   Service
    Endpoints []Endpoint
}
```

Mesma lacuna estrutural que a Tela 1 tinha antes de PLANO-SERVICEMAP-EDGES.md — o backend já tem os dados no grafo, mas não os apresenta agregados por endpoint.

### 1.2. Relações relevantes por Endpoint

| Relação | Origem no grafo | Distância |
|---|---|---|
| Handler | `(:Endpoint)-[:HANDLED_BY]->(:Function\|:Method)` | 1-hop |
| Middlewares | `(:Middleware)-[:PROTECTS]->(:Endpoint)` | 1-hop |
| Erros tratados | `(:Endpoint)-[:HANDLES_ERROR]->(:ErrorType)` | 1-hop |
| DBs tocados | Endpoint → flow → `CallDB` → `Database` | N-hop |
| Brokers | Endpoint → flow → `PublishEvent`/`ConsumeEvent` → `Broker` | N-hop |
| Serviços chamados | Endpoint → flow → `CallHTTP` resolvido → Endpoint remoto → Service | N-hop |
| Configs lidas | Endpoint → flow → nó com `USES_CONFIG` → `Config` | N-hop |

As três primeiras são triviais em Cypher. As quatro últimas exigem traversal de flow — é onde a decisão de arquitetura pesa.

---

## 2. Duas abordagens avaliadas

### 2.1. Caminho A — traversal em Cypher a cada request

Variable-length path do Endpoint pelo `CONTAINS|EXPANDS_TO|CALLS` até os IO nodes, seguido de mais um hop para o alvo IO.

- **Prós:** sempre atualizado; nenhum trabalho extra no writer/linker.
- **Contras:** múltiplos `OPTIONAL MATCH` com variable-length explodem em produto cartesiano; risco de ciclos em `CALLS`; performance ruim em serviços grandes; cada request refaz o mesmo trabalho.

### 2.2. Caminho B — materializar edges agregadas por Endpoint (escolhido)

Mesma filosofia já adotada para `DEPENDS_ON` no nível Service (`aggregates.go:20-25` — "materializada pelo linker cross-service; a query da Tela 1 apenas lê").

- **Prós:** leitura O(edges) na Tela 2 sem traversal; consistente com o design existente; o linker roda 1x pós-extração e faz o trabalho pesado uma única vez.
- **Contras:** exige passo de materialização no linker; edges precisam ser reconstruídas quando o subgrafo do endpoint muda.

**Decisão:** Caminho B. Traversal on-demand fica reservado para a Tela 3 (que é intrinsecamente de detalhe).

---

## 3. Novas edges a materializar

| Edge | Direção | Props | Semântica |
|---|---|---|---|
| `TOUCHES_DB` | `(:Endpoint)->(:Database)` | `ops: [string]` (union: "select"/"insert"/…) | Endpoint alcança este DB via flow |
| `TOUCHES_BROKER` | `(:Endpoint)->(:Broker)` | `kind: 'publish'\|'consume'`, `topics: [string]` | Fan-out de mensageria |
| `CALLS_SERVICE` | `(:Endpoint)->(:Service)` | `viaEndpointURN: URN`, `weight: int` | Chamada cross-service resolvida |
| `READS_CONFIG` | `(:Endpoint)->(:Config)` | — | Configs alcançadas via flow |

**Pré-requisito:** `HANDLED_BY` (do `ANALISE-METADADOS-NODES.md`) precisa existir. Se ainda for prop `Endpoint.HandlerURN`, este plano funciona com um `OPTIONAL MATCH` extra por URN, mas o correto é fazer a migração antes.

---

## 4. Alterações — passo a passo

### Passo 4.1 — Persistência (linker)

`infra/neo4j/linker.go` já é o ponto natural (hoje resolve `CallHTTP` cross-service). Adicionar novo passo pós-linking:

```
para cada Endpoint e:
  subgraph = traversal(e, [CONTAINS, EXPANDS_TO, CALLS]) escopado ao serviceURN
  agrupa IO nodes encontrados:
    - CallDB      -> agrupa por Database.urn, une ops    -> MERGE TOUCHES_DB
    - PublishEvent-> agrupa por Broker.urn, kind=publish -> MERGE TOUCHES_BROKER
    - ConsumeEvent-> agrupa por Broker.urn, kind=consume -> MERGE TOUCHES_BROKER
    - CallHTTP (resolvido, EXPANDS_TO cross-service) -> agrupa por Service alvo -> MERGE CALLS_SERVICE
    - nós com USES_CONFIG -> MERGE READS_CONFIG
```

Requisitos:
- **Idempotente** via `MERGE` (edges com mesma origem+destino são reutilizadas; props são substituídas na re-execução).
- **Escopado por Service** para não misturar traversal com CALLS que sai do serviço.
- **Anti-ciclo:** rastrear URNs já visitados na travessia.
- **Trigger:** rodar toda vez que uma extração completa OU um re-linking mudar o subgrafo do endpoint. Estratégia simples v1: reprocessar todos os endpoints do Service após cada extração daquele Service.

### Passo 4.2 — Query (`infra/neo4j/queries.go`)

Nova `cypherListEndpoints`:

```cypher
MATCH (s:Service {urn: $serviceURN})
OPTIONAL MATCH (s)-[:EXPOSES]->(e:Endpoint)
OPTIONAL MATCH (e)-[:HANDLED_BY]->(h)
OPTIONAL MATCH (mw:Middleware)-[:PROTECTS]->(e)
OPTIONAL MATCH (e)-[:HANDLES_ERROR]->(err:ErrorType)
OPTIONAL MATCH (e)-[td:TOUCHES_DB]->(db:Database)
OPTIONAL MATCH (e)-[tb:TOUCHES_BROKER]->(bk:Broker)
OPTIONAL MATCH (e)-[cs:CALLS_SERVICE]->(tgtSvc:Service)
OPTIONAL MATCH (e)-[:READS_CONFIG]->(cfg:Config)
WITH s, e, h,
     collect(DISTINCT mw)  AS middlewares,
     collect(DISTINCT err) AS errors,
     collect(DISTINCT cfg) AS configs,
     [x IN collect(DISTINCT CASE WHEN db IS NOT NULL
        THEN {dbURN: db.urn, ops: td.ops} END)
      WHERE x IS NOT NULL] AS touchesDb,
     [x IN collect(DISTINCT CASE WHEN bk IS NOT NULL
        THEN {brokerURN: bk.urn, kind: tb.kind, topics: tb.topics} END)
      WHERE x IS NOT NULL] AS touchesBroker,
     [x IN collect(DISTINCT CASE WHEN tgtSvc IS NOT NULL
        THEN {targetServiceURN: tgtSvc.urn,
              viaEndpointURN: cs.viaEndpointURN,
              weight: coalesce(cs.weight, 0)} END)
      WHERE x IS NOT NULL] AS callsService
RETURN s AS service,
       collect({
         node: e,
         handler: h,
         middlewares: middlewares,
         errors: errors,
         configs: configs,
         touchesDb: touchesDb,
         touchesBroker: touchesBroker,
         callsService: callsService
       }) AS endpoints
```

Pontos chave:
- Uma linha por endpoint no `collect` final — o frontend não precisa correlacionar arestas com o endpoint dono.
- `CASE ... END` + `WHERE x IS NOT NULL` para descartar rows lixo de `OPTIONAL MATCH` que não casaram (mesma técnica do PLANO-SERVICEMAP-EDGES.md).
- Sem APOC — só Cypher standard.

### Passo 4.3 — Domain (`domain/aggregates.go`)

Substituir a lista rasa por uma estrutura rica:

```go
type EndpointList struct {
    Service   Service
    Endpoints []EndpointDetail
}

// EndpointDetail é a Tela 2 enriquecida: endpoint + tudo que ele toca.
type EndpointDetail struct {
    Endpoint      Endpoint
    Handler       Node          // Function|Method|nil
    Middlewares   []Middleware
    HandledErrors []ErrorType
    Configs       []Config
    TouchesDB     []EndpointDBEdge
    TouchesBroker []EndpointBrokerEdge
    CallsService  []EndpointServiceEdge
}

type EndpointDBEdge struct {
    DatabaseURN URN
    Ops         []string
}

type EndpointBrokerEdge struct {
    BrokerURN URN
    Kind      string   // "publish" | "consume"
    Topics    []string
}

type EndpointServiceEdge struct {
    TargetServiceURN URN
    ViaEndpointURN   URN
    Weight           int
}
```

Trade-off: mudança **quebrante** no shape de `EndpointList.Endpoints` (era `[]Endpoint`, vira `[]EndpointDetail`). Consumers internos precisam ser atualizados — apenas `cmd/rhaxis-api/view.go:endpointListView`.

### Passo 4.4 — Reader (`infra/neo4j/reader.go`)

`ListEndpoints` parsing muda para consumir a nova shape por endpoint. Padrão idêntico ao já existente para `deps` e (após o PLANO-SERVICEMAP-EDGES.md) `usesDb`/`usesBroker`.

Esqueleto:
```go
for _, v := range asListProp(recordGet(rec, "endpoints")) {
    m := asMapProp(v)
    if m == nil { continue }

    epNeo, ok := asNeoNode(m["node"])
    if !ok { continue }
    epDomain, err := nodeToDomain(epNeo)
    if err != nil || epDomain.Kind() != domain.KindEndpoint { continue }

    detail := domain.EndpointDetail{ Endpoint: *epDomain.(*domain.Endpoint) }

    // handler (nó opcional)
    if h, ok := asNeoNode(m["handler"]); ok {
        if hd, ok := nodeToDomainSafe(h); ok {
            detail.Handler = hd
        }
    }
    // middlewares, errors, configs: listas de neo.Node
    // touchesDb, touchesBroker, callsService: listas de map[string]any
    // ...
    out.Endpoints = append(out.Endpoints, detail)
}
```

Nenhuma outra rota (Tela 1/3) é tocada.

### Passo 4.5 — View (`cmd/rhaxis-api/view.go`)

`endpointListView` reescrito para emitir a estrutura aninhada:

```go
func endpointListView(el domain.EndpointList) map[string]any {
    eps := make([]map[string]any, len(el.Endpoints))
    for i, d := range el.Endpoints {
        ep := d.Endpoint
        v := map[string]any{
            "node":        nodeView(&ep),
            "handler":     nodeView(d.Handler),
            "middlewares": nodesView(d.Middlewares),
            "errors":      nodesView(d.HandledErrors),
            "configs":     nodesView(d.Configs),
            "touchesDb":   edgeListDB(d.TouchesDB),
            "touchesBroker": edgeListBroker(d.TouchesBroker),
            "callsService": edgeListService(d.CallsService),
        }
        eps[i] = v
    }
    svc := el.Service
    return map[string]any{
        "service":   nodeView(&svc),
        "endpoints": eps,
    }
}
```

Contrato HTTP **quebrante** — clientes que liam `endpoints[].urn` diretamente passam a ler `endpoints[].node.urn`. Documentar como breaking change ou versionar a rota.

### Passo 4.6 — Testes

**Unidade** (reader):
- record com endpoint sem nenhuma edge → todas as listas vazias, sem panic
- record com endpoint com múltiplas edges de cada tipo → todas parseadas
- record com handler `nil` → `detail.Handler == nil`

**Integração** (`reader_integration_test.go`):
- Seed: 1 service, 2 endpoints. Endpoint A: 1 DB, 1 broker publish, chama service B. Endpoint B: 1 middleware, 1 handles_error, 1 config.
- Asserts: cada `EndpointDetail` carrega suas próprias listas corretas; nenhuma mistura entre endpoints.

**Integração do linker:**
- Após seed + extração, verificar que as 4 edges materializadas existem com props corretas (`ops` deduplicados, `topics` deduplicados, `weight` = nº de call sites).

---

## 5. Contrato HTTP resultante

`GET /services/{urn}/endpoints` passa a devolver:

```json
{
  "service": { "urn": "svc:orders", "name": "orders", ... },
  "endpoints": [
    {
      "node": { "urn": "ep:orders:POST:/orders", "httpMethod": "POST", "pathTemplate": "/orders", ... },
      "handler":     { "urn": "fn:orders:createOrder", "kind": "Function" },
      "middlewares": [ { "urn": "mw:orders:AuthGuard", "middlewareKind": "guard" } ],
      "errors":      [ { "urn": "err:BadRequest", "httpStatus": 400 } ],
      "configs":     [ { "urn": "cfg:MAX_ITEMS", "key": "MAX_ITEMS" } ],
      "touchesDb":     [ { "dbURN": "db:orders-pg", "ops": ["insert","select"] } ],
      "touchesBroker": [ { "brokerURN": "brk:events", "kind": "publish", "topics": ["order.created"] } ],
      "callsService":  [ { "targetServiceURN": "svc:billing",
                           "viaEndpointURN": "ep:billing:POST:/charges", "weight": 1 } ]
    }
  ]
}
```

---

## 6. Sequência sugerida de commits

1. **linker:** materializar as 4 edges agregadas (`TOUCHES_DB`, `TOUCHES_BROKER`, `CALLS_SERVICE`, `READS_CONFIG`) + testes de integração. Nada consome ainda; grafo cresce.
2. **domain:** novo `EndpointDetail` + tipos de edge (não usados ainda por leitores).
3. **infra/neo4j (query + reader):** trocar `cypherListEndpoints` e adaptar `ListEndpoints`. Testes atualizados.
4. **cmd/rhaxis-api (view):** nova serialização.
5. (opcional) versionar `/services/{urn}/endpoints` como `v2` se houver consumer externo.

Passos 1-2 são aditivos. 3-4 quebram o contrato interno — coordenados no mesmo release.

---

## 7. Dependências / pré-requisitos

- [ ] `HANDLED_BY` implementado (`ANALISE-METADADOS-NODES.md` §1). Sem ele, a query cai num handler `nil` — funciona, mas perde a informação.
- [ ] Linker cross-service consegue resolver `CallHTTP → Endpoint remoto` (já existe em `infra/neo4j/linker.go`, verificar cobertura).
- [ ] Constraint `node_urn_unique` em `Node` (`schema/schema.cypher:5`) — necessária para MERGE das edges agregadas ser idempotente.

---

## 8. Riscos e mitigação

| Risco | Mitigação |
|---|---|
| Materialização fica stale (extração parcial) | Reprocessar todos os endpoints do Service ao final de cada extração; documentar como job idempotente |
| Ciclos em `CALLS` explodem traversal do linker | Set de URNs visitadas por endpoint durante o walk |
| Grafos grandes: fan-out enorme por endpoint (100+ DBs?) | Improvável em ecossistemas reais; se acontecer, adicionar limite + flag `truncated` na edge |
| Breaking change no JSON da Tela 2 | Versionamento explícito da rota ou changelog coordenado |
| `topics` como lista Neo4j pode ordenar não-deterministicamente | Ordenar no linker antes de gravar |

---

## 9. Definition of Done

- [ ] Linker materializa `TOUCHES_DB`, `TOUCHES_BROKER`, `CALLS_SERVICE`, `READS_CONFIG` de forma idempotente
- [ ] `domain.EndpointDetail` implementado com edges tipadas
- [ ] Nova `cypherListEndpoints` retorna estrutura por-endpoint
- [ ] `Reader.ListEndpoints` popula `EndpointDetail` completo
- [ ] `endpointListView` emite o novo JSON
- [ ] Testes de integração do linker (materialização correta) e do reader (parsing correto) passando
- [ ] `go run ./cmd/rhaxis-api` sobe e `GET /services/{urn}/endpoints` devolve o novo shape contra o Neo4j local
