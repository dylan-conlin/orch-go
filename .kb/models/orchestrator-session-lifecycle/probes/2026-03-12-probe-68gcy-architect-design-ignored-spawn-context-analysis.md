# Probe: 68gcy Agent Ignored Architect Design — Spawn Context Knowledge-Feedback Analysis

**Model:** orchestrator-session-lifecycle
**Date:** 2026-03-12
**Status:** Complete

---

## Question

The orchestrator-session-lifecycle failure mode taxonomy (probe 2026-03-11) catalogs skill failure modes. Agent orch-go-68gcy was spawned as feature-impl to consolidate harness tools. Architect orch-go-sb13k had explicitly designed harness as a SEPARATE repo (github.com/dylan-conlin/harness) with zero orch-go dependencies. The agent instead built pkg/harness/ + cmd/harness/ inside orch-go, requiring a revert. **Does the spawn pipeline's knowledge-feedback chain reliably surface architect designs to feature-impl agents?**

---

## What I Tested

### Test 1: Recover the 68gcy issue description (primary framing)

```bash
git log --all --diff-filter=M -p -- .beads/issues.jsonl | grep -A5 "orch-go-68gcy"
```

**68gcy issue title:** "Consolidate orch harness into standalone harness CLI — single tool for structural governance"

**68gcy issue description:** "Two harness tools exist: orch harness (lock/unlock/status/verify/init/report/snapshot) and standalone harness (init/check/report). The overlap caused a bug (orch-go-s2pks/orch-go-1eo) where standalone init wrote relative paths to user-level settings. **Direction: standalone harness absorbs all functionality (including lock/unlock/verify/snapshot). orch harness becomes a thin wrapper or is removed. One tool, usable in any project.**"

### Test 2: Verify architect design was committed BEFORE 68gcy spawn

```bash
git log --format="%ai %H %s" 7cb584747..5f48ae16f
```

Timeline:
- **2026-03-11 19:05** — Architect sb13k committed design: "Extract 3 zero-dependency packages... into new github.com/dylan-conlin/harness repo... zero orch-go dependency"
- **2026-03-12 11:46** — 68gcy issue created (~16.5 hours later)
- **2026-03-12 11:47** — 68gcy agent enters Planning
- **2026-03-12 11:49** — 68gcy plans pkg/harness extraction INSIDE orch-go (2 minutes later)
- **2026-03-12 11:50** — 68gcy enters Implementation (3 minutes after Planning)
- **2026-03-12 12:00** — 68gcy commits (13 minutes total)

### Test 3: Simulate kb context query the agent would have received

```bash
# Keywords extracted from title: "consolidate harness standalone" (per ExtractKeywords logic)
kb context "consolidate harness standalone"
```

Result: The architect investigation **DID appear** in the INVESTIGATIONS section:
```
- Design Standalone Harness CLI Extracted from orch-go
  Path: .kb/investigations/2026-03-11-inv-design-standalone-harness-cli-extracted.md
```

But it appeared only as a **title + path pointer** among 6+ other investigation entries. The critical conclusion ("new separate repo with zero orch-go dependency") was NOT visible in the pointer.

### Test 4: Verify --architect-ref mechanism

Read `pkg/spawn/gates/hotspot.go` — the `--architect-ref` flag validates that an architect issue exists and is closed for hotspot gate bypass. It does NOT inject architect investigation content into SPAWN_CONTEXT.md. Content flows through kb context search only.

### Test 5: Trace spawn pipeline information injection

Read `pkg/spawn/worker_template.go` and `pkg/spawn/kbcontext.go` — investigations are injected as title + path in the "Related Investigations" section. Full investigation content is never injected into SPAWN_CONTEXT. Agents must self-discover and read the investigation file.

### Test 6: Agent phase comment timeline

From beads comments:
- Comment #2 (11:47): "Phase: Planning - Reading harness code to understand both tools"
- Comment #3 (11:49): "Phase: Planning - Designing pkg/harness extraction and cmd/harness standalone binary"
- Comment #4 (11:50): "Phase: Implementing - Creating pkg/harness/ package and cmd/harness/ standalone binary"

The agent went from Planning to a concrete in-repo design in **2 minutes**, then to implementation in **3 minutes**. No evidence it read the architect investigation.

---

## What I Observed

### Five-layer failure chain:

**Layer 1 (Strongest signal — issue description framing):**
The issue description said "consolidate", "absorb", "thin wrapper" — all verbs that imply bringing code together within a single repo. The architect designed a SEPARATE repo. The issue framing directly contradicted the architect's design without referencing it.

**Layer 2 (KB context surfaced pointer, not content):**
KB context DID return the architect investigation as a pointer in the "INVESTIGATIONS" section. But it was one of 6+ entries, listed only as title + path. The critical design decision (new repo, zero orch-go dependency) was invisible without reading the file.

**Layer 3 (No architect content injection mechanism):**
The `--architect-ref` flag is access control (hotspot gate bypass), not information injection. There is no spawn pipeline mechanism to detect "an architect investigation exists for this domain" and inject its key conclusions into SPAWN_CONTEXT.

**Layer 4 (No feature-impl checkpoint for architect designs):**
The feature-impl skill has no checkpoint like "if an architect investigation exists in kb context results, read it before designing your approach." The agent followed the strongest signal (issue description) and skipped the weaker signal (investigation pointer).

**Layer 5 (Rushed planning phase):**
The agent entered implementation 3 minutes after starting. With 2+ minutes of planning, the agent designed an approach that matched the issue description's framing without exploring the architect investigation. A longer planning phase might have surfaced the conflict.

### Counter-evidence:
The issue was created by `dylanconlin` 16.5 hours after the architect committed. The issue description itself was the primary source of the contradiction — this suggests the issue was written WITHOUT referencing the architect's design, or was written to describe a different (internal consolidation) approach that was later recognized as conflicting.

---

## Model Impact

- [x] **Extends** model with: New failure mode — "Architect Design Bypass via Issue Framing" — where an issue description's framing overrides a prior architect design that was surfaced only as a kb context pointer. This is distinct from existing failure modes in the taxonomy because the knowledge WAS in the system (committed to .kb/, surfaced by kb context) but was drowned out by the issue description's stronger framing signal.

### Proposed additions to failure mode taxonomy:

**Class: Knowledge-Feedback Inversion**
When the issue description (strongest signal in spawn context) contradicts a prior architect design (weak signal as kb context pointer), the agent follows the issue description. The knowledge feedback loop technically works (kb context found the investigation) but the signal hierarchy inverts the intended information flow (architect → issue → agent should be: architect design constrains issue framing).

**Contributing factors:**
1. Issue descriptions are injected as task framing (high salience)
2. KB context investigations are listed as title + path (low salience)
3. No mechanism to elevate architect designs above other investigation pointers
4. No feature-impl checkpoint to verify architect alignment before implementing
5. Time pressure (13 minutes total session) reduces exploration likelihood

**Prevention vectors:**
1. **Issue-level:** Reference architect issue in issue description ("Per orch-go-sb13k design: separate repo")
2. **Spawn-level:** When kb context returns architect investigations, inject their D.E.K.N. summary (not just title + path)
3. **Skill-level:** Add feature-impl planning checkpoint: "If kb context includes architect investigations, read them before designing approach"
4. **System-level:** `orch work` should detect when a closed architect issue covers the same domain and inject its conclusions

---

## Notes

- The 68gcy workspace was never committed to git (SPAWN_CONTEXT.md lives only in the workspace directory). The exact SPAWN_CONTEXT was reconstructed from keyword extraction code analysis and kb context simulation.
- The revert (b305e6d35) was committed shortly after, suggesting quick detection by the orchestrator.
- This failure mode is structurally similar to the "Architect spawns must consider tool placement before designing implementation — 'Where should this live?' is Fork 0" constraint already in kb context — but operating in reverse (feature-impl agent ignoring placement decision rather than architect failing to make one).
