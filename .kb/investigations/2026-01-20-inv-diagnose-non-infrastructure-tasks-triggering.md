<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Infrastructure gate triggers incorrectly because it's checked BEFORE project config and uses overly broad keywords like "orch-go".

**Evidence:** `spawn_cmd.go:1155` checks `isInfrastructureWork()` at priority 2.5, before `projCfg.SpawnMode` at priority 4; keywords include "orch-go" which matches any task in this project.

**Knowledge:** Config should be the primary source of truth after explicit flags; infrastructure gate should only trigger for truly critical patterns (files that restart OpenCode server).

**Next:** Implement - reorder priority (config before infra), narrow keywords, make gate advisory when config is set.

**Promote to Decision:** recommend-no (bug fix, not architectural choice)

---

# Investigation: Diagnose Non Infrastructure Tasks Triggering

**Question:** Why are non-infrastructure tasks triggering the infrastructure gate and defaulting to Claude mode instead of respecting the `spawn_mode: opencode` config?

**Started:** 2026-01-20
**Updated:** 2026-01-20
**Owner:** Dylan Conlin
**Phase:** Complete
**Next Step:** Implement the refactoring
**Status:** Complete

---

## Findings

### Finding 1: Priority order puts infrastructure gate BEFORE config

**Evidence:** In `spawn_cmd.go:1135-1194`, the backend selection priority is:
```
1. Explicit --backend flag (highest priority)
2. Explicit --opus flag (forces claude)
2.5. Infrastructure work detection (auto-apply escape hatch)  ← PROBLEM
3. Auto-selection based on --model flag
4. Config default (spawn_mode in project config)  ← TOO LATE
5. Default to claude
```

**Source:** `cmd/orch/spawn_cmd.go:1135-1194`

**Significance:** Even when user configures `spawn_mode: opencode`, if `isInfrastructureWork()` returns true, the config is never checked.

---

### Finding 2: Infrastructure keywords are too broad

**Evidence:** The `infrastructureKeywords` list includes:
- `"orch-go"` - Matches ANY task in the orch-go project
- `"orchestration"` - Very generic
- `"dashboard"` - UI doesn't affect server stability
- `"skillc"` - Skill compiler is separate from OpenCode server
- `"agents.ts"`, `"daemon.ts"` - Frontend files

**Source:** `cmd/orch/spawn_cmd.go:2285-2309`

**Significance:** The original intent was to protect agents working on OpenCode server infrastructure from being killed when they restart the server. But the keywords are so broad that nearly everything in orch-go matches.

---

### Finding 3: Config explicitly sets `spawn_mode: opencode` but it's ignored

**Evidence:** Project config at `.orch/config.yaml`:
```yaml
spawn_mode: opencode
opencode:
    model: deepseek
    server: http://localhost:4096
```

**Source:** `.orch/config.yaml`

**Significance:** The user's intent is clear - use DeepSeek via OpenCode as default. But the infrastructure gate overrides this intent.

---

## Synthesis

**Key Insights:**

1. **Priority inversion** - The infrastructure gate was designed as a "safety override" but its position in the priority chain means it overrides explicit user config, which violates the principle that config should be respected.

2. **Keyword scope creep** - The keywords grew to include general orchestration terms rather than being focused on the specific danger: restarting the OpenCode server while agents are connected to it.

3. **The real danger is narrow** - Only tasks that modify `serve.go`, `pkg/opencode/client.go`, or similar server-critical files need the escape hatch. Dashboard UI, skill compilation, and most orch-go work don't restart the server.

**Answer to Investigation Question:**

Non-infrastructure tasks are triggering the infrastructure gate because:
1. The gate is checked at priority 2.5, before the config check at priority 4
2. Keywords like "orch-go" match virtually any task in the project
3. There's no way to bypass the gate without using explicit `--backend opencode`

---

## Structured Uncertainty

**What's tested:**

- ✅ Config parsing works (`spawn_mode: opencode` is correctly parsed by config.Load())
- ✅ Infrastructure keywords are checked case-insensitively (verified in code)
- ✅ "orch-go" appears in beads issue descriptions for this project

**What's untested:**

- ⚠️ The proposed fix won't break legitimate infrastructure protection (need to test with serve.go modification)
- ⚠️ Event logging accuracy for infrastructure detection (not verified in production)

**What would change this:**

- Finding would be wrong if config parsing were broken (but we verified it's parsed correctly)
- Finding would be wrong if infrastructure gate had a bypass mechanism I missed

---

## Implementation Recommendations

### Recommended Approach: Three-Part Fix

**Reorder priority + narrow keywords + advisory mode**

**Why this approach:**
- Respects user's explicit config choice (OpenCode/DeepSeek)
- Preserves escape hatch for truly dangerous work (serve.go, pkg/opencode)
- Provides warning without forcing override for borderline cases

**Trade-offs accepted:**
- Users modifying serve.go might need manual `--backend claude --tmux` if config says opencode
- Slightly more complex logic, but clearer intent

**Implementation sequence:**
1. Reorder priority: config check before infrastructure gate
2. Narrow keywords to truly critical patterns
3. Make infrastructure detection advisory (warning) when config is explicitly set

### Alternative Approaches Considered

**Option B: Just reorder priority**
- **Pros:** Simplest change
- **Cons:** Still triggers on broad keywords, may miss legitimate infra work
- **When to use instead:** If narrowing keywords is controversial

**Option C: Add config option to disable infra detection entirely**
- **Pros:** Maximum user control
- **Cons:** Users might disable it and then break things; adds config complexity
- **When to use instead:** If advisory mode is too chatty

**Rationale for recommendation:** The three-part fix addresses all three root causes while preserving the original safety intent.

---

### Implementation Details

**What to implement first:**
1. Narrow the infrastructure keywords (safest, most impactful)
2. Reorder priority chain (config before infra)
3. Add advisory warning when config overrides infra detection

**Things to watch out for:**
- Don't remove the infrastructure gate entirely - it has a legitimate purpose
- Test with a task that mentions "serve.go" to ensure escape hatch still works
- Log when advisory mode is triggered for pattern analysis

**Success criteria:**
- `orch spawn architect "any orch-go task"` uses opencode backend when config says so
- `orch spawn architect "modify serve.go startup"` gets a warning and uses claude backend
- Explicit `--backend` flag always wins

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go:1135-1194` - Backend selection logic
- `cmd/orch/spawn_cmd.go:2273-2341` - `isInfrastructureWork()` function
- `.orch/config.yaml` - Project config with `spawn_mode: opencode`

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-10-dev-vs-prod-architecture.md` - Escape hatch rationale
- **Guide:** `.kb/guides/spawn.md` - Spawn system documentation
