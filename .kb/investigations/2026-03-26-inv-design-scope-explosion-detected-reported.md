## Summary (D.E.K.N.)

**TLDR:** The resolver already detects when a non-Anthropic model widens spawn scope by forcing a backend change, but that detection is only emitted as ad-hoc warning strings. The design should promote that moment into a structured routing-impact report that the CLI, daemon, workspace manifest, and completion review can all read consistently.

**Delta:** Non-Anthropic scope explosion is already detected in `spawn.Resolve()`, but it is not reported as a first-class artifact outside transient warning text.

**Evidence:** `pkg/spawn/resolve.go` writes `model-provider-routing` plus an auto-route warning, while `pkg/daemon/issue_adapter.go` discards successful `orch work` output and `pkg/spawn/context.go` persists only model/backend snapshots, not the routing change itself.

**Knowledge:** Detection and reporting are separate responsibilities; the canonical resolver should declare the routing change once, then every surface should render that structured result instead of reconstructing meaning from warning strings.

**Next:** Implement a structured routing-impact report in the spawn resolver, persist it in spawn artifacts/events, and render it in daemon and CLI summaries.

**Authority:** architectural - The fix crosses resolver design, daemon visibility, workspace artifacts, and completion review semantics.

---

# Investigation: Design Scope Explosion Detected Reported

**Question:** How should orch-go detect and report scope explosion when a non-Anthropic model causes spawn resolution to widen from the Anthropic-default path to a different backend/runtime path?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** OpenCode architect worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** model-access-spawn-paths

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/model-access-spawn-paths/model.md` | extends | yes | none |
| `.kb/models/defect-class-taxonomy/model.md` | extends | yes | none |

---

## Findings

### Finding 1: Resolver already detects provider-driven scope changes

**Evidence:** `Resolve()` first resolves backend/model precedence, then re-checks the resolved model and overwrites the backend when `modelBackendRequirement()` disagrees with the previously chosen backend. That path stamps `Backend.Detail = "model-provider-routing"` and appends a warning like `Auto-routed backend to opencode (...)`.

**Source:** `pkg/spawn/resolve.go:205`, `pkg/spawn/resolve.go:213`, `pkg/spawn/resolve.go:217`, `pkg/spawn/resolve.go:613`, `pkg/spawn/resolve_test.go:185`

**Significance:** The missing piece is not raw detection. The system already knows when a non-Anthropic model expands the runtime path; it just encodes that knowledge as a warning string instead of a durable routing fact.

---

### Finding 2: Reporting is stringly typed and disappears on daemon success paths

**Evidence:** `ResolveSpawnSettings()` prints resolver warnings to stderr for direct CLI use, and dry-run prints them again in a warnings section. But daemon spawning shells out to `orch work`, captures combined output, and discards that output on success. That means the exact scope-change warning exists during spawn, yet no durable consumer receives it in the normal daemon path.

**Source:** `pkg/orch/spawn_pipeline.go:208`, `cmd/orch/spawn_dryrun.go:184`, `pkg/daemon/issue_adapter.go:433`, `pkg/daemon/issue_adapter.go:449`

**Significance:** The system currently treats reporting as terminal text instead of state. That is why non-Anthropic routing changes are visible during manual experimentation but effectively invisible once the daemon successfully spawns the worker.

---

### Finding 3: Orch already has durable spawn metadata surfaces that can carry the report

**Evidence:** Workspaces persist `AGENT_MANIFEST.json` with skill, model, spawn mode, verify level, and review tier, while spawn modes emit `session.spawned` events with model/spawn metadata. SPAWN_CONTEXT also includes a config-resolution section for human review. None of these artifacts currently preserve the reason that backend/model routing changed.

**Source:** `pkg/spawn/session.go:165`, `pkg/spawn/context.go:266`, `pkg/orch/spawn_modes.go:508`, `pkg/spawn/worker_template.go:49`

**Significance:** The design does not need a new reporting subsystem. It needs one canonical routing-impact object that existing artifacts and event streams can embed.

---

## Synthesis

**Key Insights:**

1. **Detection already exists; explanation does not** - Findings 1 and 2 show that orch-go already knows when non-Anthropic models widen execution scope, but the knowledge is trapped in warning text that downstream systems cannot reliably consume.

2. **Scope explosion should be modeled as a routing impact, not as a provider-specific exception** - Findings 1 and 3 show that the risky event is a transition in execution contract (`claude/tmux` default to `opencode/headless`, or any future equivalent), not merely the appearance of `openai` or `google` strings.

3. **The right prevention pattern is allowlist + durable reporting** - This is defect class 0 exposure: a wider model/provider set changes what the resolver can return. The structural fix is to centralize the classification once and fan out structured reporting from that allowlisted source.

**Answer to Investigation Question:**

Orch-go should detect non-Anthropic scope explosion at the canonical resolver boundary, exactly where `model-provider-routing` is already decided, and convert that moment into a typed routing-impact report rather than a warning string. The report should describe the prior execution assumption, the new resolved execution contract, and the cause (`model requirement`, `provider class`, `source precedence`). It should then be persisted into existing durable surfaces - at minimum `AGENT_MANIFEST.json`, `session.spawned` events, and spawn summaries - so daemon-driven spawns, completion review, and future diagnostics all see the same fact. The main limitation in this investigation is that it did not implement the report, so the recommendation is based on code-path inspection rather than a working prototype.

---

## Structured Uncertainty

**What's tested:**

- ✅ `spawn.Resolve()` records provider-driven backend overrides today (verified by reading `pkg/spawn/resolve.go` and matching unit coverage in `pkg/spawn/resolve_test.go`).
- ✅ Direct CLI surfaces warnings, but daemon success paths do not preserve them (verified by tracing `ResolveSpawnSettings()` output handling and daemon `SpawnWork()` output handling).
- ✅ Existing spawn artifacts can carry routing reports without inventing a brand-new persistence channel (verified by reading agent manifest and `session.spawned` event code paths).

**What's untested:**

- ⚠️ Whether completion review should hard-gate on missing routing-impact metadata for non-Anthropic spawns.
- ⚠️ Whether daemon preview/orient should surface routing-impact history proactively, or only spawn-time summaries should do so.
- ⚠️ Exact schema shape for the typed report (dedicated struct vs embedded fields on `ResolvedSetting` or manifest/event payloads).

**What would change this:**

- If a successful daemon spawn already persisted resolver warnings somewhere durable, Finding 2 would be incomplete and the reporting gap would be narrower than described.
- If another canonical artifact already stores routing cause/reason fields, the recommended new object could collapse into extending that artifact instead of adding a new report structure.
- If future backend work makes non-Anthropic models no longer imply a distinct execution contract, the definition of "scope explosion" here would need to be reframed around capability changes instead of backend changes.

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Introduce a canonical routing-impact report for provider-driven backend changes and surface it across spawn artifacts/events | architectural | It spans resolver semantics, daemon observability, workspace persistence, and completion interpretation rather than a single local implementation detail |

### Recommended Approach

**Canonical routing-impact report** - Add one structured report at resolve time that describes when model/provider routing changes the execution contract, then reuse that report everywhere orch-go needs to explain the change.

**Why this approach:**
- It separates detection from presentation, which removes the current dependence on transient warning strings.
- It matches the existing architecture: resolver decides once, artifacts/events/CLI render later.
- It directly addresses the current blind spot where daemon-driven non-Anthropic spawns succeed without leaving a durable explanation of why the runtime path changed.

**Trade-offs accepted:**
- It adds a small amount of metadata plumbing across several surfaces instead of leaving reporting purely local to CLI output.
- It defers any new enforcement gate; the first step is making the routing change legible before deciding whether missing reports should block anything.

**Implementation sequence:**
1. Define a typed `RoutingImpact` or `ResolutionReport` at the resolver boundary with fields such as `trigger`, `from_backend`, `to_backend`, `provider_class`, `explicit_override`, and human-readable summary.
2. Persist that report into durable spawn artifacts (`AGENT_MANIFEST.json` and `session.spawned` event payloads) and expose it in SPAWN_CONTEXT/config resolution for review.
3. Render the report in manual spawn output, dry-run, and daemon logs/preview so non-Anthropic routing changes become visible in both interactive and autonomous workflows.

### Alternative Approaches Considered

**Option B: Keep warning strings and mirror them into more outputs**
- **Pros:** Minimal code churn; reuses current warning text.
- **Cons:** Preserves the core defect from Findings 1 and 2 because every new surface must parse or duplicate human text, and successful daemon flows still lack a canonical machine-readable record.
- **When to use instead:** Only as a stopgap if a same-day patch is needed before the structured report can be introduced.

**Option C: Detect scope explosion later with doctor or reconciliation scans**
- **Pros:** Avoids touching spawn-time resolution objects immediately.
- **Cons:** Too late in the lifecycle; the risky transition already happened, and the scan would infer behavior after the fact instead of preserving the authoritative reason at spawn time.
- **When to use instead:** As a secondary safety net to audit missed reports, not as the primary design.

**Rationale for recommendation:** Option A is the only choice that turns a currently ephemeral warning into a canonical fact with one source of truth. That directly matches the defect-class guidance and keeps future providers from repeating the same reporting gap.

---

### Implementation Details

**What to implement first:**
- A small typed report owned by `pkg/spawn/resolve.go`, not ad-hoc booleans spread across spawn callers.
- Manifest and event payload extensions before any new UI work so downstream consumers have stable data to read.
- Tests that cover OpenAI, Google, DeepSeek, explicit CLI backend overrides, and OpenClaw bypass behavior.

**Things to watch out for:**
- ⚠️ Do not redefine scope explosion as "any non-Anthropic model"; the meaningful transition is a changed execution contract, especially for future cases beyond today's provider list.
- ⚠️ Avoid duplicating routing logic in daemon and CLI presentation code; those layers should format an existing report, not rediscover it.
- ⚠️ Keep explicit user overrides visible so the report distinguishes "auto-routed due to model" from "user explicitly chose this backend."

**Areas needing further investigation:**
- Whether daemon preview should aggregate recent routing-impact events to help orchestration spot cost/capability shifts.
- Whether completion verification should require routing-impact evidence for architecture-tier spawns using non-default providers.
- Whether the same report should cover future OpenClaw/provider-multiplexing transitions.

**Success criteria:**
- ✅ A non-Anthropic spawn produces one canonical routing-impact object that can be read from the manifest or event stream without scraping CLI text.
- ✅ Manual spawn, dry-run, and daemon-driven spawn all surface the same explanation of why backend or runtime path changed.
- ✅ Resolver tests prove new providers or aliases inherit reporting automatically through the canonical classification path.

---

## References

**Files Examined:**
- `pkg/spawn/resolve.go` - Canonical model and backend resolution plus current auto-route warning behavior.
- `pkg/spawn/resolve_test.go` - Existing regression coverage for non-Anthropic auto-routing.
- `pkg/orch/spawn_pipeline.go` - Where resolver warnings are currently rendered for direct CLI use.
- `cmd/orch/spawn_dryrun.go` - Dry-run rendering of resolver warnings.
- `pkg/daemon/issue_adapter.go` - Daemon success path that drops command output, including warning text.
- `pkg/spawn/context.go` - Existing workspace artifact writing.
- `pkg/spawn/session.go` - Agent manifest schema and persistence.
- `pkg/orch/spawn_modes.go` - `session.spawned` event payloads and spawn summaries.
- `.kb/models/model-access-spawn-paths/model.md` - Existing architecture model for provider-aware backend routing.
- `.kb/models/defect-class-taxonomy/model.md` - Defect-class framing for scope expansion.

**Commands Run:**
```bash
# Verify project directory
pwd

# Create investigation artifact
kb create investigation design-scope-explosion-detected-reported --model model-access-spawn-paths

# Read issue context
bd show orch-go-u2rve
```

**External Documentation:**
- None.

**Related Artifacts:**
- **Decision:** `kb-d2ecf7` - Captures the coordination-level rule that routing changes should be reported as structured metadata from the resolver.
- **Investigation:** `.kb/investigations/2026-03-26-inv-design-scope-explosion-detected-reported.md` - Primary design artifact for this question.
- **Workspace:** `.orch/workspace/og-arch-design-scope-explosion-26mar-b366/` - Session workspace for synthesis, brief, and verification evidence.
- **Issue:** `orch-go-cubgs` - Follow-up implementation work for the recommended design.
- **Model:** `.kb/models/model-access-spawn-paths/model.md` - Existing model describing provider-aware backend routing constraints.

---

## Investigation History

**[2026-03-26 09:55]:** Investigation started
- Initial question: How should non-Anthropic scope explosion be detected and reported in spawn resolution?
- Context: Architect review requested because non-Anthropic models widen execution behavior and current reporting looked potentially transient.

**[2026-03-26 10:10]:** Current detection and reporting split identified
- Found that `spawn.Resolve()` already records provider-driven backend changes, but reporting is only emitted as warning text.

**[2026-03-26 10:20]:** Durable reporting surfaces identified
- Confirmed that manifest and `session.spawned` events already persist spawn metadata and are the natural carriers for a canonical routing-impact report.

**[2026-03-26 10:28]:** Investigation completed
- Status: Complete
- Key outcome: Recommended a canonical routing-impact report plus follow-up issue `orch-go-cubgs` for implementation.
