## Summary (D.E.K.N.)

**Delta:** Implemented `orch hotspot` command that analyzes git fix commit density and kb reflect investigation clustering to surface areas needing architect attention.

**Evidence:** Command works with both text and JSON output (verified via manual test), all 13 unit tests pass.

**Knowledge:** Hotspot detection combines two signals: (1) files with 5+ fix: commits indicate structural issues, (2) topics with 3+ investigations indicate unclear design.

**Next:** Close - implementation complete with all acceptance criteria met.

---

# Investigation: Implement Orch Hotspot Cli Command

**Question:** How to implement `orch hotspot` command for detecting areas needing architect intervention?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Git history analysis for fix commit density

**Evidence:** Git log can be parsed to count fix: commits per file. Using `git log --since="X days ago" --pretty=format:%H|%s --name-only --diff-filter=ACMR` provides commit hash, message, and affected files.

**Source:** cmd/orch/hotspot.go:analyzeFixCommits()

**Significance:** Fix commit density is a reliable signal for files needing architectural attention - files with 5+ fixes in 4 weeks indicate recurring bugs.

---

### Finding 2: kb reflect provides investigation clustering

**Evidence:** `kb reflect --type synthesis --format json` returns topics with investigation counts, files, and urgency. This maps directly to areas with unclear design.

**Source:** cmd/orch/hotspot.go:analyzeInvestigationClusters()

**Significance:** Investigation clustering (3+ investigations on same topic) indicates areas where understanding is murky and design decisions may be needed.

---

### Finding 3: Combined scoring provides actionable recommendations

**Evidence:** Sorting hotspots by score and generating severity-based recommendations (CRITICAL for 10+, HIGH for 7+, MODERATE for 5+) gives orchestrators clear next actions.

**Source:** cmd/orch/hotspot.go:generateFixRecommendation(), generateInvestigationRecommendation()

**Significance:** Recommendations that suggest specific actions (spawn architect, spawn investigation, review) make the output actionable.

---

## Structured Uncertainty

**What's tested:**

- ✅ Git log parsing correctly counts fix: commits per file (verified: ran against orch-go repo)
- ✅ kb reflect integration returns investigation clusters (verified: ran command and parsed output)
- ✅ JSON output format is valid and complete (verified: test and manual run)
- ✅ Text output displays correctly with formatting (verified: manual run)
- ✅ All unit tests pass (13 tests)

**What's untested:**

- ⚠️ Performance on very large repos (not benchmarked)
- ⚠️ kb reflect behavior when kb-cli is not installed (gracefully fails but not tested)

---

## Implementation Details

**Files created:**
- `cmd/orch/hotspot.go` - Main command implementation
- `cmd/orch/hotspot_test.go` - Unit tests

**Features implemented:**
- [x] `orch hotspot` command exists
- [x] Outputs files with hotspot scores
- [x] Includes recommendation for architect skill
- [x] --json flag for machine-readable output
- [x] --threshold flag to adjust fix commit sensitivity
- [x] --inv-threshold flag to adjust investigation sensitivity
- [x] --days flag to adjust analysis period

---

## References

**Files Examined:**
- cmd/orch/status_cmd.go - Pattern for JSON/text output
- cmd/orch/doctor.go - Pattern for health check style commands
- cmd/orch/learn.go - Pattern for kb integration

**Commands Run:**
```bash
# Build and test
go build ./cmd/orch/...
go test ./cmd/orch/... -run TestHotspot -v

# Manual verification
./orch-test hotspot
./orch-test hotspot --json
./orch-test hotspot --help
```
