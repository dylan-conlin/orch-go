<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Skill-type detection implemented in spawn_cmd.go - orchestrator-type skills now default to tmux mode.

**Evidence:** Build passes, tests pass (61.585s for cmd/orch package), changes in spawn_cmd.go and config.go verified.

**Knowledge:** Skill metadata parsing via `ParseSkillMetadata` already exists; orchestrator detection is a simple conditional based on `skill-type` field being "policy" or "orchestrator".

**Next:** Close this phase; Phase 2 (ORCHESTRATOR_CONTEXT.md template) and Phase 3 (completion verification) can proceed independently.

---

# Investigation: Phase 1 - Skill-Type Detection in Spawn

**Question:** How to detect orchestrator-type skills at spawn time and modify defaults accordingly?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Feature Implementation Agent
**Phase:** Complete
**Next Step:** None - implementation complete
**Status:** Complete

**Related-From:** `.kb/investigations/2026-01-04-inv-spawnable-orchestrator-sessions-infrastructure-changes.md`

---

## Findings

### Finding 1: Skill Metadata Parsing Already Exists

**Evidence:** The `skills.ParseSkillMetadata()` function in `pkg/skills/loader.go:168-190` extracts YAML frontmatter from skill content. The `SkillMetadata` struct includes a `SkillType` field that maps to `skill-type` in YAML.

**Source:** `pkg/skills/loader.go:21-30` (struct definition), `pkg/skills/loader.go:168-190` (parsing function)

**Significance:** No new parsing logic needed - just call `ParseSkillMetadata(skillContent)` after `LoadSkillWithDependencies`.

---

### Finding 2: Skill Loading Happens Early in runSpawnWithSkill

**Evidence:** `LoadSkillWithDependencies` is called at line 566-570 in `spawn_cmd.go`, well before the spawn mode decision at line 778-791. This gives us access to skill content early enough to detect orchestrator-type and influence defaults.

**Source:** `cmd/orch/spawn_cmd.go:566-570` (skill loading), `cmd/orch/spawn_cmd.go:778-791` (spawn mode selection)

**Significance:** The control flow already supports adding detection logic between loading and mode selection.

---

### Finding 3: Spawn Mode Selection is a Simple Conditional

**Evidence:** The spawn mode selection at line 778-791 is a straightforward if/else chain: `inline` → `tmux || attach` → default (headless). Adding orchestrator detection just extends this conditional.

**Source:** `cmd/orch/spawn_cmd.go:778-791`

**Significance:** Minimal code change required - extend the tmux condition to include `isOrchestrator`.

---

## Implementation

The following changes were made:

### 1. Added `IsOrchestrator` field to spawn.Config

```go
// pkg/spawn/config.go:131-137
IsOrchestrator bool
```

### 2. Added skill-type detection after LoadSkillWithDependencies

```go
// cmd/orch/spawn_cmd.go (after line 570)
isOrchestrator := false
if skillContent != "" {
    if metadata, err := skills.ParseSkillMetadata(skillContent); err == nil {
        isOrchestrator = metadata.SkillType == "policy" || metadata.SkillType == "orchestrator"
    }
}
```

### 3. Modified spawn mode selection to default orchestrators to tmux

```go
// cmd/orch/spawn_cmd.go (modified mode selection)
useTmux := tmux || attach || cfg.IsOrchestrator
if useTmux {
    return runSpawnTmux(...)
}
```

### 4. Passed isOrchestrator to spawn config

```go
// cmd/orch/spawn_cmd.go (in Config initialization)
IsOrchestrator: isOrchestrator,
```

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles (`go build ./...` passed)
- ✅ Existing tests pass (`go test ./...` - cmd/orch took 61.585s, all pass)
- ✅ `skills.ParseSkillMetadata` correctly parses skill-type (existing tests cover this)

**What's untested:**

- ⚠️ End-to-end test with actual orchestrator skill spawn (not run)
- ⚠️ Behavior when skill content is empty or has no frontmatter (relies on existing error handling)

**What would change this:**

- If `skill-type: policy` skills need different behavior than `skill-type: orchestrator`, we'd need separate detection
- If tmux default causes issues for orchestrators, we could add `--headless` override flag

---

## References

**Files Modified:**
- `cmd/orch/spawn_cmd.go` - Added skill-type detection and isOrchestrator logic
- `pkg/spawn/config.go` - Added IsOrchestrator field to Config struct

**Commands Run:**
```bash
# Build verification
/opt/homebrew/bin/go build ./...

# Test verification
/opt/homebrew/bin/go test ./...
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-04-inv-spawnable-orchestrator-sessions-infrastructure-changes.md` - Prior design investigation that defined this phase

---

## Investigation History

**2026-01-04 08:50:** Investigation started
- Initial question: How to detect orchestrator-type skills and modify spawn defaults?
- Context: Phase 1 of spawnable orchestrator sessions feature

**2026-01-04 09:05:** Implementation complete
- Status: Complete
- Key outcome: Skill-type detection added to spawn_cmd.go, orchestrator skills default to tmux mode
