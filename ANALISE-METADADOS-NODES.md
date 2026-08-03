# Análise — metadados de nodes candidatos a limpeza

**Contexto:** revisão dos metadados persistidos em `domain/kinds.go`, `domain/node.go` e `domain/encoders.go` à luz do modelo atual — que já tem nodes globais (`Language`, `Framework`, `Runtime`, `ErrorType`, `Config`) e um conjunto rico de arestas tipadas. Vários campos escalares nos nodes se tornaram redundantes ou mal-posicionados.

Este documento é **análise**, não plano de execução. Serve de insumo para um futuro plano de migração.

---

## 1. Redundâncias claras — props que são arestas disfarçadas

Toda prop que carrega um `URN` apontando para outro nó é conceitualmente uma edge. Duplicá-la como escalar cria dois caminhos de verdade, impede queries topológicas ("quem aponta pra X?") e obriga o extractor a manter consistência manual.

| Nó / campo | Deveria ser | Observação |
|---|---|---|
| `Endpoint.HandlerURN` | `(:Endpoint)-[:HANDLED_BY]->(:Function\|:Method)` | Não existe edge equivalente hoje — precisa ser criada |
| `CallFunction.TargetURN` | Já coberto por `EdgeCalls` + `EdgeExpandsTo` | Prop é duplicata literal — só remover |
| `CallHTTP.TargetURN` (`*URN`) | Coberto por `EXPANDS_TO`. `TargetHint` cobre pré-linker | Hoje há três formas de dizer "não linkado" |
| `CallDB.TargetURN` | `(:CallDB)-[:CALLS_DB]->(:Database)` (ou promover `USES_DB` para nível de Function) | Prop escalar apontando para Database |
| `PublishEvent.TargetURN` | Edge para o Broker | `PUBLISHES_TO` já existe no nível Service — replicar no Call é a mesma edge uma camada abaixo |
| `ConsumeEvent.TargetURN` | Edge para o Broker | Idem `CONSUMES_FROM` |
| `Method.OwnerTypeURN` | `(:Method)-[:MEMBER_OF]->(:Struct\|:Interface)` | Prop URN = edge por outro nome |
| `Struct.Methods []URN` | Inversa do `MEMBER_OF` acima — não persistir | Slice de URNs é o pior tipo de aresta: sem tipo, sem props, sem índice |
| `Interface.Methods []URN` | Idem | |
| `Struct.Fields []FieldSlot` (com `TypeURN`) | `(:Struct)-[:HAS_FIELD {name}]->(:Type)` | Achatar campos no Struct destrói a query "quem usa Type X?" — justamente o valor do knowledge map |

**Impacto:** alto. Esses campos são o que amarra o grafo — hoje estão escritos duas vezes, e queries topológicas dependem sempre da versão-edge, nunca da prop.

---

## 2. Redundâncias com nodes tech globais

O modelo já tem `Language`, `Framework`, `Runtime` como nodes globais com `WRITTEN_IN` / `USES`. As props escalares equivalentes são fósseis do modelo anterior:

| Campo | Duplicado por |
|---|---|
| `BaseNode.LanguageValue` | `(:Service)-[:WRITTEN_IN]->(:Language)` — pior: replicada em **todos** os nodes (control-flow inclusive) só herdando do Service dono |
| `Service.Framework` | `(:Service)-[:USES]->(:Framework)` |
| `Service.Runtime` | `(:Service)-[:USES]->(:Runtime)` |
| `Endpoint.Framework` | Endpoint pertence ao Service via OWNS — o framework é o do Service |

Se o extractor emite consistentemente as edges, essas strings podem sumir. Fonte única = node global. Vantagem adicional: filtragem cross-service ("todos os serviços que falam nestjs") deixa de ter dois caminhos.

---

## 3. Metadados derivados — não deveriam ser persistidos

`encoders.go:45-50` grava em **todo** node:

- `displayShape` — função pura do `Kind` (`displayShapeFor`)
- `displayCategory` — função pura do `Kind` (`displayCategoryFor`)

Isso não é metadado, é *cache do render*. Problemas concretos:

- Mudar a tabela obriga a reescrever N nodes no grafo
- Write cost e storage inflados sem contrapartida
- Frontend/view pode computar no read (ou o próprio Cypher via `CASE`)

**Recomendação:** mover para `cmd/rhaxis-api/view.go` (mesma tabela, calculada no momento do serialize) ou deixar o frontend mapear a partir do `kind`.

---

## 4. Metadados que pertencem a outro nó

`BaseNode.ExtractedAt` e `BaseNode.SourceRev` são replicados em cada node de uma mesma extração — carregam sempre o mesmo valor para todos os nodes de um Service numa dada rodada.

Duas alternativas, ambas melhores que o estado atual:

1. **Prop no Service** (`lastExtractedAt`, `sourceRev`) — simples, resolve 95% dos casos.
2. **Novo node `Extraction`** — `(:Extraction {rev, at})<-[:EXTRACTED_BY]-(:Node)` — só justificável se houver requisito de comparar snapshots históricos.

Se a resposta a "eu comparo extrações antigas?" for **não**, ir de opção 1.

---

## 5. Campos com utilidade discutível

| Campo | Comentário |
|---|---|
| `BaseNode.ResolvedValue` | Faz sentido em `Call*` e stubs. Em `Service/Database/Broker/Endpoint` é sempre `true`. Mover só para os kinds que podem estar não resolvidos (`CallHTTP`, `CallFunction`). |
| `Config.Sensitive` | Heurística por nome (documentada). OK como prop; se surgir necessidade de classificar cross-serviço (LGPD/PCI), promover a edge para `(:Classification)`. |
| `ErrorType.HTTPStatus` | Prop fixa da classe. OK — mas se o mesmo `ErrorType` mapear para status diferente por Endpoint, o status vira prop da edge `HANDLES_ERROR`, não do node. |
| `LoopNode.Kind_` / `Middleware.Kind_` / `Middleware.Phase` | Enum textual pequeno. OK como prop — não faz sentido virar nó. |
| `IfNode.ConditionText` / `Switch.Discriminant` / `Return.ExpressionText` / `Throw.ExpressionText` | Texto para display. Não candidato a edge — manter. |
| `BaseNode.Metadata map[string]any` (JSON string) | Escape hatch legítimo. Manter, mas monitorar: se sempre carrega os mesmos campos, promover a props tipadas. |

---

## 6. Resumo priorizado

Ordem sugerida por relação custo/benefício:

1. **Remover props que duplicam arestas** (§1). Maior impacto no knowledge map — viabiliza queries topológicas reais e elimina drift entre prop e edge.
   - `HandlerURN`, `TargetURN` em todas as `Call*`, `OwnerTypeURN`, `Struct.Methods`, `Interface.Methods`
   - Migrar `Struct.Fields` para edges `HAS_FIELD`

2. **Remover `displayShape` / `displayCategory` da persistência** (§3), calcular no view. Ganho imediato de storage e evita drift.

3. **Remover `Language`, `Framework`, `Runtime` como strings** (§2) — manter só via edges para nodes globais. Fonte única de verdade.

4. **Mover `ExtractedAt` / `SourceRev` para o Service** (§4). Deduplicação massiva.

5. **Mover `ResolvedValue` do `BaseNode` só para os kinds que realmente podem ser stubs** (§5). Limpeza cosmética.

---

## 7. O que **não** mudar

- `BaseNode.URNValue`, `BaseNode.KindValue`, `BaseNode.NameValue`, `BaseNode.ServiceURNValue` — identidade e agrupamento
- `BaseNode.Metadata` — escape hatch
- Todos os `*Text` de display (`ConditionText`, `Discriminant`, `ExpressionText`, `MessageTemplate`)
- Props enum pequenas (`Middleware.Kind_/Phase`, `LoopNode.Kind_`, `CallDB.Operation`, `Config.Category`, `LogNode.Level/Library`)
- `Service.RepoURL`, `Database.Engine`, `Broker.Engine` — atributos legítimos e escalares dos aggregators

---

## 8. Riscos a considerar na hora do plano

- **Queries existentes que leem as props escalares** precisam ser reescritas para percorrer as edges (mudança na Tela 1/2/3 e queries livres)
- **Extractor** precisa emitir edges consistentemente antes de remover as props — caso contrário perde-se informação
- **Backfill** dos grafos já persistidos: script `MATCH ... MERGE (edge) ... REMOVE n.prop`
- **Cross-service linker** hoje pode depender de `TargetURN` como prop — verificar `infra/neo4j/linker.go`
- **Testes de integração** com fixtures em `schema/fixtures/` provavelmente esperam as props — atualizar
