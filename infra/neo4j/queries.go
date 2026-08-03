package neo4j

// Constantes com as queries Cypher validadas em §6 do doc de design.
// Mantê-las agrupadas facilita PROFILE, revisão, e futuras extrações para
// arquivos .cypher versionados.

const cypherServiceMap = `
MATCH (s:Service)
OPTIONAL MATCH (s)-[:USES_DB]->(db:Database)
OPTIONAL MATCH (s)-[:PUBLISHES_TO|CONSUMES_FROM]->(b:Broker)
WITH collect(DISTINCT s) AS services,
     collect(DISTINCT db) AS dbs,
     collect(DISTINCT b)  AS brokers
OPTIONAL MATCH (a:Service)-[d:DEPENDS_ON]->(bt:Service)
RETURN services, dbs, brokers,
       collect({from: a.urn, to: bt.urn, via: d.via, weight: coalesce(d.weight, 0)}) AS deps
`

const cypherListEndpoints = `
MATCH (s:Service {urn: $serviceURN})
OPTIONAL MATCH (s)-[:EXPOSES]->(e:Endpoint)
RETURN s AS service, collect(e) AS endpoints
`

// cypherEndpointFlow: 1 nível de :CONTAINS + :BRANCH direto de cada child de
// controle de fluxo. Calls não são seguidos — o cliente pede via ExpandFlow.
//
// Design note: originalmente a query aninhava collect(DISTINCT ...) dentro de
// collect(DISTINCT ...) — Neo4j rejeita ("aggregate inside aggregate"). A
// forma corrente é ACHATAR os rows (uma linha por combinação child × branch ×
// branchChild) e AGRUPAR EM GO (buildStepChildren). Custo: um pouco mais de
// tráfego bruto por chamada, mas Neo4j-válido e o agrupamento em Go dedup
// naturalmente.
const cypherEndpointFlow = `
MATCH (e:Endpoint {urn: $endpointURN})
OPTIONAL MATCH (e)-[c:CONTAINS]->(child)
OPTIONAL MATCH (child)-[:CALLS|EXPANDS_TO]->(callTarget)
WITH e, child, c.index AS idx, callTarget
OPTIONAL MATCH (child)-[br:BRANCH]->(branchBlock:Block)
OPTIONAL MATCH (branchBlock)-[bc:CONTAINS]->(branchChild)
OPTIONAL MATCH (branchChild)-[:CALLS|EXPANDS_TO]->(bcCallTarget)
RETURN e AS endpoint,
       collect({
         child: child,
         idx: idx,
         callTarget: callTarget,
         branchLabel: br.label,
         branchBlock: branchBlock,
         branchChild: branchChild,
         branchIdx: bc.index,
         bcCallTarget: bcCallTarget
       }) AS rows
`

// cypherExpandFlow: mesmo formato mais enxuto — target + children diretos +
// eventual :CALLS de cada child. Sem branches profundos (só nível 1).
const cypherExpandFlow = `
MATCH (target {urn: $targetURN})
WHERE target:Function OR target:Method OR target:Endpoint OR target:Block OR target:ConsumeEvent
OPTIONAL MATCH (target)-[c:CONTAINS]->(child)
OPTIONAL MATCH (child)-[:CALLS|EXPANDS_TO]->(callTarget)
RETURN target,
       collect({
         child: child,
         idx: c.index,
         callTarget: callTarget
       }) AS children
`

