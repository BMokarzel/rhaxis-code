# Questionário 03 — Nós dos demais planos (infra, framework, governança, org)

Tipos de aresta referenciam nós que ainda não existem. Decidir se entram agora ou ficam fora do escopo.

---

## 3.1 Plano de infraestrutura

**Contexto:** Edges `DeployedOn`, `Routes`, `Peers`, `AttachedTo`, `ServiceRunsOn`, `Contains` (cloud), `CommunicatesWith` exigem nós como Compute, Persistence, Network, etc.

**Pergunta:** O projeto **vai** modelar infra ou foca só em código?
- [ ] Foco em código por ora — remover as constantes/edges de infra do `edge.go` até haver caso de uso.
- [ ] Modela infra também — criar agora os nós (lista abaixo).
- [ ] Modela infra **eventualmente** — manter constantes mas não criar structs.

Se for modelar, candidatos (marque os necessários):
- [ ] `Compute` (VM/Pod/Container/Lambda)
- [ ] `Persistence` (DB/Table/Bucket/Cache)
- [ ] `Messaging` (Topic/Queue/Exchange)
- [ ] `Network` (VPC/Subnet/ENI/LB)
- [ ] `Cluster`
- [ ] `Region`, `Account`, `Provider`
- [ ] `Secret` / `ConfigKey`

**Resposta:** ?

---

## 3.2 Plano de framework / supply chain

**Contexto:** `Uses`, `LicensedUnder`, `AffectedBy`, `PatchedIn` referenciam `Framework`, `License`, `SecurityAdvisory`. Nenhum existe.

**Pergunta:** Criar agora?
- [ ] Sim, todos.
- [ ] Só `Framework` (license e advisory ficam em Properties).
- [ ] Adiar — manter constantes mas sem struct.
- [ ] Remover as constantes até a hora.

Se criar `Framework`, sugestão:
```go
type Framework struct {
  Node
  Ecosystem string  // "npm","maven","go","pypi","cargo"
  Name      string
  Version   string  // pin observado
  Type      string  // "library","framework","tool","runtime"
}
```
Aceita?

**Pergunta extra:** Distinguir `Framework` (categoria) de `PackageVersion` (instância pinada)?
- [ ] Sim, dois nós.
- [ ] Não, um só (versão é campo).

**Resposta:** ?

---

## 3.3 Plano de governança

**Contexto:** `Delivers`, `AssignedTo`, `Serves`, `Realizes` referenciam `Feature`, `UserStory`, `Persona`, `Epic`. Nenhum existe.

**Pergunta:** Onde nasce o dado?
- [ ] Webhook do GitHub (labels/PRs) — então nós nascem do nosso lado.
- [ ] Sistema externo (Linear/Jira) — só guardar URN + ponteiro, sem schema rico.
- [ ] Ambos.

Modelar agora?
- [ ] Sim, todos (Feature, UserStory, Epic, Persona).
- [ ] Só `Feature` (resto adia).
- [ ] Adiar — manter constantes.

**Resposta:** ?

---

## 3.4 Plano organizacional

**Contexto:** `MemberOf`, `PartOf`, `ReportsTo`, `HasRole`, `LedBy`, `Owns` referenciam `Person`, `Team`, `Squad`, `Role`.

**Pergunta:** Origem do dado?
- [ ] Manual (yaml de orgchart no repo).
- [ ] Diretório corporativo (LDAP/Google/Okta).
- [ ] GitHub (CODEOWNERS, teams).
- [ ] Misto.

Modelar agora?
- [ ] Sim, todos.
- [ ] Só `Person` + `Team` (Squad e Role adiam).
- [ ] Adiar — manter constantes.

**Pergunta extra:** `Squad` vs `Team` — comentário do código diz "legado". Manter ambos?
- [ ] Só `Team`, deprecar `Squad`.
- [ ] Manter ambos.
- [ ] Generalizar para `OrgUnit` com `Kind`.

**Resposta:** ?

---

## 3.5 Plano de teste e qualidade (já tocado em 02.5)

**Pergunta:** Se `Test` virar nó, precisaremos também de:
- [ ] `CoverageReport` (artefato com agregados)?
- [ ] `Mutation` (mutation testing)?
- [ ] `Benchmark` separado de `Test`?

**Resposta:** ?

---

## 3.6 Plano de build / artefato

**Contexto:** Hoje não há nós para `Image` (docker), `Artifact` (jar/wheel/binary), `Pipeline` (CI job). Sem isso, perde-se a ponte de provenance "código → binário em produção".

**Pergunta:** Modelar?
- [ ] Sim — adicionar `Artifact`, `Image`, `Pipeline`, `Build` + edges (`BUILDS`, `PUBLISHES`, `DEPLOYS`).
- [ ] Só `Artifact`.
- [ ] Adiar.
- [ ] Fora de escopo.

**Resposta:** ?
