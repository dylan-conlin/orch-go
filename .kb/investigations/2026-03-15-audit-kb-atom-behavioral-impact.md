# KB Atom Behavioral Impact Audit — 2026-03-15

**Type:** Anti-coherence check
**Method:** Random sample of 5 KB atoms, grep events.jsonl + spawn contexts + codebase for downstream references. Measure: did the atom change any future routing, review, or spawn decision?

## Sample (5 atoms, randomly selected)

### Atom 1: `2026-01-18-inv-change-spawn-default-opencode-claude.md`
**Type:** Investigation (Jan 18)
**Downstream references:**
- SPAWN_CONTEXT auto-injections: **23** (all passive "See:" links via kb context)
- events.jsonl: **0**
- Code: **0**
- CLAUDE.md: **0**
- Other KB citations: **0** (only appeared in a directory listing in ACTIVITY.json)

**Verdict: DEAD ATOM.** Written but never read. 23 SPAWN_CONTEXT injections are all automated kb-context "See:" lines — no agent or decision ever cited this investigation as influencing a choice. The spawn default change it investigated happened long ago (Jan 2026) and the decision is now baked into code and CLAUDE.md. The investigation served its purpose at write time but has zero ongoing behavioral impact.

---

### Atom 2: `2026-02-26-three-layer-hotspot-enforcement.md`
**Type:** Decision (Feb 26)
**Downstream references:**
- SPAWN_CONTEXT injections: **7**
- events.jsonl: **0**
- Code: **0** (but `pkg/spawn/gates/hotspot.go` implements it)
- CLAUDE.md: **1** (directly cited in Accretion Boundaries section)
- Other KB citations: **22** (cited in 10+ investigations as prior art, extended by mapping/harness work)

**Verdict: HIGH IMPACT.** This decision actively shapes spawn routing (hotspot gate blocks feature-impl on CRITICAL files), is cited in CLAUDE.md (read by every agent), referenced by 10+ downstream investigations as design precedent, and has a direct code implementation. The decision changed real behavior: agents hitting hotspot files get blocked or escalated to architect.

---

### Atom 3: `2026-03-14-inv-architect-subproblem-decision-protocol-design.md`
**Type:** Investigation (Mar 14 — 1 day old)
**Downstream references:**
- SPAWN_CONTEXT injections: **6**
- events.jsonl: **16** (spawn events for sub-finding evaluation workers)
- Code: **Yes** — `daemon_decision_log.go` and `daemon_loop.go` implement decision protocol logging
- CLAUDE.md: **0**
- Other KB citations: **0**

**Verdict: ACTIVE — RECENTLY CONVERTED TO CODE.** This is a fresh atom (1 day old) that has already driven code implementation (decision protocol logging in daemon). The 16 events.jsonl references are spawn events for the architect decomposition workers that implemented it. Too early to assess long-term impact but the investigation→code pipeline worked.

---

### Atom 4: `2026-02-24-design-orchestrator-skill-behavioral-compliance.md`
**Type:** Design investigation (Feb 24)
**Downstream references:**
- SPAWN_CONTEXT injections: **982** (massive — heavily-matched kb term)
- events.jsonl: **17** (spawn events for downstream investigations)
- Code: **0** (findings are about skill template design, not Go code)
- CLAUDE.md: **0**
- Other KB citations: **31** (cited in orchestrator-skill model, skill-content-transfer model, 5+ probes, failure taxonomy)

**Verdict: HIGH IMPACT (knowledge layer).** The 17:1 signal ratio finding from this investigation reshaped the orchestrator skill's design philosophy. It's cited in 2 models as a primary evidence source, referenced in 5+ probes, and drove the skill simplification effort (Mar 2026). The 982 SPAWN_CONTEXT injections suggest over-matching on common terms. The actual behavioral impact comes from model citations, not spawn context volume.

---

### Atom 5: `2026-03-05-inv-design-orchestrator-coordination-plans-persist.md`
**Type:** Design investigation (Mar 5)
**Downstream references:**
- SPAWN_CONTEXT injections: **6**
- events.jsonl: **0**
- Code: **Yes** — `cmd/orch/plan_cmd.go` + `pkg/plan/` implement the design
- CLAUDE.md: **0**
- Other KB citations: **1** (cited in plan-beads-integration-gap investigation)

**Verdict: MEDIUM IMPACT — DESIGN→CODE PIPELINE.** The investigation drove the implementation of coordination plans (`pkg/plan/`, `plan_cmd.go`). It's cited in one follow-up investigation about integration gaps. The design→implementation pipeline worked, but the investigation itself isn't being consulted ongoing — its value was consumed at implementation time.

---

## Summary

| # | Atom | Type | Age | Impact | Behavioral Change? |
|---|------|------|-----|--------|-------------------|
| 1 | spawn-default-opencode-claude | investigation | 56d | **DEAD** | No — stale, superseded by code/CLAUDE.md |
| 2 | three-layer-hotspot-enforcement | decision | 17d | **HIGH** | Yes — blocks spawns, cited in CLAUDE.md, 10+ downstream investigations |
| 3 | architect-subproblem-decision-protocol | investigation | 1d | **ACTIVE** | Yes — drove code implementation (decision logging) |
| 4 | orchestrator-skill-behavioral-compliance | design | 19d | **HIGH** | Yes — reshaped skill design, load-bearing in 2 models |
| 5 | coordination-plans-persist | design | 10d | **MEDIUM** | Yes — drove code implementation, 1 follow-up |

**Zero-reference atoms: 1/5 (20%)**
**Behavioral impact atoms: 4/5 (80%)**

## Anti-Coherence Findings

### 1. Dead atoms are natural lifecycle, not pathological
The dead atom (Atom 1) is a 56-day-old investigation whose findings are fully absorbed into code and CLAUDE.md. This is expected decay — archive candidates, not a system failure.

### 2. Two high-impact paths for KB atoms
Investigations drive behavior primarily through two paths:
1. **Decision→CLAUDE.md→spawn gate** (Atom 2): decision documents that get cited in CLAUDE.md have the highest reach because every spawned agent reads CLAUDE.md
2. **Investigation→model/probe** (Atom 4): investigations that feed model updates have durable impact through the model layer

### 3. Investigation→code has one-shot impact
Investigations that go directly to code (Atoms 3, 5) have a one-shot impact — they drive implementation but don't persist as ongoing behavioral constraints. Once the code exists, the investigation becomes reference material only.

### 4. SPAWN_CONTEXT injection volume ≠ read rate
Atom 4's 982 SPAWN_CONTEXT injections are a false signal. High injection count means the kb-context system frequently matches the atom's keywords. It does NOT mean agents actually read or acted on the injected context. The meaningful behavioral impact comes from explicit citations in models and probes, not from passive injection volume.

### 5. False learning risk: low in this sample
No atom in this sample created a false learning loop (where wrong knowledge propagated to downstream decisions). The dead atom (1) is harmlessly inert. The active atoms (2-5) all drove correct behavioral changes. However, the 20% dead rate extrapolated to 448 atoms suggests ~90 dead atoms in the KB — potential noise source for kb-context retrieval.
