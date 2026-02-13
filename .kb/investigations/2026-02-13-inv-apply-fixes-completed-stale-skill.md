## Summary (D.E.K.N.)

**Delta:** Found and fixed 25+ stale CLI references across 8 skill files — commands that were removed or renamed since the Jan 18 revert.

**Evidence:** Grepped all ~23 skills in ~/.opencode/skill/ for every orch subcommand; cross-referenced against `orch --help` output to identify non-existent commands.

**Knowledge:** Six orch commands were referenced but don't exist: `orch health`, `orch stability`, `orch frontier`, `orch rework`, `orch reflect`, `orch reap`, `orch automation`, `orch kb archive-old`. Also `--untracked` flag on `orch clean` and `-i` shorthand for `--inline` on spawn.

**Next:** None - fixes applied directly to deployed skill files.

**Authority:** implementation - Tactical cleanup of stale references within existing patterns.

---

# Investigation: Apply Fixes Completed Stale Skill References

**Question:** Which deployed skill files reference orch CLI commands/features that don't exist at the current codebase state?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Six non-existent orch subcommands referenced

**Evidence:** `orch health`, `orch stability`, `orch frontier`, `orch rework`, `orch reflect`, `orch reap`, `orch automation`, `orch kb archive-old` — all return "unknown command" from `orch --help`.

**Source:** `go run ./cmd/orch/ {command} --help` for each; all returned errors.

**Significance:** Agents following these skill instructions would get CLI errors, wasting context and time.

### Finding 2: `orch spawn -i` shorthand doesn't exist

**Evidence:** The correct flag is `--inline`, not `-i`. Referenced in architect, feature-impl, systematic-debugging, and research skills.

**Source:** `go run ./cmd/orch/ spawn --help` — no `-i` shorthand, only `--inline`.

### Finding 3: `orch clean --untracked` flag doesn't exist

**Evidence:** `orch clean` supports `--stale`, `--ghosts`, `--windows`, etc. but not `--untracked`.

**Source:** `go run ./cmd/orch/ clean --help`

---

## Files Modified

| File | Changes |
|------|---------|
| `meta/diagnostic/SKILL.md` | Removed `orch health` (6 refs), `orch stability` (3 refs), `orch reap` → `orch clean --ghosts` (3 refs) |
| `policy/orchestrator/SKILL.md` | `orch frontier` → `orch status` (4 refs), removed `orch rework/reflect/kb archive-old/automation` block |
| `meta/orchestrator/SKILL.md` | `orch frontier` → `orch status` (2 refs), removed `orch rework/reflect/kb archive-old`, fixed `--untracked` |
| `meta/meta-orchestrator/SKILL.md` | `orch frontier` → `orch status` + `bd ready` (2 refs) |
| `worker/architect/SKILL.md` | `-i` → `--inline` (3 refs) |
| `worker/feature-impl/reference/phase-design.md` | `architect -i` → `architect --inline` |
| `worker/feature-impl/src/phases/design.md` | `architect -i` → `architect --inline` |
| `worker/systematic-debugging/SKILL.md` | `architect -i` → `architect --inline` |
| `worker/systematic-debugging/phases/phase1-root-cause.md` | `architect -i` → `architect --inline` |
| `worker/research/SKILL.md` | `architect -i` → `architect --inline` (2 refs) |

## Verified Clean

Final grep for all stale patterns returned zero matches:
- `orch (health|stability|frontier|rework|reap|automation|lint|reflect)` — 0 hits
- `orch kb archive|orch clean --untracked` — 0 hits
- `architect -i[^n]` — 0 hits
- `state.db|.orch/worktrees` — 0 hits (never existed in skills)

## Not Changed (confirmed valid)

- `orch doctor` / `orch doctor --fix` — exists
- `orch stats` — exists
- `orch reconcile` / `orch reconcile --fix --mode reset` — exists
- `orch review` / `orch review done` / `orch review --all` — exists
- `orch status --all` — exists
- `orch clean --stale` — exists
- `orch daemon reflect` — exists
- CLAUDE.md files in orch-go — no stale references found
