# Questionário 04 — Arestas faltantes e modelagem de fluxo

Dois grupos: (a) arestas com constante mas sem struct concreto; (b) arestas novas para CFG/data-flow.

---

## 4.1 Arestas declaradas sem struct concreto

`TypeDeployedOn`, `TypeRoutes`, `TypePeers`, `TypeMemberOf`, `TypePartOf`, `TypeReportsTo`, `TypeOwns` — só constante.

**Pergunta:** Política para casos sem campos extras?
- [ ] Criar struct vazio `type X struct{ Base }` por consistência (mais fácil para o type system).
- [ ] Só criar quando há campo extra; usar `Base` direto com `EdgeType` setado para os demais.
- [ ] Híbrido: criar para os "fortes" (Owns, DeployedOn, Routes) e deixar os triviais sem.

**Pergunta extra para `Owns`:** ADR-010 diz cardinalidade bifurcada (Team vs Person). Carregar isso no struct?
```go
type OwnsKind string
const (
  OwnsTeam   OwnsKind = "team"
  OwnsPerson OwnsKind = "person"
)
type Owns struct {
  Base
  Kind OwnsKind
}
```
- [ ] Sim.
- [ ] Inferir do tipo do `From` em runtime, não duplicar.

**Resposta:** ?

---

## 4.2 Arestas de controle de fluxo (CFG)

**Contexto:** Hoje `Control` e `Sequence` mal cobrem isso. Sem CFG explícito, IA não consegue raciocinar sobre execução.

**Pergunta:** Adicionar as seguintes?
- [ ] `NEXT` (Statement|Block|Call → próximo) — sequência linear.
- [ ] `BRANCHES_TO` (Control → Block|Statement) com `Outcome` (`"true"|"false"|"case:X"|"default"|"catch:Type"|"finally"`) e `Condition` (expr textual).
- [ ] `LOOPS_BACK` (Block → Control) — back-edge para detectar ciclos.
- [ ] `RETURNS` (Function → Type|Variable) — distinto de `Return []Data` que já existe (este é o caminho).
- [ ] `THROWS` / `RAISES` (Function → Type) — exceções que podem propagar.
- [ ] `BREAKS_TO` / `CONTINUES_TO` (jump statements).

**Pergunta extra:** Granularidade — se 2.6 escolheu "sem Block", essas arestas precisam adaptar (`NEXT` entre Calls dentro de uma Function).

**Resposta:** ?

---

## 4.3 Arestas de data flow

**Pergunta:** Adicionar?
- [ ] `READS` (Function|Call → Variable|Field) — leitura.
- [ ] `WRITES` (Function|Call → Variable|Field) — escrita.
- [ ] `MUTATES` (Function → Type) — modifica estrutura passada (efeito colateral).
- [ ] `CAPTURES` (Function/Closure → Variable) — closures sobre variável externa.
- [ ] `RETURNS_VALUE_OF` (Function → Variable|Literal) — proveniência do retorno.
- [ ] `PASSES_TO` (Variable → Call.Args[i]) — para tainting/security analysis.

**Pergunta extra:** Esse nível de detalhe é caro. Adotar **lazy**?
- [ ] Modelar sempre.
- [ ] Modelar só quando o coletor for explicitamente acionado (flag).
- [ ] Modelar só para funções "marcadas como sensíveis" (auth, crypto, sql).

**Resposta:** ?

---

## 4.4 Arestas de símbolos/módulos

**Pergunta:** Adicionar?
- [ ] `EXPORTS` (Module → Function|Type|Variable) — superfície pública.
- [ ] `IMPORTS_SYMBOL` (Module → Function|Type) — distinto de `Uses` (Framework).
- [ ] `OVERRIDES` (Method → Method) — distinto de `Implements` (mesma classe vs interface).
- [ ] `SHADOWS` (Variable → Variable) — variável local oculta outra de escopo externo.

**Resposta:** ?

---

## 4.5 Arestas para testes / qualidade

**Pergunta:** Se Test (2.5) entrar:
- [ ] `TESTS` (Test → Function|Endpoint).
- [ ] `COVERS` com `Coverage float32` em `Properties` (Test → Function, redundante com TESTS?).
- [ ] `EXERCISES` (Test → Call) — detalhado.
- [ ] Só `TESTS`, resto em Properties.

**Resposta:** ?

---

## 4.6 Arestas para anotações/contratos

**Pergunta:** Adicionar?
- [ ] `ANNOTATES` (Annotation → Function|Type|Variable).
- [ ] `GENERATES` (Schema → Type, ou Generator → Artifact) — para código gerado.
- [ ] `VALIDATES` (Function → Type) — guards/validators.

**Resposta:** ?

---

## 4.7 Arestas para config/secrets

**Pergunta:** Adicionar?
- [ ] `CONFIGURED_BY` (Service|Function → ConfigKey|Secret).
- [ ] `READS_ENV` (Function → ConfigKey) — caso de uso comum.
- [ ] Não modelar — fica em Properties do Function.

**Resposta:** ?

---

## 4.8 Arestas pub/sub explícitas

**Pergunta:** Hoje cai em `Targets`. Vale especializar?
- [ ] `EMITS` (Function|Service → Topic).
- [ ] `CONSUMES` (Function|Service → Topic|Queue).
- [ ] Não — manter `Targets` genérico.

**Resposta:** ?

---

## 4.9 Enriquecimento de `Control`

**Contexto:** Hoje `Control` só tem `Type`. Comentários no código questionam "what about the no condition case?".

**Pergunta:** Adicionar?
```go
type Control struct {
  Node
  Type       ControlType
  Condition  string             // expressão textual ou URN
  Branches   map[string]URN     // "true","false","case:0","default","catch:IOError","finally"
  Location   Location
}
```
- [ ] Aceita.
- [ ] `Condition` deve ser URN (apontar para nó de expressão), não string.
- [ ] `Branches` desnecessário — usar aresta `BRANCHES_TO` (resposta de 4.2 já cobre).
- [ ] Outro: ____

**Pergunta extra:** Sub-tipos faltam? Cobertura atual: `If, Switch, Try, Catch, While, For`. Faltam:
- [ ] `DoWhile`?
- [ ] `Foreach` (distinto de `For` clássico)?
- [ ] `Finally`?
- [ ] `Goto`?
- [ ] `Select` (Go channels)?
- [ ] `Match` (Rust/Scala)?
- [ ] `Guard` (Swift)?
- [ ] `When` (Kotlin)?

**Resposta:** ?
