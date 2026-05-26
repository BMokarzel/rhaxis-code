# Questionário 01 — Rodada 2 (aprofundamento)

Só os itens onde a primeira resposta abriu nova decisão ou houve dúvida. Os demais (1.1, 1.2, 1.4, 1.5, 1.6, 1.7) ficaram fechados — vão pro plano de implementação.

---

## 1.3-R2 — `Sequence` como fluxo de execução

Sua resposta: _"O edge deve criar o fluxo de steps de funções e codigos em geral. Declara var x -> chama func y -> controle de fluxo, etc."_

Isso é exatamente o que a literatura chama de **Control Flow Graph (CFG)** — e cai diretamente no escopo do bloco 4 (perguntas 4.2/4.9). Para alinhar:

### 1.3.a — Fundir ou separar?

`Sequence` (uma aresta única que liga "o próximo passo") vs família de arestas especializadas (`NEXT`, `BRANCHES_TO`, `LOOPS_BACK`, `RETURNS`, `THROWS`).

- [ ] **Manter `Sequence` única**, com `Order int` e `Type` (`Basic|Situational`). Simples, mas perde semântica de bifurcações.
- [ ] **Substituir `Sequence` pela família CFG** (`NEXT` para sequência linear, `BRANCHES_TO` para if/switch, etc.). Mais expressivo, mais arestas.
- [ ] **Híbrido**: `Sequence` linear + `BRANCHES_TO` quando há condição.

**Resposta:** ?

### 1.3.b — O que `Situational` significava?

A enum `SequenceType` tem `Basic` e `Situational`. Pode descrever a intenção? Era para diferenciar:

- [ ] Caminho sempre executado (Basic) vs condicional (Situational)?
- [ ] Caminho síncrono vs assíncrono?
- [ ] Outro: \_\_\_\_

**Resposta:** ?

### 1.3.c — Heterogeneidade dos nós ligados

Você listou: `Variable → Call → Control → ...`. Esse fluxo cruza tipos heterogêneos. Confirma que a aresta de fluxo (seja `Sequence` ou `NEXT`) pode conectar **qualquer combinação** desses?

- [ ] Sim, qualquer "passo executável" pode ser origem/destino (Variable declaration, Call, Control, Block).
- [ ] Não — só Call ↔ Call e Control ↔ Call (Variable não é passo, é dado).
- [ ] Outro recorte: \_\_\_\_

**Resposta:** ?

> Obs.: a decisão aqui afeta diretamente 2.2 (Variable como nó precisa `Kind`?) e 2.6 (Block/Statement). Vou costurar depois.

---

## 1.8-R2 — `DataNode = "type"` (explicando melhor)

Em Go, a constante tem **dois lados**:

```go
DataNode NodeType = "type"
//   ^                ^
//   |                └── valor (string que aparece no JSON serializado)
//   └── nome do símbolo (usado no código Go)
```

- No código Go você escreve `node.Type == DataNode` (usa o nome).
- No JSON ou banco aparece `"type": "type"` (usa o valor).

Como o nome diz "Data" e o valor diz "type", os dois lados não combinam. Faz quem lê o JSON pensar que é um campo `type`, não um nó de tipo.

### 1.8.a — Decisão

- [ ] Renomear o símbolo: `TypeNode NodeType = "type"` (alinha com o valor; mais natural ao ler JSON).
- [ ] Mudar o valor: `DataNode NodeType = "data"` (alinha com o símbolo; menos colisão com a palavra reservada "type").
- [ ] Não mexer.

> Recomendação: a primeira. "Type" descreve melhor o que é (struct/class/interface). E o valor `"data"` choca com `DataType` (struct/class/enum...). A confusão "type" no JSON com palavra reservada não existe — é só uma string.

**Resposta:** ?

---

## 1.9-R2 — Tags JSON ≠ Edges

Pode ter havido confusão de termos. Esclarecimento rápido:

**Tag JSON** é metadado de **serialização do struct em Go**:

```go
type Service struct {
  Repo string `json:"repo"`   // ← isto é uma tag JSON
}
```

Controla o **nome do campo no JSON gerado/lido**. Sem tag, Go usa o nome do campo Go (`Repo` → `"Repo"`, mas alguns usam `"repo"`). Não tem nenhuma relação com edges do grafo.

**Edge** é uma aresta tipada entre dois nós no grafo (`Function ─INVOKES→ Call`).

São coisas completamente diferentes. Tags vivem **dentro de um struct**; edges vivem **entre nós**.

### 1.9.a — Re-decidir

Sobre adicionar tags JSON em campos hoje sem tag (consistência de serialização):

- [ ] Adicionar em todos (recomendado — JSON sempre estável e em snake_case ou camelCase canônico).
- [ ] Só quando virar API pública.
- [ ] Manter ad-hoc.

**Resposta:** ?

### 1.9.b — Convenção

Se for adicionar, qual convenção?

- [ ] `snake_case` (já é o padrão atual: `valid_from`, `feature_tags`).
- [ ] `camelCase`.
- [ ] Manter o nome do campo Go.

**Resposta:** ?
