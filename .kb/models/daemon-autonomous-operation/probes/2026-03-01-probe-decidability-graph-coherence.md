# Probe: Daemon-Decidability Cross-Model Coherence

**Model:** daemon-autonomous-operation
**Cross-Model:** decidability-graph
**Date:** 2026-03-01
**Status:** Complete

---

## Question

Does the daemon's skill inference and issue processing respect question subtypes (factual/judgment/framing) from the decidability graph model? Specifically: if someone creates `bd create --type question -l subtype:factual -l triage:ready`, does the daemon spawn it as an investigation? Should it?

---

## What I Tested

1. Read `pkg/daemon/skill_inference.go` — checked `IsSpawnableType()` and `InferSkill()` for question type handling
2. Read `pkg/daemon/issue_queue.go` — checked `IssueFilter.Filter()` for type filtering
3. Read `pkg/daemon/daemon.go` — checked `NextIssueExcluding()` poll loop for question handling
4. Searched entire `pkg/daemon/` for any reference to "subtype" labels
5. Read all three decidability decisions for daemon integration claims
6. Ran `go test ./pkg/daemon/ -run TestIsSpawnableType -v` to confirm test coverage

```bash
# Confirmed: zero references to subtype in daemon code
rg "subtype" pkg/daemon/   # → No matches found

# Confirmed: question not in spawnable types
# IsSpawnableType() at skill_inference.go:12-18 only includes: bug, feature, task, investigation

# Confirmed: InferSkill() at skill_inference.go:26-39 doesn't handle question type

# Confirmed: Tests pass but don't test question type at all
go test ./pkg/daemon/ -run TestIsSpawnableType -v   # PASS (tests bug/feature/task/investigation/epic/chore/unknown/"")

# Preview rejection message for question type (from preview.go:159):
# "type 'question' not spawnable (must be bug/feature/task/investigation)"
```

---

## What I Observed

### Finding 1: Daemon explicitly excludes all question-type issues

`IsSpawnableType("question")` returns `false`. The daemon's `NextIssueExcluding()` calls this at line 298 and skips with a debug message. Questions never reach skill inference.

### Finding 2: Zero subtype awareness in daemon code

`rg "subtype" pkg/daemon/` returns no matches. The daemon has no concept of factual vs judgment vs framing questions. All questions are treated identically: skipped.

### Finding 3: The decidability decisions explicitly mark daemon integration as OPTIONAL future work

From `2026-01-28-question-subtype-encoding-labels.md`:
- D.E.K.N. "Next" field: "optionally extend daemon to auto-spawn factual questions"
- Structured Uncertainty: "Actual daemon behavior with `subtype:factual` questions (no questions have this label yet)" listed under "What's untested"
- Implementation item 3: "Daemon extension (optional) — Add flag to spawn factual questions as investigations"

### Finding 4: question_detector.go is about agent runtime phase, not issue types

`pkg/daemon/question_detector.go` detects agents that report `Phase: QUESTION` during execution. This is a completely separate concern from question-type beads issues. No functional overlap.

### Finding 5: Worker-authority-boundaries implies future daemon integration

From `2026-01-19-worker-authority-boundaries.md` line 42: Workers should NOT label strategic questions `triage:ready` because "Daemon shouldn't auto-process premise questions" — this phrasing implies the daemon *could* eventually process non-premise (factual) questions.

### Finding 6: What happens with `bd create --type question -l subtype:factual -l triage:ready`

The daemon will:
1. List the issue via `ListReadyIssues()`
2. Hit `IsSpawnableType("question")` → `false`
3. Skip it: "type 'question' not spawnable (must be bug/feature/task/investigation)"
4. Log the skip if verbose mode is enabled

The issue sits in the queue indefinitely until the orchestrator manually processes it.

---

## Model Impact

- [x] **Extends** model with: The daemon-autonomous-operation and decidability-graph models are coherent in intent but have an unimplemented integration point. The decidability graph defines question subtypes with clear daemon-routing semantics (factual → daemon-spawnable as investigation, judgment → orchestrator, framing → Dylan), and the encoding decision (labels) explicitly designed for daemon consumption. However, the daemon has not been extended to consume these labels. This is acknowledged in the decisions as optional future work, not a gap or contradiction.

**Specific extension needed (if implemented):**
1. Add `"question"` to `IsSpawnableType()` — conditionally, only when `subtype:factual` label is present
2. Add question→investigation mapping in `InferSkill()` for factual questions
3. The daemon would need to check `subtype:factual` before spawning, and reject `subtype:judgment` and `subtype:framing` (returning them to orchestrator queue)

**Why this is "extends" not "contradicts":**
- The decisions explicitly say daemon integration is optional
- The current behavior (skip all questions) is safe — it errs on the side of human review
- The label convention exists and works; the daemon just doesn't read it yet

---

## Notes

- The `IsSpawnableType` test suite doesn't even include `"question"` as a test case — it's tested implicitly via the `"unknown"` and `""` cases but never explicitly
- The authority × subtype matrix from `2026-01-30-recommendation-authority-classification.md` adds a second dimension (authority:implementation/architectural/strategic) that would further complicate daemon routing if implemented
- The worker-authority-boundaries decision's triage:ready/triage:review split provides a coarser-grained version of the same routing: workers label tactical questions `triage:ready` and strategic ones `triage:review`. The subtype labels add finer granularity within the question type
- If daemon question routing is implemented, it should be gated behind a config flag (matching the "optional" framing in the decisions) to allow gradual rollout
