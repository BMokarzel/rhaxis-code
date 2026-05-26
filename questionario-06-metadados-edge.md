# Questionário 06 — Metadados das arestas (`edge.Meta` + ID determinístico)

Foco em `Meta` de aresta, `Source` que está comentado mas é referenciado, e robustez do `DeterministicID`.

---

## 6.1 `Source` comentado vs referenciado

**Contexto:** `edge/edge.go:88` tem `// Source node.Source` comentado, mas docstrings em vários edges (`runs_on.go`, `code_gov.go`) dizem "Source.Properties guarda…". Inconsistência ativa.

**Pergunta:** Qual o destino?

- [ ] Definir `node.Source` (tipo novo, ex.: `{Tool, ToolVersion, RunID, Evidence []byte, ReportedAt}`) e reativar o campo.
- [ ] Remover toda referência a `Source` nos docstrings e seguir só com `Properties`.
- [ ] Mover essas infos para `Lineage` (que já existe).

Se for adicionar `Source`, ele duplica algo de `Meta.Tool/ToolVersion` (proposto em 5.1)?

- [ ] Sim — escolher um lugar (`Source` engloba `Tool/ToolVersion`).
- [ ] Não — `Source` é da edge (quem reportou esta aresta), `Tool` é do nó.

**Resposta:** ?

---

## 6.2 Campos de `edge.Meta` a adicionar

Hoje: `ValidFrom, ValidTo, ObservedAt, Confidence, Directional, Weight, Properties, Lineage`.

**Pergunta:** Adicionar?

- [ ] `Hash string` — fingerprint do payload original (idempotência).
- [ ] `EvidenceCount uint` — quantas observações apoiam.
- [ ] `Polarity string` — `positive|negative` (útil para `Communicates` que cessou, `DependsOn` removido).
- [ ] `Cardinality string` — `1:1|1:N|N:M` (hoje só comentário).
- [ ] `Reciprocal bool` — para `Peers` etc., explícito além de `Directional`.
- [ ] `Tags map[string]string` — uniformizar com nó.
- [ ] `Labels []string` — idem.
- [ ] `Tool/ToolVersion/RunID` — se 6.1 escolher Source-fora-da-edge.

**Resposta:** ?

---

## 6.3 `Confidence` granularidade

**Contexto:** `Confidence float32` em [0,1] presumido.

**Pergunta:** Documentar/normalizar?

- [ ] Padronizar faixa em `[0,1]` (assumir).
- [ ] Definir helpers/constantes (`ConfidenceHigh = 0.9`, etc.).
- [ ] Trocar por enum (`Low|Medium|High|Confirmed`).
- [ ] Manter como float livre.

**Resposta:** ?

---

## 6.4 `Weight` semântica

**Contexto:** `Communicates` documenta `Weight = BytesIn + BytesOut`. Outras arestas?

**Pergunta:** Padronizar?

- [ ] Cada edge documenta seu próprio significado.
- [ ] Adicionar `WeightUnit string` em Meta (e.g., `"bytes"`, `"calls"`, `"score"`).
- [ ] Tipo dedicado `Weight { Value float64, Unit string }`.

**Resposta:** ?

---

## 6.5 `DeterministicID` — risco de colisão

**Contexto:** Hoje: `sha256(from|type|to|validFrom)`. Dois fatos diferentes do mesmo par no mesmo instante colidem.

**Pergunta:** Adicionar discriminador?

- [ ] Incluir `Tool/RunID` no hash (coletor diferente, ID diferente).
- [ ] Incluir hash de `Properties` ordenadas (mesma aresta com payload distinto vira ID distinto).
- [ ] Incluir um `Discriminator string` opcional explícito.
- [ ] Não mudar — confiar em `ValidFrom` com precisão nanosegundo.

**Pergunta extra:** Hoje `validFrom` entra como `RFC3339Nano`. Se duas observações chegarem com timestamps idênticos (não raro em pipelines batch), colidem. Mitigar?

- [ ] Bater registro e considerar dedupe (idempotência intencional).
- [ ] Adicionar sequencial.

**Resposta:** ?

---

## 6.6 Direcionalidade

**Contexto:** `Directional bool` existe, mas a maioria das arestas é direcional por definição. `Peers` é o caso simétrico.

**Pergunta:** Manter ou inverter padrão?

- [ ] Manter `Directional` com default `true`.
- [ ] Documentar quais types são simétricos (lista canônica) e remover o campo.
- [ ] Renomear para `Symmetric` (oposto, default `false`).

**Resposta:** ?

---

## 6.7 Versionamento da aresta

**Contexto:** Nó tem `Version uint64` em `Meta`. Aresta **não**.

**Pergunta:** Aresta também precisa de `Version`?

- [ ] Sim — bitemporal + versionamento são ortogonais.
- [ ] Não — bitemporal cobre.
- [ ] Só em casos específicos (Replaces?).

**Resposta:** ?

---

## 6.8 `Replaces` específico

**Contexto:** `Replaces` modela versionamento estrutural. Tem só `Reason`.

**Pergunta:** Faltam campos?

- [ ] `FromVersion`/`ToVersion`.
- [ ] `Breaking bool`.
- [ ] `Diff URN` (referência para diff/PR).

**Resposta:** ?

---

## 6.9 Política de mutação vs reemissão

**Pergunta:** Quando algo muda (e.g., latência p99 de `Communicates`), como persistir?

- [ ] Atualizar a aresta existente (mutação) — perde histórico.
- [ ] Encerrar `ValidTo` da antiga e criar nova (bitemporal puro).
- [ ] Adicionar evento append-only separado (sample/snapshot), aresta corrente é agregada.

Isso afeta o design das edges de telemetria (`Communicates`).

**Resposta:** ?

---

## 6.10 Documentação canônica de cardinalidade

**Contexto:** Hoje cardinalidade está em comentários informais ("cardinalidade 1").

**Pergunta:** Onde formalizar?

- [ ] Em código (campo `Cardinality` em Meta — 6.2).
- [ ] Em ADR/spec separada (mantém código limpo).
- [ ] Em registro central (`edge/registry.go` com `EdgeSpec{Type, FromKinds, ToKinds, Cardinality}`).

A última opção viabiliza validação automática em runtime ("erro: `HasRole` espera Person→Role, recebeu Team→Role").

**Resposta:** ?
