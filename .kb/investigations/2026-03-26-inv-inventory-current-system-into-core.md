## Summary (D.E.K.N.)

**Delta:** The codebase is ~85% substrate/adjacent by line count, with core (thread/comprehension/knowledge) representing only ~15% of Go code — confirming the decision's insight that execution infrastructure dominates identity despite not being the differentiated product.

**Evidence:** Line-count analysis of all 55 packages and 100+ cmd files, cross-referenced against the product boundary from the thread-comprehension decision.

**Knowledge:** The biggest investment imbalance is in daemon (39K), spawn (30K), and verify (23K) — together nearly half the codebase. The core layer's two most important packages (thread at 2.3K, claims at 2.1K) are tiny compared to the substrate that supports them. The web UI has 5 routes, only 2 are core-aligned (briefs, knowledge-tree); the default landing page is a work graph.

**Next:** Use this inventory as a living classification tool for future investment decisions. Consider creating a lightweight decision gate: "Is this core, substrate, or adjacent?" before starting new features.

**Authority:** strategic — This is a scope-shaping classification that affects where new investment goes; Dylan decides priorities.

---

# Investigation: Inventory Current System Into Core/Substrate/Adjacent

**Question:** Given the thread-comprehension-layer-is-primary-product boundary, which subsystems in orch-go are core, which are substrate, and which are adjacent?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** orch-go-wgkj4
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` | extends | yes | - |
| `.kb/plans/2026-03-26-thread-comprehension-consolidation.md` | extends | yes | - |

---

## Findings

### Finding 1: Code Weight Is Overwhelmingly Substrate

**Evidence:** Total codebase is ~283K lines of Go across cmd/orch/ (90K) and pkg/ (193K).

| Layer | pkg/ lines | cmd/orch/ lines | Total | % |
|-------|-----------|----------------|-------|---|
| Core | ~22K | ~25K | ~47K | 16% |
| Substrate | ~163K | ~39K | ~202K | 72% |
| Adjacent | ~4K | ~3K | ~7K | 2% |
| Unclassified/shared | ~4K | ~23K (main, tests) | ~27K | 10% |

**Source:** `wc -l` on all Go files, classified per the product boundary decision.

**Significance:** The codebase structurally reinforces the orchestration-first identity. Any newcomer reading the code encounters 5x more spawn/daemon/verify code than thread/knowledge/comprehension code. This is the inertia the consolidation plan warned about.

---

### Finding 2: Three Packages Dominate the Substrate

**Evidence:**

| Package | Lines | % of all pkg/ | What it does |
|---------|-------|---------------|-------------|
| pkg/daemon/ | 39,331 | 20% | OODA cycle, triage, spawn execution, periodic tasks |
| pkg/spawn/ | 30,358 | 16% | Context generation, gates, spawn mechanics |
| pkg/verify/ | 23,144 | 12% | Completion checks, accretion gates, probe merge |

Together: 92,833 lines = 48% of all package code.

**Source:** `find pkg/{daemon,spawn,verify} -name '*.go' -exec wc -l`

**Significance:** These three packages represent nearly half the package code. Importantly, `pkg/verify/` straddles the boundary — probe-model-merge, consequence sensors, confidence gates, and decision enforcement are core-aligned (trust in understanding), while completion mechanics (check.go, skip.go, attempts.go) are substrate. Splitting verify's accounting: ~6.6K lines are core-aligned, ~16.5K lines are substrate.

---

### Finding 3: Core Packages Are Small but Strategically Important

**Evidence:**

| Package | Lines | What it does |
|---------|-------|-------------|
| pkg/thread/ | 2,337 | Thread lifecycle, back-propagation, work linking |
| pkg/claims/ | 2,126 | Machine-readable claim tracking, tension clusters |
| pkg/kbmetrics/ | 5,404 | KB health, evidence-tier drift, provenance |
| pkg/kbgate/ | 3,506 | Adversarial quality gates, challenge generation |
| pkg/tree/ | 2,553 | Knowledge lineage graph |
| pkg/dupdetect/ | 2,748 | Duplicate detection for completion |
| pkg/debrief/ | 1,919 | Session debrief generation |
| pkg/focus/ | 1,183 | North star / priority tracking |
| pkg/completion/ | 541 | Completion artifact validation |
| pkg/question/ | 273 | Question extraction |

Total core packages: ~22,590 lines (12% of pkg/)

**Source:** `wc -l` per package

**Significance:** The core layer is compact. This is both good (focused, not bloated) and a signal (proportionally tiny relative to the substrate it sits on). The thread package at 2.3K lines is the conceptual spine of the product but is 17x smaller than the daemon.

---

### Finding 4: Web UI Is Execution-Centric by Default

**Evidence:** 5 routes in web/src/routes/:

| Route | Layer | Purpose |
|-------|-------|---------|
| /briefs | Core | Comprehension queue reading surface |
| /knowledge-tree | Core | KB lineage visualization |
| /work-graph | Substrate | Active agents, execution state |
| /harness | Adjacent | Governance audit dashboard |
| /thinking | Ambiguous | Thinking process visualization |
| / (root +page.svelte) | Substrate | Default landing is execution dashboard (32KB) |

**Source:** `ls web/src/routes/*/`

**Significance:** The default page (root route, 32KB) is the execution dashboard. Briefs and knowledge-tree are secondary routes. This directly contradicts the product decision — a user's first impression is "execution monitoring tool" not "comprehension layer."

---

### Finding 5: The Daemon Contains Core-Relevant Logic Mixed with Substrate

**Evidence:** pkg/daemon/ (39K lines, 69 source files, 75 test files) includes:
- OODA cycle mechanics (substrate)
- Spawn execution pipeline (substrate)
- Workspace verification (substrate)
- Cycle cache, circuit breakers (substrate)
- But also: thread-first orient (`pkg/orient/orient.go` references threads), audit selection, and artifact sync — which connect to the comprehension layer

Similarly, pkg/attention/ (5K lines) has signals for model contradictions, knowledge decay, and hotspot acceleration — these are core-adjacent concepts implemented inside substrate infrastructure.

**Source:** File listings of pkg/daemon/, pkg/orient/, pkg/attention/

**Significance:** Some substrate packages serve dual roles: their infrastructure is substrate but they surface information that feeds the core comprehension layer. These are not cleanly separable today. The orient package explicitly leads with thread state, making it a bridge between layers.

---

### Finding 6: Adjacent Assets Are Small but Present

**Evidence:**

| Package/Dir | Lines | What it does |
|------------|-------|-------------|
| pkg/bench/ | 1,782 | Coordination benchmark infrastructure |
| pkg/advisor/ | 181 | Model recommendation via OpenRouter |
| pkg/scaling/ | 96 | Numeric helpers for N>2 agent experiments |
| pkg/openclaw/ | 1,200 | OpenClaw integration (migration target) |
| pkg/execution/ | 916 | Experimental execution harness |
| experiments/ | ~62 files | Coordination research results |
| cmd/orch/harness_*.go | 2,651 | Harness governance commands |
| skills/ | 210 files | Skill system source (shared/worker/meta) |

Total adjacent: ~7K Go lines + experiment/skill artifacts

**Source:** File counts and wc -l

**Significance:** Adjacent assets are relatively small in code but represent meaningful cognitive surface area (skills directory is 57K lines of markdown). The OpenClaw package is a migration artifact that may grow or die depending on strategic direction.

---

## Synthesis

**Key Insights:**

1. **The code-to-identity ratio is inverted** — 16% of code represents the differentiated product, 72% represents substrate. This isn't inherently wrong (substrate is necessary), but it means the codebase's "center of mass" pulls attention toward execution plumbing by default. New developers, new sessions, and new features naturally gravitate toward the larger surface area.

2. **Three packages own half the complexity** — daemon, spawn, and verify together are 93K lines. Future decisions about investment, refactoring, or extraction in these packages have outsized impact. The verify package specifically needs acknowledged dual identity: parts of it (probe-model-merge, confidence gates, consequence sensors) are core work and should stay invested.

3. **The web UI contradicts the product decision** — The root page is a 32KB execution dashboard. Briefs and knowledge-tree exist as secondary routes. Phase 3 of the consolidation plan (promote thread-first surfaces) will need to directly address this by changing the default landing experience.

4. **Bridge packages exist and matter** — orient, attention, and parts of verify contain logic that connects substrate execution to core comprehension (surfacing thread state, detecting knowledge decay, validating knowledge quality). These should be treated as core-aligned even though they live in substrate infrastructure. They are the connective tissue.

**Answer to Investigation Question:**

The subsystem map is presented below as the primary deliverable. The critical takeaway is that the classification is not just descriptive — it should function as a decision tool. Before new work begins, the question "does this serve core, substrate, or adjacent?" should have a fast, visible answer.

---

## Subsystem Map

### CORE — Primary investment, defines identity

**Purpose:** Systems that turn agent output into durable, legible understanding.

| Subsystem | Packages/Files | Lines | Role |
|-----------|---------------|-------|------|
| **Thread graph** | `pkg/thread/`, `cmd/orch/thread_cmd.go` | ~2.6K | Organizing artifact — work exists in service of questions |
| **Synthesis & briefs** | `cmd/orch/complete_synthesis.go`, `serve_briefs.go`, `complete_brief.go` | ~2.5K | Async comprehension surface for human review |
| **Knowledge composition** | `pkg/claims/`, `pkg/kbmetrics/`, `pkg/kbgate/`, `pkg/tree/` | ~13.6K | Claims, models, decisions, structured accumulation, lineage |
| **Comprehension queue** | `cmd/orch/review*.go`, `serve_reviews.go` | ~4.5K | What needs reading, triage, synthesis routing |
| **Knowledge quality gates** | `pkg/verify/{probe_model_merge,consequence_sensor,confidence_gate,decision_enforcement}*` | ~6.6K | Trust in understanding — probe verification, drift detection |
| **Completion pipeline** (core-aligned parts) | `cmd/orch/complete_pipeline.go`, `complete_verification.go`, `pkg/completion/`, `pkg/dupdetect/` | ~6.5K | Completion that improves trust and legibility |
| **Orient/stats** (comprehension side) | `cmd/orch/orient_cmd.go`, `stats_*.go`, `changelog*.go` | ~5K | What changed, what was learned, what remains open |
| **Debrief & focus** | `pkg/debrief/`, `pkg/focus/`, `pkg/question/` | ~3.4K | Session learning, priority tracking, question extraction |
| **Plan coordination** | `cmd/orch/plan_cmd.go`, `pkg/plan/` | ~1.7K | Multi-phase coordination management |

**Estimated total: ~47K lines (~16% of codebase)**

### SUBSTRATE — Necessary, not identity-defining

**Purpose:** Systems that make agents run. Portable and replaceable.

| Subsystem | Packages/Files | Lines | Role |
|-----------|---------------|-------|------|
| **Daemon** | `pkg/daemon/`, `pkg/daemonconfig/`, `cmd/orch/daemon_*.go` | ~46K | OODA cycle, autonomous triage, periodic tasks |
| **Spawn plumbing** | `pkg/spawn/`, `pkg/orch/`, `cmd/orch/spawn_*.go` | ~37K | Context generation, pipeline, gates, backend routing |
| **Completion mechanics** | `pkg/verify/{check,skip,attempts,accretion,duplication,model_stub}*` | ~16.5K | Mechanical verification, accretion gates, pre-commit |
| **Backend clients** | `pkg/opencode/`, `pkg/tmux/`, `pkg/model/` | ~10K | OpenCode API, tmux sessions, model resolution |
| **Agent lifecycle** | `pkg/agent/`, `pkg/discovery/`, `pkg/state/`, `pkg/identity/` | ~7.2K | Agent state, discovery, identity resolution |
| **Issue tracking** | `pkg/beads/`, `pkg/beadsutil/` | ~5.4K | Beads CLI integration |
| **Attention & orient** | `pkg/attention/`, `pkg/orient/` | ~9.5K | Work graph, attention signals, orient phase |
| **Events & telemetry** | `pkg/events/`, `pkg/action/`, `cmd/orch/telemetry.go` | ~5.6K | Event logging, action outcomes, telemetry |
| **Session management** | `pkg/session/`, `pkg/sessions/`, `cmd/orch/session*.go` | ~4K | Session tracking, history, handoff |
| **Account & config** | `pkg/account/`, `pkg/config/`, `pkg/userconfig/`, `pkg/group/` | ~8K | Multi-account, project config, user config |
| **Infrastructure** | `pkg/hook/`, `pkg/control/`, `pkg/health/`, `pkg/coaching/`, `pkg/checkpoint/` | ~6K | Hooks, governance enforcement, health monitoring |
| **HTTP server** | `cmd/orch/serve*.go` (execution surfaces) | ~20K | Dashboard API: agents, beads, system, coaching |
| **Utilities** | `pkg/{display,graph,workspace,port,service,notify,activity,claudemd,timeline,urltomd,entropy,artifactsync,skills}` | ~8K | Supporting utilities |
| **CLI infrastructure** | `cmd/orch/{main,status,clean,doctor,hotspot,swarm,deploy,learn,servers,handoff,tokens,attach}*.go` | ~20K | CLI commands for execution operations |
| **Web UI** (execution surfaces) | `web/src/routes/{+page,work-graph}/` | ~59K SVN | Default dashboard, work graph |

**Estimated total: ~202K lines (~72% of codebase)**

### ADJACENT — Valuable but separable from core identity

**Purpose:** Research, benchmarking, and experimental assets that inform methodology but should not drive product scope.

| Subsystem | Packages/Files | Lines | Role |
|-----------|---------------|-------|------|
| **Coordination research** | `experiments/`, `pkg/bench/`, `pkg/execution/`, `pkg/scaling/` | ~2.8K Go + experiment artifacts | Multi-agent coordination experiments |
| **Model benchmarking** | `pkg/advisor/` | ~181 | Provider comparison via OpenRouter |
| **Platform migration** | `pkg/openclaw/` | ~1.2K | OpenClaw integration (strategic direction TBD) |
| **Harness governance** | `cmd/orch/harness_*.go` | ~2.7K | Harness audit and report commands |
| **Web UI** (adjacent surface) | `web/src/routes/harness/` | ~14K SVN | Harness dashboard |
| **Skills system** (content) | `skills/src/` | ~58K md | Skill definitions and templates |

**Estimated total: ~7K Go lines + 58K skill markdown + experiment artifacts**

---

## Boundary Cases (Packages Straddling Core/Substrate)

These packages serve both layers and should be acknowledged as bridges:

| Package | Classified As | Core-Aligned Aspect | Substrate Aspect |
|---------|-------------|---------------------|-----------------|
| `pkg/verify/` | Split | probe-model-merge, consequence sensor, confidence gates | check.go, skip.go, accretion mechanics |
| `pkg/orient/` | Substrate with core role | Thread-first orient, measurement feedback | OODA cycle mechanics |
| `pkg/attention/` | Substrate with core role | Knowledge decay, model contradictions | Work graph prioritization |
| `pkg/events/` | Substrate | Event types feed knowledge tracking | Mechanical event logging |
| `cmd/orch/complete_*.go` | Split | Pipeline, synthesis, trust verification | Lifecycle mechanics, cleanup |

These bridges are where the product boundary gets operationally interesting. Future extraction or refactoring should preserve these connections.

---

## Structured Uncertainty

**What's tested:**

- Line counts verified via `wc -l` across all packages and cmd files
- Classification verified against decision document's explicit categorization
- Web UI route structure verified via directory listing
- Package responsibility verified against key-packages.md descriptions

**What's untested:**

- Whether the verify package split (6.6K core / 16.5K substrate) matches actual dependency flow — some "substrate" verification may be tightly coupled to core quality
- Whether skills content (58K lines of markdown) should be classified differently given it defines agent behavior
- How much of the daemon's 39K lines could be reduced if substrate-only features were pruned

**What would change this:**

- If thread-first UI testing shows users still prefer execution-centric surfaces, the core/substrate split should be re-evaluated
- If OpenClaw migration proceeds, `pkg/openclaw/` moves from adjacent to substrate
- If the daemon's orient phase becomes the primary comprehension delivery mechanism, parts of daemon shift toward core

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Use this map for investment gating | strategic | Affects where new work goes |
| Restructure web UI default page | architectural | Cross-component (backend + frontend) |
| Acknowledge verify's dual identity | implementation | Naming/documentation only |

### Recommended Approach: Living Classification Gate

Use this inventory as a lightweight decision gate before starting new features. When a new piece of work is proposed, classify it:

**Why this approach:**
- Makes the abstract product boundary concrete and operational
- Prevents substrate sprawl without requiring code restructuring today
- Gives the consolidation plan (Phase 2) its primary deliverable

**Implementation sequence:**
1. Commit this inventory as a reference artifact
2. Add a one-line classification question to spawn context: "Layer: core / substrate / adjacent"
3. Use the map to prioritize Phase 3 (thread-first UI) over substrate expansion

### Things to watch out for:
- The bridge packages (verify, orient, attention) serve both layers — don't over-prune substrate pieces that feed core
- Skills content (58K lines) is invisible in Go line counts but defines agent behavior — it deserves its own classification pass
- The daemon at 39K lines is the single largest complexity source; any consolidation work there has outsized impact

---

## References

**Files Examined:**
- All `pkg/*/` directories — line counts and file counts
- All `cmd/orch/*.go` — line counts and classification
- `web/src/routes/` — UI surface inventory
- `skills/src/` — skill content scope
- `experiments/` — research artifact scope

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` — product boundary that frames this inventory
- **Plan:** `.kb/plans/2026-03-26-thread-comprehension-consolidation.md` — Phase 2 deliverable this investigation produces
