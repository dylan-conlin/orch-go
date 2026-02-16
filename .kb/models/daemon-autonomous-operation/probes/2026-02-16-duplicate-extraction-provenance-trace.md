# Probe: Duplicate Extraction Issue Provenance Trace

**Model:** Daemon Autonomous Operation
**Date:** 2026-02-16
**Status:** Complete

---

## Question

What was the **upstream source** creating 9+ duplicate "Extract spawn flags phase 1: --mode" issues? The model claims duplicate spawns are caused by "Spawn latency" (issue labeled `triage:ready` at poll N, daemon spawns, but spawn hasn't transitioned issue to `in_progress` by poll N+1). Is this the full picture, or is there a deeper mechanism?

**Model claims being tested:**

> ### 2. Duplicate Spawns
>
> **Root cause:** Spawn latency. Issue labeled `triage:ready` at poll N, daemon spawns, but spawn hasn't transitioned issue to `in_progress` by poll N+1.
>
> **Fix:** Spawn deduplication via tracking. Track spawned beads IDs in memory, skip on subsequent polls until status confirms transition.

---

## What I Tested

1. **Searched beads issues.jsonl** for all "Extract spawn flags" issues, extracting their IDs, creation timestamps, titles, dependencies, and close reasons.

2. **Traced the parent issue** by searching for issues that depend on the extraction duplicates:
```bash
grep -E '"depends_on_id":"orch-go-(l8k2|kzqq|pg9l|cu0r|xy7n|95uh|p6k6|m8u7|a2li|ahx8)"' .beads/issues.jsonl
```
Found `orch-go-hcrg` ("spawn flags phase 1: --mode validation, advisory infra detection, fix --backend override") as the parent issue, with dependencies on ALL extraction duplicates.

3. **Read daemon extraction code** at `pkg/daemon/extraction.go`:
   - `GenerateExtractionTask()` (line 117-130): Creates title with format "Extract [concern] from [file] into [pkg]. Pure structural extraction..."
   - `inferConcernFromIssue()` (line 134-161): Strips "extract " prefix from titles starting with that verb
   - `DefaultCreateExtractionIssue()` (line 246-272): Calls `bd create` with `triage:ready` label and adds blocking dependency

4. **Read daemon Once() method** at `pkg/daemon/daemon.go:820-857`:
   - Line 826: `CheckExtractionNeeded(issue, d.HotspotChecker)` runs on EVERY issue picked up
   - Line 832: `createFunc(extraction.ExtractionTask, issue.ID)` creates new issue and blocks parent
   - Line 847-854: Replaces issue with extraction issue for spawning

5. **Verified spawn_cmd.go file size:**
```bash
$ wc -l cmd/orch/spawn_cmd.go
    2202 cmd/orch/spawn_cmd.go
```
Still 2202 lines — well above 1500-line CRITICAL threshold, even after l8k2's extraction.

6. **Reconstructed the dependency chain** from issues.jsonl:
   - `orch-go-hcrg` depends on: l8k2, kzqq, pg9l, p6k6, a2li, ahx8 (direct extraction deps)
   - `orch-go-p6k6` depends on `95uh` (cascading extraction)
   - `orch-go-95uh` depends on `xy7n` (cascading extraction)
   - `orch-go-xy7n` depends on `cu0r` and `m8u7` (cascading extraction)

---

## What I Observed

### The Parent Issue: orch-go-hcrg

`orch-go-hcrg` ("spawn flags phase 1: --mode validation, advisory infra detection, fix --backend override") was created by an orchestrator agent on Feb 15 23:03:45 as a follow-up from design issue `orch-go-82eg`. It mentions `cmd/orch/spawn_cmd.go` in its description field: "Files: cmd/orch/spawn_cmd.go (determineSpawnBackend, isInfrastructureWork)".

### Two Amplification Mechanisms

**Mechanism 1: Recurring extraction from parent (hcrg)**

Each time an extraction issue closes, the parent (hcrg) becomes unblocked and reappears in `bd ready`. The daemon runs `CheckExtractionNeeded()` again, finds spawn_cmd.go is STILL >1500 lines, and creates ANOTHER extraction issue. This cycle repeated 7+ times:

| Creation Time | Issue ID | Triggered By |
|--------------|----------|--------------|
| Feb 15 23:07 | l8k2 | hcrg picked up → extraction created |
| Feb 15 23:21 | kzqq | l8k2 closed → hcrg unblocked → new extraction |
| Feb 15 23:46 | pg9l | kzqq closed → hcrg unblocked → new extraction |
| Feb 16 00:12 | p6k6 | pg9l closed → hcrg unblocked → new extraction |
| Feb 16 09:00 | a2li | batch close → hcrg unblocked → new extraction |
| Feb 16 11:28 | ahx8 | a2li closed → hcrg unblocked → new extraction |

**Mechanism 2: Recursive extraction from extraction issues (cascading)**

Extraction issues themselves mention `cmd/orch/spawn_cmd.go` in their titles. When the daemon picks up an extraction issue, `InferTargetFilesFromIssue()` parses the filename from the title, `FindCriticalHotspot()` matches it, and a NEW extraction issue is created as a child. This creates cascading chains:

```
p6k6 → creates 95uh → creates xy7n → creates cu0r
```

Evidence: title concatenation. `inferConcernFromIssue()` strips the "extract " prefix from the parent extraction title, then wraps it in a new "Extract [concern] from [file]..." template:

- **l8k2** (clean): "Extract spawn flags phase 1: --mode from cmd/orch/spawn_cmd.go into pkg/orch/. Pure structural extraction — no behavior changes."
- **95uh** (2x): "Extract spawn flags phase 1: --mode from cmd/orch/spawn_cmd.go into pkg/orch/. **pure structural extraction — no behavior changes. from cmd/orch/spawn_cmd.go into pkg/orch/.** Pure structural extraction — no behavior changes."
- **xy7n** (3x): Title repeated 3 times
- **cu0r** (4x): Title repeated 4 times

### The Complete Provenance Chain

```
orch-go-82eg (design: rethink spawn flag semantics)
  └→ closes Feb 15 23:04
     └→ orch-go-hcrg (phase 1 impl) becomes unblocked, appears in bd ready
        └→ daemon picks hcrg, extraction creates l8k2 (Feb 15 23:07)
           └→ l8k2 does actual work (commit 41f5a781, Feb 15 23:10)
           └→ l8k2 closes (23:20) → hcrg unblocked → daemon creates kzqq (23:21)
              └→ kzqq finds work done, closes (23:46) → hcrg unblocked → creates pg9l (23:46)
                 └→ pg9l closes (00:12) → hcrg unblocked → creates p6k6 (00:12)
                    └→ p6k6 also triggers recursive extraction → creates 95uh (00:12)
                       └→ 95uh triggers recursive extraction → creates xy7n (00:13)
                          └→ xy7n triggers recursive extraction → creates cu0r (00:13)
                    └→ [later] creates m8u7 (08:38), a2li (09:00), ahx8 (11:28)
```

### Why the Model's Fix Was Insufficient

The model's claimed fix (spawn dedup via SpawnedIssueTracker) only prevents the daemon from spawning the SAME issue ID twice. It doesn't prevent:

1. **New issues with same content**: Each `bd create` call creates a new issue with a new ID. The tracker only tracks by ID, so it never detects the duplicate content.
2. **Extraction re-triggering**: The extraction logic has no memory. It doesn't check "did I already create an extraction for this parent?" or "did the file size actually decrease?"
3. **Recursive self-triggering**: Extraction issues contain the critical filename in their titles, causing them to trigger more extraction.

---

## Model Impact

- [x] **Contradicts** invariant: "Duplicate Spawns - Root cause: Spawn latency" — Spawn latency is only ONE cause. The dominant mechanism here is **extraction logic without convergence** (no termination condition) combined with **recursive self-triggering** (extraction issues mention the same critical file).

- [x] **Extends** model with: **Three bugs in the extraction subsystem that amplify into unbounded issue creation:**
  1. **No extraction memory**: Daemon doesn't track "already created extraction for file X from parent Y". Each poll cycle that picks up hcrg creates a fresh extraction.
  2. **No convergence check**: Extraction creates new issues regardless of whether the file actually decreased in size. spawn_cmd.go remained at 2202 lines throughout, so every check returned "extraction needed."
  3. **Recursive self-triggering**: `InferTargetFilesFromIssue()` parses file paths from issue titles. Extraction issue titles contain the critical filename, so they trigger more extraction when the daemon processes them. Combined with `inferConcernFromIssue()` stripping the "extract " prefix, this creates progressively concatenated titles.

- [x] **Extends** model with: **The upstream source is the daemon's extraction path (`pkg/daemon/extraction.go`), triggered by orch-go-hcrg** — a task created by an orchestrator agent as a follow-up from design issue orch-go-82eg. The orchestrator did nothing wrong; the bug is entirely in the daemon's extraction logic lacking termination conditions.

---

## Notes

- The content-aware dedup fix (orch-go-e0o3, commit d29a3c8a) addresses Mechanism 2 (recursive) by checking titles at spawn time. But it doesn't address Mechanism 1 (recurring from parent) unless the parent's extraction task generates the exact same title each time.
- The `bd create` title dedup fix (orch-go-bj0v) addresses the issue at creation time, preventing duplicate issues from being created. This is the most effective fix.
- **Missing fix**: The extraction logic itself needs a convergence check — if the file is still >1500 lines after an extraction issue was already created and closed, it should NOT create another extraction. Possible approaches: (a) track extraction history per file, (b) check if an extraction was already attempted for this file in recent history, (c) only create extraction if the file size decreased since last check.
- spawn_cmd.go at 2202 lines will continue triggering extraction for ANY issue that mentions it, even with dedup fixes, unless the extraction logic gains a convergence condition.
