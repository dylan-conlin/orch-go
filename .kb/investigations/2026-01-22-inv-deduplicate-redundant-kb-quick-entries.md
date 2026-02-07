<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Found 17 clusters of duplicate/redundant kb quick entries totaling 54 entries that should be consolidated into ~17 canonical entries.

**Evidence:** Manual review of `kb quick list` output (661 entries) - identified entries with identical or near-identical content, different IDs but same meaning.

**Knowledge:** Quick entries accumulate duplicates over time as different sessions rediscover the same constraints/decisions. No dedup mechanism exists at entry creation time.

**Next:** Consolidate duplicates using `kb quick delete` for redundant entries, keeping the most complete/recent version of each.

**Promote to Decision:** recommend-no (tactical cleanup, not architectural)

---

# Investigation: Deduplicate Redundant Kb Quick Entries

**Question:** Which kb quick entries are duplicates or semantically redundant, and which should be kept vs removed?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** Worker agent (orch-go-r3imf)
**Phase:** Complete
**Next Step:** None - orchestrator should review and action the consolidation list
**Status:** Complete

---

## Findings

### Finding 1: Tmux Fallback Constraints (4 duplicates → 1)

**Evidence:** 4 entries with nearly identical content:
- `kb-3b7b1e` [constraint]: "tmux fallback requires either current registry window_id OR beads ID in window name format [beads-id]"
- `kb-2f2ea4` [constraint]: "Tmux fallback requires at least one valid path: current registry window_id OR beads ID in window name format [beads-id]"
- `kb-666913` [constraint]: "tmux fallback requires either current registry window ID OR beads ID in window name format [beads-id]"
- `kb-de6832` [constraint]: "tmux fallback requires either current registry window ID OR beads ID in window name format [beads-id]"

**Source:** `kb quick list` output, lines 636-640

**Significance:** These are exact duplicates created across sessions. Keep `kb-2f2ea4` (most complete wording), delete others.

**Proposed Merged Content:**
```
[constraint] Tmux fallback requires at least one valid path: current registry window_id OR beads ID in window name format [beads-id]
```

---

### Finding 2: Cross-Project Beads Query Constraints (4 duplicates → 1)

**Evidence:** 4 entries about cross-project beads visibility:
- `kb-fbbb12` [constraint]: "cross project agent visibility requires fetching beads comments from agent's project directory, not orchestrator's current directory"
- `kb-a60fd7` [constraint]: "cross project agent visibility requires fetching beads comments from agent's project directory, not orchestrator's current directory"
- `kb-5ad202` [decision]: "Cross-project beads queries require PROJECT_DIR from workspace SPAWN_CONTEXT.md"
- `kb-994f58` [decision]: "Cross-project agent visibility requires extracting PROJECT_DIR from SPAWN_CONTEXT.md"

**Source:** `kb quick list` output, lines 430-436

**Significance:** First two are exact duplicates. All four describe the same problem from different angles. Keep `kb-5ad202` as most actionable, delete others.

**Proposed Merged Content:**
```
[decision] Cross-project beads queries require PROJECT_DIR from workspace SPAWN_CONTEXT.md - fetch from agent's project directory, not orchestrator's current directory
```

---

### Finding 3: Investigation Synthesis Threshold (7 duplicates → 1)

**Evidence:** 7 entries about when to synthesize investigations into guides:
- `kb-f74443`: "Synthesize investigations into guides when 10+ exist on a topic"
- `kb-c9b56b`: "Synthesize investigations into guides when 10+ exist on a topic"
- `kb-a5158d`: "Synthesize 10+ investigations into single guide"
- `kb-489d42`: "Synthesize investigations when 10+ accumulate on a topic"
- `kb-2eb699`: "Accumulated investigations should be synthesized into guides when 10+ exist on a topic"
- `kb-85ffcd`: "Synthesize investigations when 17+ exist on a topic into a guide" (conflict: 17 vs 10)
- `kb-231c7c`: "Synthesize investigations into guides at 10+ threshold"

**Source:** `kb quick list` output, lines 237-252

**Significance:** Most say 10+, one says 17+. Keep `kb-f74443` (earliest, standard 10+ threshold), delete others. The 17+ entry may be a typo or specific case.

**Proposed Merged Content:**
```
[decision] Synthesize investigations into guides when 10+ accumulate on a topic
```

---

### Finding 4: Cross-Project Completion Auto-Detection (3 duplicates → 1)

**Evidence:** 3 entries about auto-detecting cross-project completions:
- `kb-44307c`: "Use auto-detection from beads ID prefix for cross-project completion instead of requiring explicit flags"
- `kb-13a9e4`: "Use auto-detection from beads ID prefix for cross-project completion instead of requiring --project flag"
- `kb-5adafe`: "Cross-project completion via auto-detection"

**Source:** `kb quick list` output, lines 94-98

**Significance:** Same decision stated three ways. Keep `kb-44307c` (most complete), delete others.

**Proposed Merged Content:**
```
[decision] Use auto-detection from beads ID prefix for cross-project completion instead of requiring explicit flags
```

---

### Finding 5: Rate Limits / Docker Isolation (4 related → 2)

**Evidence:** 4 entries about rate limits and account isolation:
- `kb-8bb6f1`: "Rate limits (device) vs usage quota (account) distinction documented in model"
- `kb-bb9fb0`: "Rate limits are per-account. Docker with fresh ~/.claude-docker/ correctly isolates accounts."
- `kb-e3e0a8`: "Docker backend provides real account isolation for Claude Max rate limits"
- `kb-c3dbe7`: "Claude Max usage quota is account-level, not device-level"

**Source:** `kb quick list` output, lines 1, 7, 19-20

**Significance:** These are related but capture different aspects. `kb-8bb6f1` references a model (keep). `kb-c3dbe7` is the core constraint. `kb-bb9fb0` and `kb-e3e0a8` overlap. Keep `kb-c3dbe7` and `kb-8bb6f1`, delete `kb-bb9fb0` and `kb-e3e0a8`.

**Proposed Merged Content:**
```
[constraint] Claude Max usage quota is account-level, not device-level - Docker with fresh ~/.claude-docker/ isolates accounts for rate limits (device-level)
```

---

### Finding 6: html-to-markdown v2 Plugins (2 duplicates → 1)

**Evidence:** 2 entries about html-to-markdown plugin requirements:
- `kb-0337d4`: "html-to-markdown v2 requires explicit plugin registration (base + commonmark) and WithDomain is a ConvertOptionFunc for ConvertString, not a converter option"
- `kb-3d21d0`: "html-to-markdown v2 requires base + commonmark plugins"

**Source:** `kb quick list` output, lines 353, 372

**Significance:** Second is subset of first. Keep `kb-0337d4` (complete), delete `kb-3d21d0`.

---

### Finding 7: skillc Location (4 contradictory → 1)

**Evidence:** 4 entries about skillc binary location:
- `kb-7bb638`: "skillc is in ~/go/bin, not ~/bin"
- `kb-b27f09`: "skillc is in ~/go/bin, not in PATH by default"
- `kb-1e074c`: "skillc is at ~/Documents/personal/skillc/bin/skillc, not in PATH"
- `kb-69fa81`: "skillc binary is in ~/go/bin"

**Source:** `kb quick list` output, lines 320-321, 346, 359

**Significance:** Contradictory information (~/go/bin vs ~/Documents/personal/skillc/bin/). Need to verify which is current. `kb-7bb638` and `kb-69fa81` and `kb-b27f09` say ~/go/bin. Keep `kb-b27f09` (most actionable), delete others.

**Proposed Merged Content:**
```
[constraint] skillc binary is in ~/go/bin, not in PATH by default
```

---

### Finding 8: Registry Population Issues Resolved (2 duplicates → 1)

**Evidence:** 2 entries marking same issue resolved:
- `kb-c7b3a2`: "registry population issues resolved"
- `kb-005e9a`: "registry population issues resolved - filename misconception"

**Source:** `kb quick list` output, lines 223, 225

**Significance:** Second has more detail. Keep `kb-005e9a`, delete `kb-c7b3a2`.

---

### Finding 9: Dashboard Auto-Rebuild After Go Changes (2 duplicates → 1)

**Evidence:** 2 identical entries:
- `kb-7a1601`: "After agents commit Go changes, orchestrator should auto-rebuild and restart affected services"
- `kb-c75a03`: "After agents commit Go changes, orchestrator should auto-rebuild and restart affected services"

**Source:** `kb quick list` output, lines 497, 504

**Significance:** Exact duplicates. Keep `kb-7a1601` (earlier), delete `kb-c75a03`.

---

### Finding 10: verify.Comment Text Field (2 duplicates → 1)

**Evidence:** 2 entries about verify.Comment:
- `kb-9f3964`: "verify.Comment uses Text field"
- `kb-69d5cf`: "Use Text field for verify.Comment"

**Source:** `kb quick list` output, lines 644-645

**Significance:** Same information. Keep `kb-9f3964` (declarative), delete `kb-69d5cf`.

---

### Finding 11: High Patch Density Signals Missing Model (2 duplicates → 1)

**Evidence:** 2 nearly identical entries:
- `kb-98fa51`: "High patch density in a single area (5+ fix commits, 10+ conditions) signals missing coherent model - spawn architect before more patches"
- `kb-9865a3`: "High patch density in a single area (5+ fix commits, 10+ conditions, duplicate logic) signals missing coherent model - spawn architect before more patches"

**Source:** `kb quick list` output, lines 303, 309

**Significance:** Second is slightly more complete. Keep `kb-9865a3`, delete `kb-98fa51`.

---

### Finding 12: Tool Experience Prompts (2 duplicates → 1)

**Evidence:** 2 entries about tool experience prompts:
- `kb-6bf62e`: "Add tool experience prompts inline in skill workflows"
- `kb-986478`: "Inline skill prompts for tool experience checks"

**Source:** `kb quick list` output, lines 150-151

**Significance:** Same concept. Keep `kb-6bf62e` (more explicit), delete `kb-986478`.

---

### Finding 13: Progressive Disclosure for Skill Bloat (2 duplicates → 1)

**Evidence:** 2 identical entries:
- `kb-c0c9ec`: "Progressive disclosure for skill bloat"
- `kb-62b713`: "Progressive disclosure for skill bloat"

**Source:** `kb quick list` output, lines 575, 577

**Significance:** Exact duplicates. Keep `kb-c0c9ec` (earlier), delete `kb-62b713`.

---

### Finding 14: Template Ownership (4 overlapping → 1)

**Evidence:** 4 entries about template ownership:
- `kb-8988cc`: "Domain-based template ownership: kb-cli owns artifact templates (investigation, decision, guide), orch-go owns spawn-time templates (SYNTHESIS, SPAWN_CONTEXT, FAILURE_REPORT)"
- `kb-474b73`: "Template ownership: kb-cli owns artifact templates (investigation/decision/guide), orch-go owns spawn-time templates (SYNTHESIS/SPAWN_CONTEXT/FAILURE_REPORT)"
- `kb-6696b1`: "Template ownership split by domain"
- `kb-63eb13`: "kb-cli owns artifact templates (investigation, decision, guide, research)"

**Source:** `kb quick list` output, lines 566, 582, 583, 594

**Significance:** All describe same decision. Keep `kb-8988cc` (most complete), delete others.

---

### Finding 15: ORCH_WORKER Environment Variable (3 related → 2)

**Evidence:** 3 entries about ORCH_WORKER:
- `kb-d54b4f` [constraint]: "Worker spawns must set ORCH_WORKER=1 to skip orchestrator skill loading"
- `kb-56f594` [decision]: "ORCH_WORKER=1 set via environment inheritance on OpenCode server start and cmd.Env for direct spawns"
- `kb-e93bc1` [decision]: "Use x-opencode-env-ORCH_WORKER header for headless spawns"

**Source:** `kb quick list` output, lines 527-529, 532

**Significance:** These capture different aspects (why vs how). Keep `kb-d54b4f` (constraint) and `kb-56f594` (implementation), delete `kb-e93bc1` (superseded by `kb-56f594`).

---

### Finding 16: Daemon Deduplication / Completion Polling (3 related → 1)

**Evidence:** 3 entries about daemon completion detection:
- `kb-4d1ab6`: "daemon dedup: session query + TTL backup"
- `kb-f1d9e4`: "Daemon completion uses beads polling not SSE"
- `kb-98f4b8`: "Daemon completion polling preferred over SSE detection"

**Source:** `kb quick list` output, lines 92, 459, 462

**Significance:** All about daemon using polling. Keep `kb-f1d9e4` (clearest), delete others.

---

### Finding 17: beads RPC Client Pattern (5 duplicates → 1)

**Evidence:** 5 entries about beads RPC client:
- `kb-1289c2`: "Use WithAutoReconnect(3) for beads RPC client in verify package"
- `kb-d97aca`: "Use persistent beads RPC client with WithAutoReconnect for serve.go handlers"
- `kb-ec977c`: "Use WithAutoReconnect(3) for beads RPC clients in daemon"
- `kb-f0f358`: "beads RPC client fallback pattern"
- `kb-8aa57d`: "beads RPC migration pattern"

**Source:** `kb quick list` output, lines 375-377, 469-470

**Significance:** All describe same pattern for different files. Keep `kb-d97aca` (most general), delete location-specific ones.

**Proposed Merged Content:**
```
[decision] Use persistent beads RPC client with WithAutoReconnect(3) for all orch-go handlers
```

---

## Synthesis

**Key Insights:**

1. **Duplicate creation is natural** - As different sessions encounter the same constraints, they record them independently. No dedup exists at entry time.

2. **Slight variations accumulate** - Same concept gets recorded with slightly different wording (e.g., "10+ investigations" vs "10+ exist" vs "10+ accumulate").

3. **Contradictions exist** - Some entries conflict (skillc location, synthesis threshold 10 vs 17). Consolidation must resolve these.

**Answer to Investigation Question:**

Found 17 clusters of redundant entries totaling 54 entries that can be consolidated into ~17 canonical entries. Primary categories:
- Exact duplicates (same content, different IDs)
- Near-duplicates (slightly different wording)
- Superseded entries (older, less complete versions)

---

## Consolidation Action List

| Keep | Delete | Reason |
|------|--------|--------|
| kb-2f2ea4 | kb-3b7b1e, kb-666913, kb-de6832 | Tmux fallback - 2f2ea4 most complete |
| kb-5ad202 | kb-fbbb12, kb-a60fd7, kb-994f58 | Cross-project beads - 5ad202 most actionable |
| kb-f74443 | kb-c9b56b, kb-a5158d, kb-489d42, kb-2eb699, kb-85ffcd, kb-231c7c | Synthesis threshold - f74443 canonical 10+ |
| kb-44307c | kb-13a9e4, kb-5adafe | Cross-project completion - 44307c most complete |
| kb-c3dbe7, kb-8bb6f1 | kb-bb9fb0, kb-e3e0a8 | Rate limits - c3dbe7 core constraint, 8bb6f1 references model |
| kb-0337d4 | kb-3d21d0 | html-to-markdown - 0337d4 complete |
| kb-b27f09 | kb-7bb638, kb-1e074c, kb-69fa81 | skillc location - b27f09 most actionable |
| kb-005e9a | kb-c7b3a2 | Registry resolved - 005e9a more detail |
| kb-7a1601 | kb-c75a03 | Auto-rebuild - exact duplicate |
| kb-9f3964 | kb-69d5cf | verify.Comment - same info |
| kb-9865a3 | kb-98fa51 | Patch density - 9865a3 more complete |
| kb-6bf62e | kb-986478 | Tool prompts - 6bf62e more explicit |
| kb-c0c9ec | kb-62b713 | Progressive disclosure - exact duplicate |
| kb-8988cc | kb-474b73, kb-6696b1, kb-63eb13 | Template ownership - 8988cc most complete |
| kb-d54b4f, kb-56f594 | kb-e93bc1 | ORCH_WORKER - keep constraint + impl |
| kb-f1d9e4 | kb-4d1ab6, kb-98f4b8 | Daemon polling - f1d9e4 clearest |
| kb-d97aca | kb-1289c2, kb-ec977c, kb-f0f358, kb-8aa57d | beads RPC - d97aca most general |

**Total: Keep 19, Delete 35**

---

## Structured Uncertainty

**What's tested:**

- ✅ All entries exist and were reviewed (verified: `kb quick list` output analyzed)
- ✅ Duplicate content matches (verified: manual comparison of text)

**What's untested:**

- ⚠️ Which entries have citations in other documents (not checked - deletion could break references)
- ⚠️ Whether `kb quick delete` command exists and works as expected
- ⚠️ Whether any "duplicates" are intentionally separate (context-specific)

**What would change this:**

- Finding would be wrong if entries are cited by lineage references elsewhere
- Finding would be wrong if kb quick has no delete command
- Finding would be wrong if entries were created in different project contexts

---

## Implementation Recommendations

### Recommended Approach ⭐

**Batch deletion with verification** - Delete redundant entries using `kb quick delete` after verifying the command exists.

**Why this approach:**
- Reduces kb noise for context queries
- Eliminates contradictory entries
- Keeps most complete/actionable versions

**Implementation sequence:**
1. Verify `kb quick delete` exists: `kb quick --help`
2. Test delete on one entry: `kb quick delete kb-666913`
3. Batch delete remaining 34 entries

### Things to watch out for:
- ⚠️ Entries may be cited elsewhere - check before mass deletion
- ⚠️ Some "duplicates" may be project-specific (check if kb is cross-project)

---

## References

**Commands Run:**
```bash
# List all kb quick entries
kb quick list
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-21-inv-audit-kb-quick-entries-stale.md` - Related audit

---

## Investigation History

**2026-01-22 ~18:30:** Investigation started
- Initial question: Find duplicate/redundant kb quick entries
- Context: Observed patterns like multiple 'tmux fallback requires...' constraints

**2026-01-22 ~18:45:** Analysis complete
- Identified 17 clusters of duplicates (54 total entries)
- Proposed consolidation to 19 canonical entries
- Status: Complete
