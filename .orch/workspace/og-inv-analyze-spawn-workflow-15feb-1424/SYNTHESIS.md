# Session Synthesis

**Agent:** og-inv-analyze-spawn-workflow-15feb-1424
**Issue:** orch-go-untracked-1771204529 (ad-hoc spawn, no tracking)
**Duration:** 2026-02-15 17:15 → 2026-02-15 17:28
**Outcome:** success

---

## Plain-Language Summary

I analyzed the spawn workflow mechanics to verify the Spawn Architecture model's claims about how `orch spawn` creates workspaces, delivers context, and initializes sessions. I traced through the codebase to understand the 14-step spawn orchestration sequence, examined workspace name generation (kebab-case from task description with stop word filtering), verified KB context integration (tiered local→global search with 5-second timeout), confirmed the tier system (full vs light based on skill type), and validated session scoping via HTTP headers. The investigation confirmed most model invariants but found one outdated failure mode: cross-project spawns now correctly set session directories via the `x-opencode-directory` header, contradicting the model's claim that this remains broken.

---

## TLDR

Analyzed spawn workflow mechanics by examining source code (`pkg/spawn/*`, `cmd/orch/spawn_cmd.go`) and current workspace artifacts. Confirmed 6 model invariants (workspace naming, KB context, tier system, beads tracking), contradicted 1 outdated failure mode (cross-project directory setting is now fixed), and extended model with documentation of the 14-step spawn orchestration sequence.

---

## Delta (What Changed)

### Files Created
- `.kb/models/spawn-architecture/probes/2026-02-15-spawn-workflow-mechanics-analysis.md` - Probe documenting verification of spawn architecture model claims

### Files Modified
None (investigation only, no code changes)

### Commits
- Pending: Probe file and SYNTHESIS.md to be committed

---

## Evidence (What Was Observed)

### Code Examined

1. **Workspace Name Generation** (`pkg/spawn/config.go:254-374`)
   - Format: `{project-prefix}-{skill-prefix}-{task-slug}-{date}-{unique}`
   - `generateSlug` extracts meaningful words, filters 54 stop words
   - Project prefix: 2-part names use first letter of each part ("orch-go" → "og")
   - Unique suffix: 4-char hex (2 random bytes) prevents same-day collisions

2. **KB Context Integration** (`pkg/spawn/kbcontext.go:112-203`)
   - Tiered search: local first, global if <3 matches
   - 5-second timeout per query to prevent hangs
   - Post-filters global results to orch ecosystem repos
   - Applies per-category limits

3. **Tier System** (`pkg/spawn/config.go:26-48`)
   - Full tier: investigation, architect, research, codebase-audit, design-session, systematic-debugging
   - Light tier: feature-impl, reliability-testing, issue-creation
   - Default: TierFull (conservative)

4. **Session Directory Setting** (`pkg/spawn/backends/headless.go:88-124`)
   - Uses `x-opencode-directory` header in CreateSession API call
   - Passes `cfg.ProjectDir` explicitly (not CWD)
   - **Contradicts model failure mode** - this IS working

5. **Spawn Workflow** (`cmd/orch/spawn_cmd.go:1034-1145`)
   - 14-step sequence: pre-flight → resolve dir → load skill → setup beads → resolve model → gather context → extract repro → build usage → determine backend → load design → build context → build config → validate/write → dispatch

### Current Workspace Artifacts

```bash
# Workspace structure
.orch/workspace/og-inv-analyze-spawn-workflow-15feb-1424/
├── .beads_id           # orch-go-untracked-1771204529
├── .session_id         # OpenCode session ID
├── .spawn_mode         # claude (using Claude CLI backend)
├── .spawn_time         # 2026-02-15T17:15:31-08:00
├── .tier               # full
├── AGENT_MANIFEST.json # Complete metadata
├── screenshots/        # Visual artifacts directory
└── SPAWN_CONTEXT.md    # 57KB context delivery

# Untracked beads ID format
orch-go-untracked-1771204529
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/spawn-architecture/probes/2026-02-15-spawn-workflow-mechanics-analysis.md` - Verification probe

### Decisions Made
None (investigation only)

### Constraints Discovered
None new (confirmed existing model constraints)

### Model Updates Needed

**Spawn Architecture model should be updated:**

1. **Remove outdated failure mode** (Section: Why This Fails → Cross-Project Spawn Sets Wrong Session Directory)
   - **Model claim:** "Session directory is set from spawn caller's CWD, not `--workdir` target"
   - **Current reality:** Fixed via `x-opencode-directory` header in `pkg/spawn/backends/headless.go:93-110`
   - **Evidence:** `CreateSession` and `SendMessageInDirectory` explicitly pass `cfg.ProjectDir`

2. **Add spawn workflow sequence** (New invariant)
   - 14-step orchestration not documented in model
   - Key phases: pre-flight → context gathering → validation → dispatch
   - KB context integration happens in step 6 (gatherSpawnContext)
   - Tier determination happens in step 11 (buildSpawnConfig)

---

## Next (What Should Happen)

**Recommendation:** close

### Completion Checklist
- [x] All deliverables complete
  - [x] Probe file created in `.kb/models/spawn-architecture/probes/`
  - [x] Model claims verified via code examination
  - [x] Current workspace artifacts examined
  - [x] SYNTHESIS.md created
- [x] Probe status updated to Complete
- [x] Ready for commit and `/exit`

---

## Unexplored Questions

**Questions that emerged during this session:**

1. **Token estimation accuracy** - Model mentions "token estimation at 4 chars/token" but didn't verify actual calculation or accuracy. Worth investigating if this causes frequent warnings/errors.

2. **Skill content stripping for --no-track** - Model claims "Skill content stripped for --no-track" but only verified beads-related content is stripped, not whether entire skill content is removed. May need clarification.

3. **Session TTL values** - Observed worker sessions get 4-hour TTL, orchestrators get 0 (no expiration), but didn't trace where this is configured or if it's documented elsewhere.

---

## Session Metadata

**Skill:** investigation (probe mode - testing spawn-architecture model)
**Model:** anthropic/claude-sonnet-4-5-20250929
**Workspace:** `.orch/workspace/og-inv-analyze-spawn-workflow-15feb-1424/`
**Probe:** `.kb/models/spawn-architecture/probes/2026-02-15-spawn-workflow-mechanics-analysis.md`
**Beads:** orch-go-untracked-1771204529 (ad-hoc, no tracking)

---

## Verification Contract

**Link:** See workspace for VERIFICATION_SPEC.yaml (not created for probe-only investigation)

**Key Outcomes:**
- ✅ Confirmed 6 model invariants
- ✅ Contradicted 1 outdated failure mode (cross-project directory setting)
- ✅ Extended model with 14-step spawn workflow documentation
- ✅ Identified 2 model sections needing updates
