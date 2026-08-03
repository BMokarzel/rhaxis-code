// Fixture: dois serviços minúsculos para validar as 4 queries do Reader.
//
//   orders-api  ──HTTP──►  users-api  (dependência cross-service)
//   orders-api  ──DB─────►  postgres
//
// orders-api tem 2 endpoints:
//   GET  /orders/:id   — flat, sem branch
//   POST /orders       — middleware → if(user.isAdmin) → callDB / callHTTP
//                        → serialize
//
// Idempotente: MERGE por urn. Rodar quantas vezes quiser.

// ---------- SERVICES ----------
MERGE (orders:Node:Service {urn: 'urn:cg:orders-api:_:service'})
  SET orders.kind='Service', orders.name='orders-api', orders.language='ts',
      orders.resolved=true, orders.repoURL='git@example/orders-api.git',
      orders.framework='nestjs', orders.runtime='node20',
      orders.lastExtractedAt=datetime(), orders.sourceRev='deadbeef01';

MERGE (users:Node:Service {urn: 'urn:cg:users-api:_:service'})
  SET users.kind='Service', users.name='users-api', users.language='ts',
      users.resolved=true, users.framework='nestjs',
      users.lastExtractedAt=datetime();

// ---------- EXTERNALS ----------
MERGE (pg:Node:Database {urn: 'urn:cg:orders-api:_:database:main'})
  SET pg.kind='Database', pg.name='main-db', pg.engine='postgres',
      pg.serviceURN='urn:cg:orders-api:_:service', pg.resolved=true;

MATCH (orders:Service {urn:'urn:cg:orders-api:_:service'}),
      (pg:Database {urn:'urn:cg:orders-api:_:database:main'})
MERGE (orders)-[:USES_DB]->(pg);

// ---------- ENDPOINTS ----------
MERGE (ep1:Node:Endpoint {urn:'urn:cg:orders-api:ts:endpoint:GET /orders/{id}'})
  SET ep1.kind='Endpoint', ep1.name='getOrder', ep1.language='ts',
      ep1.serviceURN='urn:cg:orders-api:_:service', ep1.resolved=true,
      ep1.httpMethod='GET', ep1.pathTemplate='/orders/:id',
      ep1.framework='nestjs',
      ep1.handlerURN='urn:cg:orders-api:ts:method:src/orders.controller.ts#OrdersController.get';

MERGE (ep2:Node:Endpoint {urn:'urn:cg:orders-api:ts:endpoint:POST /orders'})
  SET ep2.kind='Endpoint', ep2.name='createOrder', ep2.language='ts',
      ep2.serviceURN='urn:cg:orders-api:_:service', ep2.resolved=true,
      ep2.httpMethod='POST', ep2.pathTemplate='/orders',
      ep2.framework='nestjs',
      ep2.handlerURN='urn:cg:orders-api:ts:method:src/orders.controller.ts#OrdersController.create';

MATCH (s:Service {urn:'urn:cg:orders-api:_:service'}),
      (ep1:Endpoint {urn:'urn:cg:orders-api:ts:endpoint:GET /orders/{id}'}),
      (ep2:Endpoint {urn:'urn:cg:orders-api:ts:endpoint:POST /orders'})
MERGE (s)-[:EXPOSES]->(ep1)
MERGE (s)-[:EXPOSES]->(ep2)
MERGE (s)-[:OWNS]->(ep1)
MERGE (s)-[:OWNS]->(ep2);

// ---------- FUNCTIONS ----------
MERGE (fnAuth:Node:Function {urn:'urn:cg:orders-api:ts:function:src/middleware/auth.ts#authMiddleware'})
  SET fnAuth.kind='Function', fnAuth.name='authMiddleware',
      fnAuth.language='ts', fnAuth.serviceURN='urn:cg:orders-api:_:service',
      fnAuth.resolved=true, fnAuth.signature='(req,res,next) => void',
      fnAuth.isAsync=false;

MERGE (fnFbd:Node:Function {urn:'urn:cg:orders-api:ts:function:src/utils/forbidden.ts#forbidden'})
  SET fnFbd.kind='Function', fnFbd.name='forbidden',
      fnFbd.language='ts', fnFbd.serviceURN='urn:cg:orders-api:_:service',
      fnFbd.resolved=true, fnFbd.signature='(res) => void', fnFbd.isAsync=false;

MERGE (fnSer:Node:Function {urn:'urn:cg:orders-api:ts:function:src/utils/serialize.ts#serialize'})
  SET fnSer.kind='Function', fnSer.name='serialize',
      fnSer.language='ts', fnSer.serviceURN='urn:cg:orders-api:_:service',
      fnSer.resolved=true, fnSer.signature='(x) => JSON', fnSer.isAsync=false;

// authMiddleware possui corpo simples (só para ExpandFlow ter algo):
MERGE (fnAuthReturn:Node:Block {urn:'urn:cg:orders-api:ts:block:authMiddleware/body'})
  SET fnAuthReturn.kind='Block', fnAuthReturn.name='body',
      fnAuthReturn.language='ts', fnAuthReturn.resolved=true,
      fnAuthReturn.serviceURN='urn:cg:orders-api:_:service';

MATCH (f:Function {urn:'urn:cg:orders-api:ts:function:src/middleware/auth.ts#authMiddleware'}),
      (b:Block {urn:'urn:cg:orders-api:ts:block:authMiddleware/body'})
MERGE (f)-[:CONTAINS {index:0}]->(b);

// ---------- FLOW: POST /orders ----------
// step 0: chamada de middleware
MERGE (c0:Node:CallFunction {urn:'urn:cg:orders-api:ts:callFunction:endpoints/post-orders#0-auth'})
  SET c0.kind='CallFunction', c0.name='auth()', c0.language='ts',
      c0.serviceURN='urn:cg:orders-api:_:service', c0.resolved=true,
      c0.targetURN='urn:cg:orders-api:ts:function:src/middleware/auth.ts#authMiddleware';

MATCH (c0:CallFunction {urn:'urn:cg:orders-api:ts:callFunction:endpoints/post-orders#0-auth'}),
      (fn:Function {urn:'urn:cg:orders-api:ts:function:src/middleware/auth.ts#authMiddleware'})
MERGE (c0)-[:CALLS]->(fn);

// step 1: IF
MERGE (ifN:Node:IfNode {urn:'urn:cg:orders-api:ts:ifNode:endpoints/post-orders#1'})
  SET ifN.kind='IfNode', ifN.name='if user.isAdmin', ifN.language='ts',
      ifN.serviceURN='urn:cg:orders-api:_:service', ifN.resolved=true,
      ifN.conditionText='user.isAdmin';

// step 2: chamada de serialize
MERGE (c2:Node:CallFunction {urn:'urn:cg:orders-api:ts:callFunction:endpoints/post-orders#2-serialize'})
  SET c2.kind='CallFunction', c2.name='serialize()', c2.language='ts',
      c2.serviceURN='urn:cg:orders-api:_:service', c2.resolved=true,
      c2.targetURN='urn:cg:orders-api:ts:function:src/utils/serialize.ts#serialize';

MATCH (c2:CallFunction {urn:'urn:cg:orders-api:ts:callFunction:endpoints/post-orders#2-serialize'}),
      (fn:Function {urn:'urn:cg:orders-api:ts:function:src/utils/serialize.ts#serialize'})
MERGE (c2)-[:CALLS]->(fn);

// liga POST /orders → steps
MATCH (ep:Endpoint {urn:'urn:cg:orders-api:ts:endpoint:POST /orders'}),
      (c0:CallFunction {urn:'urn:cg:orders-api:ts:callFunction:endpoints/post-orders#0-auth'}),
      (ifN:IfNode {urn:'urn:cg:orders-api:ts:ifNode:endpoints/post-orders#1'}),
      (c2:CallFunction {urn:'urn:cg:orders-api:ts:callFunction:endpoints/post-orders#2-serialize'})
MERGE (ep)-[:CONTAINS {index:0}]->(c0)
MERGE (ep)-[:CONTAINS {index:1}]->(ifN)
MERGE (ep)-[:CONTAINS {index:2}]->(c2);

// ---------- Branches do IF ----------
// then: [callDB(insert), callHTTP(users-api /notify)]
MERGE (bkT:Node:Block {urn:'urn:cg:orders-api:ts:block:endpoints/post-orders#1/then'})
  SET bkT.kind='Block', bkT.name='then', bkT.language='ts',
      bkT.serviceURN='urn:cg:orders-api:_:service', bkT.resolved=true;

MERGE (cdb:Node:CallDB {urn:'urn:cg:orders-api:ts:callDB:endpoints/post-orders#1/then/0'})
  SET cdb.kind='CallDB', cdb.name='insert order', cdb.language='ts',
      cdb.serviceURN='urn:cg:orders-api:_:service', cdb.resolved=true,
      cdb.operation='insert',
      cdb.targetURN='urn:cg:orders-api:_:database:main';

MERGE (chttp:Node:CallHTTP {urn:'urn:cg:orders-api:ts:callHTTP:endpoints/post-orders#1/then/1'})
  SET chttp.kind='CallHTTP', chttp.name='POST /notify', chttp.language='ts',
      chttp.serviceURN='urn:cg:orders-api:_:service',
      chttp.resolved=false,                              // ainda não linkado
      chttp.httpMethod='POST', chttp.pathTemplate='/notify',
      chttp.targetHint='USERS_API_URL';

MATCH (ifN:IfNode {urn:'urn:cg:orders-api:ts:ifNode:endpoints/post-orders#1'}),
      (bkT:Block {urn:'urn:cg:orders-api:ts:block:endpoints/post-orders#1/then'}),
      (cdb:CallDB {urn:'urn:cg:orders-api:ts:callDB:endpoints/post-orders#1/then/0'}),
      (chttp:CallHTTP {urn:'urn:cg:orders-api:ts:callHTTP:endpoints/post-orders#1/then/1'}),
      (pg:Database {urn:'urn:cg:orders-api:_:database:main'})
MERGE (ifN)-[:BRANCH {label:'then'}]->(bkT)
MERGE (bkT)-[:CONTAINS {index:0}]->(cdb)
MERGE (bkT)-[:CONTAINS {index:1}]->(chttp)
MERGE (cdb)-[:CALLS]->(pg);

// else: [forbidden()]
MERGE (bkE:Node:Block {urn:'urn:cg:orders-api:ts:block:endpoints/post-orders#1/else'})
  SET bkE.kind='Block', bkE.name='else', bkE.language='ts',
      bkE.serviceURN='urn:cg:orders-api:_:service', bkE.resolved=true;

MERGE (cfb:Node:CallFunction {urn:'urn:cg:orders-api:ts:callFunction:endpoints/post-orders#1/else/0'})
  SET cfb.kind='CallFunction', cfb.name='forbidden()', cfb.language='ts',
      cfb.serviceURN='urn:cg:orders-api:_:service', cfb.resolved=true,
      cfb.targetURN='urn:cg:orders-api:ts:function:src/utils/forbidden.ts#forbidden';

MATCH (ifN:IfNode {urn:'urn:cg:orders-api:ts:ifNode:endpoints/post-orders#1'}),
      (bkE:Block {urn:'urn:cg:orders-api:ts:block:endpoints/post-orders#1/else'}),
      (cfb:CallFunction {urn:'urn:cg:orders-api:ts:callFunction:endpoints/post-orders#1/else/0'}),
      (fn:Function {urn:'urn:cg:orders-api:ts:function:src/utils/forbidden.ts#forbidden'})
MERGE (ifN)-[:BRANCH {label:'else'}]->(bkE)
MERGE (bkE)-[:CONTAINS {index:0}]->(cfb)
MERGE (cfb)-[:CALLS]->(fn);

// ---------- DEPENDS_ON agregada (materializada pelo linker) ----------
MATCH (a:Service {urn:'urn:cg:orders-api:_:service'}),
      (b:Service {urn:'urn:cg:users-api:_:service'})
MERGE (a)-[d:DEPENDS_ON]->(b)
  SET d.via='http', d.weight=1;
