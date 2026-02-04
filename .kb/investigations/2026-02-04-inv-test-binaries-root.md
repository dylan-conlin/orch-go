## Summary (D.E.K.N.)

**Delta:** Test binaries at root are created by MANUAL `go test -c ./...` commands, not by any automated script or CI - beads issue orch-go-21212 documented this as the repro step.

**Evidence:** Ran `go test -c ./...` -> created 38 binaries. Searched entire codebase for `go test -c` -> only found in beads issue description. pkg/verify/build_verification.go uses `go test -run=^$` (no artifacts).

**Knowledge:** `go test -c` compiles test binaries to CWD; `go test -run=^$` compiles without artifacts. Issue repro steps taught humans/agents the wrong command.

**Next:** Add quick constraint to prevent future use of `go test -c`. No code changes needed.

**Authority:** implementation - Documentation fix within existing patterns

---

# Investigation: Test Binaries Root

**Question:** What creates *.test files at project root despite gitignore fix?

**Started:** 2026-02-04
**Updated:** 2026-02-04
**Owner:** Worker agent (orch-go-21273)
**Phase:** Complete
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## Findings

### Finding 1: No automated source of `go test -c` exists

**Evidence:** Searched entire codebase:
- `rg "go test -c" --type go` -> no results
- `rg "test -c" Makefile*` -> only cleanup (`rm -f *.test`)
- `rg "go test -c" scripts/` -> no results
- CI uses bloat-check but no test compilation command

**Source:** Makefile, pkg/verify/build_verification.go

**Significance:** The binaries are NOT created by automated scripts/CI. Source is manual execution.

### Finding 2: Beads issue orch-go-21212 documents `go test -c ./...` as repro step

**Evidence:** `.beads/issues.jsonl` contains:
"repro":"Run 'go test -c ./...' in orch-go root"

**Source:** .beads/issues.jsonl

**Significance:** This is the teaching vector - humans/agents following the issue's repro step learned the wrong command.

### Finding 3: Build verification gate uses the correct command

**Evidence:** pkg/verify/build_verification.go:136-138 uses:
`exec.Command("go", "test", "-run=^$", "./...")`

The `-run=^$` flag compiles all code but creates NO artifacts.

**Source:** pkg/verify/build_verification.go:136-138

**Significance:** Automated verification already uses correct approach. Only manual invocations create binaries.

### Finding 4: Test confirmed - `go test -c ./...` creates 38 binaries

**Evidence:** 
```
$ rm -f *.test && go test -c ./...
$ ls *.test | wc -l
38
```

**Source:** Direct test in project root

**Significance:** Confirms the mechanism. Each package creates {package}.test binary in CWD.

---

## Synthesis

**Answer:** No script, CI, or automated process runs `go test -c`. The binaries appear when humans/agents manually run `go test -c ./...` following the repro step in beads issue orch-go-21212. The verification gate in pkg/verify/build_verification.go correctly uses `go test -run=^$ ./...` which compiles without creating artifacts.

---

## Structured Uncertainty

**Tested:**
- `go test -c ./...` creates 38 binaries at root (verified: ran command)
- `go test -run=^$ ./...` does NOT create binaries (verified: checked gate code)
- No automated source in codebase (verified: rg search)
- Beads issue documents the command (verified: checked issues.jsonl)

**Untested:**
- Whether agents read issue descriptions (assumed, not observed)

---

## Recommendations

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add kb quick constraint | implementation | Knowledge externalization |

**Recommended:** Add constraint: "Use `go test -run=^$` instead of `go test -c` for compilation checks"

No code changes needed - the issue is documentation, not implementation.

---

## References

**Commands Run:**
```bash
rg "go test -c" --type go  # no results
rm -f *.test && go test -c ./... && ls *.test | wc -l  # 38
```

**Investigation History:**
- 2026-02-04: Started, found beads issue with repro step, tested, concluded
