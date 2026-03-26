## Summary (D.E.K.N.)

**Delta:** Defined the concrete first-release bundle for orch-go: artifact formats + composition guide + thread CLI + curated examples, shipped with a comprehension-first init flow. Contradicts the matrix's Wave 1 assumption that artifact formats alone are self-documenting.

**Evidence:** Scored all 7 artifact types for standalone comprehensibility (avg 2.6/5). Counted 29+ orch-specific references that become noise without system context. Audited `orch init` flow against README promise — found execution-first "Next steps" contradicts comprehension-first positioning.

**Knowledge:** The minimum release has four layers: (1) must-ship surfaces that teach the method, (2) deferrable analysis tools, (3) substrate that ships but is backgrounded, (4) surfaces that wait. The critical insight is that Wave 1 needs a composition guide — without documenting how artifacts relate, the formats are opaque.

**Next:** Three implementation tasks: (1) write method guide documenting the composition model, (2) rewrite init "Next steps" to lead with threads, (3) curate example artifacts with IDs cleaned.

**Authority:** strategic — First-release definition constrains product perception irreversibly.

---

# Investigation: Define Minimum Open Release Bundle

**Question:** What is the smallest open release that lets a new user experience the method clearly, without the repo presenting itself as just another orchestration/execution CLI?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** orch-go-ehper
**Phase:** Complete
**Next Step:** Implementation tasks created below
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-03-26-inv-define-openness-boundary-matrix-productization.md` | extends — matrix provides 28-surface classification; this defines which subset ships first | Yes | **Contradicts** claim that artifact formats are "self-documenting" — they score 2.6/5 standalone |
| `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` | constrains — bundle must center on comprehension, not execution | Yes | None |
| `.kb/decisions/2026-03-26-open-boundary-opinionated-core.md` | constrains — bundle must be open at boundary, opinionated at core | Yes | None |
| `.kb/decisions/archived/2026-03-10-kb-product-identity-open-source-cli-targeting-sol.md` | constrains — target audience is solo technical researchers, bottom-up adoption | Yes | None |

---

## Problem Framing

**Design question:** What must ship together in the first open release so that a new user:
1. Understands what the product IS (comprehension layer, not orchestration CLI)
2. Can DO something immediately (not just read documentation)
3. Experiences the method's value without needing the full system running
4. Doesn't see a 72%-substrate codebase and conclude "this is a spawn tool"

**Success criteria:**
- A user can `orch init` in a repo and, within 10 minutes, create a thread, understand the artifact cycle, and feel the difference between this and "just taking notes"
- The README + init + first commands present a coherent story that matches the product decision
- Substrate exists and works but is not the first thing encountered

**Constraints:**
- Product identity decision: comprehension is core, execution is substrate
- Open-boundary decision: artifact formats must be inspectable and portable
- Target audience: solo technical researchers who adopt tools bottom-up
- Code ratio: 16% core / 72% substrate — opening everything equally makes execution dominate

---

## Findings

### Finding 1: Artifact formats are not self-documenting (contradicts matrix assumption)

**Evidence:** Scored all 7 artifact types for standalone comprehensibility:

| Artifact | Score | Key Blocker |
|---|---|---|
| Thread | 2/5 | 7 unexplained orch IDs in frontmatter |
| Brief | 3.5/5 | Best format (Frame/Resolution/Tension clear), filename IDs confusing |
| Model | 2/5 | Dense cross-refs, claim IDs opaque |
| Investigation | 3/5 | D.E.K.N. undefined, Phase/Authority jargon |
| Probe | 2/5 | Claim ref syntax opaque, verdict undocumented |
| Decision | 4/5 | Standard ADR, most standalone |
| KB README | 2/5 | Documents 4 of 7+ types, missing threads and briefs |

29+ orch-specific references across artifacts. No composition model documented anywhere.

**Significance:** Wave 1 of the three-wave release ("publish artifact format documentation") is necessary but not sufficient. Without a composition guide explaining how artifacts relate, the formats teach you what each piece looks like but not how the cycle works.

---

### Finding 2: The init flow contradicts the product decision

**Evidence:** After `orch init`, the user sees:

```
Next steps:
  1. Edit CLAUDE.md with project-specific details
  2. Create a beads issue: bd create "task description"
  3. Spawn an agent: orch spawn investigation "explore codebase"
```

The README leads with "Threads: the organizing spine." The init leads with "Spawn an agent." A user following init's guidance skips the primary product and jumps to substrate.

Thread commands (`orch thread new/list/show/append/resolve/link`) exist and work, but are never mentioned in the first-contact flow.

**Significance:** This is the most impactful single fix — changing 3 lines of "Next steps" output would realign the first contact with the product decision.

---

### Finding 3: The KB README documents 4 of 7+ artifact types

**Evidence:** `kb init` creates `.kb/README.md` documenting: models, investigations, decisions, quick entries. Missing: threads (the "primary organizing artifact"), briefs (the comprehension surface), probes (the evidence mechanism), SYNTHESIS.md (session completion artifact).

This means 3 of the 5 most method-defining artifacts (threads, briefs, probes) are invisible to a user who follows the official onboarding.

**Significance:** A user who reads the KB README after init sees a "standard ADR-plus" knowledge base, not the thread-centric method. The missing artifact types are exactly the ones that differentiate the method.

---

### Finding 4: The composition model is the missing link

**Evidence:** No document in the system describes how artifacts relate to each other:
- Threads organize questions and spawn investigations
- Investigations produce findings that generate probes
- Probes test claims against models
- Models are updated by probe findings
- Briefs synthesize agent work for human comprehension
- Decisions record resolved branches of threads

This cycle IS the method. Each individual artifact makes sense. The relationships between them are what make the system non-generic. But those relationships are implicit in the CLI tooling and the skill system — never documented for a new user.

**Significance:** This is the highest-leverage missing artifact. A single "method guide" that diagrams the composition cycle would do more for first-contact comprehension than any amount of artifact format documentation.

---

## Decision Forks

### Fork 1: What is the entry point type?

**Option A: Read-first (documentation)** — Publish artifact format docs and method guide. Users read about the method, adopt artifacts manually.
**Option B: Do-first (thread CLI)** — Users `orch init` and create their first thread immediately. The method is experienced, not described.
**Option C: Full binary, comprehension-first init** — Ship everything, but the first-run experience leads with threads.

**Recommendation: Option C.** Principle: "Day-one adoption should be additive, not replacement" (open-boundary decision). Users should DO something. But shipping just the thread CLI (Option B) orphans it from the substrate that makes threads valuable (agents fill threads with evidence). Ship the full binary, change the front door.

### Fork 2: How to handle the 16/72 code ratio?

**Option A: Change the ratio** (invest heavily in core code before release)
**Option B: Change the front door** (rewrite init, help text, docs to lead with comprehension)
**Option C: Ship a separate `kb` binary** (extract knowledge layer into standalone tool)

**Recommendation: Option B.** The orch-go-wgkj4 brief already identified this fork: "The front door change is a weekend. The ratio change is a quarter." For a minimum release, change what appears first. The ratio can shift over time through normal investment bias.

### Fork 3: What examples to curate?

**Option A: Synthetic examples** (clean but artificial)
**Option B: Real examples from orch-go** (authentic but noisy with IDs)
**Option C: Cleaned real examples** (real content, IDs replaced with explanations)

**Recommendation: Option C.** Real examples from orch-go are the strongest evidence that the method works. But raw IDs (orch-go-sispn) are noise. Clean them: replace IDs with descriptive labels, add inline comments explaining metadata fields. The briefs are the best candidate — Frame/Resolution/Tension is the clearest format and the most distinctive.

### Fork 4: What ships together vs. what can follow?

This is the core question. See Synthesis below.

---

## Synthesis

### The Minimum Open Release Bundle

**Principle applied:** "Is this a boundary concern or a method concern?" (open-boundary decision practical rule). The bundle includes everything needed to teach the method on first contact, with substrate present but backgrounded.

#### Layer 1: MUST-SHIP (teaches the method)

These must ship together. Removing any one breaks the first-contact story.

| Surface | Why it's essential | State |
|---|---|---|
| **Method guide** (new doc) | Documents the composition cycle: thread → investigation → probe → model → brief. This is the single most important missing artifact. | Does not exist — must be written |
| **Thread commands** (`orch thread new/list/show/append/resolve/link`) | The organizing spine. Users need to create and see threads on day 1. | Exists, works |
| **Artifact format documentation** (all 7 types) | Shows what each piece looks like. But only useful alongside the composition guide. | Partially exists (KB README has 4 of 7) |
| **Comprehension-first init flow** | Init must lead with threads, not spawn. 3 lines of output change. | Exists but wrong — init.go "Next steps" needs rewrite |
| **Curated examples** (2-3 per type) | Real artifacts cleaned of orch IDs, with inline explanations. Briefs are the strongest examples. | Raw material exists — needs curation |
| **The README** (already rewritten) | Leads with comprehension. Already aligned with product decision. | Done |
| **Expanded KB README** (all 7 artifact types) | Current `kb init` README documents 4 of 7+ types. Must include threads, briefs, probes, and the composition model. | Needs rewrite |
| **Brief format + template** | The most self-explanatory artifact. Frame/Resolution/Tension teaches the synthesis discipline. | Exists |

#### Layer 2: NICE-TO-HAVE (adds depth, not essential for first contact)

| Surface | Why deferrable | When |
|---|---|---|
| `kb ask` (knowledge queries) | Requires populated KB to be useful | After user has created artifacts |
| `kb claims` / `kb orphans` / `kb autolink` | Analysis tools — useful for maintenance, not first contact | After KB has grown |
| `orch comprehension` queue | Assumes agents have completed work | After first agent cycle |
| `orch review synthesize` | Batch review — requires existing completions | After first agent cycle |
| Detailed VERIFICATION_SPEC documentation | Governance detail — useful for teams, not solo first contact | With team features |

#### Layer 3: SUBSTRATE (ships but backgrounded)

| Surface | How to background it | Why it still ships |
|---|---|---|
| `orch spawn` + all spawn commands | In docs: appears under "Execution Substrate" section, not the introduction | Users need it to run agents |
| Daemon | Documented as "advanced: autonomous operation" | Power users want it |
| Backend routing (Claude CLI, OpenCode) | Documented as "configurable substrate" in CLAUDE.md | Required for agents to work |
| Dashboard (web UI) | Ships but isn't the first thing referenced. Init doesn't open it. | Useful for monitoring |
| Beads integration | Present as issue tracking, not foregrounded | Necessary for the lifecycle |
| Account management | Background infrastructure | Multi-account users need it |

#### Layer 4: WAIT (explicitly deferred)

| Surface | Why wait | Signal to ship |
|---|---|---|
| Hosted comprehension UX | Doesn't exist. Building it before first release is scope creep. | When adoption generates demand |
| Ranking intelligence | Doesn't exist. Needs usage data. | When comprehension queue has real traffic |
| Collaborative knowledge | Requires multi-user infrastructure. Solo tool first. | When team adoption happens |
| Coordination benchmark / research harness | Adjacent, not core. Confuses the product story. | Ship separately, not as orch |
| Full skill system documentation | Authoring guide for custom skills is a power-user concern | After users want to extend |

---

### First-Contact Walkthrough (what the bundle produces)

A new user's experience with the minimum release:

```
$ orch init
  Created: .orch/workspace/, .orch/templates/
  Knowledge base initialized (.kb/)
  ...

  Next steps:
    1. Edit CLAUDE.md with project-specific details
    2. Start a thread: orch thread new "How does [your question]?"
    3. See the method guide: cat .kb/GUIDE.md
    4. When ready for agents: orch spawn investigation "explore [topic]"
```

Step 2 gives them something to DO immediately — create a thinking thread. Step 3 shows them the composition cycle. Step 4 introduces agents when they're ready, not as the first thing.

After creating a thread, they see:

```
$ orch thread list
  [*] how-does-token-refresh-work    2026-03-26  How does token refresh...
```

They now have a named line of thinking. When they spawn an agent, the agent's findings feed back into this thread. The brief that results is readable without context. The cycle compounds.

---

### What "teaches the method" means concretely

The method is the cycle: **question → evidence → synthesis → knowledge → new question.**

The minimum release teaches it through four channels:
1. **Structure** — `orch init` creates the places artifacts go
2. **Action** — `orch thread new` makes the cycle start with a question, not a task
3. **Documentation** — the method guide diagrams the composition cycle
4. **Example** — curated real artifacts show what each stage looks like

All four are needed. Structure alone (Wave 1 from the matrix) doesn't teach — it just creates empty directories. Action without documentation is confusing. Documentation without examples is abstract. Examples without structure don't compound.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|---|---|---|
| Write method guide (`.kb/GUIDE.md` or `docs/method.md`) | strategic | First-release narrative — defines how the product is perceived |
| Rewrite init "Next steps" to lead with threads | implementation | 3 lines of output in `cmd/orch/init.go` |
| Expand KB README to document all 7 artifact types | implementation | Content change in `cmd/orch/kb_init.go` |
| Curate 2-3 example artifacts per type | implementation | Content curation, no code change |
| Ship full binary with comprehension-first ordering in `--help` | implementation | Help text ordering, no functionality change |

### Recommended Approach

**Ship the full `orch` binary with four changes:**

1. **Write a method guide** (~2-3 pages) documenting the composition cycle with a diagram. This is the one new artifact that ties everything together. Place it where `orch init` can reference it. Include the cycle diagram:

```
Question (thread)
    ↓
Investigation → Evidence
    ↓
Probe → Tests claim in model
    ↓
Model updated (confirm/contradict/extend)
    ↓
Brief → Synthesizes for human
    ↓
Decision (if resolved) → or → New question (thread)
```

2. **Rewrite `orch init` "Next steps"** to lead with thread creation, reference the method guide, and background spawn as "when ready":

```go
// In init.go, printInitResult()
fmt.Println("Next steps:")
fmt.Println("  1. Edit CLAUDE.md with project-specific details")
fmt.Println("  2. Start a thread: orch thread new \"How does [your question]?\"")
fmt.Println("  3. See the method: cat .kb/GUIDE.md")
fmt.Println("  4. When ready for agents: orch spawn investigation \"explore [topic]\"")
```

3. **Expand `kb init` README** to document all 7 artifact types (add threads, briefs, probes) and include a 5-line composition model summary.

4. **Curate example artifacts** — select 2-3 real briefs, one thread, one model excerpt, one decision. Clean orch-specific IDs, add inline comments explaining metadata. Ship in `docs/examples/` or `.kb/examples/`.

### Trade-offs accepted

- **Not shipping a separate `kb` binary.** This would cleanly separate the method from the substrate, but it adds a packaging/distribution concern and fragments the CLI surface. The full binary with comprehension-first ordering is simpler.
- **Not waiting for the comprehension dashboard to be ready.** The method can be taught through the CLI alone. The dashboard is nice-to-have for day 1.
- **Not changing the `orch` name.** "Orch" implies orchestration, which is the substrate framing. But renaming is a strategic decision beyond this scope. The README and method guide can frame the name correctly.

### Things to watch out for

- The method guide must be SHORT. More than 3 pages and it becomes documentation, not teaching. Lead with the diagram, then one paragraph per artifact type.
- Curated examples must be REAL, not synthetic. Synthetic examples feel like marketing. Real examples feel like evidence. This is consistent with the model's epistemic principles.
- The init "Next steps" change is the highest-impact, lowest-effort item. Ship it even if nothing else ships.

### Areas needing further investigation

- **Name tension:** Should the first-release position "orch" as a name for the method layer, or does the name need to change? (Strategic decision, beyond this scope.)
- **Packaging:** Binary distribution (Homebrew tap, Go install, release binaries). Technical question, not addressed here.
- **Whether `orch thread` alone is compelling enough** without agent-produced evidence. A user who creates threads but never spawns agents might feel like they're just using a glorified notes tool. The method guide needs to address this: threads become powerful when agents fill them with evidence.

### Success criteria

- A new user can `orch init` → `orch thread new` → read the method guide and understand the cycle within 10 minutes
- The README, init output, and method guide tell the same story (comprehension, not execution)
- Someone browsing the repo sees threads/briefs/models before they see spawn/daemon/backend
- The first release announcement can lead with "what agents learn matters as much as what they build" and the released surface supports that claim

---

## Structured Uncertainty

**What's tested:**
- ✅ Artifact formats scored for standalone comprehensibility against real artifacts (N=7 types)
- ✅ Init flow audited against README promise — confirmed mismatch
- ✅ KB README coverage gap measured (4/7+ types documented)
- ✅ Thread commands verified as existing and functional

**What's untested:**
- ⚠️ Whether the method guide will actually teach the cycle (needs user testing after it's written)
- ⚠️ Whether threads alone (without agent evidence) feel compelling on day 1
- ⚠️ Whether "cleaned real examples" are better than synthetic examples (assumption based on epistemic principle, not A/B test)
- ⚠️ Whether the `orch` name is a barrier to comprehension-first perception

**What would change this:**
- If the method guide turns out to be 10+ pages to explain the cycle, the cycle is too complex and needs simplification before release
- If users consistently skip threads and go straight to spawn, the method guide isn't teaching — it's decorating
- If a competitor ships a comprehension layer under a non-orchestration name and gets traction, the naming question becomes urgent

---

## References

**Files Examined:**
- `.kb/threads/2026-03-24-threads-as-primary-artifact-thinking.md` — Thread format
- `.kb/briefs/orch-go-sispn.md`, `.kb/briefs/orch-go-pityd.md` — Brief format
- `.kb/models/knowledge-accretion/model.md` — Model format
- `.kb/investigations/2026-03-26-inv-define-openness-boundary-matrix-productization.md` — Investigation format
- `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` — Decision format
- `.kb/decisions/2026-03-26-open-boundary-opinionated-core.md` — Open boundary decision
- `.kb/decisions/archived/2026-03-10-kb-product-identity-open-source-cli-targeting-sol.md` — Product identity
- `cmd/orch/init.go` — Init command and "Next steps" output
- `cmd/orch/kb_init.go` — KB init and README content
- `cmd/orch/thread_cmd.go` — Thread commands
- `cmd/orch/comprehension_cmd.go` — Comprehension queue
- `cmd/orch/kb.go` — KB commands
- `README.md` — Current product framing
- `.orch/templates/BRIEF.md` — Brief template
- `.orch/templates/SYNTHESIS.md` — Synthesis template

**Related Artifacts:**
- **Probe:** `.kb/models/knowledge-accretion/probes/2026-03-26-probe-minimum-open-release-bundle-definition.md`
- **Decision:** `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md`
- **Decision:** `.kb/decisions/2026-03-26-open-boundary-opinionated-core.md`
- **Matrix:** `.kb/investigations/2026-03-26-inv-define-openness-boundary-matrix-productization.md`

---

## Investigation History

**2026-03-26:** Investigation started
- Design question: What is the minimum open release that teaches the method?
- Spawned 2 exploration agents: artifact comprehensibility audit + init flow analysis

**2026-03-26:** Evidence gathered
- Artifact comprehensibility scores: avg 2.6/5 — formats NOT self-documenting
- Init flow: leads with execution, contradicts product decision
- KB README: missing 3 of 7 artifact types
- Composition model: nowhere documented

**2026-03-26:** 4 forks identified and navigated
- Entry point: full binary with comprehension-first init (not docs-only or separate kb binary)
- Code ratio: change front door, not ratio (a weekend, not a quarter)
- Examples: cleaned real artifacts (authentic but readable)
- Bundle: 4 layers — must-ship / deferrable / backgrounded substrate / explicitly wait

**2026-03-26:** Investigation completed
- Minimum release defined as: artifact formats + composition guide + thread CLI + curated examples + comprehension-first init
- Key correction to three-wave strategy: Wave 1 needs more than formats alone
