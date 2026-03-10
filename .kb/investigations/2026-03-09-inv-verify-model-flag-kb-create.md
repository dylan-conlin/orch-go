## Summary (D.E.K.N.)

**Delta:** The `--model` flag on `kb create investigation` IS shipped and enforced — the gate was implemented in kb-cli commit 8d2a424, with 13 passing tests including 4 gate-specific tests.

**Evidence:** Ran `kb create investigation test-no-model` (failed with required error), `kb create investigation test --model spawn-architecture` (succeeded with Model field), all 13 kb-cli investigation tests pass.

**Knowledge:** The prior issue orch-go-vfd6v was closed at Planning phase but the work WAS completed (commit 8d2a424 references it). However, 6+ skill docs still reference `kb create investigation` without `--model`, creating agent confusion. The gate enforcement is solid but documentation is stale.

**Next:** Create issue for stale skill docs that reference `kb create investigation` without `--model` flag. No code changes needed.

**Authority:** implementation - Stale docs are a cleanup task within existing patterns.

---

# Investigation: Verify --model flag on kb create investigation

**Question:** Is the `--model` flag on `kb create investigation` actually shipped and enforced, or was orch-go-vfd6v closed prematurely at Phase: Planning?

**Started:** 2026-03-09
**Updated:** 2026-03-09
**Owner:** Agent (orch-go-q3m3m)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** spawn-architecture

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| orch-go-vfd6v (beads issue) | confirms | yes | Issue closed at Planning, but code was shipped |

---

## Findings

### Finding 1: Gate enforcement IS shipped in kb-cli

**Evidence:** Running `kb create investigation test-no-model` without `--model` or `--orphan` produces:
```
Error: --model is required for investigations (links to a knowledge model).
Use --model <model-name> to couple this investigation to a model,
or --orphan to explicitly create without model coupling.
List available models: ls .kb/models/
```

Running `kb create investigation test --model spawn-architecture` succeeds and produces a file with `**Model:** spawn-architecture` on line 48.

Running `kb create investigation test --orphan` succeeds and produces a file without Model field and without Lineage section.

**Source:** kb-cli commit `8d2a424` ("feat: hard knowledge gate -- require --model on kb create investigation (orch-go-vfd6v)"), `kb-cli/cmd/kb/create.go` lines 1614-1622

**Significance:** The gate is a hard error, not a warning. Agents cannot create investigations without explicitly choosing model coupling or orphan status.

---

### Finding 2: 13 tests pass, including 4 gate-specific tests

**Evidence:** `go test ./cmd/kb/ -run "TestCreateInvestigation" -v` produces:
- `TestCreateInvestigationRequiresModelOrOrphan` - PASS
- `TestCreateInvestigationModelGateBypassedByOrphan` - PASS
- `TestCreateInvestigationModelGateSatisfiedByModel` - PASS
- `TestCreateInvestigationWithModelFlag` - PASS
- Plus 9 other investigation creation tests, all passing (0.013s total)

**Source:** `kb-cli/cmd/kb/create_test.go` (113 lines added in commit 8d2a424)

**Significance:** The enforcement is well-tested with edge cases covered. Not just a runtime check.

---

### Finding 3: Prior issue orch-go-vfd6v was closed at Planning but work WAS done

**Evidence:** `bd show orch-go-vfd6v` shows 2 comments, both at Planning phase. No "Phase: Complete" was ever reported. However, `git log --grep="orch-go-vfd6v"` in kb-cli shows commit 8d2a424 with the feature tagged to that issue. The issue was closed at 14:31, the commit was at 13:49 — the work was done before the issue was formally closed.

**Source:** `bd show orch-go-vfd6v`, `git log` in kb-cli

**Significance:** The closure was premature in terms of protocol (no Phase: Complete reported), but NOT premature in terms of work completion. The feature was shipped with tests before closure.

---

### Finding 4: Stale skill docs reference `kb create investigation` without `--model`

**Evidence:** Grep for `kb create investigation` in skills/ found 6+ locations missing `--model`:
- `skills/src/worker/codebase-audit/SKILL.md:48` - `kb create investigation "audit/dimension-description"` (no --model)
- `skills/src/worker/codebase-audit/SKILL.md:82` - same
- `skills/src/worker/codebase-audit/.skillc/SKILL.md:48,82` - same
- `skills/src/worker/ux-audit/SKILL.md:103` - `kb create investigation "audit/ux-{page-slug}"` (no --model)
- `skills/src/worker/investigation/SKILL.md:103,147` - `kb create investigation {slug}` (no --model)
- `skills/src/worker/investigation/.skillc/reference/template.md:3` - same

Note: `skills/src/worker/codebase-audit/.skillc/phases/common-overview.md:90` DOES include `--model`. The compiled SKILL.md overrides don't include it.

Also: `pkg/spawn/context.go:274` correctly includes `--model <model-name>` in the spawn context template.

**Source:** `grep -rn "kb create investigation" skills/ pkg/spawn/`

**Significance:** Agents using codebase-audit, ux-audit, or investigation skills from the compiled SKILL.md examples will hit the gate error. The spawn context template (context.go) is correct, but the skill reference docs are stale.

---

## Synthesis

**Key Insights:**

1. **Gate is shipped and enforced** - The `--model` flag requirement works as designed with a hard error and `--orphan` escape hatch. 4 dedicated tests validate the gate logic.

2. **Prior issue closure was procedurally premature but substantively correct** - orch-go-vfd6v had the code shipped (commit 8d2a424) but never reported Phase: Complete. The verification gap was in protocol adherence, not in feature delivery.

3. **Documentation drift creates agent friction** - 6+ skill docs still show `kb create investigation` without `--model`, which will cause agents to hit the gate error and waste cycles. The spawn context template (context.go) is correct.

**Answer to Investigation Question:**

Yes, the `--model` flag on `kb create investigation` IS shipped and enforced. The prior issue orch-go-vfd6v was closed at Phase: Planning (protocol violation) but the work was completed — commit 8d2a424 in kb-cli implements the hard gate with tests. The one gap is stale skill documentation that doesn't include `--model` in examples.

---

## Structured Uncertainty

**What's tested:**

- `kb create investigation` without `--model` or `--orphan` fails with descriptive error (verified: ran command)
- `kb create investigation --model spawn-architecture` succeeds and populates Model field (verified: ran command, read file)
- `kb create investigation --orphan` succeeds without Model field (verified: ran command, read file)
- 13 unit tests pass in kb-cli (verified: `go test ./cmd/kb/ -run TestCreateInvestigation -v`)
- Commit 8d2a424 exists in kb-cli referencing orch-go-vfd6v (verified: `git log --grep`)

**What's untested:**

- Whether agents spawned via codebase-audit or ux-audit skills hit the gate error in practice (likely yes, based on stale docs)
- Orphan rate impact since gate deployment (no metrics checked)

**What would change this:**

- If kb-cli binary on PATH is not the latest build (would miss the gate)
- If skillc deploy recompiles SKILL.md and overwrites the stale examples

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Update stale skill docs with --model examples | implementation | Fix within existing patterns, no architectural decisions needed |

### Recommended Approach

**Update skill docs** - Add `--model <model-name>` (or `--orphan`) to all `kb create investigation` examples in skill sources.

**Files to update:**
- `skills/src/worker/codebase-audit/.skillc/SKILL.md` (lines 48, 82)
- `skills/src/worker/codebase-audit/SKILL.md` (lines 48, 82)
- `skills/src/worker/ux-audit/SKILL.md` (line 103)
- `skills/src/worker/investigation/SKILL.md` (lines 103, 147)
- `skills/src/worker/investigation/.skillc/reference/template.md` (line 3)

Then run `skillc deploy` to recompile and deploy.

---

## References

**Files Examined:**
- `kb-cli/cmd/kb/create.go` (lines 826, 1614-1622) - Gate implementation
- `kb-cli/cmd/kb/create_test.go` - Gate tests
- `pkg/spawn/context.go:274` - Spawn context template (correctly includes --model)
- Multiple skill docs in `skills/src/` - Stale references

**Commands Run:**
```bash
# Test gate enforcement (no flags)
kb create investigation test-no-model
# Result: Error requiring --model or --orphan

# Test with --model
kb create investigation test-with-model --model spawn-architecture
# Result: Created .kb/investigations/2026-03-09-inv-test-with-model.md

# Test with --orphan
kb create investigation test-orphan --orphan
# Result: Created .kb/investigations/2026-03-09-inv-test-orphan.md

# Run kb-cli tests
cd ~/Documents/personal/kb-cli && go test ./cmd/kb/ -run "TestCreateInvestigation" -v
# Result: 13 tests pass (0.013s)

# Check prior issue
bd show orch-go-vfd6v
# Result: Closed with only Phase: Planning comments

# Check kb-cli git history
cd ~/Documents/personal/kb-cli && git log --oneline --grep="model" --grep="investigation" --all-match
# Result: 8d2a424 feat: hard knowledge gate (orch-go-vfd6v)
```

**Related Artifacts:**
- **Issue:** orch-go-vfd6v - Prior issue that implemented the gate (closed at Planning)
- **Commit:** kb-cli 8d2a424 - The gate implementation
