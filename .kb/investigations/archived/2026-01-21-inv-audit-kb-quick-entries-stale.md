<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Found 17 kb quick entries that are stale/superseded due to later events: plugin workarounds that failed, model default contradictions, inaccurate registry removal claims, and spawn mode documentation conflicts.

**Evidence:** Cross-referenced kb entries against actual codebase (pkg/model/model.go shows Opus default, pkg/registry still exists, Jan 9 investigation shows plugin 0.0.7 got re-blocked).

**Knowledge:** kb quick entries accumulate contradictions without cleanup - same topic can have 3+ entries with conflicting claims. Need periodic audit.

**Next:** Run `kb quick obsolete <id>` for the 17 entries identified below. Consider establishing quarterly kb hygiene review.

**Promote to Decision:** recommend-no (tactical cleanup, not architectural)

---

# Investigation: Audit Kb Quick Entries Stale

**Question:** Which kb quick entries are no longer valid due to later events (OpenCode plugin workarounds that failed, old model defaults, registry patterns claimed removed, old spawn modes)?

**Started:** 2026-01-21
**Updated:** 2026-01-21
**Owner:** feature-impl worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: OpenCode Plugin Workaround Entries (Failed Jan 9)

**Entries to archive:**

| ID | Content | Reason Stale |
|----|---------|--------------|
| kb-81f105 | Use opencode-anthropic-auth@0.0.7 to bypass Opus auth gate | Plugin got re-blocked within 6 hours per Jan 9 investigation |
| kb-264489 | Update opencode plugin to 0.0.7 | Same - workaround no longer works |

**Evidence:** Investigation `2026-01-09-inv-anthropic-oauth-community-workarounds.md` documents that 0.0.7 fix was released Jan 9 ~4:30 AM UTC, re-blocked by ~2:30 PM UTC (6 hours). The investigation's D.E.K.N. explicitly recommends "Abandon Claude Max OAuth until official OpenCode upstream fix arrives."

**Source:** `.kb/investigations/2026-01-09-inv-anthropic-oauth-community-workarounds.md:4-6, 94-100`

**Significance:** These entries actively recommend a solution that doesn't work anymore.

---

### Finding 2: Model Default Contradictions

**Entries to archive:**

| ID | Content | Reason Stale |
|----|---------|--------------|
| kb-9daa5a | Always use flash as default spawn model | Actual code has Opus as default (`pkg/model/model.go:20-23`) |

**Conflicting but correct entry:**

| ID | Content | Status |
|----|---------|--------|
| kb-290db1 | orch-go DefaultModel should be Opus | Matches actual code - KEEP |
| kb-deaacb | Opus default, Gemini escape hatch | Matches actual code - KEEP |

**Evidence:** `pkg/model/model.go` lines 17-23 explicitly set DefaultModel to Opus:
```go
var DefaultModel = ModelSpec{
    Provider: "anthropic",
    ModelID:  "claude-opus-4-5-20251101",
}
```

kb-9daa5a claims "Anthropic Max subscription no longer available outside of Claude Code" but this is incorrect - the codebase still uses Opus as default.

**Source:** `pkg/model/model.go:17-23`, `kb quick get kb-9daa5a`

**Significance:** Entry contradicts actual implementation and could mislead agents.

---

### Finding 3: Inaccurate Registry Removal Claims

**Entries to archive:**

| ID | Content | Reason Stale |
|----|---------|--------------|
| kb-bba8ba | Agent registry removal complete - all state from OpenCode API + beads | Registry package still exists and is actively imported |

**Evidence:**
- `pkg/registry/registry.go` exists (14,311 bytes, last modified Jan 20)
- `cmd/orch/spawn_cmd.go` imports `github.com/dylan-conlin/orch-go/pkg/registry`
- `cmd/orch/abandon_cmd.go` imports same package
- 5 files actively import pkg/registry

**Source:** `ls -la pkg/registry/`, `rg "pkg/registry" --files-with-matches`

**Significance:** Entry claims registry is removed but it's still a core dependency.

---

### Finding 4: Spawn Mode Documentation Conflicts

**Entries with partial accuracy (need clarification):**

| ID | Content | Issue |
|----|---------|-------|
| kb-318507 | Tmux is the default spawn mode in orch-go, not headless | Only true for orchestrators, workers default to headless |
| kb-6f7dd1 | Default spawn mode is headless with --tmux opt-in | Only true for workers, orchestrators default to tmux |

**Evidence:** `spawn_cmd.go:1329-1340`:
```go
// Orchestrator-type skills default to tmux mode (visible interaction)
// Workers default to headless mode (automation-friendly)
useTmux := tmux || attach || cfg.IsOrchestrator
if useTmux { ... return runSpawnTmux(...) }
// Default for workers: Headless mode
return runSpawnHeadless(...)
```

**Source:** `cmd/orch/spawn_cmd.go:1329-1340`

**Significance:** Both entries are partially correct but incomplete - neither captures the orchestrator vs worker distinction.

---

### Finding 5: Registry-Related Entries That Need Review

**Entries referencing removed/changed patterns:**

| ID | Content | Status |
|----|---------|--------|
| kb-c7b3a2 | registry population issues resolved | Likely stale - references old issue |
| kb-005e9a | registry population issues resolved - filename misconception | Likely stale - references old issue |
| kb-3b7b1e | orch tail tmux fallback requires registry window ID OR beads ID | May need update if registry role changed |
| kb-2f2ea4 | Tmux fallback requires registry window_id OR beads ID | Duplicate of above |
| kb-666913 | Same as above | Duplicate |
| kb-de6832 | Same as above | Duplicate |
| kb-a50596 | Registry updates must happen before beads close | May be outdated |
| kb-4a8b7e | Registry respawn workflow uses slot reuse pattern | May be outdated |
| kb-7829b4 | orch-go agent state exists in four layers including registry | Registry layer exists but claim of removal is false |

**Evidence:** Registry still exists and is imported, but there are 4 near-duplicate entries about tmux fallback requirements.

**Source:** `kb quick list | grep -i registry`

**Significance:** Multiple duplicate entries and potential confusion about registry status.

---

### Finding 6: Duplicate/Redundant Entries

**Entries that are duplicates:**

| IDs | Content | Action |
|-----|---------|--------|
| kb-3b7b1e, kb-2f2ea4, kb-666913, kb-de6832 | All say "tmux fallback requires registry window_id OR beads ID" | Keep one, archive 3 |
| kb-c7b3a2, kb-005e9a | Both say "registry population issues resolved" | Keep one, archive 1 |

**Evidence:** `kb quick list` shows 4 entries with nearly identical content about tmux fallback requirements.

**Source:** kb quick list output lines 635-639

**Significance:** Duplicates add noise and can become inconsistent over time.

---

## Synthesis

**Key Insights:**

1. **Plugin workarounds decay rapidly** - The Jan 9 OpenCode plugin fix had a 6-hour lifespan before Anthropic re-blocked it. Entries recommending this approach are immediately stale.

2. **Contradictory entries accumulate** - Model defaults have 3+ entries with conflicting claims (Flash vs Opus as default). Without cleanup, agents may follow outdated guidance.

3. **Registry status is misrepresented** - The "registry removal complete" entry is false - pkg/registry is actively used. This creates confusion about architecture.

4. **Spawn mode documentation is incomplete** - Neither kb entry captures the orchestrator/worker distinction for default spawn modes.

5. **Duplicates create maintenance burden** - 4 near-identical entries about tmux fallback requirements dilute signal.

**Answer to Investigation Question:**

17 kb quick entries are candidates for archival:
- 2 entries recommending failed OpenCode plugin workaround (kb-81f105, kb-264489)
- 1 entry with wrong model default (kb-9daa5a)
- 1 entry falsely claiming registry removal (kb-bba8ba)
- 2 entries about spawn modes that are incomplete (kb-318507, kb-6f7dd1 - need clarification rather than removal)
- 2 entries about resolved registry issues (kb-c7b3a2, kb-005e9a)
- 3 duplicate entries about tmux fallback (keep kb-3b7b1e, archive kb-2f2ea4, kb-666913, kb-de6832)
- 5 entries referencing registry patterns that may need update (kb-a50596, kb-4a8b7e, kb-7829b4, kb-3ffc51, kb-ec1343)

## Recommended Actions

### Priority 1: Archive Definitively Stale (6 entries)

```bash
# OpenCode plugin workarounds that failed
kb quick obsolete kb-81f105 --reason "Plugin 0.0.7 re-blocked within 6 hours per Jan 9 investigation"
kb quick obsolete kb-264489 --reason "Plugin 0.0.7 workaround no longer works"

# Wrong model default
kb quick obsolete kb-9daa5a --reason "Contradicts actual code - pkg/model/model.go uses Opus as default"

# False registry removal claim
kb quick obsolete kb-bba8ba --reason "Registry package still exists and is actively imported in spawn_cmd.go and abandon_cmd.go"

# Old resolved issues
kb quick obsolete kb-c7b3a2 --reason "Old resolved issue - no longer actionable"
kb quick obsolete kb-005e9a --reason "Duplicate of kb-c7b3a2, old resolved issue"
```

### Priority 2: Deduplicate (3 entries)

```bash
# Keep kb-3b7b1e as canonical, archive duplicates
kb quick obsolete kb-2f2ea4 --reason "Duplicate of kb-3b7b1e"
kb quick obsolete kb-666913 --reason "Duplicate of kb-3b7b1e"
kb quick obsolete kb-de6832 --reason "Duplicate of kb-3b7b1e"
```

### Priority 3: Clarify Rather Than Archive (2 entries)

The spawn mode entries (kb-318507, kb-6f7dd1) are partially correct. Rather than archiving, consider:

```bash
kb quick supersede kb-318507 --reason "Only true for orchestrators; workers default to headless"
kb quick supersede kb-6f7dd1 --reason "Only true for workers; orchestrators default to tmux"

# Then create a corrected entry:
kb quick decide "Spawn mode defaults: orchestrators→tmux, workers→headless, --tmux opts workers into tmux" \
  --reason "Per spawn_cmd.go:1329-1340, IsOrchestrator flag determines default mode"
```

### Priority 4: Review Registry-Related (5 entries)

These entries may still be valid but need verification against current architecture:
- kb-a50596: Registry updates must happen before beads close
- kb-4a8b7e: Registry respawn workflow uses slot reuse pattern
- kb-7829b4: orch-go agent state exists in four layers
- kb-3ffc51: Session_id stored in workspace file not registry
- kb-ec1343: Session ID resolution pattern

**Recommendation:** Defer these to a follow-up investigation focused on registry's current role.

---

## Structured Uncertainty

**What's tested:**

- ✅ Default model is Opus (verified: read pkg/model/model.go:17-23)
- ✅ Registry package exists and is imported (verified: ls pkg/registry/, grep for imports)
- ✅ Plugin 0.0.7 failed (verified: read Jan 9 investigation D.E.K.N.)
- ✅ Spawn mode logic distinguishes orchestrators from workers (verified: read spawn_cmd.go:1329-1340)

**What's untested:**

- ⚠️ Registry-related entries (Priority 4) may still be valid - needs architecture review
- ⚠️ Running `kb quick obsolete` commands - syntax not verified
- ⚠️ Full impact of spawn mode clarification on downstream guidance

**What would change this:**

- If OpenCode plugin workaround gets fixed upstream, kb-81f105/kb-264489 would become relevant again
- If model defaults change in config, kb-9daa5a might become correct
- If registry is actually deprecated in favor of OpenCode API, kb-bba8ba would need update rather than archive

---

## Implementation Recommendations

**Purpose:** Clean up stale kb quick entries to reduce noise and prevent agents from following outdated guidance.

### Recommended Approach ⭐

**Batch Obsolete via kb quick obsolete** - Run the commands in Priority 1 and 2 sections to archive 9 definitively stale entries.

**Why this approach:**
- Non-destructive: `kb quick obsolete` marks entries as obsolete, doesn't delete
- Immediate impact: Removes contradictory/outdated guidance from kb context results
- Low risk: Can be reversed if an entry is found to still be valid

**Trade-offs accepted:**
- Deferring Priority 4 (registry-related) entries that need architecture review
- Not fully resolving spawn mode documentation (needs new corrected entry)

**Implementation sequence:**
1. Run Priority 1 commands (6 definitively stale entries)
2. Run Priority 2 commands (3 duplicate entries)
3. Create corrected spawn mode entry (Priority 3)
4. Create follow-up issue for registry architecture review (Priority 4)

### Alternative Approaches Considered

**Option B: Full registry audit first**
- **Pros:** Would resolve all registry-related uncertainty
- **Cons:** Scope creep - registry architecture review is a separate task
- **When to use instead:** If registry changes are actively planned

**Option C: Leave entries and add superseding entries**
- **Pros:** Preserves history, doesn't change existing entries
- **Cons:** Doesn't reduce noise, kb context still surfaces stale entries
- **When to use instead:** If audit findings are uncertain

**Rationale for recommendation:** Prioritize clearing obvious stale entries now, defer uncertain cases.

---

### Implementation Details

**What to implement first:**
- Priority 1 obsolete commands - 6 definitively stale entries
- Priority 2 deduplication - 3 duplicate entries

**Things to watch out for:**
- ⚠️ `kb quick obsolete` command syntax - verify it works before batch execution
- ⚠️ Some entries may have been cited in decisions/guides - search before obsoleting
- ⚠️ Registry entries (Priority 4) need careful review before any changes

**Areas needing further investigation:**
- Registry's current role in orch-go architecture
- Whether OpenCode upstream has fixed the auth issue (would make plugin entries relevant again)
- Full audit of kb quick entries for other stale categories not covered here

**Success criteria:**
- ✅ 9 entries marked obsolete (6 stale + 3 duplicates)
- ✅ New corrected spawn mode entry created
- ✅ Follow-up issue created for registry review
- ✅ `kb context` no longer returns obsoleted entries in search results

---

## References

**Files Examined:**
- `pkg/model/model.go:17-23` - Verified DefaultModel is Opus, not Flash
- `cmd/orch/spawn_cmd.go:1309-1340` - Verified spawn mode logic (orchestrator vs worker defaults)
- `pkg/registry/registry.go` - Verified registry package still exists (14KB, modified Jan 20)
- `.kb/investigations/2026-01-09-inv-anthropic-oauth-community-workarounds.md` - Verified plugin 0.0.7 failure

**Commands Run:**
```bash
# List all kb quick entries
kb quick list

# Get specific entry details
kb quick get kb-bba8ba
kb quick get kb-9daa5a
kb quick get kb-81f105
kb quick get kb-264489

# Check registry package existence and usage
ls -la pkg/registry/
rg "pkg/registry" --files-with-matches

# Check model defaults in code
rg "DefaultModel" pkg/model/

# Check spawn mode logic
rg "default.*headless|tmux.*default" cmd/orch/
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-09-inv-anthropic-oauth-community-workarounds.md` - Documents plugin 0.0.7 failure
- **Decision:** `.kb/decisions/2026-01-14-registry-contract-spawn-cache-only.md` - May clarify current registry role

---

## Investigation History

**2026-01-21 23:55:** Investigation started
- Initial question: Which kb quick entries are stale due to later events?
- Context: Scheduled audit for kb hygiene

**2026-01-21 23:58:** Found 660 total kb quick entries
- Identified focus areas: plugin workarounds, model defaults, registry claims, spawn modes

**2026-01-22 00:05:** Verified key contradictions
- Plugin 0.0.7 failed (per Jan 9 investigation)
- Model default is Opus (per code)
- Registry still exists (per file system)
- Spawn mode has orchestrator/worker distinction (per code)

**2026-01-22 00:15:** Investigation completed
- Status: Complete
- Key outcome: 17 entries identified for cleanup (9 definitive + 8 needing review/clarification)
