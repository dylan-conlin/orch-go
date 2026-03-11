# Session Synthesis

**Agent:** og-inv-investigate-creates-empty-11mar-ca17
**Issue:** orch-go-hxsd1
**Outcome:** success

---

## Plain-Language Summary

Investigated what creates empty model stubs in `.kb/models/` and found that no empty stubs currently exist — all 37 models are fully filled. Three creation vectors were identified: `kb create model` (without `--from`) produces scaffold files with bracket-enclosed placeholder text, `kb create model --from` fills some but not all placeholders, and agents can write model files directly. The risk is that an agent dies or abandons work after creating a scaffold but before filling it, leaving a committed stub. To prevent this, a model-stub pre-commit gate was implemented (`orch precommit model-stub`) that detects 7 template placeholder patterns in staged `.kb/models/*/model.md` files and blocks the commit. The gate follows the same architecture as the existing accretion and knowledge gates. Wiring it into the pre-commit hook script requires orchestrator action (governance-protected file).

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace directory.

Key outcomes:
- 17 tests pass for model-stub detection (8 integration, 3 unit, 6 path-matching, 1 formatting)
- `orch precommit model-stub` command registers and runs
- Full build compiles

---

## TLDR

Investigated empty model stub creation vectors, found zero current stubs but three potential creation paths. Implemented `orch precommit model-stub` gate with 17 passing tests to block committing unfilled model templates. Gate needs wiring into pre-commit hook by orchestrator (governance file).

---

## Delta (What Changed)

### Files Created
- `pkg/verify/model_stub_precommit.go` - Pre-commit gate that checks staged model.md files for template placeholders
- `pkg/verify/model_stub_precommit_test.go` - 17 tests covering detection, path matching, formatting, and edge cases
- `.kb/models/knowledge-physics/probes/2026-03-11-probe-empty-model-stub-creation-vectors.md` - Probe documenting creation vectors and findings

### Files Modified
- `cmd/orch/precommit_cmd.go` - Added `model-stub` subcommand to `orch precommit`
- `.kb/models/knowledge-physics/model.md` - Merged probe findings (invariant 1 updated, probe reference added)

---

## Evidence (What Was Observed)

- All 37 model.md files in .kb/models/ contain substantial content (3K-26K words each), zero stubs
- `kb create model` without `--from` produces 7 detectable placeholder patterns
- Git history shows 11 commits created 41 models across Feb 25 - Mar 10, 2026
- Model creation rate (~41 total) is far lower than investigation creation rate (~1,166+)
- Pre-commit hook infrastructure already supports extensible gates via `orch precommit` subcommands

### Tests Run
```bash
go test -v -count=1 -run "TestCheckStagedModelStubs$|TestIsModelStubCandidate$|TestFormatStagedModelStubError$|TestFindPlaceholders$" ./pkg/verify/
# PASS: 17 tests passing (0.704s)

go build ./cmd/orch/
# Build OK

go run ./cmd/orch/ precommit model-stub --help
# Command registered and functional
```

---

## Architectural Choices

### Gate as blocking pre-commit check, not advisory warning
- **What I chose:** Hard block (exit 1) when placeholder patterns detected
- **What I rejected:** Advisory warning (like the 800-line accretion threshold)
- **Why:** Empty model stubs provide zero value — unlike large files which may be legitimate, an unfilled template is always a mistake
- **Risk accepted:** Agent needs to use `FORCE_MODEL_STUB=1` if legitimately committing a scaffold for later filling

### Check both new AND modified model files
- **What I chose:** `--diff-filter=ACM` (Added, Copied, Modified)
- **What I rejected:** `--diff-filter=A` (Added only, like the knowledge gate)
- **Why:** A model could be reset to template state during a refactoring session, not just newly created

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/knowledge-physics/probes/2026-03-11-probe-empty-model-stub-creation-vectors.md` - Creation vector analysis

### Constraints Discovered
- `scripts/pre-commit-exec-start-cleanup.sh` is governance-protected — workers cannot wire new gates into it
- The pre-commit script exits early when no Go files are staged, meaning knowledge gates for .md files must be placed outside the Go-only guard
- `isModelFile` function already exists in `probe_model_merge.go` — had to use `isModelStubCandidate` to avoid name collision

---

## Next (What Should Happen)

**Recommendation:** close (after orchestrator wires the gate into pre-commit hook)

### Orchestrator Action Required
The `scripts/pre-commit-exec-start-cleanup.sh` governance file needs updating to:
1. Move Go-only gates inside a `if [ -n "$STAGED_GO" ]` guard
2. Add model-stub gate outside the guard, triggered when `.kb/models/*/model.md` files are staged

Already reported via: `bd comments add orch-go-hxsd1 "DISCOVERED: governance file scripts/pre-commit-exec-start-cleanup.sh needs update..."`

---

## Unexplored Questions

- Should the knowledge gate (`orch precommit knowledge`) also run outside the Go-only guard? Currently it only runs when Go files are staged, which means .md-only commits skip it too.
- Should `kb create model` itself refuse to commit the scaffold, or is the pre-commit gate sufficient?

---

## Friction

- `governance`: Pre-commit hook script is governance-protected, so the gate implementation is complete but not wired. This is correct behavior (control plane immutability) but means the task requires orchestrator follow-up.

---

## Session Metadata

**Skill:** investigation
**Workspace:** `.orch/workspace/og-inv-investigate-creates-empty-11mar-ca17/`
**Probe:** `.kb/models/knowledge-physics/probes/2026-03-11-probe-empty-model-stub-creation-vectors.md`
**Beads:** `bd show orch-go-hxsd1`
