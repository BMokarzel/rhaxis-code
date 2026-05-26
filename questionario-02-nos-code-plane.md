# Questionário 02 — Nós faltantes no plano de código

Nós declarados em `NodeType` mas sem struct, e nós novos que viabilizam fluxo/análise.

---

## 2.1 `Call` (CRÍTICO — referenciado em todo o ADR-007)

**Contexto:** `CallNode` existe na enum, struct não. Sem ele, `Invokes`/`Targets` ficam sem origem/destino válido.

**Pergunta:** Quais campos `Call` deve carregar?

Sugestão mínima:

```go
type CallKind string
const (
  CallDirect     CallKind = "direct"     // f(x)
  CallMethod     CallKind = "method"     // obj.f(x)
  CallDynamic    CallKind = "dynamic"    // reflexão, ponteiro p/ func
  CallHttp       CallKind = "http"       // cliente HTTP
  CallDB         CallKind = "db"         // query/exec
  CallQueue      CallKind = "queue"      // publish/consume
  CallRPC        CallKind = "rpc"        // grpc, etc.
  CallExternal   CallKind = "external"   // syscall, FFI
)

type Call struct {
  Node
  Kind     CallKind
  Caller   URN              // Function que invoca
  Callee   URN              // Function/Endpoint resolvido (opcional)
  Receiver URN              // para método: o tipo do receiver
  Ordinal  int              // ordem dentro do Caller
  Args     []URN            // Variable/Parameter/Literal
  Location Location
}
```

- [ ] Aceita sugestão completa.
- [ ] Aceita mas remova: \_\_\_\_
- [ ] Adicione também: \_\_\_\_

**Pergunta extra:** `Args` deve ser URN (referência a nó Variable/Parameter) ou estrutura inline com tipo? Tem implicações: URN exige criar nós para literais.

- [ ] URN sempre (cria Variable/Literal para tudo).
- [ ] URN só quando há nó; literais ficam em `Meta.Properties`.
- [ ] Estrutura inline `{Name, TypeURN, Literal string}`.

**Resposta:** Aceito a primeira pergunta, mas nao use KIND, use Type. URN para nós, mas o args talvez deva ser um map[string]urn para que os parametros sejam nomeados, nao?

---

## 2.2 `Variable`

**Contexto:** `VariableNode` existe sem struct. Cobre const, var, package-level, local, parâmetro?

**Pergunta:** Granularidade?

```go
type VarKind string
const (
  VarConst    VarKind = "const"
  VarPackage  VarKind = "package"   // var no nível do pacote
  VarLocal    VarKind = "local"     // dentro de função
  VarField    VarKind = "field"     // campo de struct (separado de FieldSlot?)
  VarParam    VarKind = "param"     // parâmetro de função
)

type Variable struct {
  Node
  Kind     VarKind
  TypeURN  URN
  Owner    URN       // Module|Function|Type (escopo)
  Mutable  bool
  Location Location
}
```

- [ ] OK.
- [ ] Locals não devem ser nós (alto volume, baixo valor) — só const/package/field.
- [ ] Outro corte: \_\_\_\_

**Pergunta extra:** `FieldSlot` (em `Data`) e `Variable` com `Kind=field` se sobrepõem. Unificar?

- [ ] Sim, remover `FieldSlot`, usar `Variable`.
- [ ] Manter ambos (FieldSlot é embarcado, Variable é nó independente).

**Resposta:** ?

---

## 2.3 `Schema`

**Contexto:** `SchemaNode` na enum, referenciado em `SerializesAs`, `Imports`, `Targets`. Sem struct.

**Pergunta:** Sugestão:

```go
type SchemaFormat string
const (
  SchemaProto    SchemaFormat = "protobuf"
  SchemaOpenAPI  SchemaFormat = "openapi"
  SchemaJSON     SchemaFormat = "jsonschema"
  SchemaAvro     SchemaFormat = "avro"
  SchemaGraphQL  SchemaFormat = "graphql"
)

type Schema struct {
  Node
  Format    SchemaFormat
  Version   string
  Path      string   // arquivo de origem
  Namespace string
}
```

- [ ] OK.
- [ ] Mudanças: \_\_\_\_

**Resposta:** ?

---

## 2.4 `Parameter` como nó autônomo

**Contexto:** Hoje `FunctionParam` é slot embarcado em `Function`. Não dá para ligar fluxo (READS/WRITES) a um parâmetro específico.

**Pergunta:** Promover parâmetros a nós?

- [ ] Sim — reuso da estrutura `Variable` com `Kind=param`.
- [ ] Sim — tipo dedicado `Parameter`.
- [ ] Não — manter como slot.

**Resposta:** ?

---

## 2.5 `Test` / `TestCase`

**Contexto:** Testes não são cidadãos do grafo. Sem isso, não dá para responder "qual cobertura desta função?".

**Pergunta:** Adicionar nó `Test`?

```go
type TestKind string
const (
  TestUnit        TestKind = "unit"
  TestIntegration TestKind = "integration"
  TestE2E         TestKind = "e2e"
  TestBenchmark   TestKind = "benchmark"
)

type Test struct {
  Node
  Kind     TestKind
  Targets  []URN     // Function|Endpoint sob teste
  Location Location
  Skip     bool
}
```

E aresta `TESTS` (Test → Function/Endpoint) com `Coverage float32` em `Properties`.

- [ ] Sim.
- [ ] Sim mas mudanças: \_\_\_\_
- [ ] Adiar.

**Resposta:** ?

---

## 2.6 `Block` / `Statement` (granularidade para CFG)

**Contexto:** Sem isso, `Control` não tem como apontar "branches para qual bloco".

**Pergunta:** Nível desejado?

- [ ] Granularidade de **bloco** (sequência de statements sem desvio interno).
- [ ] Granularidade de **statement** (cada linha relevante).
- [ ] Não modelar — `Control` carrega tudo via Properties.
- [ ] Modelar só quando há necessidade (lazy).

Lembrar: granularidade fina explode volume; bloco é o usual em CFG.

**Resposta:** ?

---

## 2.7 `Annotation` / `Decorator`

**Contexto:** Decorators Python, anotações Java, atributos C#, tags Go struct, JSDoc — relevantes para IA entender semântica.

**Pergunta:** Modelar como nó?

- [ ] Sim, nó `Annotation` + aresta `ANNOTATES`.
- [ ] Não, guardar em `Meta.Properties["annotations"]`.

**Resposta:** ?

---

## 2.8 `Literal`

**Contexto:** Valores literais (strings, números) que aparecem em chamadas — relevante para detectar SQL injection, URLs hardcoded, secrets.

**Pergunta:** Nó próprio?

- [ ] Sim, `Literal{Kind, Value, Location}`.
- [ ] Não, inline em `Call.Args` ou Properties.

**Resposta:** ?
