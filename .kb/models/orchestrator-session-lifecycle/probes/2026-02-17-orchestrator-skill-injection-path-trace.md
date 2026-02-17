# Probe: Orchestrator Skill Injection Path Trace

**Model:** orchestrator-session-lifecycle
**Date:** 2026-02-17
**Status:** Complete

---

## Question

The model claims orchestrator sessions operate in a three-tier hierarchy and that skill content is loaded via spawn context. But interactive OpenCode orchestrator sessions are reportedly running with stale skill versions (pre-gate, with "Spawning Checklist", "Post-Completion Verification" sections). Where does skill content get injected into OpenCode sessions, and why might old versions win?

---

## What I Tested

### 1. Identified all files containing orchestrator skill content

```bash
# Found 7 copies of the orchestrator skill across the system:
md5 ~/.claude/skills/orchestrator/SKILL.md ~/.claude/skills/meta/orchestrator/SKILL.md
# Both: 1c549b3fe17233068cb5d5f600bde84b (same file, newest version)

find ~/.opencode/skill/ -name "SKILL.md" -exec md5 {} \;
# ~/.opencode/skill/meta/orchestrator/SKILL.md  = 98f0b17... (checksum 658d5bd26dd9, Feb 15)
# ~/.opencode/skill/policy/orchestrator/SKILL.md = e390bf2... (checksum 1bf2c60bcef0, Feb 8)
# ~/.opencode/skill/SKILL.md                     = bfe7cc9... (checksum 0d727b600a4a, Feb 14)

find ~/.claude/skills/src/ -name "SKILL.md" | grep orchestrator
# ~/.claude/skills/src/meta/orchestrator/SKILL.md (checksum c35bd189eed1, Feb 7)
# ~/.claude/skills/src/meta/SKILL.md              (checksum d935d9468684, Feb 6)
```

### 2. Traced all injection paths

```bash
# Path 1: OpenCode plugin (orchestrator-session.ts)
cat orch-go/plugins/orchestrator-session.ts
# Line 133: skillPath = join(homedir(), ".claude", "skills", "meta", "orchestrator", "SKILL.md")
# Line 141-142: Caches content in memory at plugin init time
# Line 215-237: experimental.chat.system.transform injects cached content

# Path 2: Claude Code SessionStart hook
cat ~/.orch/hooks/load-orchestration-context.py
# Line 27: skill_path = Path.home() / '.claude' / 'skills' / 'orchestrator' / 'SKILL.md'
# BUG: Missing 'meta/' prefix — works by accident (duplicate file exists at that path)

# Path 3: orch spawn template
cat pkg/spawn/orchestrator_context.go
# Line 94: {{.SkillContent}} embedded in ORCHESTRATOR_CONTEXT.md
# Reads from pkg/skills/loader.go at spawn time

# Path 4: OpenCode Skill tool discovery
# OpenCode's src/skill/skill.ts scans:
#   ~/.opencode/skill/**/ + ~/.claude/skills/**/ = 7+ "orchestrator" skills found

# Path 5: OpenCode orch-hud.ts plugin
cat ~/.config/opencode/plugin/orch-hud.ts
# Only injects spawn state/backlog health — NOT skill content
```

### 3. Checked SessionStart hooks scope

```bash
cat ~/.claude/settings.json  # Lines 206-268: SessionStart hooks
# load-orchestration-context.py is in Claude Code settings.json
# OpenCode does NOT use Claude Code's hooks system
# OpenCode uses plugin hooks instead (experimental.chat.system.transform)
```

### 4. Verified OpenCode plugin behavior

```bash
# plugins/orchestrator-session.ts vs plugins/orchestrator-session.ts.backup:
# Backup used config hook (read file per instruction load)
# Current uses experimental.chat.system.transform (cached at init)
# The current version reads ONCE at init and caches — stale after skillc deploy
```

---

## What I Observed

### Complete Injection Path Map

| # | Path | Mechanism | Reads From | Caching | Applies To |
|---|------|-----------|------------|---------|------------|
| 1 | `orchestrator-session.ts` plugin | `experimental.chat.system.transform` | `~/.claude/skills/meta/orchestrator/SKILL.md` | **Init-time cache** | OpenCode sessions (interactive + spawned) |
| 2 | `load-orchestration-context.py` hook | Claude Code SessionStart | `~/.claude/skills/orchestrator/SKILL.md` (wrong path!) | Fresh read per session | Claude Code sessions only |
| 3 | `orchestrator_context.go` template | `{{.SkillContent}}` in ORCHESTRATOR_CONTEXT.md | `pkg/skills/loader.go` | Fresh at spawn time | `orch spawn`-ed orchestrators |
| 4 | OpenCode Skill tool | Agent calls Skill tool | All discovered skill dirs | Per-call discovery | Any OpenCode session (manual) |
| 5 | `orch-hud.ts` plugin | `experimental.chat.system.transform` | N/A | N/A | Does NOT inject skill content |

### Version Inventory (7 copies)

| Location | Checksum | Compiled | Version Identity |
|----------|----------|----------|------------------|
| `~/.claude/skills/meta/orchestrator/SKILL.md` | 94ffc2baf1c2 | Feb 16 | **Newest** (Strategic Comprehender + Orientation) |
| `~/.claude/skills/orchestrator/SKILL.md` | 94ffc2baf1c2 | Feb 16 | Duplicate at wrong path |
| `~/.opencode/skill/meta/orchestrator/SKILL.md` | 658d5bd26dd9 | Feb 15 | Intermediate (Strategic Comprehender, no Orientation) |
| `~/.opencode/skill/SKILL.md` | 0d727b600a4a | Feb 14 | Old (wrong level in dir) |
| `~/.opencode/skill/policy/orchestrator/SKILL.md` | 1bf2c60bcef0 | Feb 8 | Old ("What changed" section) |
| `~/.claude/skills/src/meta/orchestrator/SKILL.md` | c35bd189eed1 | Feb 7 | Stale (old src/ directory) |
| `~/.claude/skills/src/meta/SKILL.md` | d935d9468684 | Feb 6 | Stale (wrong directory level) |

### Bugs Found

1. **Plugin init-time caching (ROOT CAUSE for stale versions):** `orchestrator-session.ts` reads `~/.claude/skills/meta/orchestrator/SKILL.md` once at plugin init and caches in memory. `skillc deploy` does not trigger OpenCode server restart. Sessions get whatever version was on disk when the server last started.

2. **Wrong path in hook:** `load-orchestration-context.py` line 27 uses `~/.claude/skills/orchestrator/SKILL.md` (missing `meta/` prefix). Works by accident because a duplicate file happens to exist at that path. Fragile.

3. **7 stale copies never cleaned:** `skillc deploy` writes to new canonical paths but never cleans old deployment locations (`~/.opencode/skill/policy/`, `~/.opencode/skill/`, `~/.claude/skills/src/`). These are discoverable by OpenCode's skill scanner.

4. **Multi-skill name collision:** OpenCode discovers all SKILL.md files in `~/.opencode/skill/` and `~/.claude/skills/`. Multiple skills named "orchestrator" exist at different paths with different versions. If the Skill tool is called, first-match-wins behavior may return a stale version.

### The "Oldest Version" Mystery

None of the current files on disk contain the pre-gate sections ("Spawning Checklist" as section header, "Post-Completion Verification", "Amnesia-Resilient Artifact Design", "Common Red Flags"). These existed in the hand-written orchestrator skill (pre-skillc era, before Jan 29 2026). The most likely explanations:
- **If OpenCode server was running since before Jan 29**, the init-time cache in `orchestrator-session.ts` would serve that pre-skillc version
- **Old ORCHESTRATOR_CONTEXT.md files** in archived workspaces (from January) still contain the old skill content embedded at spawn time
- **The "Spawning Checklist" still exists** as a deprecated reference file at `~/.claude/skills/meta/orchestrator/reference/spawning-checklist.md` and `~/.claude/skills/reference/spawning-checklist.md`

---

## Model Impact

- [x] **Extends** model with: The model describes the three-tier hierarchy but does not document the skill injection paths. Five distinct injection paths exist, with the OpenCode `orchestrator-session.ts` plugin being the primary path for interactive sessions. The plugin's init-time caching is a systemic vulnerability: `skillc deploy` doesn't signal the server to reload, causing sessions to run with stale skill versions until manual restart.

- [x] **Extends** model with: Skill version sprawl — 7 copies of the orchestrator skill exist across the filesystem at different versions. `skillc deploy` creates new copies without cleaning old deployment locations, creating a growing graveyard of stale skill files that OpenCode's skill discovery can find.

- [x] **Confirms** invariant: The model's claim that orchestrators produce SESSION_HANDOFF.md (not SYNTHESIS.md) is confirmed by the `orchestrator_context.go` template. The ORCHESTRATOR_CONTEXT.md template embeds `{{.SkillContent}}` for spawned orchestrators, creating a fresh-at-spawn-time snapshot of the skill.

---

## Notes

### Recommended Fixes (Priority Order)

1. **Fix plugin caching:** Change `orchestrator-session.ts` to re-read the skill file periodically (e.g., check mtime) or on each transform call (file reads are cheap at ~35KB). The backup version used `config.instructions.push(skillPath)` which let OpenCode handle reading — reverting to that approach would also fix this.

2. **Fix hook path:** Change `load-orchestration-context.py` line 27 from `'.claude' / 'skills' / 'orchestrator' / 'SKILL.md'` to `'.claude' / 'skills' / 'meta' / 'orchestrator' / 'SKILL.md'`.

3. **Clean stale copies:** Delete old deployment locations:
   - `~/.opencode/skill/policy/orchestrator/`
   - `~/.opencode/skill/SKILL.md` (orchestrator at root)
   - `~/.claude/skills/src/meta/orchestrator/`
   - `~/.claude/skills/src/meta/SKILL.md`
   - `~/.claude/skills/orchestrator/` (non-canonical duplicate)

4. **Add cleanup to `skillc deploy`:** When deploying to a new path, remove files at old paths to prevent stale copies accumulating.

### Cross-Reference

- Prior probe: `2026-02-15-orchestrator-skill-deployment-sync.md`
- Prior probe: `2026-02-16-orchestrator-skill-orientation-redesign.md`
- Constraint: "Worker spawns must set ORCH_WORKER=1 to skip orchestrator skill loading"
- Constraint: "orch-knowledge repo is at ~/orch-knowledge"
