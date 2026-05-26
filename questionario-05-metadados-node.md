# Questionário 05 — Metadados nos nós (`Meta` + extensões por tipo)

Foco em campos que afetam análise de IA, observabilidade da pipeline e queries comuns.

---

## 5.1 `Meta` base — campos transversais

Hoje `Meta` tem: `Version, ValidFrom, ValidTo, ObservedAt, ExtractionMethod, Confidence, Labels, Properties, Lineage`.

**Pergunta:** Adicionar campo a campo:

- [ ] `Hash string` — content hash da declaração (para detectar mudança real vs reformatação; idempotência de coletor).
- [ ] `Embedding []float32` + `EmbeddingModel string` — vetor semântico para busca/RAG.
- [ ] `Summary string` — resumo gerado por LLM.
- [ ] `Docstring string` — comentário do código (tipado, não em Properties).
- [ ] `Visibility string` — `public|private|protected|internal|package`.
- [ ] `Deprecated bool` + `DeprecatedSince string` + `Replacement URN`.
- [ ] `Stability string` — `experimental|stable|frozen`.
- [ ] `Tool string` + `ToolVersion string` — qual extrator/coletor produziu.
- [ ] `EvidenceCount uint` — companheiro de Confidence.
- [ ] `Tags map[string]string` — hoje só em Service.
- [ ] `Location Location` — uniformizar (hoje só Function/Data têm campo paralelo).

**Resposta:** ?

---

## 5.2 Localização

**Contexto:** `Location` aparece em `Function`, `Data`, sugerido para `Call`, `Variable`, `Test`, etc. Ter o campo em todo struct é repetitivo.

**Pergunta:** Onde fica?

- [ ] Em `Meta.Location` (único ponto, todos os nós herdam).
- [ ] Em cada struct concreto (mais explícito).
- [ ] Em ambos (incoerente, descartar).

**Pergunta extra:** `Location` cobre arquivo + range. Estender com:

- [ ] `Commit string` (SHA do commit observado)?
- [ ] `Blob string` (hash do conteúdo do arquivo)?
- [ ] `Anchor string` (anchor estável tipo `file#L10-L20`)?

**Resposta:** ?

---

## 5.3 `Properties` — catch-all tipado vs livre

**Contexto:** Hoje `Properties map[string]any`. Flexível, mas chaves variam livremente entre coletores.

**Pergunta:** Política?

- [ ] Manter livre (rapidez).
- [ ] Catalogar chaves conhecidas em constantes (`PropKeyPRUrl = "pr_url"`).
- [ ] Tipar `Properties` por tipo de nó (cada struct expõe sua sub-struct, não map).
- [ ] Híbrido: chaves comuns viram campo, raras ficam no map.

**Resposta:** ?

---

## 5.4 Lineage

**Contexto:** `Lineage{DerivedFrom []URN, Rule string}` — bom para fatos inferidos.

**Pergunta:** Adicionar?

- [ ] `Inputs []URN` — entradas além dos DerivedFrom (ex.: arquivos lidos).
- [ ] `RuleVersion string` — qual versão da regra produziu.
- [ ] `RunID string` — execução do coletor que gerou.

**Resposta:** ?

---

## 5.5 Específico de `Function`

**Pergunta:** Adicionar (no struct, não em Properties)?

- [ ] `Complexity { Cyclomatic, Cognitive, Halstead int }` — métricas.
- [ ] `LinesOfCode int`.
- [ ] `Churn { Commits int, LastModified time.Time, LastAuthor string }`.
- [ ] `Pure bool` — pureza funcional (ajuda IA a refatorar).
- [ ] `SideEffects []SideEffect` — `net|fs|db|env|stdout|time|rand|crypto|panic`.
- [ ] `IsAsync bool`, `IsGenerator bool`, `IsCoroutine bool`.
- [ ] `Receiver { TypeURN URN, Pointer bool }` — substitui `Class URN` por algo mais expressivo.
- [ ] `Visibility` (sobe para Meta, em 5.1).
- [ ] `TestCoverage float32`.
- [ ] `Annotations []URN` (se Annotation virar nó — 2.7).

**Pergunta extra:** `Param` hoje é `[]FunctionParam{Name, Type Data}`. Mudar para `[]URN` (apontando para `Variable` com Kind=param)?

- [ ] Sim (consistente com data flow).
- [ ] Não (parâmetros são parte da assinatura, não nós).
- [ ] Manter ambos (slot + URN).

**Resposta:** ?

---

## 5.6 Específico de `Data` (Type)

**Pergunta:** Adicionar?

- [ ] `Visibility` (Meta).
- [ ] `Generic bool` + `TypeParams []TypeParam{Name, Constraint URN}` — genéricos/templates.
- [ ] `Generated bool` + `GeneratedBy URN` — código gerado.
- [ ] `Immutable bool` (records, frozen, sealed).
- [ ] `Sealed bool` (Kotlin/Scala/Rust).
- [ ] `Size int` (bytes, se conhecido) — útil para perf.

**Resposta:** ?

---

## 5.7 Específico de `Service`

**Pergunta:** Adicionar?

- [ ] `LanguageVersion string` (Go 1.22, Python 3.11).
- [ ] `Runtime string` (Node18, JVM21, dotnet8).
- [ ] Tipar `ExternalDependencies` como `[]Dependency{Ecosystem, Name, Version, Constraint, Kind}` onde `Kind = build|runtime|dev|optional`.
- [ ] `BuildSystem string` (make, bazel, gradle, etc.).
- [ ] `RootPath string` (caminho do manifest dentro do repo).
- [ ] `EntryPoints []URN` (Functions/Endpoints "main").

**Resposta:** ?

---

## 5.8 Específico de `Endpoint`

**Pergunta:** Adicionar?

- [ ] `Auth { Required bool, Schemes []string }` — auth declarada.
- [ ] `RateLimit string`.
- [ ] `StatusCodes []int` — códigos possíveis documentados.
- [ ] `Middlewares []URN` — cadeia.
- [ ] `OperationID string` (OpenAPI).
- [ ] `Tags []string` (separar de `FeatureTags`).
- [ ] `Deprecated` (sobe para Meta — 5.1).

**Pergunta extra:** `Request.QueryParam/Header/Body` são todos `Data`. Isso força criar `Data` para query params (estranho). Manter ou trocar por estrutura mais leve (`map[string]ParamSpec`)?

- [ ] Manter `Data`.
- [ ] Trocar por estrutura própria.
- [ ] Híbrido.

**Resposta:** ?

---

## 5.9 Específico de `Module`

**Pergunta:** Adicionar?

- [ ] `Path string` (caminho dentro do repo).
- [ ] `Public bool` — exporta símbolos para fora?
- [ ] `Imports []URN` — modules referenciados (ou só via aresta?).
- [ ] `LanguageSpecific map[string]any` — Go-mod-replace, namespace XML, etc.

**Resposta:** ?

---

## 5.10 `ExtractionMethod` — cobertura

Hoje: `API, Inferred, Declared, Imported`.

**Pergunta:** Adicionar?

- [ ] `AST` — parse direto da árvore.
- [ ] `LSP` — via servidor de linguagem.
- [ ] `Runtime` — instrumentação em execução.
- [ ] `LLM` — extraído por modelo.
- [ ] `Manual` — entrada humana.
- [ ] `Webhook` — evento externo.

**Resposta:** ?
