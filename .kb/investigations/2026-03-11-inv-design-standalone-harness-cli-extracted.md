## Summary (D.E.K.N.)

**Delta:** orch-go already contains a clean standalone harness path — `harness_init.go`'s standalone mode, `pkg/verify/accretion*.go`, and `pkg/control/` have zero orch-go dependencies and can be extracted with minimal modification.

**Evidence:** Code audit of 6 packages: `pkg/verify/accretion*.go` (298+225 lines), `pkg/control/` (245 lines), `cmd/orch/harness_init.go` standalone functions, `cmd/orch/hotspot.go` bloat analysis, `pkg/events/logger.go`, `pkg/orch/governance.go`. First 3 have zero internal imports.

**Knowledge:** The standalone mode already built into `harness_init.go` proves the extraction boundary — orch detected "no orch infrastructure" and generated self-contained artifacts. The harness CLI is extracting the standalone path into its own binary.

**Next:** Create `github.com/dylan-conlin/harness` repo with 3 commands (init/check/report), extracting from the identified sources. Implementation issues below.

**Authority:** strategic — New repo, new binary, public-facing artifact tied to blog post. Dylan decides.

---

# Investigation: Design Standalone Harness CLI Extracted from orch-go

**Question:** What should a standalone `harness` CLI look like that gives blog post readers a working tool, with zero dependency on orch-go, beads, kb, or opencode?

**Started:** 2026-03-11
**Updated:** 2026-03-11
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None — recommendations ready for implementation
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| Blog post (harness-engineering-draft.md) | extends — CLI operationalizes the Day 1 checklist | Yes — lines 223-250 define scope | None |
| .kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md | extends — hotspot gates are Layer 1 | Yes — extraction candidates confirmed | None |

---

## Findings

### Finding 1: Three packages have zero orch-go internal dependencies

**Evidence:**
- `pkg/verify/accretion.go` imports only `fmt`, `os/exec`, `path/filepath`, `strconv`, `strings`
- `pkg/verify/accretion_precommit.go` imports only `fmt`, `os/exec`, `strings`
- `pkg/control/control.go` imports only stdlib (`os`, `os/exec`, `encoding/json`, `path/filepath`, `strings`)
- These three packages contain the core logic for all three `harness` commands

**Source:** `pkg/verify/accretion.go:1-10`, `pkg/verify/accretion_precommit.go:1-7`, `pkg/control/control.go` imports

**Significance:** Clean extraction boundary. These can be copied with only import path changes, no logic modifications. This is the strongest signal that the standalone CLI is a copy+rename, not a rewrite.

---

### Finding 2: harness_init.go already implements standalone mode

**Evidence:** `cmd/orch/harness_init.go:58-69` has `detectStandaloneMode()` which checks for absence of `~/.orch/hooks/`. When standalone:
- Deny rules use `standaloneDenyRules()` (lines 693-700) — 4 rules protecting settings.json only
- Hook scripts are generated inline as Python (lines 808-901, `standaloneGitAddAllHook`)
- Pre-commit gate is a self-contained bash script (lines 1029-1112, `standalonePreCommitScript`)
- Control plane lock works unchanged (uses `pkg/control/`)

The standalone mode proves the extraction is viable. The blog post's Day 1 checklist maps 1:1 to the standalone init steps.

**Source:** `cmd/orch/harness_init.go:58-69` (detection), `92-96` (mode display), `117-157` (standalone deny rules), `160-182` (standalone hooks), `255-297` (standalone pre-commit)

**Significance:** The standalone path is already tested and working inside orch-go. Extraction means lifting this path into its own binary, not reimplementing it.

---

### Finding 3: Bloat analysis is decoupled from hotspot spawn gates

**Evidence:** `cmd/orch/hotspot.go` has `analyzeBloatFiles()` (lines 195-260) which:
- Walks project directory
- Filters via `skipBloatDirs` (line 37-53) and `isSourceFile()`
- Counts lines via `countLines()` (lines 263-287) — pure Go, no exec
- Returns `[]Hotspot` with path, score (line count), and recommendation

This is completely independent of the spawn gate logic in `hotspot_spawn.go` (which needs orch-specific types like `SpawnHotspotResult`, `HotspotChecker`).

**Source:** `cmd/orch/hotspot.go:195-260` (analyzeBloatFiles), `263-287` (countLines), `37-53` (skipBloatDirs)

**Significance:** The `harness check` command can extract `analyzeBloatFiles` + `countLines` + `isSourceFile` + `skipBloatDirs` without touching spawn infrastructure. This is the bloat scanner.

---

### Finding 4: Events logger is over-scoped for harness but pattern is right

**Evidence:** `pkg/events/logger.go` has 40+ event types and 30+ logging methods, almost all orch-specific (daemon decisions, exploration phases, agent rework cycles). But the core JSONL appender pattern is ~30 lines:
```go
type Logger struct { Path string }
func (l *Logger) Log(event Event) error { ... append JSONL ... }
```

**Source:** `pkg/events/logger.go` full file

**Significance:** Write fresh for harness. Need only 4 event types: `gate.fired` (pre-commit gate triggered), `gate.bypassed`, `init.completed`, `check.completed`. The JSONL pattern is trivial to reimplement; extracting the full logger would bring 36 unused event types.

---

### Finding 5: Blog post scope (lines 223-250) maps cleanly to 3 commands

**Evidence:** The blog post's Getting Started defines:
- **Day 0** (lines 225-227): Directory structure → `harness init` creates `.harness/` dir
- **Day 1** (lines 229-245): 7 items, of which 5 map to `harness init`:
  1. Deny rules → `init` step 1
  2. Agent can't self-close → out of scope (requires issue tracker)
  3. No git add -A → `init` hook generation
  4. Event emission → out of scope (requires issue tracker)
  5. Pre-commit growth gate → `init` step 4
  6. Control plane lock → `init` step 5
  7. Governance CLAUDE.md → `init` step (new, generate template)
- **Week 1** (lines 247-249): Verification → `harness check` + `harness report`

Items 2 and 4 are explicitly out of scope for standalone (they require beads/orch). The standalone CLI covers items 1, 3, 5, 6, 7 plus manual check/report.

**Source:** `.kb/publications/harness-engineering-draft.md:223-250`

**Significance:** 5 of 7 Day 1 items are implementable standalone. The 2 that aren't (self-close gate, event emission) require issue tracking infrastructure. This is a clean scope boundary.

---

## Synthesis

**Key Insights:**

1. **The standalone path is already proven** — harness_init.go's standalone mode IS the extraction. The CLI wraps this tested logic in its own binary with zero orch imports.

2. **Three clean extraction packages** — `pkg/verify/accretion*.go` (pre-commit gate), `pkg/control/` (deny rules + locks), and `analyzeBloatFiles` from `hotspot.go` (bloat scanner) form the complete core. All stdlib-only.

3. **Write fresh where orch-specific** — Events logger and governance checks are too orch-coupled. Simpler to write ~100 lines of fresh code for gate event logging than to extract and prune 800+ lines.

**Answer to Investigation Question:**

The standalone `harness` CLI should be a new repo (`github.com/dylan-conlin/harness`) with 3 commands (`init`, `check`, `report`). Extract `pkg/verify/accretion*.go`, `pkg/control/control.go`, standalone functions from `harness_init.go`, and `analyzeBloatFiles`/`countLines`/`isSourceFile` from `hotspot.go`. Write fresh: CLI entry point, simplified event logger (4 event types), CLAUDE.md governance template, and the `report` command. Total estimated new code: ~800 lines. Total extracted code: ~600 lines.

---

## Structured Uncertainty

**What's tested:**

- ✅ pkg/verify/accretion*.go has zero orch-go imports (verified: read imports in both files)
- ✅ pkg/control/ has zero orch-go imports (verified: explored package)
- ✅ harness_init.go standalone mode generates self-contained artifacts (verified: read standalone functions)
- ✅ analyzeBloatFiles uses only os.Walk + countLines, no orch dependencies (verified: read hotspot.go:195-260)
- ✅ Blog post Day 1 items 1,3,5,6,7 require no orchestration infrastructure (verified: read lines 229-245)

**What's untested:**

- ⚠️ Whether `go install github.com/dylan-conlin/harness@latest` works cross-platform (needs build + test)
- ⚠️ Whether the control plane lock works on Linux (chflags is macOS; chattr +i is Linux equivalent, not yet implemented)
- ⚠️ Whether blog post readers find 3 commands sufficient or need more (user testing)
- ⚠️ Whether the 800-line threshold is right for codebases that aren't Go (may need per-language defaults)

**What would change this:**

- If Claude Code changes settings.json schema, deny rule format would break
- If blog post readers primarily use Cursor (not Claude Code), deny rules/hooks format differs
- If accretion thresholds need to be language-aware, `isSourceFile` needs config rather than hardcoding

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Create new `harness` repo | strategic | New public artifact, tied to blog publication, irreversible naming decision |
| Extract identified packages | implementation | Copy + rename, no architectural decisions |
| Write fresh event logger | implementation | Tactical choice within clear scope |
| Add CLAUDE.md template generator | implementation | Content creation within scope |
| Support Cursor/non-Claude-Code tools | architectural | Cross-tool compatibility changes scope |

### Recommended Approach ⭐

**Extract + Minimal Fresh Code** — Copy the 3 zero-dependency packages, wrap in a new CLI with 3 commands.

**Why this approach:**
- Proven code: the accretion gate and control plane lock already work in production (12 weeks, 50+ agents/day)
- Minimum invention: ~800 lines of new code (CLI scaffolding, template, report) vs ~600 lines of extracted code
- Blog post alignment: `harness init` maps directly to the Day 1 checklist in the Getting Started section

**Trade-offs accepted:**
- No Cursor/Windsurf support in MVP (their settings format differs from Claude Code's)
- No per-language thresholds (800 lines hardcoded, override via flag)
- No config file (flags only, simpler for blog post readers)

**Implementation sequence:**
1. Create repo with `cmd/harness/main.go` (Cobra, 3 commands) — foundational
2. Copy `pkg/accretion/` from verify, `pkg/control/` as-is — core logic
3. Implement `init` command (extracted from standalone mode in harness_init.go) — primary value
4. Implement `check` command (extracted from analyzeBloatFiles) — manual gate
5. Implement `report` command (git-log based velocity + JSONL gate history) — observability
6. Add CLAUDE.md governance template — soft harness orientation

### Alternative Approaches Considered

**Option B: Shell script distribution (no binary)**
- **Pros:** Simpler distribution (curl | bash), works everywhere
- **Cons:** Can't do pre-commit gate portably (bash vs zsh vs fish), no structured output, poor UX for `harness check`, harder to test
- **When to use instead:** If target audience is non-Go developers who won't have `go install`

**Option C: npm package**
- **Pros:** Larger audience (JS/TS developers), easier `npx harness init`
- **Cons:** Requires Node runtime, adding a language dependency to a Go tool's blog post, reimplementation cost
- **When to use instead:** If blog post audience is primarily TypeScript developers

**Option D: Plugin within existing tools (e.g., GitHub Action only)**
- **Pros:** CI/CD integration, no local install
- **Cons:** Doesn't help with Day 1 governance (pre-commit is local), doesn't address the immediate "nothing to try" gap
- **When to use instead:** Future addition after CLI proves value

**Rationale for recommendation:** The blog post is about Go codebase governance, the audience uses Go, and the extracted code is Go. `go install` is the natural distribution path.

---

### Repo Structure

```
github.com/dylan-conlin/harness/
├── cmd/harness/
│   └── main.go                # Cobra CLI: init, check, report
├── pkg/
│   ├── accretion/
│   │   ├── check.go           # From pkg/verify/accretion.go (completion gate)
│   │   ├── precommit.go       # From pkg/verify/accretion_precommit.go (pre-commit gate)
│   │   ├── source.go          # isSourceFile, nonSourceDirs, countLines
│   │   ├── check_test.go      # From pkg/verify/accretion_test.go
│   │   └── precommit_test.go  # From pkg/verify/accretion_precommit_test.go
│   ├── control/
│   │   ├── control.go         # From pkg/control/ (lock/unlock/deny rules)
│   │   └── control_test.go    # From pkg/control/control_test.go
│   ├── scaffold/
│   │   ├── init.go            # From harness_init.go standalone functions
│   │   ├── denylist.go        # Deny rule management
│   │   ├── hooks.go           # Hook generation + registration
│   │   ├── precommit.go       # Pre-commit gate installation
│   │   └── template.go        # CLAUDE.md governance template
│   ├── report/
│   │   ├── velocity.go        # Git-log based accretion velocity
│   │   └── history.go         # Gate firing history from JSONL
│   └── events/
│       └── events.go          # Simplified JSONL logger (4 event types)
├── templates/
│   └── governance.md          # CLAUDE.md governance template content
├── go.mod                     # module github.com/dylan-conlin/harness
├── go.sum
├── Makefile
└── README.md
```

### Extraction Map (What Comes From Where)

| Harness Package | Source in orch-go | Modification |
|----------------|-------------------|-------------|
| `pkg/accretion/check.go` | `pkg/verify/accretion.go` | Rename package, remove `VerifyAccretionForCompletion` (orch-specific), keep types + `getGitDiffWithLineCounts` + `isSourceFile` |
| `pkg/accretion/precommit.go` | `pkg/verify/accretion_precommit.go` | Rename package, keep as-is |
| `pkg/accretion/source.go` | `pkg/verify/accretion.go` + `cmd/orch/hotspot.go` | Extract `isSourceFile`, `nonSourceDirs`, `countLines`, `skipBloatDirs` into shared file |
| `pkg/control/control.go` | `pkg/control/control.go` | Copy as-is (already zero deps) |
| `pkg/scaffold/init.go` | `cmd/orch/harness_init.go` standalone functions | Extract standalone mode only, remove full mode, remove orch imports |
| `pkg/scaffold/hooks.go` | `cmd/orch/harness_init.go:808-1024` | Extract `standaloneGitAddAllHook`, `ensureStandaloneHookScripts`, `ensureStandaloneHookRegistration` |
| `pkg/scaffold/precommit.go` | `cmd/orch/harness_init.go:1029-1157` | Extract `standalonePreCommitScript`, `ensureStandalonePreCommitGate` |
| `pkg/report/velocity.go` | **Fresh** | Parse `git log --numstat` for file growth over time |
| `pkg/events/events.go` | **Fresh** (pattern from `pkg/events/logger.go`) | 4 event types: `gate.fired`, `gate.bypassed`, `init.completed`, `check.completed` |
| `templates/governance.md` | **Fresh** | CLAUDE.md template with accretion boundaries, authority delegation |

### Command Specifications

#### `harness init`

```
harness init [--dry-run] [--threshold N]

Steps:
1. Deny rules → add to ~/.claude/settings.json (prevents agents editing control plane)
2. Hook script → generate .claude/hooks/gate-git-add-all.py (blocks git add -A)
3. Hook registration → register in settings.json PreToolUse
4. Pre-commit gate → install .git/hooks/pre-commit (accretion warnings + blocks)
5. Control plane lock → chflags uchg on settings.json + hooks (macOS)
6. Governance template → generate CLAUDE.md with accretion boundaries (if not exists)

Output: Step-by-step progress (same format as current harness init)
Exit 0 on success, 1 on errors
Idempotent: safe to run multiple times
```

#### `harness check`

```
harness check [--threshold N] [--json] [path...]

Scans project for bloated files:
- Default threshold: 800 lines
- Reports all source files above threshold
- Checks staged files for accretion violations
- Optionally restrict to specific paths

Output modes:
  Text (default): Table with file path, lines, threshold, severity
  JSON (--json):  Machine-readable for CI integration

Exit codes:
  0: No files above threshold
  1: Files above threshold found
```

#### `harness report`

```
harness report [--days N] [--json]

Shows accretion health:
- File growth velocity (lines/week over last N days, from git log)
- Current hotspot files (above threshold)
- Gate firing history (from .harness/events.jsonl)
- Trend: accelerating, stable, or decelerating

Output: Summary report with key metrics
```

### CLAUDE.md Governance Template

The `harness init` command generates a governance section for CLAUDE.md:

```markdown
## Accretion Boundaries

Files above 800 lines require extraction before additions.
Run `harness check` to see current file sizes.

### Authority Boundaries

**Agents can decide:** Implementation details, test strategy, refactoring within scope.
**Agents must escalate:** Architectural changes, new patterns, unclear requirements.

### Conventions

- Stage files explicitly: `git add path/to/file.go` (never `git add -A` or `git add .`)
- Keep files under 800 lines; extract into packages when approaching the limit
- The pre-commit hook will warn on file growth; extract before committing
```

### Things to Watch Out For

- ⚠️ **Claude Code settings.json location** differs by platform: `~/.claude/settings.json` on macOS, unknown on Linux. The `DefaultSettingsPath()` function in `pkg/control/` handles this.
- ⚠️ **Control plane lock is macOS-only** (`chflags uchg`). Linux needs `chattr +i` (requires root). MVP should warn on non-macOS but not fail.
- ⚠️ **Pre-commit hook append** vs create: If `.git/hooks/pre-commit` already exists, must append, not overwrite. This is already handled in the extracted code.
- ⚠️ **Cursor compatibility**: Cursor uses `~/.cursor/settings.json` with different deny rule format. Out of MVP scope but should be the first post-MVP addition.

### Success Criteria

- ✅ `go install github.com/dylan-conlin/harness@latest` works
- ✅ `harness init` in a fresh git repo with Claude Code creates all 5 governance artifacts
- ✅ `harness init --dry-run` previews without modifying anything
- ✅ `harness check` reports files above threshold with correct line counts
- ✅ `harness report` shows accretion velocity from git history
- ✅ All tests pass: `go test ./...`
- ✅ Zero imports from `github.com/dylan-conlin/orch-go`

---

## Implementation Issues (for decomposition)

**Issue 1: Create harness repo scaffolding**
- Type: task
- Scope: go.mod, cmd/harness/main.go (Cobra), Makefile, README.md stub
- Skill: feature-impl
- Dependencies: none

**Issue 2: Extract accretion package**
- Type: task
- Scope: Copy + adapt pkg/verify/accretion*.go → pkg/accretion/
- Skill: feature-impl
- Dependencies: Issue 1

**Issue 3: Extract control package**
- Type: task
- Scope: Copy pkg/control/ → pkg/control/
- Skill: feature-impl
- Dependencies: Issue 1

**Issue 4: Implement `harness init` command**
- Type: feature
- Scope: Extract standalone mode from harness_init.go, add CLAUDE.md template generation
- Skill: feature-impl
- Dependencies: Issues 2, 3

**Issue 5: Implement `harness check` command**
- Type: feature
- Scope: Bloat scanner from hotspot.go, staged accretion check, text + JSON output
- Skill: feature-impl
- Dependencies: Issue 2

**Issue 6: Implement `harness report` command**
- Type: feature
- Scope: Git-log velocity analysis, JSONL gate history reader
- Skill: feature-impl
- Dependencies: Issues 4, 5 (needs gate event data)

**Issue 7: Integration — end-to-end test of `harness init` → `harness check` → `harness report` workflow**
- Type: task
- Scope: Integration test in temp git repo, verify all artifacts created, gates fire correctly
- Skill: feature-impl
- Dependencies: Issues 4, 5, 6

---

## References

**Files Examined:**
- `pkg/verify/accretion.go` — Core accretion types + completion gate (298 lines)
- `pkg/verify/accretion_precommit.go` — Pre-commit staged accretion check (225 lines)
- `pkg/verify/accretion_test.go` — Accretion test suite (592 lines)
- `pkg/verify/accretion_precommit_test.go` — Pre-commit test suite (547 lines)
- `pkg/control/control.go` — Control plane lock/unlock/deny rules (~245 lines)
- `cmd/orch/harness_init.go` — Harness init with standalone mode (1204 lines)
- `cmd/orch/hotspot.go` — Bloat analysis + hotspot reporting (387 lines)
- `cmd/orch/hotspot_spawn.go` — Spawn gate hotspot matching (340 lines)
- `cmd/orch/precommit_cmd.go` — Pre-commit CLI commands (190 lines)
- `pkg/spawn/gates/hotspot.go` — Spawn hotspot gate (157 lines)
- `pkg/events/logger.go` — Event logging infrastructure
- `pkg/orch/governance.go` — Governance path protection
- `.kb/publications/harness-engineering-draft.md` — Blog post (311 lines)

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md` — Three-layer gate design
- **Publication:** `.kb/publications/harness-engineering-draft.md` — Blog post defining scope

---

## Investigation History

**2026-03-11 17:30:** Investigation started
- Initial question: What should a standalone harness CLI look like?
- Context: Blog post ready to publish, readers need a tool to try

**2026-03-11 17:45:** Found zero-dependency extraction boundary
- pkg/verify/accretion*.go, pkg/control/, and standalone mode in harness_init.go are all stdlib-only

**2026-03-11 18:00:** Mapped blog post Day 1 checklist to commands
- 5 of 7 items implementable standalone, 2 require issue tracker (out of scope)

**2026-03-11 18:15:** Investigation completed
- Status: Complete
- Key outcome: Extract 3 packages + standalone mode, write ~800 lines fresh, 3 commands (init/check/report)
