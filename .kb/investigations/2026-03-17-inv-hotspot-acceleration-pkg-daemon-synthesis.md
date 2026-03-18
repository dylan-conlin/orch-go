## Summary (D.E.K.N.)

**Delta:** synthesis_auto_create_test.go hotspot is a false positive — file born 2026-03-12, entire 404-line existence counted as 30d growth.

**Evidence:** `git log --diff-filter=A` shows single creation commit `77b8640` on 2026-03-12 with +404/-0. Zero subsequent commits.

**Knowledge:** This is the 4th false positive from the same detection pattern (file birth counted as growth). Prior: status_infra.go, pidlock.go, mock_test.go.

**Next:** No action needed. File is healthy (404 lines testing 263 lines of production code, 12 tests all passing, growth drivers exhausted).

**Authority:** implementation - False positive classification within established pattern, no architectural impact.

---

# Investigation: Hotspot Acceleration Pkg Daemon Synthesis

**Question:** Is pkg/daemon/synthesis_auto_create_test.go (+404 lines/30d, now 404 lines) a real hotspot that needs extraction?

**Started:** 2026-03-17
**Updated:** 2026-03-17
**Owner:** orch-go-2k4ba
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `5b0f99c0f` inv: status_infra.go hotspot false positive | confirms | yes | - |
| `78ae22f47` inv: pidlock.go hotspot false positive | confirms | yes | - |
| `245b16b0c` inv: mock_test.go hotspot false positive | confirms | yes | - |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Pattern matches exactly — file birth counted as growth.
**Conflicts:** None.

---

## Findings

### Finding 1: File born in single commit, zero subsequent growth

**Evidence:**
```
$ git log --diff-filter=A --format='%H %ai %s' -- pkg/daemon/synthesis_auto_create_test.go
77b864023 2026-03-12 22:06:37 -0700 feat: add daemon auto-synthesis for investigation clusters without models (orch-go-8qswb)

$ git log --format='%H %ai' --numstat -- pkg/daemon/synthesis_auto_create_test.go
77b864023 2026-03-12 22:06:37 -0700
404  0  pkg/daemon/synthesis_auto_create_test.go
```

Only 1 commit. All 404 lines added at birth. Zero growth since.

**Source:** `git log --diff-filter=A` and `git log --numstat`

**Significance:** The hotspot detector counts the entire file birth as "30-day growth." Since the file is only 5 days old and has had no modifications, the +404 lines/30d metric is misleading — it represents initial creation, not accumulation.

---

### Finding 2: Test file is proportional and healthy

**Evidence:**
- Production file: `synthesis_auto_create.go` — 263 lines, 6 functions
- Test file: `synthesis_auto_create_test.go` — 404 lines, 12 test functions
- Ratio: 1.5x test-to-production (healthy for Go tests with mock setup)
- All 12 tests pass

**Source:** `wc -l`, `grep -c 'func Test'`, `go test ./pkg/daemon/ -v`

**Significance:** The test file is well-structured — 43 lines of mock struct + 12 focused test functions covering all branches (not-due, no-suggestions, below-threshold, creates-issue, skips-model-exists, skips-dedup, multiple-clusters, load-error, service-not-configured, updates-scheduler, topic-to-slug, default-config). No redundancy or bloat.

---

### Finding 3: Growth drivers exhausted

**Evidence:**
- Production file has 6 public functions/methods
- Test file already covers all branches: happy path, error paths, edge cases (multiple clusters), scheduling behavior
- No new features pending for synthesis auto-create (feature shipped in orch-go-8qswb)

**Source:** Code review of both files

**Significance:** The test file is unlikely to grow significantly unless the production API surface expands. At 404 lines it is well under the 1,500-line extraction threshold and would need to nearly quadruple before becoming a genuine concern.

---

## Synthesis

**Key Insights:**

1. **Birth-as-growth false positive** - The hotspot detector counts new file creation within the 30-day window as growth. This is the 4th instance of this pattern in recent commits.

2. **File is healthy** - At 404 lines with a 1.5x test-to-production ratio, the file is well-structured with no signs of accretion problems.

3. **No extraction needed** - The file is 27% of the 1,500-line threshold. Growth drivers (test cases for existing functions) are exhausted.

**Answer to Investigation Question:**

No, this is not a real hotspot. The +404 lines/30d metric is entirely from the file's birth on 2026-03-12. There has been zero subsequent growth. The file is healthy, proportional, and well under the extraction threshold.

---

## Structured Uncertainty

**What's tested:**

- ✅ File has exactly 1 commit — creation only (verified: `git log --oneline --follow`)
- ✅ All 404 lines added in birth commit (verified: `git log --numstat`)
- ✅ All 12 tests pass (verified: `go test ./pkg/daemon/ -v`)

**What's untested:**

- ⚠️ Whether hotspot detector should exclude files younger than their measurement window (not in scope)

**What would change this:**

- Finding would be wrong if subsequent commits had added significant lines (git log shows they didn't)
- Finding would change if the production API surface expanded significantly (no evidence of pending changes)

---

## References

**Files Examined:**
- `pkg/daemon/synthesis_auto_create_test.go` - The hotspot file (404 lines, 12 tests)
- `pkg/daemon/synthesis_auto_create.go` - Production counterpart (263 lines, 6 functions)

**Commands Run:**
```bash
# Check file birth
git log --diff-filter=A --format='%H %ai %s' -- pkg/daemon/synthesis_auto_create_test.go

# Check all commits with line counts
git log --format='%H %ai' --numstat -- pkg/daemon/synthesis_auto_create_test.go

# Run tests
go test ./pkg/daemon/ -run TestDaemon_RunPeriodicSynthesisAutoCreate -count=1 -v
go test ./pkg/daemon/ -run TestTopicToSlug -count=1 -v
go test ./pkg/daemon/ -run TestDefaultConfig_IncludesSynthesisAutoCreate -count=1 -v
```
