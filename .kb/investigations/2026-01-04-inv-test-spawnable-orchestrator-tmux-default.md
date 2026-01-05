<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Orchestrator-type skills default to tmux mode when spawned, but the `--headless` flag does not override this default (a bug).

**Evidence:** Spawning with `orch spawn orchestrator "test"` attempted tmux (failed due to PATH); spawning with `--headless` flag also attempted tmux. Code at spawn_cmd.go:789 shows logic: `useTmux := tmux || attach || cfg.IsOrchestrator`.

**Knowledge:** The spawnable orchestrator infrastructure is functional - it creates ORCHESTRATOR_CONTEXT.md, embeds the orchestrator skill, and uses different completion protocols (SESSION_HANDOFF.md instead of SYNTHESIS.md). However, there's no way to force headless mode for orchestrator skills.

**Next:** Create beads issue for the `--headless` flag bug (it should override IsOrchestrator).

---

# Investigation: Test Spawnable Orchestrator Tmux Default

**Question:** Does the spawnable orchestrator infrastructure work correctly, and do orchestrator-type skills default to tmux mode?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Meta-orchestrator worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Orchestrator-type skills are correctly detected

**Evidence:** The spawn command detects orchestrator-type skills by checking the skill metadata `skill-type` field. Skills with `skill-type: policy` or `skill-type: orchestrator` are treated as orchestrator spawns.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go:570-576`
```go
isOrchestrator := false
rawSkillContent, err := loader.LoadSkillContent(skillName)
if err == nil {
    if metadata, err := skills.ParseSkillMetadata(rawSkillContent); err == nil {
        isOrchestrator = metadata.SkillType == "policy" || metadata.SkillType == "orchestrator"
    }
}
```

**Significance:** This enables differentiated handling of orchestrator spawns vs worker spawns.

---

### Finding 2: Orchestrator-type skills default to tmux mode

**Evidence:** At line 789, the spawn mode logic is:
```go
useTmux := tmux || attach || cfg.IsOrchestrator
```

When I ran `orch spawn orchestrator "test spawn tmux default" --no-track`, it attempted to spawn in tmux mode and failed with:
```
Error: failed to ensure tmux session: failed to create session: exec: "tmux": executable file not found in $PATH
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go:787-793`

**Significance:** Confirms the design intent - orchestrator spawns use tmux for interactive visibility, while workers default to headless.

---

### Finding 3: The `--headless` flag does NOT override orchestrator default

**Evidence:** Running `orch spawn orchestrator "test" --no-track --headless` still attempted tmux mode. The `headless` parameter is passed to `runSpawnWithSkill` but is never used in the mode selection logic.

**Source:** Function signature shows `headless bool` parameter but it's not checked at line 789.

**Significance:** This is a bug - users cannot force headless mode for orchestrator skills when desired.

---

### Finding 4: ORCHESTRATOR_CONTEXT.md is generated correctly

**Evidence:** Spawning created workspaces with proper orchestrator context files:
- `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-work-test-spawn-tmux-04jan/ORCHESTRATOR_CONTEXT.md`
- `.orchestrator` marker file with content "orchestrator-spawn"

The context file includes:
- Session Goal from task description
- Skill guidance (full orchestrator skill embedded)
- Different completion protocol (SESSION_HANDOFF.md instead of SYNTHESIS.md)
- Uses `orch session end` instead of `/exit`

**Source:** Examined generated files in workspace directory.

**Significance:** The spawnable orchestrator infrastructure is functionally complete.

---

### Finding 5: meta-orchestrator skill exists and is spawnable

**Evidence:** The meta-orchestrator skill exists at `/Users/dylanconlin/.claude/skills/meta/meta-orchestrator/SKILL.md` with:
- skill-type: policy
- dependencies: orchestrator (inherits full orchestrator skill)
- Describes three-tier hierarchy: Worker → Orchestrator → Meta-Orchestrator

**Source:** `/Users/dylanconlin/.claude/skills/meta/meta-orchestrator/SKILL.md`

**Significance:** The meta-orchestrator concept has skill infrastructure in place and is spawnable via `orch spawn meta-orchestrator "goal"`.

---

## Synthesis

**Key Insights:**

1. **Spawnable orchestrator infrastructure is functional** - The system correctly detects orchestrator-type skills, generates appropriate context (ORCHESTRATOR_CONTEXT.md), and uses different completion protocols suitable for orchestrator sessions.

2. **tmux default for orchestrators is intentional** - The comment at line 787 explicitly states "Orchestrator-type skills default to tmux mode (visible interaction)" while "Workers default to headless mode (automation-friendly)". This reflects the design principle that orchestrators need visibility for interactive work.

3. **Bug: `--headless` flag is ignored for orchestrator skills** - The flag is passed but not used in the mode selection logic. This prevents forcing headless mode when desired (e.g., for CI/CD or automated testing).

**Answer to Investigation Question:**

Yes, the spawnable orchestrator infrastructure works correctly. Orchestrator-type skills (identified by `skill-type: policy` or `skill-type: orchestrator`) default to tmux mode as designed. The infrastructure generates ORCHESTRATOR_CONTEXT.md with appropriate session-oriented guidance, uses SESSION_HANDOFF.md for completion instead of SYNTHESIS.md, and instructs agents to use `orch session end` instead of `/exit`.

However, there's a bug: the `--headless` flag should be able to override the orchestrator default to tmux, but currently it doesn't.

---

## Structured Uncertainty

**What's tested:**

- ✅ Orchestrator skill detection works (verified: spawned orchestrator skill, detected as IsOrchestrator=true)
- ✅ tmux is default for orchestrators (verified: spawn attempt failed with "tmux not found" error)
- ✅ ORCHESTRATOR_CONTEXT.md is generated (verified: file exists with correct content)
- ✅ .orchestrator marker file is created (verified: contains "orchestrator-spawn")

**What's untested:**

- ⚠️ Full orchestrator session lifecycle (spawn → work → SESSION_HANDOFF.md → orch session end) not tested end-to-end
- ⚠️ `orch complete` behavior for orchestrator tier spawns not tested
- ⚠️ Integration with `orch review` for orchestrator sessions not tested

**What would change this:**

- Finding would be wrong if spawning in tmux actually worked (proving tmux IS in PATH in other contexts)
- The `--headless` bug finding would be wrong if there's a different code path that handles the flag

---

## Implementation Recommendations

### Recommended Approach ⭐

**Fix the `--headless` flag to override orchestrator default** - Modify the mode selection logic to respect the `--headless` flag.

**Why this approach:**
- Preserves the intentional default (orchestrators use tmux)
- Enables override when needed (CI/CD, automated testing, environments without tmux)
- Maintains backward compatibility

**Trade-offs accepted:**
- None significant

**Implementation sequence:**
1. Modify spawn_cmd.go:789 to check headless flag: `useTmux := (tmux || attach || cfg.IsOrchestrator) && !headless`
2. Add test case for orchestrator spawn with --headless
3. Update help text to note that --headless overrides orchestrator default

### Alternative Approaches Considered

**Option B: Add --force-headless flag**
- **Pros:** Explicit, doesn't change existing flag semantics
- **Cons:** Redundant flag, --headless already exists
- **When to use instead:** If there's concern about breaking existing scripts

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go` - Main spawn command implementation
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/orchestrator_context.go` - ORCHESTRATOR_CONTEXT.md template
- `/Users/dylanconlin/.claude/skills/meta/meta-orchestrator/SKILL.md` - Meta-orchestrator skill definition
- `/Users/dylanconlin/.claude/skills/meta/orchestrator/SKILL.md` - Orchestrator skill definition

**Commands Run:**
```bash
# Test spawn orchestrator
~/bin/orch spawn orchestrator "test spawn tmux default" --no-track

# Test with headless flag
~/bin/orch spawn orchestrator "test spawn headless override" --no-track --headless

# Check generated files
ls -la /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-work-test-spawn-tmux-04jan/
```

**Related Artifacts:**
- **Investigation:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-04-inv-test-spawnable-orchestrator-infrastructure.md` - Prior investigation template (empty)
- **Workspace:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-work-test-spawn-tmux-04jan/` - Test spawn workspace

---

## Investigation History

**[2026-01-04 17:15]:** Investigation started
- Initial question: Does the spawnable orchestrator infrastructure work, and do orchestrator-type skills default to tmux?
- Context: Testing meta-orchestrator skill spawning capability

**[2026-01-04 17:18]:** Key finding - orchestrator spawns default to tmux
- Attempted spawn failed with "tmux not found" - proving tmux IS the default mode
- Confirmed code logic at spawn_cmd.go:789

**[2026-01-04 17:20]:** Bug discovered - --headless flag ignored
- Tested with --headless flag, still tried tmux
- Identified root cause: headless parameter not used in mode selection

**[2026-01-04 17:25]:** Investigation completed
- Status: Complete
- Key outcome: Spawnable orchestrator infrastructure works; bug found in --headless override
