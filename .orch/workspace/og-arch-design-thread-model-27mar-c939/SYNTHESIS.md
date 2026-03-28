# Session Synthesis

**Agent:** og-arch-design-thread-model-27mar-c939
**Issue:** orch-go-t37xi
**Duration:** 2026-03-27 → 2026-03-27
**Outcome:** success

---

## Plain-Language Summary

Converged threads are a dead end — they represent completed thinking that has nowhere to go. This design adds a promotion lifecycle: `orch thread promote <slug> --as model|decision` takes a thread that's done thinking and scaffolds it into a durable artifact (model directory with provenance, or decision record with context). The thread gets a new `promoted` status and `promoted_to` field pointing at what it became. Orient starts surfacing unpromoted converged threads as "ready to promote," the same way it surfaces unread briefs. The design was validated against two live test cases: the generative-systems thread (→ model) and the product-surface thread (→ decision).

## Verification Contract

See `VERIFICATION_SPEC.yaml` — design outputs verified: investigation complete, 4 implementation issues created with dependencies reported, 3 composition claims documented, defect class exposure analyzed.

---

## TLDR

Designed the thread-to-model promotion lifecycle: new `promoted` status, `promoted_to` frontmatter field, `orch thread promote --as model|decision` command with artifact scaffolding and provenance, orient integration surfacing converged threads as actionable. Four implementation issues created.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-27-inv-design-thread-model-promotion-lifecycle.md` - Full architect investigation with 5 findings, synthesis, recommendations, and composition claims

### Commits
- (pending) - Investigation and workspace artifacts

---

## Evidence (What Was Observed)

- `pkg/thread/lifecycle.go:14-19`: `IsResolved()` returns true for converged — this makes converged threads invisible to orient
- `pkg/thread/thread.go:18-32`: Thread struct has no PromotedTo field
- `cmd/orch/orient_cmd.go:475-496`: collectActiveThreads calls thread.ActiveThreads which filters on !IsResolved — no path for surfacing converged threads
- `.kb/models/TEMPLATE.md`: No Promoted From provenance field
- Two live test cases confirm different promotion targets needed (model vs decision)
- `updateFrontmatter()` only updates existing fields — may need field insertion capability for promoted_to

---

## Architectural Choices

### Promotion as new status vs. overloading converged
- **What I chose:** New `StatusPromoted` terminal status
- **What I rejected:** Reusing `converged` with just a `promoted_to` field
- **Why:** Converged means "thinking done but not yet externalized." Promoted means "became an artifact." These are semantically distinct states. A thread can sit converged for days before someone decides what artifact it should become. Orient needs to distinguish "converged and waiting" from "converged and already promoted."
- **Risk accepted:** 6th status increases lifecycle complexity slightly

### Multi-target promotion vs. model-only
- **What I chose:** `--as model|decision` flag on promote command
- **What I rejected:** Promotion always creates a model
- **Why:** Live test cases prove different targets: generative-systems is a model (describes mechanism), product-surface is a decision (defines a choice). Principle promotion deferred — principles are hand-curated in a single file.
- **Risk accepted:** More complex command, but the orchestrator (not Dylan) uses the flag

### Probe-first claims bootstrap vs. auto-populated claims table
- **What I chose:** Promotion creates scaffold with thread's core claim as initial thesis; probes fill claims organically
- **What I rejected:** Auto-generating claims from thread entries, or requiring an architect session to decompose claims
- **Why:** Aligns with named-incompleteness principle — the model starts with a named gap (one claim, no probes) rather than premature structure

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-27-inv-design-thread-model-promotion-lifecycle.md` - Full design for promotion lifecycle

### Decisions Made
- Decision: Thread promotion targets multiple artifact types (model, decision), not just models — because live test cases prove different needs
- Decision: `promoted` is a new terminal status distinct from `converged` — because the two states have different orient visibility requirements
- Decision: Bidirectional provenance (thread→artifact AND artifact→thread lineage) — because absorbed threads lose contribution credit without it

### Constraints Discovered
- `updateFrontmatter()` only updates existing fields — inserting new fields (promoted_to) requires either field pre-population in thread creation or a new insertion function
- Orient renders 3 elements (threads, briefs, tensions) — promotion-ready is a 4th element that needs careful placement

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation, issues, workspace artifacts)
- [x] Design validated against live test cases
- [x] Investigation file has Status: Complete
- [x] Ready for `orch complete orch-go-t37xi`

**Implementation order:**
1. orch-go-jp1ti (thread package — foundation)
2. orch-go-5ithk + orch-go-yf3xa (command + orient — can parallelize, both depend on thread pkg)
3. orch-go-6kjp7 (integration verification — depends on all three)

---

## Unexplored Questions

- Should `orch compose` (digest creation) include promotion-ready threads as a signal for the digest?
- Should the daemon auto-spawn promotion work when converged threads accumulate past a threshold?
- How should `kb context` handle promoted threads — include them in search results as provenance for the model they became?
- Should principle promotion (`--as principle`) be supported, or is that always a Dylan-direct action?

---

## Friction

Friction: ceremony: `bd dep add` blocked by governance hook — had to report dependencies via comment instead. Minimal time cost (~30s redirect).

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-arch-design-thread-model-27mar-c939/`
**Investigation:** `.kb/investigations/2026-03-27-inv-design-thread-model-promotion-lifecycle.md`
**Beads:** `bd show orch-go-t37xi`
