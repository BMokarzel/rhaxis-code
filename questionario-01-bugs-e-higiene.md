# Questionário 01 — Bugs, inconsistências e higiene

Itens objetivos. Maioria são correções pontuais; algumas têm impacto em consumidores.

---

## 1.1 Typo em `Endpoint.Hanlder`

**Contexto:** `node/endpoint.go:15` — `Hanlder URN` (typo).

**Pergunta:** Renomear para `Handler`?

- [ ] Sim, agora (ainda não há consumidores estáveis).
- [ ] Sim, mas manter alias deprecated.
- [ ] Não, manter como está.

**Resposta:** sim, agora

---

## 1.2 `HttpMethod` singletons sem URN

**Contexto:** `node/httpMethod.go` declara `MethodGet/Post/Put/Patch/Delete` todos com `URN: ""`. Indistinguíveis.

**Pergunta:** Que URN canônica adotar?

- [ ] `urn:http:method:GET` (etc.)
- [ ] `http://method/GET`
- [ ] Outro padrão: \_\_\_\_

**Pergunta extra:** `HttpMethod` deve continuar sendo nó do grafo, ou pode virar **enum simples** (string)?
Hoje `Endpoint.Method` é `URN`, o que implica nó. Vale o custo?

- [ ] Manter como nó (justifica queries do tipo "todos endpoints GET").
- [ ] Rebaixar para enum (`Endpoint.Method string`).

**Resposta:** quero facilitar consultas, então nó. pode colocar o urn:http:method:GET mesmo

---

## 1.3 `Sequence` quebrada

**Contexto:** `edge/sequence.go:11` embute a **interface** `Edge` em vez de `Base`. Não tem `From/To/ID`.

**Pergunta:** Qual o propósito real de `Sequence`?

- [ ] Ordem de execução entre statements (CFG `NEXT`).
- [ ] Ordem de argumentos em um Call.
- [ ] Ordem de campos em um struct.
- [ ] Outro: \_\_\_\_

Dependendo da resposta, sugiro renomear/refatorar. Pode descrever o caso de uso original?

**Resposta:** O edge deve criar o fluxo de steps de funções e codigos em geral. Declara var x -> chama func y -> controle de fluxo, etc.

---

## 1.4 `ControlNode` ausente em `NodeType`

**Contexto:** `node/control.go` define struct `Control`, mas `node.go` não tem `ControlNode` na enum. Sem discriminador na serialização.

**Pergunta:** Adicionar `ControlNode NodeType = "control"` à enum?

- [ ] Sim.
- [ ] Não — `Control` é sub-estrutura, não nó autônomo (então deveria sair do package node ou virar parte de Function).

**Resposta:** sim

---

## 1.5 `DefinedIn` deprecated

**Contexto:** `edge/defined.go` + comentário em `edge.go:32` dizem para usar `Contains`.

**Pergunta:** Já há consumidores reais de `DefinedIn`? Quando remover?

- [ ] Remover agora (não há consumidores).
- [ ] Manter por N versões — quais?
- [ ] Manter indefinidamente como alias.

**Resposta:** agora

---

## 1.6 `Data.Field` (singular) vs `Methods` (plural)

**Contexto:** `node/data.go:27` — `Field []FieldSlot`.

**Pergunta:** Renomear para `Fields`?

- [ ] Sim.
- [ ] Não.

**Resposta:** sim

---

## 1.7 `FunctionInterface` mal-formada

**Contexto:** `node/function.go:18` embute o **struct** `Node` em vez da **interface** `NodeInterface`. Método `isMethod()` é unexported (não usável via interface).

**Pergunta:** Qual intenção?

- [ ] Era para ser interface de fato — corrigir para embutir `NodeInterface` e exportar `IsMethod()`.
- [ ] É só um helper interno — deletar a interface e deixar só o método em `*Function`.
- [ ] Outro: \_\_\_\_

**Resposta:** corrigir para interface

---

## 1.8 `DataNode = "type"`

**Contexto:** Nome do símbolo (`DataNode`) difere do valor da string (`"type"`).

**Pergunta:** Unificar?

- [ ] Renomear símbolo para `TypeNode` (mantém valor `"type"`).
- [ ] Mudar valor para `"data"` (mantém símbolo).
- [ ] Não mexer.

**Resposta:** nao entendi

---

## 1.9 Tags JSON ausentes em vários campos

**Contexto:** `Service.ExternalDependencies`, `Data.Field`, `Data.Methods`, `Function.Param`, `Function.Return`, `Function.Class`, `Endpoint.*` — vários sem tag JSON. `Control.Type` tem.

**Pergunta:** Política?

- [ ] Adicionar tags JSON em **tudo** (consistência + serialização estável).
- [ ] Só quando virar API pública.
- [ ] Manter ad-hoc.

**Resposta:** acredito que ao invés de tags daria para ter edges, isso talvez facilitasse as queries
