# Empirical Audit: Code Displaced by Governance Hook Denials

**TLDR:** Found 2 confirmed displaced code instances (115 lines), 1 probable displacement from documentation mismatch (182 lines), and 5 sessions where agents were fully or partially blocked without displacing code (they escalated instead). A key amplifier: worker-base skill claims ALL of `pkg/verify/*` is protected, but the hook only blocks 2 specific files.

**Status:** Complete

## D.E.K.N. Summary

- **Delta:** 2 confirmed displaced code instances totaling 115 lines (concurrency gate + governance check), 1 probable (182 lines, artifact validator in pkg/completion/ instead of pkg/verify/), plus a documentation mismatch that overstates the protected zone
- **Evidence:** git blame, SYNTHESIS.md from blocked sessions, beads CONSTRAINT comments, governance hook regex vs worker-base skill text comparison
- **Knowledge:** Hard deny hooks have two failure modes: (1) code displaced to wrong location, (2) agent blocked entirely and work deferred. The documentation-enforcement mismatch creates a phantom protected zone wider than actual enforcement.
- **Next:** Architect session to fix documentation mismatch (narrow `pkg/verify/*` claim to actual protected files), add redirect hints to deny message, consider migrating displaced code to correct locations

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| .kb/investigations/2026-03-19-design-analysis-hard-deny-hooks-attractors.md | Extends | yes | none |
| .kb/models/claude-code-agent-configuration/probes/2026-03-12-probe-hook-infrastructure-audit.md | Extends | yes | none |
| .orch/workspace/archived/og-debug-resolve-self-review-13mar-9a37/SYNTHESIS.md | Extends | yes | none (first to note documentation mismatch) |

## Question

How much code exists in the wrong location because governance hooks blocked agents from writing to protected files?

## Findings

### Finding 1: Starting approach
Read governance hook (`~/.orch/hooks/gate-governance-file-protection.py`) to identify exact protected patterns, then scanned all non-protected locations for gate-like and verify-like logic.

### Finding 2: Confirmed displacement — Concurrency gate in spawn_preflight.go

**File:** `pkg/orch/spawn_preflight.go` lines 10-12 (type def) + 30-41 (gate logic)
**Lines displaced:** 15 lines of gate logic
**Agent:** scs-sp-8dm (commit `61e12c298`)
**Should be in:** `pkg/spawn/gates/concurrency.go` (file does not exist)
**Evidence:** Commit message explicitly says "reinstates concurrency gate for manual spawns"; beads CONSTRAINT comment documents the block; code pattern (check condition → log decision → return error) matches gates/{triage,hotspot,agreements,question}.go exactly.

### Finding 3: Confirmed displacement — Governance path checking in pkg/orch/

**File:** `pkg/orch/governance.go` (87 lines) + `pkg/orch/governance_test.go` (110 lines)
**Lines displaced:** ~100 lines of gate logic (excluding boilerplate)
**Agent:** orch-go-enwt2 (commit `7102b4722`)
**Should be in:** `pkg/spawn/gates/governance.go`
**Evidence:** This is a spawn-time gate that checks task descriptions against protected paths and warns. It follows the exact gate pattern: receives inputs, evaluates conditions, returns result struct, logs decisions. It's called from `RunPreFlightChecks` alongside other `gates.Check*` functions. The only reason it's in `pkg/orch/` instead of `pkg/spawn/gates/` is that the worker could not write to the protected directory.

### Finding 4: Probable displacement — Artifact validator in pkg/completion/

**File:** `pkg/completion/artifact.go` (182 lines)
**Agent:** orch-go-lqiel (commit `f45b40a12`)
**Should probably be in:** `pkg/verify/artifact.go` or `pkg/verify/completion_artifact.go`
**Evidence:** This is a completion verification gate — it validates COMPLETION.yaml artifacts and is called from `complete_verification.go` alongside other `verify.*` functions. The worker-base skill says `pkg/verify/*` is protected, but the actual hook only protects `precommit.go` and `accretion.go`. The worker likely avoided `pkg/verify/` entirely because the skill told them to. This is **documentation-driven displacement** — the code could have been written to pkg/verify/ safely, but the perceived protection zone was wider than actual enforcement.

### Finding 5: Documentation-enforcement mismatch (amplifier)

**Worker-base skill** (`skills/src/shared/worker-base/SKILL.md` line 56):
> "Files in `pkg/spawn/gates/*` and `pkg/verify/*` cannot be modified by workers — governance hooks will block the edit."

**Actual hook** (`gate-governance-file-protection.py` lines 53-55):
```python
re.compile(r'pkg/verify/precommit\.go$'),
re.compile(r'pkg/verify/accretion\.go$'),
```

The skill claims the ENTIRE `pkg/verify/` directory is protected. The hook only protects 2 specific files. This creates a phantom protected zone — workers avoid 67 files in `pkg/verify/` that they could safely modify. Agent og-debug-resolve-self-review-13mar-9a37 was the first to notice this discrepancy (SYNTHESIS.md line 81).

### Finding 6: Sessions blocked without displacement (correct escalation)

Five sessions hit governance blocks and escalated correctly instead of displacing code:

| Session | Outcome | What was blocked | How they responded |
|---|---|---|---|
| og-debug-populate-hotspot-bypass-11mar-9327 | **blocked** | pkg/spawn/gates/hotspot.go, hotspot_test.go | Produced VERIFICATION_SPEC.yaml with exact patches, escalated to orchestrator |
| og-debug-stop-hook-escape-10mar-0966 | **partial** | ~/.orch/hooks/enforce-phase-complete.py | Fixed schema in pkg/hook/, produced patch for hook, escalated |
| og-debug-claude-print-output-11mar-be96 | **partial** | ~/.orch/hooks/enforce-phase-complete.py | Wrote tests in cmd/orch/, produced patch for hook, escalated |
| og-feat-accretion-gates-block-17mar-9f1a | **partial** | All accretion gate files (spawn/gates + verify/) | Produced decision doc + exact code changes for orchestrator |
| og-debug-fix-concurrency-cap-27feb-8772 | partial | pkg/spawn/gates/concurrency.go | Led to scs-sp-8dm displacement (Finding 2) |

### Finding 7: Not displaced — Explain-back gate in pkg/orch/completion.go

The subagent identified `RunExplainBackGate` (82 lines) and `RecordGate2Checkpoint` (18 lines) in `pkg/orch/completion.go` as potentially displaced. **This is NOT displacement** — these were intentionally extracted from `cmd/orch/` via deliberate refactoring (commits `db0d39755`, `caa5e3dd1`, `5a8227d6c`). The explain-back gate was not created by a governance-blocked worker. It calls `verify.FormatExplainBack()` from pkg/verify/, demonstrating that the developer who wrote it had access to pkg/verify/.

### Finding 8: Not displaced — Gap gating in spawn_kb_context.go

`checkGapGating` (16 lines in `pkg/orch/spawn_kb_context.go`) is a gate function but it's tightly coupled to KB context generation. It was extracted from a larger file via architect-driven refactoring (`592e0984c`), not displaced by a governance block. Its placement in the KB context flow is architecturally defensible.

### Finding 9: complete_verification.go — gate orchestration in cmd/orch/

`cmd/orch/complete_verification.go` (365 lines) contains verification gate orchestration code. While it calls into `pkg/verify/` and `pkg/completion/` for the actual gate logic, the orchestration itself (checkpoint enforcement, skip filters, liveness checks) lives in cmd/orch/. This is **not displacement** — it's normal Cobra command-level orchestration. The gates it calls are in the right packages; the command-level wiring should be in cmd/orch/.

## Test Performed

1. Read actual governance hook patterns (`gate-governance-file-protection.py` GOVERNANCE_PATTERNS list)
2. Compared against protected paths claimed in worker-base skill
3. For each candidate displaced file: checked git log to determine authorship (worker agent vs orchestrator), checked whether the file follows gate patterns (condition check → decision → result struct), verified whether the canonical location was actually protected
4. Searched SYNTHESIS.md files for "governance", "blocked", "CONSTRAINT" patterns across 1786+ workspaces
5. Checked beads issues for CONSTRAINT comments referencing governance
6. Verified false positives by checking git history for intentional refactoring vs. governance-forced displacement

## Conclusion

**~115-297 lines of code are in wrong locations due to governance hook enforcement.**

- **Confirmed displaced:** 115 lines across 2 instances (concurrency gate in spawn_preflight.go, governance check in pkg/orch/governance.go)
- **Probable displaced (documentation-driven):** 182 lines (artifact validator in pkg/completion/ instead of pkg/verify/)
- **Not displaced (false positives):** explain-back gate, gap gating, complete_verification.go orchestration — all intentionally placed

**The bigger issue is the documentation mismatch.** The worker-base skill tells agents that ALL 69 files in `pkg/verify/` are protected. The hook only protects 2. This phantom protection zone is a stronger displacement force than the actual hook — it prevents workers from writing to pkg/verify/ even when the hook would allow it.

**Recommended actions:**
1. Fix documentation mismatch: Change worker-base skill from `pkg/verify/*` to list the 2 actually-protected files
2. Migrate confirmed displaced code: Move concurrency gate to `pkg/spawn/gates/concurrency.go`, move governance check to `pkg/spawn/gates/governance.go` (requires orchestrator session)
3. Consider migrating artifact validator: Move `pkg/completion/artifact.go` to `pkg/verify/completion_artifact.go`
4. Add redirect hints to governance hook denial message (per design analysis investigation recommendation)
