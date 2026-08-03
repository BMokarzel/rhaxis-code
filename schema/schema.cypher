// Constraints e índices do codegraph (Neo4j 5.x).
// Idempotente: pode ser reexecutado sem erro.

// Identidade global — constraint gera índice automático.
CREATE CONSTRAINT node_urn_unique IF NOT EXISTS
  FOR (n:Node) REQUIRE n.urn IS UNIQUE;

// Constraints por label — defesa em profundidade contra duplicatas
// silenciosas em MERGE(:Label {urn}). A constraint global :Node já cobre,
// mas as por-label garantem que o planner use o índice específico da
// label ao invés de scan pela constraint global, e evitam bugs em
// operações que só olham a label específica.
CREATE CONSTRAINT service_urn_unique IF NOT EXISTS
  FOR (n:Service) REQUIRE n.urn IS UNIQUE;

CREATE CONSTRAINT endpoint_urn_unique IF NOT EXISTS
  FOR (n:Endpoint) REQUIRE n.urn IS UNIQUE;

CREATE CONSTRAINT function_urn_unique IF NOT EXISTS
  FOR (n:Function) REQUIRE n.urn IS UNIQUE;

CREATE CONSTRAINT method_urn_unique IF NOT EXISTS
  FOR (n:Method) REQUIRE n.urn IS UNIQUE;

CREATE CONSTRAINT database_urn_unique IF NOT EXISTS
  FOR (n:Database) REQUIRE n.urn IS UNIQUE;

CREATE CONSTRAINT broker_urn_unique IF NOT EXISTS
  FOR (n:Broker) REQUIRE n.urn IS UNIQUE;

// Aceleram macro-view e resolução por serviço.
CREATE INDEX service_urn IF NOT EXISTS
  FOR (s:Service) ON (s.urn);

CREATE INDEX endpoint_by_service IF NOT EXISTS
  FOR (e:Endpoint) ON (e.serviceURN);

// Linker cross-service: busca por (método, path) em endpoints.
CREATE INDEX endpoint_route IF NOT EXISTS
  FOR (e:Endpoint) ON (e.httpMethod, e.pathTemplate);

// Fila de stubs pendentes de resolução.
CREATE INDEX unresolved_calls IF NOT EXISTS
  FOR (c:CallHTTP) ON (c.resolved);
