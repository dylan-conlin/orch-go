## Summary (D.E.K.N.)

**Delta:** Fixed cross-project agent visibility by scanning OpenCode session storage directly to discover all project directories, bypassing the chicken-and-egg problem.

**Evidence:** Before fix: pw-2o1x not visible. After fix: pw-2o1x appears (verified via curl http://localhost:3349/api/agents).

**Knowledge:** OpenCode stores sessions per-project in `~/.local/share/opencode/storage/session/{partition_hash}/`, where each session JSON contains a `directory` field with the project path.

**Next:** Close - implementation complete and verified.

---

# Investigation: Fix Cross Project Agent Visibility

**Question:** How to make cross-project agents visible in orch serve without prior knowledge of their project directories?

**Started:** 2025-12-29
**Updated:** 2025-12-29
**Owner:** Agent og-feat-fix-cross-project-29dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Chicken-and-egg problem in extractUniqueProjectDirs

**Evidence:** The existing `extractUniqueProjectDirs` function could only find project directories from sessions that were already visible, but sessions weren't visible without knowing their project directories first.

**Source:** `cmd/orch/serve.go:354-384` - function relied on already-visible sessions to extract directories

**Significance:** New cross-project agents would never appear in the dashboard because their project directory wasn't known yet.

---

### Finding 2: OpenCode session storage structure reveals all projects

**Evidence:** OpenCode stores sessions in `~/.local/share/opencode/storage/session/{partition_hash}/{session_id}.json`. Each session JSON file contains a `directory` field with the full project path.

**Source:** 
```bash
ls ~/.local/share/opencode/storage/session/
cat ~/.local/share/opencode/storage/session/aca13819f57d62c96e5f8c734d7ef8e50377d4fb/ses_494c972f0ffe2iwVwijFYHUZOX.json
# {"directory": "/Users/dylanconlin/.../price-watch", ...}
```

**Significance:** By scanning this directory structure, we can discover ALL projects that have ever had OpenCode sessions, regardless of whether they're currently visible.

---

## Structured Uncertainty

**What's tested:**

- ✅ pw-2o1x now appears in /api/agents (verified: curl after fix showed "Found pw-2o1x agents: 1")
- ✅ Cross-project sessions from 9 different projects now visible (verified: By project count showed pw:9, glass:8, etc.)
- ✅ Build succeeds with new code (verified: go build ./cmd/orch/)

**What's untested:**

- ⚠️ Performance impact of scanning session storage on systems with many partitions (not benchmarked)

---

## References

**Files Examined:**
- `cmd/orch/serve.go` - Main serve implementation with handleAgents function
- `~/.local/share/opencode/storage/session/` - OpenCode session storage structure

**Commands Run:**
```bash
# List OpenCode session partitions
ls ~/.local/share/opencode/storage/session/

# View session JSON structure
cat ~/.local/share/opencode/storage/session/*/ses_*.json | head -20

# Test before fix
curl -s http://localhost:3348/api/agents | grep pw-2o1x  # No results

# Test after fix
curl -s http://localhost:3349/api/agents | grep pw-2o1x  # Found agent
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-29-inv-opencode-session-storage-model-cross.md` - Prior investigation on OpenCode session storage model
