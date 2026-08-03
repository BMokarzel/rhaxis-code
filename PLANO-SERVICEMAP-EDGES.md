# Plano de implementação — arestas explícitas no `ServiceMap` (Abordagem B)

**Objetivo:** fazer com que `LoadServiceMap` devolva, além dos nós, **todas as arestas** necessárias para o frontend desenhar o grafo de topologia:

- `Service → Service` (`DEPENDS_ON`) — já existe hoje via `deps`
- `Service → Database` (`USES_DB`) — **faltando**
- `Service → Broker` (`PUBLISHES_TO` / `CONSUMES_FROM`) — **faltando**

Abordagem escolhida: **manter a forma da query atual** (listas separadas `services` / `dbs` / `brokers`) e **coletar as arestas na mesma etapa** em que o pareamento `(s, db)` / `(s, b)` ainda existe, antes do `collect` colapsar as linhas.

---

## 1. Diagnóstico

### 1.1. Onde está o problema
`infra/neo4j/queries.go` → `cypherServiceMap`:

```cypher
WITH collect(DISTINCT s) AS services,
     collect(DISTINCT db) AS dbs,
     collect(DISTINCT b)  AS brokers
```

Após esse `WITH`, o pareamento `(s, db)` e `(s, b)` é descartado. Só sobra a existência de cada tipo de nó, sem quem-usa-quem.

### 1.2. Estado da persistência
As arestas já existem no grafo (`USES_DB`, `PUBLISHES_TO`, `CONSUMES_FROM`, `DEPENDS_ON`). **Nenhuma mudança de modelo é obrigatória** para essa entrega. Ajustes opcionais estão listados em §6.

---

## 2. Escopo

### 2.1. Dentro do escopo
- Nova versão de `cypherServiceMap` que devolve **listas de arestas explícitas** para todos os tipos acima.
- Novos campos no agregado `domain.ServiceMap` para transportar essas arestas.
- Parsing das novas listas em `infra/neo4j/reader.go` (`LoadServiceMap`).
- Serialização JSON no `view.go` de `cmd/rhaxis-api`.
- Testes de unidade e integração cobrindo os novos campos.

### 2.2. Fora do escopo
- Mudança na Tela 2 (`ListEndpoints`) ou Tela 3 (`LoadEndpointFlow` / `ExpandFlow`).
- Redesenho do frontend (o backend passa a fornecer os dados; render é outra frente).
- Introdução de novos tipos de nós/arestas no schema Cypher.

---

## 3. Alterações — passo a passo

### Passo 3.1 — Domain (`domain/aggregates.go`)

Estender `ServiceMap` com listas de arestas. Manter compatibilidade dos campos existentes (nenhuma remoção).

```go
type ServiceMap struct {
    Services        []Service
    ExternalSystems []Node
    Dependencies    []ServiceDependency // Service→Service (já existe)
    UsesDB          []ExternalEdge      // Service→Database
    UsesBroker      []ExternalEdge      // Service→Broker (com Kind = PUBLISHES_TO|CONSUMES_FROM)
}

// ExternalEdge é uma aresta de um Service para um sistema externo (DB/Broker).
// Kind carrega o tipo da relação Cypher para o frontend diferenciar
// publish vs consume no caso de brokers.
type ExternalEdge struct {
    From URN
    To   URN
    Kind string // "USES_DB" | "PUBLISHES_TO" | "CONSUMES_FROM"
}
```

**Rationale:** um único tipo `ExternalEdge` para `DB` e `Broker` — a diferença de semântica fica em `Kind`. Evita explosão de tipos e mantém o frontend uniforme.

### Passo 3.2 — Query (`infra/neo4j/queries.go`)

Nova versão de `cypherServiceMap`:

```cypher
MATCH (s:Service)
OPTIONAL MATCH (s)-[:USES_DB]->(db:Database)
OPTIONAL MATCH (s)-[rb:PUBLISHES_TO|CONSUMES_FROM]->(b:Broker)
WITH
  collect(DISTINCT s)  AS services,
  collect(DISTINCT db) AS dbs,
  collect(DISTINCT b)  AS brokers,
  [x IN collect(DISTINCT
     CASE WHEN db IS NOT NULL
          THEN {from: s.urn, to: db.urn, kind: 'USES_DB'} END)
   WHERE x IS NOT NULL] AS usesDb,
  [x IN collect(DISTINCT
     CASE WHEN b IS NOT NULL
          THEN {from: s.urn, to: b.urn, kind: type(rb)} END)
   WHERE x IS NOT NULL] AS usesBroker
OPTIONAL MATCH (a:Service)-[d:DEPENDS_ON]->(bt:Service)
RETURN services, dbs, brokers, usesDb, usesBroker,
  [x IN collect({from: a.urn, to: bt.urn, via: d.via,
                 weight: coalesce(d.weight, 0)})
   WHERE x.from IS NOT NULL] AS deps
```

Pontos chave:
- `usesDb` e `usesBroker` são coletados **antes** do `WITH` colapsar — o pareamento `(s, db)` / `(s, b)` sobrevive.
- `type(rb)` distingue `PUBLISHES_TO` de `CONSUMES_FROM` num único match.
- Comprehension `[x IN collect(...) WHERE x IS NOT NULL]` remove o "row lixo" que o `OPTIONAL MATCH` gera quando não casa.
- O filtro do `deps` foi trocado para `WHERE x.from IS NOT NULL` (bug menor da query atual, aproveitando a mudança).

### Passo 3.3 — Reader (`infra/neo4j/reader.go`)

Em `LoadServiceMap`, adicionar parsing das duas novas listas depois do parsing de `deps`. Padrão idêntico ao já existente:

```go
for _, v := range asListProp(recordGet(rec, "usesDb")) {
    m := asMapProp(v)
    if m == nil { continue }
    from := asStringProp(m["from"])
    to   := asStringProp(m["to"])
    if from == "" || to == "" { continue }
    out.UsesDB = append(out.UsesDB, domain.ExternalEdge{
        From: domain.URN(from),
        To:   domain.URN(to),
        Kind: asStringProp(m["kind"]),
    })
}
// idêntico para "usesBroker" → out.UsesBroker
```

Nenhuma outra rota (Tela 2/3) é tocada.

### Passo 3.4 — View (`cmd/rhaxis-api/view.go`)

Em `serviceMapView`, emitir os novos campos como listas JSON no mesmo formato de `dependencies` (chaves camelCase para consistência com o restante do arquivo):

```go
usesDB := make([]map[string]any, len(sm.UsesDB))
for i, e := range sm.UsesDB {
    usesDB[i] = map[string]any{"from": e.From, "to": e.To, "kind": e.Kind}
}
usesBroker := make([]map[string]any, len(sm.UsesBroker))
for i, e := range sm.UsesBroker {
    usesBroker[i] = map[string]any{"from": e.From, "to": e.To, "kind": e.Kind}
}
return map[string]any{
    "services":        services,
    "externalSystems": ext,
    "dependencies":    deps,
    "usesDB":          usesDB,
    "usesBroker":      usesBroker,
}
```

### Passo 3.5 — Testes

**Unidade** (`infra/neo4j/reader_test.go`, se existir; senão criar):
- record mockado com `usesDb`/`usesBroker` bem formados → parsing correto
- record com listas vazias → `nil` slices, sem panic
- record com entradas parciais (`from` ou `to` vazios) → ignoradas

**Integração** (`infra/neo4j/reader_integration_test.go`):
- Seed com: 2 services, 1 db (usado por ambos), 1 broker (um publica, outro consome), 1 `DEPENDS_ON` entre os services.
- Asserts: `len(UsesDB) == 2`, `len(UsesBroker) == 2`, `Kind` correto por aresta, `len(Dependencies) == 1`.

---

## 4. Contrato HTTP resultante

`GET /service-map` passa a devolver:

```json
{
  "services":        [ /* nodes */ ],
  "externalSystems": [ /* nodes */ ],
  "dependencies": [
    { "from": "svc:orders", "to": "svc:billing", "via": "http", "weight": 3 }
  ],
  "usesDB": [
    { "from": "svc:orders", "to": "db:orders-pg", "kind": "USES_DB" }
  ],
  "usesBroker": [
    { "from": "svc:orders", "to": "brk:events", "kind": "PUBLISHES_TO" },
    { "from": "svc:billing","to": "brk:events", "kind": "CONSUMES_FROM" }
  ]
}
```

Aditivo — não quebra clientes que só liam `services`/`dependencies`.

---

## 5. Sequência sugerida de commits

1. **domain:** adicionar `ExternalEdge` e campos em `ServiceMap` (sem uso ainda) → compila, testes verdes.
2. **infra/neo4j (query + reader):** trocar `cypherServiceMap` e adicionar parsing. Integração passa a preencher os campos novos.
3. **cmd/rhaxis-api (view):** expor no JSON.
4. **testes:** cobertura de unidade + integração.

Cada commit é independentemente verde. Rollback do último não afeta os anteriores.

---

## 6. Ajustes de persistência (opcionais / futuros)

Nenhum é bloqueante para esta entrega. Cada um resolve um problema real diferente:

| Situação | Ajuste sugerido | Motivação |
|---|---|---|
| Nós sem `urn` estável | `CREATE CONSTRAINT` de unicidade em `urn` por label | Evita duplicatas silenciosas que quebram `DISTINCT` e o pareamento |
| Precisa distinguir leitura de escrita no DB | Substituir `USES_DB` por `READS_FROM` / `WRITES_TO`, ou adicionar propriedade `mode` na aresta | Análises de fan-out de escrita, blast radius |
| Broker com múltiplos tópicos | Modelar tópico como nó: `(:Service)-[:PUBLISHES_TO]->(:Topic)-[:HOSTED_ON]->(:Broker)` | Hoje uma aresta por par (service, broker) — não representa "qual tópico" |
| Dependência com múltiplos canais (HTTP + evento) | Manter multi-aresta `DEPENDS_ON` com `via` distinto, ou consolidar via `weight` | Já suportado; documentar |
| Performance em grafos grandes | Índice `CREATE INDEX FOR (s:Service) ON (s.urn)` (equivalente para `Database`, `Broker`) | O `MATCH (s:Service)` inicial escaneia toda label; índice acelera filtros futuros |

**Recomendação:** priorizar a constraint de `urn` (linha 1) mesmo antes desta feature — é barato e evita bugs de duplicação difíceis de rastrear.

---

## 7. Riscos e mitigação

| Risco | Mitigação |
|---|---|
| Explosão de tamanho do JSON em grafos grandes | Campos novos são listas de dicts pequenos (3 chaves); ainda O(E). Se virar problema, paginação/filtro no futuro (`ServiceMapFilter` já existe). |
| Cypher com `CASE ... WHEN ... END` interpretado errado por versão antiga do driver | Testar contra Neo4j 5.x (versão-alvo). Query é standard Cypher, sem APOC. |
| Falso-positivo de "aresta vazia" no `deps` | O `WHERE x.from IS NOT NULL` já cuida (comentado no §3.2). |
| Frontend legado consumindo o endpoint | Mudança é aditiva; nenhum campo removido/renomeado. |

---

## 8. Definition of Done

- [ ] `cypherServiceMap` atualizado e revisado (PROFILE opcional para conferir plano)
- [ ] `domain.ServiceMap` com `UsesDB` e `UsesBroker`
- [ ] `Reader.LoadServiceMap` popula as duas listas
- [ ] `serviceMapView` emite `usesDB` e `usesBroker` no JSON
- [ ] Teste de integração com seed cobrindo `USES_DB`, `PUBLISHES_TO`, `CONSUMES_FROM`, `DEPENDS_ON` — passando
- [ ] `go run ./cmd/rhaxis-api` sobe e `GET /service-map` devolve os novos campos populados contra o Neo4j local
