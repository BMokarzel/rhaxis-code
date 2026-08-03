# Plano-mestre — Melhorias unificadas do codegraph

**Objetivo:** consolidar em um único roteiro faseado as quatro frentes de melhoria já analisadas isoladamente:

- `PLANO-SERVICEMAP-EDGES.md` — Tela 1: expor `USES_DB`, `PUBLISHES_TO`, `CONSUMES_FROM` no ServiceMap.
- `PLANO-ENDPOINT-EDGES.md` — Tela 2: materializar edges agregadas por Endpoint (`TOUCHES_DB`, `TOUCHES_BROKER`, `CALLS_SERVICE`, `READS_CONFIG`).
- `PLANO-FLOW-CFG.md` — Tela 3: substituir a árvore atual por CFG explícito (`Entry`/`Exit`/`Join`/`NEXT`/`BRANCH`/`FLOWS_TO`).
- `ANALISE-METADADOS-NODES.md` — Limpeza: remover props que duplicam edges, tech strings duplicadas por nodes globais, campos derivados persistidos, metadata de extração replicada.

Cada plano foi bom em isolamento — mas há **dependências cruzadas**, **fixture única** que precisa servir a todos, e **um único extrator externo** que precisa ser atualizado uma vez só. Este documento reordena tudo em fases coerentes com avaliação entre elas, e substitui os quatro planos como referência de execução (eles ficam como spec detalhada de cada tópico).

---

## 1. Análise cruzada — o que cada plano assume do outro

### 1.1. Dependências identificadas

| Origem | Depende de | Efeito se ignorado |
|---|---|---|
| PLANO-ENDPOINT-EDGES | `HANDLED_BY` edge (ANALISE §1) | Handler cai como `nil` na Tela 2 |
| PLANO-ENDPOINT-EDGES | Linker traversal via `CONTAINS/EXPANDS_TO/CALLS` | Sem isso, edges agregadas ficam vazias |
| PLANO-FLOW-CFG | `buildExpansionSlot` hoje lê `Call*.TargetURN` prop; ANALISE §1 quer remover | Se remover antes, slot fica vazio; precisa passar a usar edge |
| PLANO-FLOW-CFG | Extrator emitindo CFG (Entry/Exit/NEXT/…) | Grafos ficam esparsos — não quebra, mas perde o valor |
| ANALISE §1 | Extrator emitindo edges consistentemente antes de remover props | Perda de informação em nodes antigos |
| Todos | Constraint `node_urn_unique` (schema/schema.cypher:5) | Já existe — nada a fazer |
| Todos | Fixture de teste que exercite as 3 telas simultaneamente | Hoje há `nestjs-orders.cypher` só — precisa evoluir |

### 1.2. Redundâncias e sinergias entre planos

- **`PROTECTS` na Tela 2** (edge agregada) e **middlewares na cadeia NEXT** (CFG da Tela 3) são complementares, não redundantes: `PROTECTS` responde "quem protege este endpoint" (afinidade), `NEXT` responde "em que ordem" (fluxo).
- **`TOUCHES_DB` na Tela 2** e **`CallDB → Database` no interior do fluxo** (Tela 3): mesma informação em granularidades diferentes. `TOUCHES_DB` é o **rollup** que a query da Tela 2 lê direto sem traversal — vale a duplicação, é a mesma justificativa arquitetural de `DEPENDS_ON` no Service.
- **Extrator v2**: um único evento coordena tudo (Entry/Exit/NEXT/JOIN/BRANCH/FLOWS_TO/HANDLED_BY/HAS_FIELD/MEMBER_OF/USES_CONFIG/PROTECTS/THROWS/CATCHES/HANDLES_ERROR). Fragmentar em N releases do extrator é pior que consolidar.
- **Fixture única**: `schema/fixtures/nestjs-orders.cypher` pode ser evoluída para exercitar todas as edges, servindo simultaneamente aos testes de integração das 3 telas.

### 1.3. Riscos de ordenação

- **Remover props antes do reader deixar de consultá-las → quebra Tela 3 atual.** `buildExpansionSlot` (`infra/neo4j/reader.go:303-334`) lê `CallFunction.TargetURN`, `CallHTTP.TargetURN`, etc. Precisa migrar para edge **antes** de qualquer remoção.
- **Introduzir CFG sem coordenar com o extrator → view devolve grafo esparso.** Não é quebra, mas quebra a percepção de "melhoria". Extrator v2 deve estar pronto para release conjunto com Fase 3.
- **Materializar TOUCHES_* sem HANDLED_BY → handler nulo na resposta.** Executar HANDLED_BY na fase anterior.

---

## 2. Estrutura do plano faseado

Seis fases. Cada fase é independentemente entregável (compila, testa, roda). Entre fases há um **checkpoint de avaliação** que decide se a próxima começa.

```
Fase 0 — Infra base                            [baixo risco, aditivo]
   ↓
Fase 1 — Declarações de novas edges + fixture   [aditivo]
   ↓
Fase 2 — Tela 1 rica (ServiceMap)               [aditivo no JSON]
   ↓
Fase 3 — CFG na Tela 3                          [BREAKING no JSON]
   ↓
Fase 3.5 — Extrator v2 (fora deste repo)        [coordenado]
   ↓
Fase 4 — Linker rico + Tela 2                   [BREAKING no JSON]
   ↓
Fase 5 — Limpeza de props duplicadas            [BREAKING no JSON de nodes]
```

Fases 0-2 podem sair em qualquer ordem entre si (todas aditivas). A partir da Fase 3 há ordem obrigatória.

---

## 3. Fase 0 — Infra base

**Escopo:** melhorias de baixo risco que beneficiam todas as fases seguintes. Nada que dependa de decisões maiores.

### 3.1. Ações

1. **Confirmar constraint `node_urn_unique`** (já em `schema/schema.cypher:5`). Adicionar constraint equivalente por label para `Service`, `Endpoint`, `Function`, `Method`, `Database`, `Broker` se PROFILE mostrar benefício — `MATCH (s:Service {urn: X})` já usa índice de constraint global, mas duplicatas silenciosas em label específica são um bug conhecido de MERGE (`PLANO-SERVICEMAP-EDGES.md` §6).
2. **Remover `displayShape` / `displayCategory` da persistência** (`ANALISE-METADADOS-NODES.md` §3). Passar cálculo para `cmd/rhaxis-api/view.go` ou deixar frontend derivar do `kind`.
3. **Mover `ExtractedAt` / `SourceRev`** de `BaseNode` para `Service` como `LastExtractedAt` / `SourceRev` (`ANALISE-METADADOS-NODES.md` §4, opção 1). Backfill: script que copia o valor de qualquer node do serviço para o próprio Service e apaga das folhas.
4. **Restringir `ResolvedValue`** aos kinds que realmente podem ser stubs (`CallHTTP`, `CallFunction`, `ConsumeEvent`) — `Service/Database/Broker/Endpoint` são sempre `true` (`ANALISE-METADADOS-NODES.md` §5). Mudança no encoder/decoder; contrato Node continua igual (default = true nos casos removidos).

### 3.2. Testes

- Snapshot do JSON antes/depois de nodes que perderam `displayShape` — não deve haver campo `displayShape` na resposta HTTP.
- Integração: `Service.lastExtractedAt` populada após load; nodes filhos sem essa prop.

### 3.3. Checkpoint de avaliação

- [ ] Nenhuma regressão nas 4 rotas atuais (`/service-map`, `/services/{urn}/endpoints`, `/endpoints/{urn}/flow`, `/flow/{urn}/expand`).
- [ ] Tamanho médio de node no Neo4j diminuiu (medir com `CALL db.stats.retrieve('GRAPH COUNTS')`).
- **Gate:** só avança se front confirmou que não lia `displayShape`/`displayCategory` do wire.

---

## 4. Fase 1 — Declarações de novas edges + fixture canônica

**Escopo:** preparar o vocabulário de edges e uma fixture única que servirá aos testes de todas as fases seguintes. Nenhuma consulta lê ainda — puramente aditivo.

### 4.1. Ações

1. **`domain/node.go`** — adicionar constantes:
   ```go
   EdgeHandledBy    EdgeType = "HANDLED_BY"     // Endpoint → Function|Method
   EdgeMemberOf     EdgeType = "MEMBER_OF"      // Method → Struct|Interface
   EdgeHasField     EdgeType = "HAS_FIELD"      // Struct → Type (props: name)
   EdgeFlowsTo      EdgeType = "FLOWS_TO"       // step → step (props: role, via)
   EdgeTouchesDB    EdgeType = "TOUCHES_DB"     // Endpoint → Database (props: ops)
   EdgeTouchesBroker EdgeType = "TOUCHES_BROKER" // Endpoint → Broker (props: kind, topics)
   EdgeCallsService EdgeType = "CALLS_SERVICE"  // Endpoint → Service (props: viaEndpointURN, weight)
   EdgeReadsConfig  EdgeType = "READS_CONFIG"   // Endpoint → Config
   ```
2. **`loader/loader.go`** — estender `isKnownEdgeType` para as novas edges.
3. **`domain/registry.go`** — nenhum kind novo em v1. Confirmar que `EntryNode`, `ExitNode`, `JoinNode` já têm decoder/encoder registrados.
4. **Fixture canônica reescrita** (`schema/fixtures/nestjs-orders.cypher`) — passa a exercitar:
   - CFG completo em cada endpoint (regras consolidadas em §7.1.1): `Endpoint` é container executável direto (sem wrapper `Block('body')`); `EntryNode` como `CONTAINS index=0` do Endpoint; `ExitNode` único como último `CONTAINS`; `JoinNode` APENAS quando há convergência real (não emitir se o único caminho vivo é o fallthrough do IfNode com then-terminal); `NEXT` conectando todos os passos; `BRANCH` com labels (`then`/`else`/`try`/`catch:T`/`finally`); terminals (Return/Throw) → `NEXT {isImplicit:true}` → `ExitNode`; fallthrough do IfNode (else implícito) marcado com `NEXT {isImplicit:true, label:'else'}` quando converge em Join.
   - Middlewares na cadeia `NEXT` + `PROTECTS`.
   - `HANDLED_BY` do Endpoint para o Handler.
   - `FLOWS_TO` best-effort em pontos óbvios (retorno de call vira condição de if, param propaga como arg).
   - `HAS_FIELD` / `MEMBER_OF` em um Struct+Method exemplo.
   - `THROWS` / `CATCHES` / `HANDLES_ERROR` já existentes preservados.
   - `USES_CONFIG` / `LOGS` já existentes preservados.
5. **Documentar a fixture como contrato do extrator** — inline no arquivo, prosa curta.

### 4.2. Testes

- Compilação verde.
- Testes de integração existentes contra a fixture continuam passando (a fixture cresceu, mas não removeu nada).
- Novo teste: `MATCH (e:Endpoint)-[:HANDLED_BY]->(h) RETURN count(*)` > 0 após seed.

### 4.3. Checkpoint de avaliação

- [ ] Fixture cobre 100% das edges do vocabulário v1.
- [ ] Loader aceita todas as novas edges sem skip.
- [ ] Ninguém consome ainda — nenhum teste de reader mudou de comportamento.
- **Gate:** só avança quando a fixture está estável e revisada.

---

## 5. Fase 2 — Tela 1 rica (ServiceMap)

**Escopo:** implementação integral do `PLANO-SERVICEMAP-EDGES.md`. É aditivo no JSON — clientes existentes continuam funcionando.

### 5.1. Ações

Execução conforme `PLANO-SERVICEMAP-EDGES.md` §3 (Passos 3.1-3.5) sem alterações. Resumo:

1. `domain/aggregates.go` — `ServiceMap` recebe `UsesDB []ExternalEdge` e `UsesBroker []ExternalEdge`.
2. `infra/neo4j/queries.go` — `cypherServiceMap` reescrita coletando arestas antes do `WITH` colapsar.
3. `infra/neo4j/reader.go` — parsing das novas listas.
4. `cmd/rhaxis-api/view.go` — emite `usesDB` e `usesBroker` no JSON.
5. Testes conforme §3.5 do plano original.

### 5.2. Checkpoint de avaliação

- [ ] `GET /service-map` devolve `usesDB` e `usesBroker` populados contra fixture da Fase 1.
- [ ] Tela 1 do front renderiza arestas Service→DB e Service→Broker (ou aceita que ainda não renderiza; contrato está entregue).
- **Gate:** verde ⇒ Fase 3 pode começar. Se front demorar a consumir, sem problema — nada quebra.

---

## 6. Fase 3 — CFG na Tela 3

**Escopo:** implementação integral do `PLANO-FLOW-CFG.md`. **Breaking change no JSON da Tela 3.** Precisa release coordenado com front e com o extrator v2 (Fase 3.5, executada em paralelo).

### 6.1. Ações

Execução conforme `PLANO-FLOW-CFG.md` §3 (Passos 3.1-3.7). Resumo:

1. Domain: `EndpointFlow`, `FlowNode`, `FlowEdge` refletindo grafo.
2. `cypherEndpointFlow` e `cypherExpandFlow` reescritas para retornar `{nodes, edges}`.
3. `Reader.LoadEndpointFlow` / `Reader.ExpandFlow` populam o novo shape; `buildStepChildren` removido.
4. `endpointFlowView` emite `{entryURN, exitURN, nodes[], edges[]}`.
5. **Ajuste crítico coordenado com fase futura**: `buildExpansionSlot` continua lendo `Call*.TargetURN` como prop (compat com extrator atual). Marcar `// TODO: Fase 5 — migrar para edge EXPANDS_TO`.

### 6.2. Testes

- Suite completa do `PLANO-FLOW-CFG.md` §3.7.
- Integração exercita fluxo linear, if+join, if terminando em return (sem join, terminals→exit), loop com back-edge, EXPANDS_TO externo, FLOWS_TO com props.

### 6.3. Checkpoint de avaliação

- [ ] `GET /endpoints/{urn}/flow` devolve novo shape contra a fixture (que já tem CFG).
- [ ] Front atualizado para consumir grafo (nodes + edges) — coordenado no mesmo release.
- [ ] Extrator v2 (Fase 3.5) já está pronto ou em cutover.
- **Gate crítico:** este é o **primeiro release quebrante**. Confirmar com front e com dono do extrator antes de merge.

---

## 7. Fase 3.5 — Extrator v2 (paralelo à Fase 3, fora deste repo)

**Escopo:** atualizar o extrator TypeScript (ou qualquer outro) para emitir tudo que as fases seguintes vão consumir. Executado em paralelo com Fase 3, entregue no mesmo release.

### 7.1. Ações

O extrator passa a emitir, além do que já emite:

- CFG completo (regras da §3.3 do PLANO-FLOW-CFG): `EntryNode`, `ExitNode` único, `JoinNode` condicional, `NEXT`, `BRANCH` com labels, `NEXT {backEdge:true}` em loops, terminals → `NEXT` → `ExitNode`.
- `HANDLED_BY` do Endpoint para o handler (Function/Method).
- `MEMBER_OF` do Method para o Owner (Struct/Interface).
- `HAS_FIELD` do Struct para o Type do campo, com prop `name`.
- Middlewares na cadeia `NEXT` do Endpoint (mantendo `PROTECTS`).
- `FLOWS_TO` best-effort onde conseguir inferir estaticamente.

Ainda **mantém** as props que serão removidas na Fase 5 (`Call*.TargetURN`, `Endpoint.HandlerURN`, `Method.OwnerTypeURN`, `Struct.Methods`, `Struct.Fields`, etc). O loader continua aceitando ambos os formatos.

### 7.1.1. Regras de emissão do CFG (consolidadas do fix do extrator)

Correções aplicadas ao extrator TS antes de qualquer consumidor da Fase 3 codificar contra o novo shape. Estas regras substituem o modelo antigo (árvore CONTAINS com Block("body") intermediário e JoinNode indiscriminado):

- **Container executável direto**: `Endpoint`/`Method`/`Function` são pais diretos de `EntryNode`, steps e `ExitNode` via `CONTAINS`. Sem wrapper `Block(name:'body')` — o nome sugeria HTTP body e a camada só duplicava CONTAINS.
- **Terminals (`ReturnNode`/`ThrowNode`) encerram o chain**: emitem uma única `NEXT` (marcada `isImplicit:true`) direto para o `ExitNode` do container executável mais próximo. Nada de NEXT saindo depois disso — statements sequenciais pós-terminal são unreachable (dead-code) e não são conectados.
- **`IfNode` com `else` explícito**: NUNCA cria `NEXT` linear do IfNode para o próximo sibling. Convergência é feita por `JoinNode` que recebe os tails vivos das branches. Se ambas branches terminam (Return/Throw em ambas), chain principal também termina — próximos statements ficam dead-code.
- **`IfNode` sem `else` explícito**: se o `then` sobreviveu, emite `JoinNode` com duas entradas — (a) fallthrough do próprio `IfNode` (edge `NEXT {isImplicit:true, label:'else'}`) e (b) tail vivo do then. Se o `then` terminou, NÃO emite Join — o `IfNode` fica como prev do chain e o próximo sibling recebe `NEXT` direto (interpretado como else implícito no fluxo pai).
- **`JoinNode` só existe quando há convergência real**: nunca emitido para "if sem else e sem tail sobrevivente".
- **Exceção documentada à sibling-invariant do NEXT**: normalmente NEXT não atravessa fronteira de CONTAINS. As duas exceções canônicas são `terminal→ExitNode` (do interior de qualquer branch para o Exit do container executável) e `branch-tail→JoinNode` (do interior de Block(then|else) para o Join sibling do IfNode).

Implicações práticas:

- A fixture da §4 (Fase 1) deve refletir exatamente esta topologia — em particular, os endpoints devem ter `Entry/steps/Exit` como filhos diretos do Endpoint (sem `Block(body)`), e cada `Return/Throw` deve ter `NEXT→Exit`.
- O Reader da Fase 3 (`cypherEndpointFlow`/`ExpandFlow`) não deve assumir nenhum wrapper entre Endpoint e Entry — traversal parte de `(ep)-[:CONTAINS]->(entry:EntryNode)` direto.
- `buildExpansionSlot` (Fase 5) continua lendo `Call*.TargetURN` até migrar para edge `EXPANDS_TO`. Nenhuma mudança nas props devido ao fix do CFG.

### 7.2. Checkpoint de avaliação

- [ ] Extrator novo produz payload que passa por `loader.LoadInto` sem `NodesSkipped > 0`.
- [ ] Um serviço real re-extraído gera fixture equivalente à canônica em cobertura de edges.
- **Gate:** só avança para Fase 4 quando existir pelo menos 1 serviço real com extração v2 completa.

---

## 8. Fase 4 — Linker rico + Tela 2

**Escopo:** implementação integral do `PLANO-ENDPOINT-EDGES.md`. **Breaking change no JSON da Tela 2.**

Depende de Fase 3.5 (edges como `HANDLED_BY` populadas) e da Fase 1 (edges declaradas).

### 8.1. Ações

Execução conforme `PLANO-ENDPOINT-EDGES.md` §4. Ajustes por conta do plano-mestre:

1. **Linker** (`infra/neo4j/linker.go`) — adicionar passo pós-linking que traversa `CONTAINS/EXPANDS_TO/CALLS` a partir de cada Endpoint e materializa `TOUCHES_DB`, `TOUCHES_BROKER`, `CALLS_SERVICE`, `READS_CONFIG`.
   - **Otimização aproveitando Fase 3**: com Entry/Exit emitidos, o linker pode delimitar o escopo do traversal usando `(entry)-[:NEXT|BRANCH*]->(exit)` — mas `CONTAINS*` também funciona e é mais simples. Manter `CONTAINS*` em v1.
2. **`domain/aggregates.go`** — novo `EndpointDetail` conforme §4.3 do plano.
3. **`cypherListEndpoints`** reescrita conforme §4.2.
4. **`Reader.ListEndpoints`** popula `EndpointDetail` completo.
5. **`endpointListView`** emite novo JSON.

### 8.2. Testes

- Suite completa do `PLANO-ENDPOINT-EDGES.md` §4.6.
- Integração do linker: rodar sobre fixture e verificar que edges agregadas batem com o subgrafo (weight, ops, topics deduplicados).
- Idempotência: rodar linker 2x → contagem de edges igual.

### 8.3. Checkpoint de avaliação

- [ ] `GET /services/{urn}/endpoints` devolve `endpoints[].node`, `.handler`, `.middlewares`, `.errors`, `.configs`, `.touchesDb`, `.touchesBroker`, `.callsService`.
- [ ] Front atualizado (coordenado).
- [ ] Linker roda em <30s em fixture de 100 endpoints (benchmark simples).
- **Gate:** verde ⇒ pode iniciar Fase 5. Se houver serviço grande onde o linker é lento, priorizar otimização antes.

---

## 9. Fase 5 — Limpeza de props duplicadas

**Escopo:** execução do `ANALISE-METADADOS-NODES.md` §1-2 (props que duplicam edges + tech strings duplicadas). **Breaking change no JSON de nodes** (`nodeView` deixa de emitir `handlerURN`, `targetURN`, `ownerTypeURN`, `framework` em Endpoint, `language` em nodes de código).

Só executável após Fase 3.5 — extrator já emite as edges equivalentes; grafos novos não têm as props.

### 9.1. Ações

1. **Migrar `buildExpansionSlot`** (`infra/neo4j/reader.go:303-334`) para ler edge `EXPANDS_TO`/`CALLS` do resultado da query em vez de props `Call*.TargetURN`. Query já retorna `EXPANDS_TO` como edge (Fase 3) — usar dela.
2. **Backfill script** (`schema/migrations/YYYY-props-to-edges.cypher`):
   ```
   // handler
   MATCH (e:Endpoint) WHERE e.handlerURN IS NOT NULL
   MATCH (h {urn: e.handlerURN})
   MERGE (e)-[:HANDLED_BY]->(h)
   REMOVE e.handlerURN;

   // method owner
   MATCH (m:Method) WHERE m.ownerTypeURN IS NOT NULL
   MATCH (o {urn: m.ownerTypeURN})
   MERGE (m)-[:MEMBER_OF]->(o)
   REMOVE m.ownerTypeURN;

   // struct methods / interface methods → apenas remover (inversa de MEMBER_OF)
   MATCH (s:Struct) REMOVE s.methods;
   MATCH (i:Interface) REMOVE i.methods;

   // struct fields → HAS_FIELD
   // (mais elaborado: depende de como fields estão persistidos hoje)

   // call targets
   MATCH (c:CallFunction) WHERE c.targetURN IS NOT NULL
   MATCH (t {urn: c.targetURN})
   MERGE (c)-[:CALLS]->(t)
   MERGE (c)-[:EXPANDS_TO]->(t)
   REMOVE c.targetURN;
   // idem CallHTTP (resolvido), CallDB, PublishEvent, ConsumeEvent

   // tech strings (só depois de garantir edges USES/WRITTEN_IN)
   MATCH (s:Service) REMOVE s.framework, s.runtime;
   MATCH (n:Node) WHERE NOT n:Language AND NOT n:Framework AND NOT n:Runtime
   REMOVE n.language;
   MATCH (ep:Endpoint) REMOVE ep.framework;
   ```
3. **Domain** — remover campos das structs em `domain/kinds.go`: `Endpoint.HandlerURN`, `Method.OwnerTypeURN`, `Struct.Methods`, `Struct.Fields`, `Interface.Methods`, `CallFunction.TargetURN`, `CallHTTP.TargetURN`, `CallDB.TargetURN`, `PublishEvent.TargetURN`, `ConsumeEvent.TargetURN`, `Service.Framework`, `Service.Runtime`, `Endpoint.Framework`, `BaseNode.LanguageValue`.
4. **Encoders/decoders** — atualizar para não escrever/ler essas props.
5. **`view.go`** — remover campos correspondentes de `nodeView`.
6. **Loader** — se payload antigo trouxer essas props, silenciosamente ignora (compat leve).
7. **Fixture** — reescrever removendo props duplicadas.

### 9.2. Testes

- Reader integrado: nenhum serviço perdido; edges `HANDLED_BY` populadas em vez de `handlerURN` prop.
- View: `nodeView(*Endpoint)` não tem chave `handlerURN`.
- Snapshot de tamanho médio de node no Neo4j — redução esperada.

### 9.3. Checkpoint de avaliação

- [ ] Backfill executado em ambientes de teste e prod sem perda de informação.
- [ ] Nenhum front lê chaves removidas do wire (auditar chamadas conhecidas).
- [ ] Tamanho médio de node reduziu no benchmark.
- **Gate final:** verde ⇒ plano-mestre concluído.

---

## 10. Fora do escopo v1 (parking lot)

Registrados aqui para não perder:

- **Kinds `ParamSlot` / `ReturnSlot`** (PLANO-FLOW-CFG §2.7 v2) — habilita queries "quem consome param X de Y?".
- **`Extraction` node** (ANALISE §4 opção 2) — só se surgir requisito de comparar snapshots históricos.
- **`Config.Sensitive` → classificação cross-serviço** (ANALISE §5) — só se requisito LGPD/PCI aparecer.
- **`ErrorType.HTTPStatus` como prop da edge `HANDLES_ERROR`** (ANALISE §5) — só quando o mesmo error mapear para status diferente por endpoint.
- **Análise de reachability** (dead code, unreachable branches) — habilitada pelo CFG mas não é view.
- **Métricas derivadas** (cyclomatic complexity, path count) — idem.
- **Modelar tópico como nó** (`PLANO-SERVICEMAP-EDGES.md` §6) — quando fizer sentido diferenciar tópicos.

---

## 11. Matriz de acompanhamento

Uma linha por fase. Marca-se `x` na coluna correspondente.

| Fase | Aditivo | Breaking wire | Precisa extrator v2 | Precisa front | Blocked-by |
|---|:-:|:-:|:-:|:-:|:-:|
| 0 — Infra base | x |  |  |  | — |
| 1 — Edges + fixture | x |  |  |  | 0 |
| 2 — Tela 1 | x |  |  | opcional | 1 |
| 3 — CFG Tela 3 |  | x | x | x | 1 |
| 3.5 — Extrator v2 | x |  | — |  | 1 |
| 4 — Tela 2 |  | x | x | x | 1, 3.5 |
| 5 — Limpeza props |  | x | x | auditar | 3, 3.5 |

---

## 12. Regras gerais de execução

1. **Cada fase é um PR (ou séquência curta de PRs).** Fase 3, 3.5 e 4 podem exigir múltiplos PRs mas devem ser mergeadas conjuntamente para não deixar wire quebrado em produção.
2. **Checkpoint entre fases não é ritual — é gate real.** Se o critério falhar, corrigir antes de avançar.
3. **Grafos existentes continuam legíveis em todas as fases até Fase 5.** Fase 5 exige backfill.
4. **`isKnownEdgeType` do loader deve ser atualizada junto com cada nova edge** — evita skip silencioso.
5. **Testes de integração usam a fixture canônica.** Se a fase acrescenta comportamento, a fixture cresce junto.
6. **Documentação dos planos originais** (`PLANO-SERVICEMAP-EDGES.md`, `PLANO-ENDPOINT-EDGES.md`, `PLANO-FLOW-CFG.md`, `ANALISE-METADADOS-NODES.md`) permanece como spec detalhada — este documento coordena, não substitui.

---

## 13. Definition of Done (plano-mestre)

- [ ] Fases 0-5 concluídas conforme critérios de cada §.
- [ ] Extrator v2 em produção emitindo o vocabulário completo.
- [ ] Nenhuma prop URN que duplique edge persiste no Neo4j.
- [ ] Front consome os três novos shapes (Tela 1/2/3) sem fallback a versão antiga.
- [ ] Fixture canônica documenta todo o vocabulário v1.
- [ ] Nenhum plano deste projeto em estado "parcialmente implementado" — cada fase encerrada.
