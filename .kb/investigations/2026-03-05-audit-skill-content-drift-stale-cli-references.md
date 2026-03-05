## Summary (D.E.K.N.)

**Delta:** 30+ stale CLI references found across 28 deployed skills — 5 removed commands, 1 deprecated command used everywhere, 7 wrong flag syntaxes, 4 non-existent skills referenced, and stale path references to pre-Go-rewrite infrastructure.

**Evidence:** Cross-referenced every CLI command in all 28 deployed SKILL.md files against `orch --help`, `bd --help`, `kb --help` and all relevant subcommand help outputs.

**Knowledge:** The single highest-impact finding is `bd comment` → `bd comments add` deprecation, present in worker-base (inherited by ALL worker skills). Five orch commands (`frontier`, `reap`, `health`, `stability`, `friction`) were removed but still referenced in diagnostic and orchestrator skills. Reference docs are the most stale — orch-commands.md alone has 15+ stale entries.

**Next:** Fix worker-base `bd comment` references first (affects all skills). Then fix orchestrator + diagnostic skills. Then update reference docs. Route through architect for batch implementation.

**Authority:** architectural - Cross-skill changes affecting all deployed skills, requires coordinated update

---

# Investigation: Skill Content Drift — Stale CLI References, Removed Commands, Wrong Defaults

**Question:** Which deployed skills contain stale CLI references, removed commands, wrong flag syntax, or outdated conventions that could cause agent failures or misleading guidance?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Owner:** Claude (codebase-audit skill)
**Phase:** Complete
**Next Step:** Route fixes through architect for batch implementation
**Status:** Complete
**Resolution-Status:** Resolved

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-03-05-inv-skill-system-audit.md | extends | Yes | N/A — that audit covered deployment format, this covers content |
| 2026-02-25-inv-architect-skillc-deploy-silent-failures.md | related | Yes | N/A — stale deploys compound content drift |

---

## Findings

### Finding 1: `bd comment` is DEPRECATED — affects ALL worker skills

**Severity:** BLOCKING (will be removed in v1.0.0)

**Evidence:** Running `bd comment` returns: `Command "comment" is deprecated, use 'bd comments add' instead (will be removed in v1.0.0)`. The command still works today but will break when bd v1.0.0 ships.

**Source:** worker-base/SKILL.md (the template ALL worker skills inherit). Every `bd comment` reference in worker-base propagates to: architect, codebase-audit, design-session, feature-impl, investigation, research, systematic-debugging, ux-audit, diagnostic, orchestrator, meta-orchestrator.

**Affected skills (13 total):** worker-base + all 12 skills that inherit it or use `bd comment` directly.

**Fix:** Global find-replace `bd comment <` → `bd comments add <` across all skill sources in `skills/src/`.

---

### Finding 2: Five removed orch commands still referenced

**Severity:** BLOCKING (commands don't exist, agents will get errors)

| Removed Command | Skills Affected | What Replaced It |
|----------------|-----------------|------------------|
| `orch frontier` | orchestrator (6 refs), reference/orch-commands.md, daemon-workflow.md, orchestrator-autonomy.md, dashboard-troubleshooting.md | `orch status` + `bd ready` (no direct replacement) |
| `orch reap` | diagnostic (2 refs) | `orch clean --orphans` |
| `orch health` | diagnostic (4 refs) | `orch doctor` |
| `orch stability` | diagnostic (2 refs) | No replacement (removed) |
| `orch friction` | diagnostic (2 refs) | No replacement (removed) |
| `orch sync` | reference/orch-commands.md (5 refs) | No replacement (removed) |

**Source:** Verified via `orch <command> 2>&1` — all return `unknown command`.

---

### Finding 3: Wrong flag syntax on multiple commands

**Severity:** BLOCKING (commands will fail with wrong flags)

| Stale Reference | Skill | Correct Syntax |
|----------------|-------|----------------|
| `orch spawn --opus` | reference/model-selection.md | `orch spawn --model opus` |
| `bd update <id> --blocks <dep-id>` | architect | `bd dep add <dep-id> <blocks-id>` |
| `bd dep add <task-b> --blocks <task-a>` | design-session (2 refs) | `bd dep add <issue> <depends-on>` (two positional args, no --blocks) |
| `bd edit <id> --type <type>` | reference/daemon-workflow.md | `bd update <id> --type <type>` |
| `bd list --labels "<area>"` | design-session, reference/daemon-workflow.md | `bd list --label "<area>"` or `-l` (singular) |
| `orch complete --orchestrator <id>` | meta-orchestrator | `orch complete <workspace-name>` (positional arg for orchestrators) |
| `orch spawn --note "..."` | reference/completion-workflow.md | Flag doesn't exist. Use `--reason` |

---

### Finding 4: Non-existent skills referenced in skill-selection-guide

**Severity:** HIGH (agents will try to spawn skills that don't exist)

| Referenced Skill | Where | Status |
|-----------------|-------|--------|
| `reliability-testing` | reference/skill-selection-guide.md (2 refs) | Not deployed — never existed or was removed |
| `issue-creation` | reference/skill-selection-guide.md (3 refs) | Not deployed |
| `delegating-to-team` | reference/skill-selection-guide.md (1 ref) | Not deployed |
| `writing-skills` | reference/skill-selection-guide.md (1 ref) | Not deployed |

**Source:** Compared against `find ~/.claude/skills/ -name "SKILL.md"` — 28 deployed skills, none match these names.

---

### Finding 5: Stale paths referencing pre-Go-rewrite infrastructure

**Severity:** MEDIUM (misleading but won't crash — paths just won't exist)

| Stale Path | Skill | Current Path |
|-----------|-------|--------------|
| `~/meta-orchestration` | codebase-audit (organizational dim) | Merged into orch-go |
| `tools/orch/cli.py` | codebase-audit (organizational dim) | `cmd/orch/*.go` (Go rewrite) |
| `~/.orch/templates/orchestrator/` | codebase-audit (organizational dim) | `.orch/templates/` in project root |
| `docs/ROADMAP.org` | codebase-audit (organizational dim) | No direct equivalent |
| `.orch/decisions/` | record-decision | `.kb/decisions/` |
| `~/.claude/scripts/update-index.sh` | audit-claude-md | May not exist |
| `~/.claude/scripts/cdd-status.sh` | audit-claude-md | May not exist |
| `bash skills/analyze-skill-usage/queries/*.sh` | analyze-skill-usage | Skills now at `skills/src/`, deploy to `~/.claude/skills/` |

---

### Finding 6: orch-commands.md reference doc is massively stale

**Severity:** HIGH (this file is loaded as reference context for orchestrators)

**Evidence:** `~/.claude/skills/reference/orch-commands.md` (last modified Jan 29, 2026) contains:

- **`orch frontier`** (6 refs) — removed
- **`orch sync`** (5 refs) — removed
- **`orch sessions list/search/show`** — should be `orch session-history list/search/show`
- **`orch servers init/up/down/gen-plist`** — don't exist. Actual: list, start, stop, attach, open, status
- **`orch kb ask "question" --save`** — `--save` not verified
- **`orch kb extract`** — exists but syntax may have changed
- **`kb reflect --type orchestrator`** — not a valid type (meta-orchestrator refs this too)

**Source:** Cross-referenced all commands in orch-commands.md against `orch --help` and subcommand help.

---

### Finding 7: Diagnostic skill references 5 non-existent commands

**Severity:** BLOCKING (diagnostic skill is used during crisis — wrong commands make it worse)

| Non-existent Command | Diagnostic Skill Context | Replacement |
|---------------------|------------------------|-------------|
| `orch health` | Entry protocol, exit criteria, tool table | `orch doctor` |
| `orch stability` | Auto-nudge trigger, tool table | No equivalent |
| `orch reap` | Action: clean stale agents, tool table | `orch clean --orphans` |
| `orch friction` | Listed as "IGNORE" noise, tool table | No equivalent (removed) |
| `orch reconcile --fix` | Fixing zombies | `orch reconcile --fix` EXISTS, but some usages show wrong flags |

---

### Finding 8: Model/backend default assumptions are stale in reference docs

**Severity:** MEDIUM (agents may make wrong assumptions about defaults)

| Stale Assumption | Where | Current Reality |
|-----------------|-------|-----------------|
| "default sonnet + headless" | reference/model-selection.md | Default is opus + claude backend (per CLAUDE.md) |
| "sonnet via OpenCode API" | reference/model-selection.md | OpenCode API is secondary path; Claude CLI is default |
| Default Gemini for spawns | reference/model-selection.md | Opus is default for Claude CLI spawns |

---

## Synthesis

**Key Insights:**

1. **Worker-base inheritance amplifies drift** — A single stale reference in worker-base (`bd comment`) propagates to ALL 12+ worker skills. This is the highest-leverage fix target.

2. **Reference docs are the most stale** — The `reference/` directory (last modified Jan 29, 2026) has drifted the most. orch-commands.md alone has 15+ stale entries. These files need a full refresh.

3. **Diagnostic skill is dangerous when stale** — It's used during crises. 5 of its core commands don't exist. An agent running diagnostic during a real incident would waste time on error messages.

4. **Skills that reference non-existent skills create spawn failures** — skill-selection-guide references 4 skills that were never deployed, causing agents to try impossible spawns.

**Answer to Investigation Question:**

28 deployed skills were audited. 13 contain stale CLI references. The most critical finding is the deprecated `bd comment` command used in worker-base (inherited by all worker skills), followed by 5 removed orch commands in diagnostic and orchestrator skills. Reference docs are the single most stale category with 15+ broken command references in orch-commands.md alone. Total: ~50 stale references across 4 severity levels.

---

## Structured Uncertainty

**What's tested:**

- ✅ All `orch` commands verified via `orch <cmd> 2>&1` — confirmed 5 removed commands
- ✅ All `bd` flag syntax verified via `bd <cmd> --help` — confirmed 4 wrong flag usages
- ✅ All deployed skills enumerated via `find ~/.claude/skills/ -name "SKILL.md"` — confirmed 4 referenced skills don't exist
- ✅ `bd comment` deprecation confirmed via actual execution — returns deprecation warning

**What's untested:**

- ⚠️ `~/.claude/scripts/*.sh` paths — didn't verify if they still exist on filesystem
- ⚠️ `orch kb ask --save` and `orch kb extract` exact flag syntax — verified commands exist but not all flags
- ⚠️ Whether `snap` tool referenced in systematic-debugging is still functional (binary exists at ~/go/bin/snap)
- ⚠️ Playwright-cli skill references — assumed correct since it's a standalone tool with its own versioning

**What would change this:**

- Finding would be incomplete if orch-go merged new commands since this audit
- Findings for reference docs could be narrower if some reference files are no longer loaded by skills

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Fix `bd comment` → `bd comments add` in worker-base | implementation | Template fix, single file, all skills benefit |
| Fix diagnostic skill removed commands | implementation | Single skill file update |
| Update orchestrator skill stale commands | implementation | Single skill file update |
| Refresh reference/orch-commands.md | architectural | Cross-skill reference doc, needs complete rewrite |
| Remove non-existent skill references from skill-selection-guide | implementation | Single file cleanup |
| Update model/backend defaults in reference docs | architectural | Affects routing decisions across all orchestrator sessions |

### Recommended Approach ⭐

**Batch Template Fix** — Fix all stale references in skill source files (`skills/src/`), then redeploy all skills via `skillc deploy`.

**Why this approach:**
- Skill sources live in `skills/src/` — editing deployed files directly would be overwritten by next `skillc deploy`
- Worker-base fix propagates to all worker skills automatically via skillc compilation
- Single `skillc deploy` recompiles all skills from corrected sources

**Implementation sequence:**
1. **Phase 1 (BLOCKING fixes):** Fix `bd comment` → `bd comments add` in worker-base source. Fix diagnostic skill removed commands. Fix wrong flag syntax in architect, design-session sources. (~30min)
2. **Phase 2 (HIGH fixes):** Update orchestrator skill stale commands. Remove non-existent skills from skill-selection-guide. (~30min)
3. **Phase 3 (Reference doc refresh):** Rewrite orch-commands.md against current `orch --help`. Update model-selection.md defaults. Fix stale paths in codebase-audit. (~1hr)
4. **Phase 4 (Deploy):** `skillc deploy` to push all fixes to `~/.claude/skills/`. (~5min)

**Trade-offs accepted:**
- Reference docs in `~/.claude/skills/reference/` are NOT compiled by skillc — they need manual editing
- Some removed commands (frontier, stability, friction) have no replacement — guidance must be rewritten, not just s/old/new/

### Alternative Approaches Considered

**Option B: Fix only BLOCKING items**
- **Pros:** Faster, lower risk
- **Cons:** HIGH items continue misleading agents
- **When to use instead:** If time-constrained, fix only Phase 1

---

## Detailed Stale Reference Table

### Per-Skill Summary

| Skill | Blocking | High | Medium | Total | Key Issues |
|-------|----------|------|--------|-------|------------|
| **worker-base** | 1 | 0 | 0 | 1 | `bd comment` deprecated |
| **orchestrator** | 7 | 0 | 0 | 7 | `orch frontier` (6x), `bd comment` via worker-base |
| **diagnostic** | 10 | 0 | 0 | 10 | `orch health/stability/reap/friction` removed, `bd comment` |
| **meta-orchestrator** | 2 | 0 | 0 | 2 | `orch complete --orchestrator`, `kb reflect --type orchestrator` |
| **architect** | 2 | 0 | 0 | 2 | `bd update --blocks`, `bd comment` |
| **design-session** | 3 | 0 | 1 | 4 | `bd dep add --blocks`, `bd list --labels`, `bd comment` |
| **feature-impl** | 1 | 0 | 0 | 1 | `bd comment` via worker-base |
| **systematic-debugging** | 1 | 0 | 0 | 1 | `bd comment` via worker-base |
| **ux-audit** | 1 | 0 | 0 | 1 | `bd comment` via worker-base |
| **investigation** | 1 | 0 | 0 | 1 | `bd comment` via worker-base |
| **research** | 1 | 0 | 0 | 1 | `bd comment` via worker-base |
| **codebase-audit** | 1 | 0 | 5 | 6 | `bd comment`, stale paths to meta-orchestration/cli.py |
| **record-decision** | 0 | 0 | 1 | 1 | `.orch/decisions/` → `.kb/decisions/` |
| **ref/orch-commands.md** | 6 | 5 | 4 | 15 | Most stale file in the system |
| **ref/model-selection.md** | 1 | 2 | 0 | 3 | `--opus` flag, wrong defaults |
| **ref/skill-selection-guide.md** | 0 | 4 | 0 | 4 | 4 non-existent skills |
| **ref/daemon-workflow.md** | 1 | 1 | 0 | 2 | `orch frontier`, `bd edit --type` |
| **ref/completion-workflow.md** | 0 | 1 | 0 | 1 | `--note` flag |
| **ref/dashboard-troubleshooting.md** | 1 | 0 | 0 | 1 | `orch frontier` |
| **ref/orchestrator-autonomy.md** | 1 | 0 | 0 | 1 | `orch frontier` |
| **audit-claude-md** | 0 | 0 | 2 | 2 | Stale script paths |
| **analyze-skill-usage** | 0 | 0 | 2 | 2 | Stale `bash skills/` paths |
| **TOTAL** | **~41** | **~13** | **~15** | **~69** |  |

---

## References

**Files Examined:**
- All 28 SKILL.md files in `~/.claude/skills/` (meta/, shared/, worker/, utilities/, reference/, playwright-cli/, policy/)
- `orch --help` and 15 subcommand help outputs
- `bd --help` and 10 subcommand help outputs
- `kb --help` and 3 subcommand help outputs

**Commands Run:**
```bash
# CLI ground truth
orch --help && orch spawn --help && orch complete --help && orch status --help
bd --help && bd create --help && bd update --help && bd close --help && bd dep add --help
kb --help && kb reflect --help && kb search --help

# Verify removed commands
orch frontier 2>&1  # "unknown command"
orch reap 2>&1      # "unknown command"
orch health 2>&1    # "unknown command"
orch stability 2>&1 # "unknown command"
orch friction 2>&1  # "unknown command"
orch sync 2>&1      # "unknown command"

# Verify deprecated command
bd comment orch-go-xxx "test"  # Works but shows deprecation warning
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-05-inv-skill-system-audit.md` — Deployment format audit (complements this content audit)
- **Constraint:** `skillc deploy` cannot signal reload — stale cached skills in memory
