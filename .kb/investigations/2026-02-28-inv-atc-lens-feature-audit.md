# Investigation: ATC Lens Feature Audit

**TLDR:** Audited all 33 orch-go commands + 40 packages through the ATC-not-conductor lens. The system is ~70% ATC-shaped (spawn context, gates, daemon sequencing, completion verification), ~20% conductor-shaped (tail, send, question, attention collectors, coaching metrics — real-time observation/direction infrastructure), and ~10% under-invested from ATC perspective (post-flight knowledge capture, handoff quality, separation/deconfliction).

## D.E.K.N. Summary

- **Delta:** Mapped every orch-go feature to ATC/conductor/under-invested categories. The system's architecture already leans heavily ATC, but significant conductor infrastructure exists in monitoring subsystems that could be reframed or deprioritized.
- **Evidence:** Read all cmd/orch/*.go command definitions (33 commands), all pkg/ packages (40+), serve endpoints (20+), and the ATC decision document. Categorization based on actual code behavior, not naming.
- **Knowledge:** Three conductor features have legitimate ATC functions if reframed (status as radar, tail as black-box recorder, question as distress signal). Three features are pure conductor with no ATC equivalent (send as mid-flight direction, coaching metrics as technique scoring, attention collectors as "watch everything"). The biggest ATC gap is post-flight knowledge capture — the system invests heavily in pre-flight briefing (spawn context) but under-invests in debrief-to-knowledge pipeline.
- **Next:** Recommend architect review for: (1) reframing monitoring features as ATC instruments, (2) investing in post-flight debrief pipeline, (3) evaluating conductor features for deprecation or scope reduction.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| .kb/decisions/2026-02-28-atc-not-conductor-orchestrator-reframe.md | extends | yes | - |
| .kb/investigations/2026-02-13-inv-audit-orch-go-implementation-model.md | extends | pending | - |

## Question

How does the actual orch-go feature set map to the ATC-not-conductor reframe? Which features are conductor-shaped (real-time observation/direction/micro-management), which are ATC-shaped (sequencing, separation, clearance, context injection), and where is the system under-invested from an ATC perspective?

## Status: Complete

---

## Methodology

Inventoried every command registered in `cmd/orch/main.go` (33 commands), every package in `pkg/` (40+), all HTTP API endpoints in `serve*.go` (20+), and the daemon subsystem. For each, assessed:

1. **ATC function mapping** — Does it map to sequencing, separation, clearance, context injection, weather advisory, emergency handling, or handoff? (Table from ATC decision)
2. **Conductor function mapping** — Does it involve real-time direction, technique review, micro-management, or "watching the performance"?
3. **Under-investment assessment** — Does the ATC model imply this function should exist but doesn't, or exists but is underdeveloped?

---

## Finding 1: Complete Feature Categorization

### Category 1: ATC-Shaped (Built for sequencing, separation, clearance, context injection)

These features directly map to ATC functions and need no reframing.

#### Pre-Flight Briefing (Context Injection)

| Feature | ATC Function | What It Does |
|---|---|---|
| **`orch spawn`** | Sequencing + Clearance + Context injection | 14-stage spawn pipeline with pre-flight gates, context generation, backend dispatch |
| **pkg/spawn/context.go** | Pre-flight briefing | Generates SPAWN_CONTEXT.md with 300+ lines of structured guidance, constraints, authority, deliverables |
| **pkg/spawn/gates/** | Clearance gates | 5 independent gates: hotspot, verification, concurrency, rate limit, triage bypass |
| **`orch orient`** | Weather advisory | Pre-session orientation: throughput metrics, ready work, model freshness, KB context enrichment |
| **pkg/orient/** | Weather advisory system | Computes throughput baselines, model staleness, knowledge attachment for session start |
| **`orch init`** | Runway preparation | Initializes .orch and .beads directories, idempotent |

**Assessment:** This is the system's strongest ATC area. Spawn context generation is extensive and well-evolved. The pre-flight briefing metaphor maps perfectly — agents get a comprehensive briefing before "takeoff" and are expected to fly autonomously from there.

#### Sequencing & Scheduling

| Feature | ATC Function | What It Does |
|---|---|---|
| **`orch daemon run`** | Traffic sequencer | Worker pool processes triage:ready issues, respects concurrency limits, polls periodically |
| **daemon/skill_inference.go** | Flight plan routing | Maps issue properties to skill assignment (title patterns, description heuristics, type inference) |
| **daemon/architect_escalation.go** | Altitude change clearance | Routes feature-impl to architect when targeting hotspot areas |
| **daemon/orphan_detector.go** | Missing aircraft detection | Detects in_progress issues with no active session, resets for re-spawn after 1h threshold |
| **daemon/phase_timeout.go** | Radio silence alert | Flags agents with no phase update for >30min (advisory, does not auto-correct) |
| **daemon/completion_processing.go** | Landing clearance | Polls for Phase: Complete agents, runs verification gates, closes verified work |
| **daemon/session_dedup.go** | Separation | Prevents duplicate spawns for the same issue |
| **`orch daemon preview`** | Flight plan preview | Shows what would be processed next without processing |
| **`orch daemon once`** | Single clearance | Processes one issue and exits |
| **`orch work`** | Clearance from issue | Spawns from beads issue with skill inference |
| **`orch backlog cull`** | Airspace cleanup | Surfaces stale P3/P4 issues for keep-or-close decision |

**Assessment:** Daemon is thoroughly ATC-shaped. Worker pool is a scheduler. Skill inference is routing. Concurrency limits are separation. Orphan detection is missing-aircraft protocol. Phase timeouts are radio silence alerts.

#### Post-Flight Verification (Clearance to Close)

| Feature | ATC Function | What It Does |
|---|---|---|
| **`orch complete`** | Landing clearance verification | 4-phase pipeline: resolve target → verification gates → advisories → lifecycle transition |
| **pkg/verify/** | Safety checks | 13 verification gates (phase, synthesis, test evidence, build, constraints, etc.) |
| **pkg/checkpoint/** | Checkpoint persistence | Two-tier verification: gate1 (comprehension), gate2 (behavioral) with JSONL audit trail |
| **`orch review`** | Post-flight debrief queue | Batch view of pending completions grouped by project, shows SYNTHESIS.md summaries |
| **`orch review done`** | Batch debrief processing | Completes all Phase: Complete agents for a project |
| **`orch rework`** | Go-around (missed approach) | Reopens failed completion, spawns rework agent with prior context |
| **`orch abandon`** | Emergency landing | Marks stuck agent as abandoned, exports transcript, generates failure report |
| **`orch clean`** | Hangar cleanup | Removes completed workspaces, stale tmux windows, orphaned resources |

**Assessment:** Completion pipeline is ATC's "landing sequence" — structured, gated, with explicit clearance (verification gates) before closing. The tier system maps well to aircraft category (Tier 1 = heavy requiring both gates, Tier 3 = light with minimal requirements). Rework is the "go-around" — missed approach that re-enters the pattern.

#### Priority & Alignment

| Feature | ATC Function | What It Does |
|---|---|---|
| **`orch focus`** | Priority airspace designation | Sets/views north star priority goal |
| **`orch drift`** | Deviation detection | Checks if active work aligns with focus, verdicts: on-track/drifting/unverified |
| **`orch next`** | Suggested clearance | Recommends next action based on focus + system state |

**Assessment:** Clean ATC pattern. Focus is "designated priority airspace." Drift is "deviation from flight plan." Next is "suggested routing."

#### Infrastructure & Configuration

| Feature | ATC Function | What It Does |
|---|---|---|
| **`orch account`** | Resource allocation | Manages Claude Max accounts for rate limit distribution |
| **`orch usage`** | Capacity monitoring | Shows account usage for capacity planning |
| **`orch version`** | System identification | Version and staleness check |
| **`orch hook`** | Safety system testing | Test, validate, and trace hook execution |
| **`orch doctor`** | System health check | Validates orch services health |
| **`orch port`** | Port allocation | Manages port assignments for project servers |
| **`orch servers`** | Ground infrastructure | Manages project dev servers (start/stop/attach/status) |
| **`orch retries`** | Retry management | Manages failed operation retries |

**Assessment:** All infrastructure/config commands are clearly ATC-shaped — system management, not agent direction.

---

### Category 2: Conductor-Shaped (Built for real-time observation/direction/micro-management)

These features invest in "seeing what's happening right now" or "directing agents mid-flight" — conductor patterns.

#### Real-Time Observation During Flight

| Feature | Conductor Pattern | ATC Reframe Possible? |
|---|---|---|
| **`orch tail`** | Watch agent output in real-time | **Yes** — Reframe as "flight data recorder" (black box). Post-incident analysis, not live monitoring. Currently used for debugging, which is post-incident. |
| **`orch monitor`** | SSE event stream watching | **Partial** — Desktop notifications on completion are ATC (landing notification). But continuous SSE monitoring is conductor ("watch the performance"). The daemon already handles completion processing, making monitor largely redundant for automation. |
| **`orch status`** | Snapshot of all agent activity | **Yes** — Reframe as "radar display." ATC needs situational awareness. Status is the primary radar. But the detailed per-agent view (phase, runtime, activity) trends toward conductor granularity. Compact mode is more ATC; `--all` mode is more conductor. |
| **`orch wait`** | Poll until agent reaches phase | **Partial** — Used in scripting/automation (ATC sequencing dependency). But also used by orchestrators to "watch" a specific agent, which is conductor. |
| **pkg/attention/** | Composable signal collection (11+ collectors) | **Partial** — StuckCollector and OrphanCollector are ATC (emergency handling). But CompetingCollector, GitCollector, DuplicateCandidateCollector trend toward "see everything" conductor mentality. The whole system is designed around the question "what needs my attention right now?" which is conductor framing. ATC framing would be "what clearance is pending?" |

#### Real-Time Direction During Flight

| Feature | Conductor Pattern | ATC Reframe Possible? |
|---|---|---|
| **`orch send`** | Send message to running agent mid-execution | **No** — This is explicitly "flying the plane." ATC doesn't send instructions to pilots during flight beyond clearances. If an agent is stuck, the ATC response is to surface a distress signal and let a human intervene, not to send course corrections. Send is the single most conductor-shaped feature. |
| **`orch question`** | Extract pending question from agent | **Yes** — Reframe as "distress signal detection." Agent is blocked, ATC detects the block and surfaces it. This is closer to "emergency handling" than "direction." |

#### Behavioral Observation & Scoring

| Feature | Conductor Pattern | ATC Reframe Possible? |
|---|---|---|
| **pkg/coaching/metrics.go** | Records behavioral metrics (loop detection, thrashing) | **Partial** — Loop detection injected as pain (tool-layer friction) is ATC (weather advisory). But aggregate coaching metrics for "performance scoring" is conductor. Recording is fine; using it to score agents is conductor. |
| **`orch patterns`** | Surfaces behavioral patterns for orchestrator awareness | **No** — "Orchestrator should be aware of agent behavioral patterns" is conductor framing. ATC doesn't analyze pilot technique. ATC cares: did they land safely? Pattern analysis is technique review. |

---

### Category 3: Under-Invested from ATC Perspective

These are ATC functions that the decision document implies should exist but are missing or underdeveloped.

#### 3.1: Post-Flight Debrief → Knowledge Pipeline (BIGGEST GAP)

**ATC parallel:** After every flight, there's a debrief. Findings from the flight inform future pre-flight briefings. The cycle is: briefing → flight → debrief → update briefings.

**Current state:** The system invests heavily in pre-flight briefing (SPAWN_CONTEXT.md with extensive context injection) but the debrief-to-knowledge pipeline is manual and leaky:

- SYNTHESIS.md is required but often placeholder quality
- `orch complete` checks SYNTHESIS.md exists but not that it feeds back into knowledge
- `kb quick` entries are created manually by agents during sessions
- No automated extraction from SYNTHESIS.md → decision records, guide updates, or constraint additions
- The "Leave it Better" skill instruction asks agents to externalize knowledge, but there's no verification that this actually happens
- `orch review` shows SYNTHESIS.md content but the orchestrator must manually decide what to promote

**Gap:** No `orch debrief` or automated SYNTHESIS.md → knowledge promotion pipeline. The ATC model says the orchestrator's investment should be in pre-flight briefing and post-flight debrief, but the debrief step is largely manual.

**Specific missing capabilities:**
- Automated SYNTHESIS.md parsing for decision/constraint/learning extraction
- `orch debrief <agent-id>` command that walks through knowledge promotion
- Knowledge delta tracking (what did this session add to the knowledge base?)
- Feedback loop verification (did the debrief actually update briefings for future agents?)

#### 3.2: Separation & Deconfliction (MODERATE GAP)

**ATC parallel:** ATC's core job is preventing collisions — ensuring two aircraft don't occupy the same space.

**Current state:**
- `daemon/session_dedup.go` prevents duplicate spawns for the same issue
- Concurrency gate limits total active agents
- But there's no mechanism to detect when two agents are modifying the same files
- No "edit conflict detection" before or during spawn
- The `CompetingCollector` in pkg/attention detects agents on the same issue, but not agents on different issues that target overlapping files
- Hotspot gates check file size but not concurrent modification

**Gap:** File-level separation is missing. ATC would ensure two agents don't land on the same runway simultaneously. Currently, parallel agents can and do create edit conflicts (mentioned in MEMORY.md: "Other agents frequently modify files concurrently").

**Specific missing capabilities:**
- File-lock or file-advisory system at spawn time ("these files are being modified by agent X")
- Spawn gate that warns when target files overlap with active agent's workspace
- Deconfliction routing (delay spawn until conflicting agent completes)

#### 3.3: Handoff Quality (MODERATE GAP)

**ATC parallel:** Handoff between ATC sectors is highly structured — the receiving controller gets a complete picture before accepting responsibility.

**Current state:**
- `orch handoff` exists but generates a session handoff document
- `handoff_content` is a verification gate in completion pipeline
- But inter-session handoff (when orchestrator session ends and new one starts) relies on MEMORY.md auto-memory + ad-hoc `bd prime` context recovery
- Cross-project handoff has no structured protocol
- The `orch orient` command provides session-start context but doesn't consume prior session's handoff document

**Gap:** No structured handoff-receive protocol. Current handoff is one-directional (generate document) without a corresponding "accept handoff" step that verifies the receiving orchestrator has situational awareness.

**Specific missing capabilities:**
- `orch orient` consuming prior `orch handoff` output
- Structured handoff acceptance checklist
- Cross-project handoff protocol (when work spans repos)

#### 3.4: Emergency Procedures (MINOR GAP)

**ATC parallel:** ATC has well-defined emergency procedures for different failure modes.

**Current state:**
- `orch abandon` handles stuck agents
- Orphan detection resets in_progress with no session
- Phase timeout surfaces silent agents
- But there's no unified emergency classification or escalation ladder

**Gap:** Emergency handling is ad-hoc rather than structured by severity. No `orch emergency` or severity classification that routes differently based on failure type.

---

## Finding 2: Feature Distribution Analysis

### By Category

| Category | Count | % of Features |
|---|---|---|
| ATC-shaped | ~25 commands + daemon subsystem + spawn gates + verification pipeline | ~70% |
| Conductor-shaped | 7 features (tail, monitor, status, send, question, patterns, attention) | ~20% |
| Under-invested ATC | 4 gaps (debrief pipeline, separation, handoff, emergency) | ~10% |

### By Investment (Lines of Code)

| Area | Approximate LOC | ATC/Conductor |
|---|---|---|
| Spawn (cmd + pkg) | ~3,000 | ATC (pre-flight) |
| Completion pipeline | ~2,000 | ATC (post-flight verification) |
| Daemon | ~2,500 | ATC (sequencing/scheduling) |
| Status/monitoring | ~1,500 | Conductor (observation) |
| Serve (HTTP API) | ~2,000 | Mixed (dashboard for both) |
| Attention collectors | ~800 | Conductor (observation) |
| Verification/checkpoint | ~1,200 | ATC (clearance) |
| Orient/focus/drift | ~600 | ATC (weather advisory) |

The LOC distribution confirms the system is architecturally ATC-shaped. The conductor investment is concentrated in status/monitoring/attention — features that make the orchestrator's job about "watching" rather than "managing."

---

## Finding 3: Conductor Features — Deprecation vs. Reframe Assessment

For each conductor-shaped feature, assessed: pure conductor (deprecate candidate) or has ATC function if reframed?

| Feature | Verdict | Reasoning |
|---|---|---|
| **`orch tail`** | **Reframe** as flight data recorder | Used for post-incident debugging, not live monitoring. ATC function: black box data for incident analysis. Keep but clarify purpose. |
| **`orch monitor`** | **Deprecation candidate** | Daemon handles completion processing. Desktop notifications could be a daemon feature. SSE watching is conductor. Only ATC function is completion notification, which daemon already does. |
| **`orch status`** | **Reframe** as radar display | ATC needs situational awareness. Compact mode is radar. `--all` mode is conductor granularity. Recommend: keep compact, reduce detail in full mode to ATC-relevant signals only (clearance pending, emergencies, capacity). |
| **`orch wait`** | **Keep** for scripting, **deprecate** for manual use | Scripting use (automation sequencing) is ATC. Manual "watch this agent" use is conductor. |
| **`orch send`** | **Deprecation candidate** | Most conductor-shaped feature. ATC never sends mid-flight instructions. If agent is stuck, surface via `orch question` + human intervention, don't send course corrections through the system. |
| **`orch question`** | **Reframe** as distress signal | Agent is blocked = pilot declaring emergency. ATC surfaces the distress signal for human response. This is valid ATC. |
| **`orch patterns`** | **Deprecation candidate** | Technique analysis is conductor. ATC doesn't review pilot flying patterns. The coaching system's friction injection (pain-as-signal) is ATC; the pattern analysis/reporting is conductor. |
| **pkg/attention/** | **Reframe + Scope reduction** | Keep: StuckCollector, OrphanCollector, VerifyFailedCollector (emergency detection). Evaluate: CompetingCollector (deconfliction is ATC), DuplicateCandidateCollector, GitCollector (observability, low ATC value). The unified "what needs attention" model is conductor framing; ATC would be "what clearance is pending." |
| **pkg/coaching/metrics.go** | **Keep recording, deprecate scoring** | Injecting friction (pain-as-signal) is ATC weather advisory. Recording metrics is data collection. But using metrics to score agent performance is conductor. |

### Summary: 3 deprecation candidates, 4 reframes, 2 keep-with-scope-changes

**Deprecation candidates:** `orch monitor`, `orch send`, `orch patterns`

**Reframe targets:** `orch tail` → flight recorder, `orch status` → radar, `orch question` → distress detection, `pkg/attention/` → clearance queue + emergency detection

---

## Finding 4: Under-Investment Priority Ranking

| Gap | Priority | Effort | Why This Matters for ATC |
|---|---|---|---|
| **Post-flight debrief → knowledge pipeline** | HIGH | Medium | ATC's briefing quality depends on debrief quality. Without automated debrief→briefing feedback, spawn context gradually becomes stale. The whole pre-flight investment degrades without post-flight capture. |
| **File-level separation/deconfliction** | MEDIUM | Medium | Core ATC job is preventing collisions. Current system prevents same-issue duplicates but not file-level conflicts between different issues. Parallel agents routinely conflict. |
| **Handoff receive protocol** | MEDIUM | Low | Handoff generation exists but acceptance verification doesn't. New sessions start with partial context. |
| **Emergency classification** | LOW | Low | Ad-hoc emergency handling works at current scale. Structured classification would matter more with higher agent counts. |

---

## Conclusion

The orch-go system is already predominantly ATC-shaped (~70% by feature count, ~75% by LOC). This is because the system organically evolved toward ATC patterns before the metaphor was explicitly named:

1. **Spawn context (pre-flight briefing)** is the most developed subsystem — extensive context injection, multiple clearance gates, skill-based routing.
2. **Completion pipeline (landing verification)** is well-structured with tiered gates and skip documentation.
3. **Daemon (traffic sequencer)** handles scheduling, routing, and periodic health checks without directing agents.

The conductor investment (~20%) is concentrated in monitoring subsystems that evolved when the system was smaller and manual oversight was the primary coordination mechanism. Most of these features (tail, status, question) have legitimate ATC interpretations if reframed. Three features (send, monitor, patterns) are pure conductor with no clear ATC function.

The biggest ATC gap is the post-flight debrief → knowledge pipeline. The system invests heavily in pre-flight briefing (SPAWN_CONTEXT.md) but the feedback loop — capturing learnings from completed work and feeding them back into future briefings — is manual and inconsistent. This is the highest-leverage investment from an ATC perspective: improving debrief quality directly improves briefing quality for all future agents.

---

## Test Performed

Verified categorizations against actual code:
- Read all 33 command definitions in `cmd/orch/main.go`
- Read spawn pipeline, daemon subsystem, completion pipeline, attention collectors, monitoring commands
- Verified ATC decision document's function mapping table against actual feature behavior
- Confirmed conductor features by checking whether they involve real-time agent direction or technique observation

---

## Recommendations

### Immediate (Low Effort)
1. **Reframe documentation** for tail, status, question to use ATC language (flight recorder, radar, distress signal)
2. **Scope `orch review`** to focus on "safe landing verification" not "technique review"

### Short-term (Medium Effort)
3. **Evaluate `orch send` deprecation** — if agents need help, they should declare distress (question), not receive mid-flight instructions
4. **Build `orch debrief`** — structured post-completion knowledge extraction from SYNTHESIS.md
5. **Add file-advisory to spawn** — warn when target files overlap with active agents

### Long-term (Higher Effort)
6. **Automated SYNTHESIS.md → knowledge promotion** — parse synthesis for decisions, constraints, learnings and propose `kb quick` entries
7. **Structured handoff acceptance** — `orch orient` consumes prior handoff, verifies continuity
8. **Reframe attention system** — from "what needs attention" (conductor) to "what clearance is pending" (ATC)
