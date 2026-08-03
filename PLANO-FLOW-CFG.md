# Plano de implementação — Fluxo como CFG (Tela 3)

**Objetivo:** substituir a representação atual do fluxo interno de Endpoints/Functions/Methods — hoje uma árvore ordenada por `CONTAINS {index}` — por um **grafo de fluxo de controle (CFG) explícito**, com arestas `NEXT` para sequência, `BRANCH` para bifurcação, `FLOWS_TO` para propagação de dado entre steps, e um `EntryNode` ancorando o início. Terminações (`Return`/`Throw`) opcionalmente confluem para um `ExitNode` sintético para facilitar queries.

A API da Tela 3 (`LoadEndpointFlow` / `ExpandFlow`) passa a devolver um par `{nodes, edges}` — grafo genuíno, não árvore.

**Escopo:** control-flow interno de containers executáveis. Ortogonal ao `PLANO-ENDPOINT-EDGES.md` (edges agregadas por endpoint, Tela 2) e complementar ao `ANALISE-METADADOS-NODES.md` §1 (remoção de props que duplicam edges).

---

## 1. Diagnóstico consolidado

### 1.1. O que o modelo já contempla

- **Kinds** existem em `domain/kinds.go`: `EntryNode`, `ExitNode`, `JoinNode`, `IfNode`, `SwitchNode`, `LoopNode`, `TryNode`, `Block`, `ReturnNode`, `ThrowNode`.
- **Edges** existem em `domain/node.go`: `EdgeNext` (`NEXT`), `EdgeBranch` (`BRANCH`), `EdgeContains` (`CONTAINS`), `EdgeCalls`, `EdgeExpandsTo`.
- Comentários da spec (`kinds.go:126-144`) já descrevem Entry→…→Exit via `NEXT` e Join reconvergindo após branchy.

### 1.2. O que a extração/persistência/consulta realmente fazem

| Camada | Estado atual | Gap |
|---|---|---|
| Extração (fixture `nestjs-orders.cypher`) | Ordena steps por `CONTAINS {index}`; sem Entry/Exit/Join; sem `NEXT` | CFG não é emitido — a sequência é implícita pela ordem no array |
| Persistência (`Writer`) | Genérica: `UpsertNode` + `UpsertEdge` funcionam para qualquer kind/edge | Nenhuma — mudanças aqui são aditivas |
| Query (`cypherEndpointFlow`, `cypherExpandFlow`) | Percorre só `CONTAINS` + 1 nível de `BRANCH` | `NEXT` está declarado mas **nunca navegado**; sem noção de Entry→Exit |
| Reader (`buildStepChildren`) | Deduplica produto cartesiano do OPTIONAL MATCH aninhado | Complexidade desnecessária — some quando query voltar `{nodes, edges}` |
| Domain (`FlowNode`) | Árvore: `Children`, `Branches map[label][]FlowNode` | Não expressa reconvergência; frontend não sabe onde branches confluem |

### 1.3. Consequência prática

- Após um `IfNode`, o front recebe branches como sub-árvores. Não há como saber que a saída de `then` e `else` levam ao mesmo step subsequente.
- `LoopNode` não expressa back-edge.
- Dado que flui entre steps (retorno de call vira condição de if, param propaga como arg) não é modelado — é só texto de display em `IfNode.ConditionText` etc.
- Middlewares são "tags" (`PROTECTS`), não passos do fluxo.

---

## 2. Decisões de design

### 2.1. Grafo genuíno na resposta da Tela 3 (Opção 2)

`EndpointFlow` deixa de ser árvore. Passa a devolver todos os nodes do container em uma lista plana + todas as arestas entre eles. Frontend desenha direto — sem inferir ordem por índice.

**Trade-off aceito:** breaking change no shape JSON. Necessário coordenar com o consumer da API.

### 2.2. `EntryNode` sempre; `ExitNode` opcional mas recomendado

`Return`/`Throw` já são terminais naturais (comentário em `domain/kinds.go:111-124`). Não são estritamente necessários como sinks.

**Decisão: emitir `ExitNode` único por container, com todos os `ReturnNode`/`ThrowNode` desse container ligados a ele via `NEXT`.**

Motivação:
- **Simetria** com `EntryNode` — âncora clara de fim de fluxo para o front (pin de "end" no render).
- **Queries triviais** de "todas as terminações desta função" via `(container)-[:CONTAINS]->(exit:ExitNode)<-[:NEXT]-(term)`.
- **Path queries** viáveis: `shortestPath((entry)-[:NEXT|BRANCH*]->(exit))` cobre qualquer caminho de execução do container.
- **Custo:** N arestas extras (uma por Return/Throw) e um node por container. Marginal.
- **`ExitNode` também recebe `NEXT` do último step "normal" de um Block que caia por fluxo natural** (função void que termina sem `return` explícito).

### 2.3. `JoinNode` só quando há continuação

Emitir `JoinNode` **apenas quando existe statement após o branchy no escopo pai**. Se todos os ramos terminam em `Return`/`Throw`, cada terminal aponta direto para o `ExitNode` — não há a que juntar.

### 2.4. Data-flow como aresta separada

Novo `EdgeDataFlow` (`FLOWS_TO`) para propagação de dado entre steps. Props:
- `role: 'return'|'arg:0'|'arg:1'|…|'condition'|'discriminant'|'thrown'|'caught'`
- `via: string` (opcional; nome da variável intermediária, best-effort)

`FLOWS_TO` é **ortogonal a `NEXT`** — expressa "de onde este valor veio" enquanto `NEXT` expressa "qual passo executa em seguida". Duas queries diferentes, dois eixos diferentes.

### 2.5. Middlewares na cadeia `NEXT` do Endpoint

Middlewares deixam de ser só nós satélites de `PROTECTS`. Passam a estar na sequência `NEXT` do endpoint:

```
(entry) -NEXT-> (mw0:Middleware) -NEXT-> (mw1:Middleware) -NEXT-> (call:CallFunction→handler) -NEXT-> (exit)
```

`PROTECTS` continua existindo como affordance agregada da Tela 2 (`PLANO-ENDPOINT-EDGES.md`). Redundância aceita: ganho de expressividade > custo de duplicar informação.

### 2.6. Loops com back-edge explícita

`LoopNode` emite:
- `BRANCH {label:'body'}` → primeiro step do body Block.
- Último step do body → `NEXT {backEdge:true}` → `LoopNode` (back-edge).
- `LoopNode` → `NEXT` → step de saída (após o loop).

Prop `backEdge:true` na aresta permite ao front detectar ciclo sem análise topológica.

### 2.7. Parâmetros e retorno de Function/Method — faseado

**v1:** não emitir `ParamSlot`/`ReturnSlot` como kinds. `FLOWS_TO` liga direto:
- `EntryNode` como origem (semântica: "parâmetro N do container", codificado em `role: 'arg:N'`).
- `ReturnNode` como destino (semântica: "valor de retorno do container").
- Do call externo, `FLOWS_TO {role:'return'}` sai do `CallFunction` e entra no consumidor.

**v2 (futuro):** promover a kinds `ParamSlot`/`ReturnSlot` filhos de Function via `CONTAINS`, se queries topológicas exigirem ("quem consome o param X de Y?").

---

## 3. Alterações — passo a passo

### Passo 3.1 — Domain (aditivo)

**`domain/node.go`:**

```go
const (
    // ... existentes ...
    EdgeFlowsTo EdgeType = "FLOWS_TO"
)
```

**`domain/registry.go` (via `decoders.go`/`encoders.go`):**
- Nenhum kind novo em v1.
- `EntryNode`, `ExitNode`, `JoinNode` já registrados — verificar decoders/encoders existentes.

**Whitelist do loader** (`loader/loader.go:118-145`):
- Adicionar `domain.EdgeFlowsTo` à lista `isKnownEdgeType`.

### Passo 3.2 — Nova view de fluxo (`domain/aggregates.go`)

Substituir `EndpointFlow` e `FlowNode`:

```go
type EndpointFlow struct {
    Endpoint Endpoint
    EntryURN URN
    ExitURN  URN            // vazio se container não emitir ExitNode
    Nodes    []FlowNode     // todos os nodes do subgrafo do container
    Edges    []FlowEdge     // NEXT | BRANCH | FLOWS_TO | EXPANDS_TO
}

type FlowNode struct {
    Node      domain.Node
    Expansion *ExpansionSlot   // continua igual: Call*/ConsumeEvent
}

type FlowEdge struct {
    From  URN
    To    URN
    Type  EdgeType
    Label string             // "then" | "else" | "case:x" | "body" | "arg:0" | "condition" | ...
    Props map[string]any     // {backEdge:true}, {via:"user.id"}, ...
}
```

`ExpandFlow` devolve o mesmo shape (`EndpointFlow` renomeado como `ContainerFlow` conceitualmente, ou reuso literal — decidir no PR).

### Passo 3.3 — Extração / Fixture (`schema/fixtures/nestjs-orders.cypher`)

Reescrever a fixture como referência viva do novo shape. Regras a garantir por container executável (Endpoint, Function, Method):

1. `EntryNode` com URN `<container>.entry`, filho de body Block via `CONTAINS {index:0}`.
2. `ExitNode` com URN `<container>.exit`, filho de body Block via `CONTAINS {index:last+1}`.
3. `Entry` → `NEXT` → primeiro step.
4. Cada par consecutivo de steps → `NEXT`.
5. Último step de fluxo natural → `NEXT` → `Exit`.
6. Cada `ReturnNode`/`ThrowNode` → `NEXT` → `Exit` (mesmo Exit do container).
7. `IfNode`/`SwitchNode`/`TryNode`:
   - `BRANCH {label}` para cada ramo (primeiro step do branch Block).
   - Se há continuação: `JoinNode` compartilhado; último step de cada ramo → `NEXT` → `Join`; `Join` → `NEXT` → próximo step.
   - Se todos os ramos terminam em Return/Throw: sem Join; terminais vão direto para o Exit.
8. `LoopNode`: `BRANCH {label:'body'}`; último step do body → `NEXT {backEdge:true}` → `LoopNode`; `LoopNode` → `NEXT` → step de saída.
9. Middlewares do Endpoint: `Entry` → `NEXT` → `Middleware₀` → `NEXT` → … → `NEXT` → invocação do handler → `NEXT` → `Exit`. `PROTECTS` mantido.
10. `FLOWS_TO` best-effort onde o extrator conseguir: retorno de call para condição de if seguinte; args de call vindos de param do container ou de step anterior.

### Passo 3.4 — Queries (`infra/neo4j/queries.go`)

**Nova `cypherEndpointFlow`:**

```cypher
MATCH (e:Endpoint {urn: $endpointURN})
// Todos os nodes do container via CONTAINS transitivo (não segue EXPANDS_TO/CALLS)
OPTIONAL MATCH (e)-[:CONTAINS*0..]->(n)
WITH e, collect(DISTINCT n) AS nodes
// Arestas de fluxo entre nodes desse conjunto + EXPANDS_TO como affordance
UNWIND nodes AS src
OPTIONAL MATCH (src)-[r:NEXT|BRANCH|FLOWS_TO|EXPANDS_TO]->(dst)
WHERE dst IN nodes OR type(r) = 'EXPANDS_TO'
WITH e, nodes,
     collect(DISTINCT CASE WHEN r IS NOT NULL THEN {
       from: src.urn, to: dst.urn, type: type(r),
       label: coalesce(r.label, ''), props: properties(r)
     } END) AS rawEdges
RETURN e AS endpoint,
       nodes,
       [x IN rawEdges WHERE x IS NOT NULL] AS edges
```

Pontos-chave:
- **Sem produto cartesiano** — cada linha do collect é uma edge única.
- `CONTAINS*0..` é seguro (sem ciclos por design de `CONTAINS`).
- Cycles em `NEXT` (loops) não recursam — só coletamos arestas entre nodes já materializados no conjunto.
- `EXPANDS_TO` incluída sempre — apontamento para alvo fora do container (Call→Function/Endpoint) preserva contrato lazy.
- Retornar `properties(r)` inclui `backEdge:true`, `via:"..."`, etc, sem enumerar props no query.
- `EntryURN`/`ExitURN` derivam no Go a partir do kind dos nodes (evita ida extra ao Neo4j).

**Nova `cypherExpandFlow`:**

```cypher
MATCH (target {urn: $targetURN})
WHERE target:Function OR target:Method OR target:Endpoint
   OR target:Block OR target:ConsumeEvent
OPTIONAL MATCH (target)-[:CONTAINS*0..]->(n)
WITH target, collect(DISTINCT n) AS nodes
UNWIND nodes AS src
OPTIONAL MATCH (src)-[r:NEXT|BRANCH|FLOWS_TO|EXPANDS_TO]->(dst)
WHERE dst IN nodes OR type(r) = 'EXPANDS_TO'
WITH target, nodes,
     collect(DISTINCT CASE WHEN r IS NOT NULL THEN {
       from: src.urn, to: dst.urn, type: type(r),
       label: coalesce(r.label, ''), props: properties(r)
     } END) AS rawEdges
RETURN target,
       nodes,
       [x IN rawEdges WHERE x IS NOT NULL] AS edges
```

Estrutura idêntica à Tela 3 — o cliente lida com a mesma shape em ambos os endpoints.

### Passo 3.5 — Reader (`infra/neo4j/reader.go`)

**Some completamente** o `buildStepChildren` (linhas 191-298). Substitui por parsing linear:

```go
func (r *Reader) LoadEndpointFlow(ctx context.Context, endpointURN domain.URN) (domain.EndpointFlow, error) {
    // ... boilerplate de sessão + Run ...
    rec := result.(*neo4j.Record)

    epNeo, _ := asNeoNode(recordGet(rec, "endpoint"))
    epDomain, err := nodeToDomain(epNeo)
    // ... type-assert Endpoint ...

    flow := domain.EndpointFlow{Endpoint: *ep}

    for _, v := range asListProp(recordGet(rec, "nodes")) {
        n, ok := asNeoNode(v)
        if !ok { continue }
        d, ok := nodeToDomainSafe(n)
        if !ok { continue }
        fn := domain.FlowNode{Node: d}
        if slot := buildExpansionSlot(d, nil); slot != nil {
            fn.Expansion = slot
        }
        flow.Nodes = append(flow.Nodes, fn)
        // Deriva Entry/Exit por kind
        switch d.Kind() {
        case domain.KindEntry:
            flow.EntryURN = d.URN()
        case domain.KindExit:
            flow.ExitURN = d.URN()
        }
    }

    for _, v := range asListProp(recordGet(rec, "edges")) {
        m := asMapProp(v)
        if m == nil { continue }
        flow.Edges = append(flow.Edges, domain.FlowEdge{
            From:  domain.URN(asStringProp(m["from"])),
            To:    domain.URN(asStringProp(m["to"])),
            Type:  domain.EdgeType(asStringProp(m["type"])),
            Label: asStringProp(m["label"]),
            Props: asMapProp(m["props"]),
        })
    }
    return flow, nil
}
```

Mesma estrutura para `ExpandFlow` — devolve o mesmo tipo (renomeado ou não).

`buildExpansionSlot` continua sendo chamado por node, mas **sem passar `callTargetRaw`** — o EXPANDS_TO agora vem como edge, não como coluna do record. O slot é construído a partir dos props do próprio node (`CallFunction.TargetURN` etc), e o front usa a edge `EXPANDS_TO` da resposta para navegar. Alternativa: emitir `Expansion` no reader olhando a lista de edges antes — decidir no PR conforme o que ficar mais limpo.

### Passo 3.6 — View HTTP (`cmd/rhaxis-api/view.go`)

Reescrever `flowNodeView` e `endpointFlowView`:

```go
func endpointFlowView(ef domain.EndpointFlow) map[string]any {
    ep := ef.Endpoint
    nodes := make([]map[string]any, len(ef.Nodes))
    for i, fn := range ef.Nodes {
        v := map[string]any{"node": nodeView(fn.Node)}
        if fn.Expansion != nil {
            v["expansion"] = map[string]any{
                "targetURN":      fn.Expansion.TargetURN,
                "targetKind":     fn.Expansion.TargetKind,
                "targetResolved": fn.Expansion.TargetResolved,
            }
        }
        nodes[i] = v
    }
    edges := make([]map[string]any, len(ef.Edges))
    for i, e := range ef.Edges {
        edges[i] = map[string]any{
            "from":  e.From,
            "to":    e.To,
            "type":  e.Type,
            "label": e.Label,
            "props": e.Props,
        }
    }
    return map[string]any{
        "endpoint": nodeView(&ep),
        "entryURN": ef.EntryURN,
        "exitURN":  ef.ExitURN,
        "nodes":    nodes,
        "edges":    edges,
    }
}
```

Idem para `ExpandFlow` (se compartilhar tipo, compartilha view).

### Passo 3.7 — Testes

**Unidade (reader):**
- Record com `nodes=[]`, `edges=[]` → `FlowGraph` vazio, sem panic.
- Record com Entry, Exit, um Call, arestas Entry→Call, Call→Exit → 3 nodes, 2 edges, `EntryURN`/`ExitURN` preenchidos.
- Record com IfNode + then/else + Join → 6 nodes (if, blockThen, stepT, blockElse, stepE, join), edges com labels corretos.
- Record com LoopNode + back-edge → edge com `props.backEdge == true`.
- Record com CallFunction + EXPANDS_TO para alvo fora do conjunto → edge EXPANDS_TO presente na resposta.

**Integração (`reader_integration_test.go`):**
- Seed com nova fixture. Assert que o grafo devolvido pelo `LoadEndpointFlow` cobre:
  - Entry único, Exit único.
  - Todo Return/Throw alcança Exit via NEXT.
  - Toda edge da fixture aparece com props preservadas.
  - `ExpandFlow` de uma Function devolve o interior dela com seu próprio Entry/Exit.

**Contrato:**
- Verificar `GET /endpoints/{urn}/flow` devolve o novo JSON contra Neo4j real.

---

## 4. Contrato HTTP resultante

`GET /endpoints/{urn}/flow`:

```json
{
  "endpoint": { "urn": "…", "httpMethod": "POST", "pathTemplate": "/orders", "..." : "..." },
  "entryURN": "urn:cg:orders-api:ts:entry:endpoints/post-orders.entry",
  "exitURN":  "urn:cg:orders-api:ts:exit:endpoints/post-orders.exit",
  "nodes": [
    { "node": { "urn": "…", "kind": "EntryNode", "..." : "..." } },
    { "node": { "urn": "…", "kind": "Middleware", "name": "AuthGuard", "..." : "..." } },
    { "node": { "urn": "…", "kind": "CallFunction", "name": "create()", "..." : "..." },
      "expansion": { "targetURN": "…", "targetKind": "Method", "targetResolved": true } },
    { "node": { "urn": "…", "kind": "IfNode", "conditionText": "user.isAdmin" } },
    { "node": { "urn": "…", "kind": "CallDB", "..." : "..." } },
    { "node": { "urn": "…", "kind": "JoinNode", "after": "if" } },
    { "node": { "urn": "…", "kind": "ReturnNode", "expressionText": "order" } },
    { "node": { "urn": "…", "kind": "ExitNode" } }
  ],
  "edges": [
    { "from": "…entry",   "to": "…mw",     "type": "NEXT",       "label": "", "props": {} },
    { "from": "…mw",      "to": "…call",   "type": "NEXT",       "label": "", "props": {} },
    { "from": "…call",    "to": "…if",     "type": "NEXT",       "label": "", "props": {} },
    { "from": "…call",    "to": "…if",     "type": "FLOWS_TO",   "label": "condition", "props": {} },
    { "from": "…if",      "to": "…cdb",    "type": "BRANCH",     "label": "then", "props": {} },
    { "from": "…if",      "to": "…join",   "type": "BRANCH",     "label": "else", "props": {} },
    { "from": "…cdb",     "to": "…join",   "type": "NEXT",       "label": "", "props": {} },
    { "from": "…join",    "to": "…return", "type": "NEXT",       "label": "", "props": {} },
    { "from": "…return",  "to": "…exit",   "type": "NEXT",       "label": "", "props": {} },
    { "from": "…call",    "to": "…handler","type": "EXPANDS_TO", "label": "", "props": {} }
  ]
}
```

`GET /flow/{urn}/expand` devolve o mesmo shape para o interior da Function/Method/Block alvo.

---

## 5. Sequência sugerida de commits

1. **domain aditivo:** `EdgeFlowsTo` + whitelist do loader. Grafos antigos continuam válidos.
2. **domain tipos novos:** `FlowEdge`, novo `FlowNode`, novo `EndpointFlow`. Ainda não usados pelo reader — só o tipo.
3. **fixture reescrita:** `schema/fixtures/nestjs-orders.cypher` com Entry/Exit/Join/NEXT/FLOWS_TO no shape novo. Serve de referência para o extrator externo.
4. **queries + reader:** trocar `cypherEndpointFlow`/`cypherExpandFlow` e reescrever `LoadEndpointFlow`/`ExpandFlow`. Remover `buildStepChildren`. Testes de integração adaptados.
5. **view:** nova serialização em `cmd/rhaxis-api/view.go`. **Breaking change no JSON** — coordenar com front.
6. **extractor (fora deste repo):** atualizar para emitir CFG completo. Sem esse passo, endpoints extraídos com o extrator antigo devolvem grafo pobre (sem Entry/Exit/NEXT), mas nada quebra — as arestas simplesmente não existem.

Passos 1-3 são aditivos. 4-5 são coordenados em um único release. 6 é independente e pode ir depois.

---

## 6. Riscos e mitigação

| Risco | Mitigação |
|---|---|
| Grafos antigos sem Entry/Exit/NEXT devolvem grafo esparso | Aceito — `entryURN`/`exitURN` vazios, front renderiza como fallback (lista de nodes sem arestas). Documentar. |
| Ciclos por `NEXT` (loops) em variable-length | Query só coleta arestas entre nodes do conjunto materializado — sem recursão em path patterns. |
| Fan-out enorme em containers grandes | Aceitável para Tela 3. Se virar problema, aplicar `LIMIT` no `CONTAINS*` + flag `truncated`. |
| Breaking change no JSON da Tela 3 | Coordenado com front no mesmo release; versionar rota (`/v2/flow`) se houver consumer externo. |
| `EXPANDS_TO` em edge vs `Expansion` em node — informação duplicada | Escolher no PR: manter `Expansion` derivado no reader (compat com o campo atual do `FlowNode`) e emitir `EXPANDS_TO` como edge só quando o alvo estiver fora do conjunto. |
| Extrator emite Return/Throw sem `NEXT` para Exit | Reader não força — Exit fica órfão; alguns terminals ficam sem alcançar o sink. Documentar como responsabilidade do extrator. Adicionar teste de invariante no CI do extrator (fora deste repo). |
| `FLOWS_TO` best-effort — extrator não consegue rastrear tudo | Aceito. `FLOWS_TO` é enriquecimento, não requisito. Ausência não quebra a Tela 3. |

---

## 7. Definition of Done

- [ ] `EdgeFlowsTo` (`FLOWS_TO`) declarada em `domain/node.go` e aceita pelo `loader.isKnownEdgeType`.
- [ ] `domain.EndpointFlow`, `domain.FlowNode`, `domain.FlowEdge` refletindo grafo (nodes + edges + entry/exit URN).
- [ ] `schema/fixtures/nestjs-orders.cypher` reescrita com Entry/Exit/Join/NEXT/FLOWS_TO conforme regras da §3.3.
- [ ] `cypherEndpointFlow` e `cypherExpandFlow` reescritas para retornar `{endpoint|target, nodes, edges}`.
- [ ] `Reader.LoadEndpointFlow` e `Reader.ExpandFlow` populam o novo shape; `buildStepChildren` removido.
- [ ] `endpointFlowView` emite JSON com `entryURN`, `exitURN`, `nodes[]`, `edges[]`.
- [ ] Testes unitários do reader cobrindo: fluxo linear, if com join, if terminando em return (sem join), loop com back-edge, EXPANDS_TO para alvo externo, FLOWS_TO com props.
- [ ] Teste de integração contra fixture nova.
- [ ] `go run ./cmd/rhaxis-api` sobe e `GET /endpoints/{urn}/flow` devolve o novo shape contra Neo4j local.

---

## 8. Fora de escopo (para plano futuro)

- Promover `ParamSlot`/`ReturnSlot` a kinds próprios (§2.7 v2).
- Materializar edges agregadas por Endpoint (`PLANO-ENDPOINT-EDGES.md`) — ortogonal, complementar.
- Remoção das props `Call*.TargetURN`, `Endpoint.HandlerURN` etc (`ANALISE-METADADOS-NODES.md` §1) — ortogonal; se feito antes deste plano, ajustar `buildExpansionSlot` para depender só das edges.
- Análise de reachability (dead code, unreachable branches) — habilitada pelo CFG, mas fora deste plano.
- Métricas derivadas (complexity, path count) — idem.
