## Summary (D.E.K.N.)

**Delta:** Semantic topic clustering now works - investigations cluster by domain (dashboard, daemon, spawn) instead of meaningless action verbs (add, fix, implement).

**Evidence:** Before: 41 investigations under "add" (unrelated items). After: 36 under "dashboard", 20 under "daemon", 34 under "spawn" (semantically related). All tests pass.

**Knowledge:** Investigation filenames follow `action-domain-description` pattern. First word is usually an action verb, domain keywords appear later. Must scan entire topic for domain keywords first.

**Next:** Close - implementation complete in kb-cli, ready for installation.

---

# Investigation: KB Reflect Semantic Clustering

**Question:** How can we replace keyword grouping with semantic topic clustering in `kb reflect --type synthesis`?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Phase:** Complete
**Status:** Complete

---

## Findings

### Finding 1: Current clustering uses first meaningful word

**Evidence:** The `normalizeTopicForClustering` function takes the FIRST word that passes filters:
- "add-dashboard-feature" → clusters under "add"
- "fix-daemon-polling" → clusters under "fix"
- "implement-sse-events" → clusters under "implement"

**Source:** `kb-cli/cmd/kb/reflect.go:446-476`

**Significance:** This creates meaningless clusters where 41 unrelated investigations all land under "add" because they all start with "add-".

### Finding 2: Action verbs dominate filename prefixes

**Evidence:** Analyzed orch-go's 482 investigations:
- 41 start with "add-"
- 31 start with "implement-"
- 26 start with "test-"
- 23 start with "fix-"

These action verbs tell us WHAT was done, not WHAT DOMAIN it's about.

**Source:** `kb reflect --type synthesis --format json` output

**Significance:** The pattern is `action-domain-details`. Domain keywords appear AFTER the action verb.

### Finding 3: Domain keywords exist throughout filenames

**Evidence:** Good clusters already existed when the domain keyword happened to be first:
- "dashboard-*" (24 investigations) - all dashboard-related
- "daemon-*" (13 investigations) - all daemon-related
- "glass-*" (7 investigations) - all glass-related

**Source:** Analysis of synthesis output before changes

**Significance:** When domain keywords are first, clustering works. Need to find them regardless of position.

---

## Synthesis

**Key Insights:**

1. **Action verbs are noise for clustering** - "add", "fix", "implement" tell us WHAT was done, not WHAT it was done to.

2. **Domain keywords are signal** - "dashboard", "daemon", "spawn", "opencode" represent actual semantic topics.

3. **Two-pass algorithm needed** - First scan for domain keywords anywhere, then fall back to first non-action word.

**Answer to Investigation Question:**

Replace first-word extraction with semantic topic detection:
1. Define a list of domain-specific keywords (orchestration domains)
2. Define action verbs to skip
3. Scan entire topic for domain keywords first
4. Fall back to first meaningful non-action word

---

## Implementation

The solution was implemented in `kb-cli/cmd/kb/reflect.go`:

```go
// Domain-specific topic keywords to look for
domainTopics := []string{
    "daemon", "spawn", "agent", "session", "orchestrator", "worker",
    "dashboard", "glass", "playwright",
    "http", "api", "sse", "endpoint",
    "skill", "investigation", "synthesis", "reflection",
    ...
}

// Action verbs to skip
actionVerbs := map[string]bool{
    "add": true, "create": true, "implement": true,
    "update": true, "fix": true, "enhance": true,
    "test": true, "investigate": true, "explore": true,
    ...
}
```

**Results:**
- Before: "add" (41), "implement" (31), "test" (26), "fix" (23)
- After: "orch" (45), "dashboard" (36), "spawn" (34), "daemon" (20)

All investigations now cluster by semantic domain, not by action verb.

---

## References

**Files Modified:**
- `kb-cli/cmd/kb/reflect.go:446-536` - Updated `normalizeTopicForClustering` function

**Commands Run:**
```bash
# Test before changes
cd orch-go && kb reflect --type synthesis --format json

# Build and test after changes
cd kb-cli && go build -o kb ./cmd/kb
go test ./cmd/kb/... -run TestReflect -v

# Verify results
cd orch-go && ./kb reflect --type synthesis --format json
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-21-kb-reflect-command-interface.md` - kb reflect design
- **Issue:** orch-go-s03z - Knowledge fragmentation investigation that discovered this problem
