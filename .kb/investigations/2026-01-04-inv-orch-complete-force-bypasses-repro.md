## Summary (D.E.K.N.)

**Delta:** `--force` was bypassing repro verification for bugs, defeating the Bug Reproducibility Gates epic (jjd4).

**Evidence:** Code at main.go:936 had condition `if !completeForce && !completeSkipReproCheck` - --force disabled repro gate.

**Knowledge:** Repro verification is critical for bug fixes and should always run. --force should only skip phase/liveness verification.

**Next:** Close. Implementation complete, tests passing.

---

# Investigation: Orch Complete Force Bypasses Repro

**Question:** Why does `orch complete --force` bypass repro verification for bugs, and how to fix it?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: --force condition gated repro verification

**Evidence:** Line 936 in cmd/orch/main.go had:
```go
if !completeForce && !completeSkipReproCheck {
```
This meant `--force` would completely bypass the repro verification block.

**Source:** cmd/orch/main.go:936

**Significance:** This defeated the purpose of the Bug Reproducibility Gates epic (jjd4). The repro gate should be mandatory for bug issues.

---

### Finding 2: --force was described as "skip phase verification" but did more

**Evidence:** Flag description said "Skip phase verification" but it actually skipped:
1. Phase verification (line 851)
2. Liveness check (line 890)
3. Repro verification (line 936)

**Source:** cmd/orch/main.go:230, 851, 890, 936

**Significance:** The documentation and behavior were misaligned. Users might not realize `--force` was bypassing critical bug verification.

---

### Finding 3: Explicit --skip-repro-check flag already exists

**Evidence:** There's already a `--skip-repro-check` flag (line 235) with a required `--skip-repro-reason` flag for explicit bypass with documentation.

**Source:** cmd/orch/main.go:235-236

**Significance:** The proper bypass mechanism already exists. `--force` should not also bypass repro verification.

---

## Synthesis

**Key Insights:**

1. **Separation of concerns** - `--force` should skip procedural checks (phase, liveness), not substantive gates (repro verification).

2. **Explicit bypass is better** - `--skip-repro-check --skip-repro-reason` provides documented, auditable bypass.

3. **Documentation must match behavior** - Updated help text and error messages to reflect actual behavior.

**Answer to Investigation Question:**

The bug was caused by `--force` being used as a catch-all bypass. The fix removes `!completeForce` from the repro verification condition, making repro verification mandatory for bugs regardless of `--force`. Users who need to bypass repro verification must explicitly use `--skip-repro-check --skip-repro-reason`.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code builds successfully (verified: go build passed)
- ✅ All existing tests pass (verified: go test ./cmd/orch/... passed)
- ✅ --force no longer appears in repro condition (verified: code change at line 936)

**What's untested:**

- ⚠️ End-to-end bug completion flow (would require actual bug issue with repro)

**What would change this:**

- Finding would be wrong if there's a legitimate reason --force should bypass repro

---

## Implementation Recommendations

### Recommended Approach ⭐

**Remove !completeForce from repro gate** - Keep repro verification mandatory for bugs

**Why this approach:**
- Preserves purpose of Bug Reproducibility Gates epic
- Explicit bypass already exists via --skip-repro-check
- Minimal code change

**Implementation sequence:**
1. Remove `!completeForce &&` from line 936 condition
2. Update help text to clarify --force behavior
3. Update error messages to not suggest --force for repro bypass

---

## References

**Files Modified:**
- cmd/orch/main.go - Main fix and documentation updates
- cmd/gendoc/main.go - Documentation generation updates

**Commands Run:**
```bash
# Build verification
go build -o /dev/null ./cmd/orch/...

# Test verification
go test ./cmd/orch/...
```

---

## Investigation History

**2026-01-04:** Investigation started and completed
- Fixed the issue by removing `!completeForce` from repro verification condition
- Updated help text, flag descriptions, and error messages
- All tests passing
