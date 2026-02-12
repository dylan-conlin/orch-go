## Summary (D.E.K.N.)

**Delta:** The compound flags `--opus` and `--infra` conflate intent (model, resilience) with mechanism (backend, tmux), creating semantic confusion and broken combinations (`--opus --headless` is silently ignored).

**Evidence:** Code analysis of `resolveBackend()` (backend.go:28-151) shows `--opus` forces claude backend at priority 2; `dispatchSpawn()` (spawn_pipeline.go:727-758) then routes claude backend exclusively to `runSpawnClaude` (tmux) regardless of `--headless` flag. Tests confirm: `--opus` always resolves to claude, `--headless` is never consulted for claude-backend spawns.

**Knowledge:** The compound flags were correct when created (Jan 2026: Opus only via Claude CLI Max). With OpenCode OAuth stealth mode (Jan 26) and multi-account (Feb 12), Opus is available on all backends. The flags now encode stale assumptions.

**Next:** Decompose compound flags into atomic intent flags. Implementation requires architectural decision on deprecation strategy and timeline.

**Authority:** architectural - Cross-component change affecting spawn pipeline, daemon, CLAUDE.md docs, and operator muscle memory.

---

# Investigation: Review Orch Spawn Flag Semantics

**Question:** Given multi-account, multi-backend, multi-model reality (Feb 2026), how should orch spawn flags be restructured to separate intent from mechanism?

**Started:** 2026-02-12
**Updated:** 2026-02-12
**Owner:** architect agent (orch-go-49030)
**Phase:** Complete
**Next Step:** None - promote recommendations to decision if accepted
**Status:** Complete

**Patches-Decision:** `.kb/decisions/2026-02-08-multi-model-routing-strategy-spawn-daemon.md`

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `2026-01-20-inv-backend-selection-logic-spawn-cmd.md` (archived) | extends | Yes - backend.go largely unchanged | None |
| `2026-02-12-inv-opencode-per-session-auth-isolation.md` | deepens | Yes - per-session auth constraint confirmed | None |
| `orch-go-tzdeb` (issue) | confirms | Yes - `--opus --headless` silently drops headless | None |

---

## Findings

### Finding 1: `--opus` bundles three independent concerns

**Evidence:**
- Flag definition (spawn_cmd.go:198): `"Use Opus via Claude CLI in tmux (Max subscription, implies claude backend + tmux mode)"`
- `resolveBackend()` (backend.go:65-68): `--opus` forces `backend = "claude"` at priority 2
- `dispatchSpawn()` (spawn_pipeline.go:729-733): claude backend always routes to `runSpawnClaude()` which creates tmux window
- Line 751: `useTmux := p.tmux || p.attach || p.cfg.IsOrchestrator || spawnInfra` - note `spawnOpus` is NOT checked here because claude backend already bypassed this code path entirely

**Source:** `cmd/orch/spawn_cmd.go:198`, `cmd/orch/backend.go:65-68`, `cmd/orch/spawn_pipeline.go:729-751`

**Significance:** `--opus` conflates: (1) model selection (Opus), (2) backend selection (claude), (3) display mode (tmux). User cannot get "Opus model, opencode backend, headless mode" - which is now valid via OAuth stealth mode. The issue `orch-go-tzdeb` confirms this is actively frustrating.

---

### Finding 2: `--infra` bundles resilience intent with specific mechanism

**Evidence:**
- Flag definition (spawn_cmd.go:199): `"Infrastructure work: use claude+tmux backend (survives service crashes)"`
- `resolveBackend()` (backend.go:77-79): `--infra` forces `backend = "claude"`
- `dispatchSpawn()` (spawn_pipeline.go:751): `spawnInfra` forces tmux mode
- However, docker backend also survives OpenCode crashes (separate process entirely)
- And Claude CLI inline mode (`--backend claude --inline`) also survives crashes without tmux

**Source:** `cmd/orch/backend.go:77-79`, `cmd/orch/spawn_pipeline.go:751`

**Significance:** `--infra` means "survives server crashes" but hardcodes one solution (claude+tmux). Users wanting crash-resilient headless spawns (e.g., from daemon) or crash-resilient docker spawns have no flag for the intent, only for the mechanism.

---

### Finding 3: Display mode flags have inconsistent precedence with backend

**Evidence:**
The dispatch logic (spawn_pipeline.go:727-758) has this priority:
1. claude backend → always tmux (or inline if `--inline`)
2. docker backend → always tmux-like
3. `--inline` → inline mode
4. `--headless` → headless mode
5. `--tmux` or `--attach` or orchestrator or `--infra` → tmux mode
6. Default → headless

Problem: `--headless` at priority 4 is unreachable when backend is claude (priority 1) or docker (priority 2). So `--opus --headless` silently ignores `--headless`.

**Source:** `cmd/orch/spawn_pipeline.go:727-758`

**Significance:** Silent flag conflicts are worse than errors. Users don't know their intent was dropped. The `--backend opencode --model opus` workaround exists but is unintuitive.

---

### Finding 4: `--account` lacks a config default, creating repetitive flag usage

**Evidence:**
- `maybeSwitchSpawnAccount()` (spawn_account_isolation.go:19-42) only activates when `accountName != ""`
- `resolveSpawnClaudeConfigDir()` (spawn_account_isolation.go:44-67) handles auto-switched accounts from usage checks
- No `default_account` field exists in `userconfig.Config` (userconfig.go:108-134) or project `config.Config`
- The model routing decision (2026-02-08) has explicit `skill_models` for model-per-skill, but no equivalent for accounts
- Current workflow requires `--account work` on every claude backend spawn

**Source:** `pkg/userconfig/userconfig.go:108-134`, `cmd/orch/spawn_account_isolation.go:19-42`

**Significance:** With two Max accounts (personal 5x, work 20x), account selection is a frequent per-spawn decision. A config default (`default_account` or `backend_accounts: {claude: work, opencode: personal}`) would eliminate repetition while preserving `--account` override.

---

### Finding 5: Backend-model compatibility validation is incomplete

**Evidence:**
- `validateBackendModelCompatibility()` (backend.go:172-177) only warns about `opencode + opus`
- `validateModeModelCombo()` (spawn_cmd.go:303-309) also only warns about `opencode + opus`
- But with stealth mode, `opencode + opus` is now VALID (it works via OAuth)
- No validation exists for other invalid combos: e.g., `--backend claude --model gpt-5.3-codex` (Claude CLI can't run GPT models)
- No validation for `--backend claude --model deepseek` (same issue)

**Source:** `cmd/orch/backend.go:172-177`, `cmd/orch/spawn_cmd.go:303-309`

**Significance:** The validation logic encodes the Jan 2026 reality (Opus = Claude CLI only). It's now stale AND missing validations for actually broken combos.

---

### Finding 6: The `--headless` flag is documented as redundant but serves as intent signaling

**Evidence:**
- Flag definition (spawn_cmd.go:202): `"Run headless via HTTP API (default behavior, flag is redundant)"`
- In `dispatchSpawn()` (spawn_pipeline.go:746-748): `if p.headless { return runSpawnHeadless(...) }` — it does route to headless mode
- But it only works when backend is opencode (claude/docker paths short-circuit above)
- It serves as an explicit signal: "I want headless even if other flags might suggest otherwise"

**Source:** `cmd/orch/spawn_cmd.go:202`, `cmd/orch/spawn_pipeline.go:746-748`

**Significance:** `--headless` is not truly redundant — it should be an intent signal that overrides compound flag display defaults. Currently it's silently ignored by claude/docker backends.

---

## Synthesis

**Key Insights:**

1. **Flags should express intent, not mechanism** — `--opus` should mean "I want Opus model" not "I want Claude CLI in tmux". The system should resolve intent to mechanism based on current capabilities. This is the core decomposition principle.

2. **Display mode and backend are orthogonal axes** — Backend (opencode/claude/docker) determines the runtime. Display mode (headless/tmux/inline) determines visibility. Currently they're entangled: claude backend forces tmux. They should be independently selectable with the system warning about genuinely incompatible combinations.

3. **Compound flags served a valid purpose when the system was simpler** — `--opus` was created Jan 2026 when Opus literally required Claude CLI. `--infra` bundled the only crash-resilient combination. With 3 backends, 2 accounts, and OAuth stealth mode, the space grew but the flags didn't.

**Answer to Investigation Question:**

The spawn flag surface needs decomposition along three orthogonal axes: **model** (`--model`), **backend** (`--backend`), and **display** (`--headless`/`--tmux`/`--inline`). Compound flags (`--opus`, `--infra`) should be deprecated in favor of atomic compositions. A new `--resilient` flag should replace `--infra` with backend-agnostic semantics. Account defaults should be configurable per-backend to reduce flag repetition.

---

## Structured Uncertainty

**What's tested:**

- ✅ `--opus` forces claude backend (verified: backend_test.go passes, `resolveBackend` code path traced)
- ✅ `--headless` is unreachable when backend=claude (verified: `dispatchSpawn` code path — claude short-circuits at line 729)
- ✅ Backend resolution tests pass with current priority chain (verified: `go test -run TestResolveBackend ./cmd/orch/` — all pass)
- ✅ OpenCode OAuth stealth mode enables Opus without Claude CLI (verified: model stack doc, commit `77e60ac7e`)

**What's untested:**

- ⚠️ Whether `--backend opencode --model opus` actually works in practice today (stealth mode enabled, but no end-to-end spawn test run)
- ⚠️ Whether daemon-driven spawns would benefit from `--resilient` semantics (hypothesis only)
- ⚠️ Whether removing `--opus` would break orchestrator scripts/muscle memory significantly (no usage frequency data)

**What would change this:**

- If OpenCode OAuth stealth mode is unreliable for Opus (intermittent auth failures), `--opus` forcing claude backend would remain correct
- If Claude CLI adds headless support natively, the backend/display entanglement would resolve differently
- If account isolation becomes possible per-session in OpenCode (not just per-server), `--account` semantics would change

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Deprecate `--opus`, replace with `--model opus` | architectural | Affects spawn UX, docs, daemon, orchestrator scripts |
| Replace `--infra` with `--resilient` | architectural | Changes backend selection semantics across pipeline |
| Add `default_account` to config | implementation | Additive config field, no behavioral change |
| Fix `--headless` to override compound flags | implementation | Bug fix within existing architecture |
| Update backend-model validation | implementation | Bug fix — remove stale warning, add missing ones |

### Recommended Approach ⭐

**Phased Decomposition** — Decompose compound flags over two phases: immediate fixes (Phase 1) then deprecation (Phase 2).

**Why this approach:**
- Phase 1 fixes the active bug (`--opus --headless`) without breaking existing scripts
- Phase 2 cleanly removes legacy semantics with a deprecation period
- Matches "long-term solution" preference while providing immediate relief

**Trade-offs accepted:**
- Two-phase rollout means compound flags live longer (acceptable: only Dylan uses orch)
- `--opus` meaning changes (from backend+model to just model) — acceptable with deprecation warning

**Implementation sequence:**

#### Phase 1: Immediate Fixes (implementation authority)

1. **Fix `--headless` to override display mode for all backends**
   - In `dispatchSpawn()`, check `p.headless` BEFORE backend-specific routing
   - Claude backend + headless = error ("Claude CLI requires tmux or inline mode") OR fall back to opencode backend
   - Best: make `--headless` override backend display default, warn if incompatible

2. **Add `default_account` to config**
   - Add `default_account` field to `userconfig.Config`
   - Add `backend_accounts` map (optional): `{claude: "work", opencode: ""}`
   - `maybeSwitchSpawnAccount()` checks config when `--account` not specified

3. **Update backend-model validation**
   - Remove stale `opencode + opus` warning (now valid via stealth mode)
   - Add `claude + non-anthropic model` error (Claude CLI can only run Claude models)
   - Add `docker + non-anthropic model` error (same constraint)

#### Phase 2: Flag Decomposition (architectural authority)

4. **Deprecate `--opus` with warning**
   - `--opus` emits: `"⚠️ --opus is deprecated. Use --model opus --backend claude --tmux instead. For headless opus, use --model opus."`
   - Behavior unchanged during deprecation period
   - Remove after 30 days

5. **Replace `--infra` with `--resilient`**
   - `--resilient` means: "choose a backend that survives OpenCode server crashes"
   - Resolution: prefer claude > docker > opencode (reverse of normal priority)
   - Display mode remains independently selectable (`--resilient --headless` is valid if backend supports it)
   - `--infra` becomes alias for `--resilient --tmux` during deprecation

6. **Add flag conflict detection**
   - Explicit error when flags produce impossible combinations
   - E.g., `--backend opencode --resilient` → warning: "opencode doesn't survive its own crashes"
   - E.g., `--backend claude --headless` → error: "Claude CLI requires tmux or inline mode"

### Alternative Approaches Considered

**Option B: Keep compound flags, fix edge cases only**
- **Pros:** Zero breaking changes, minimal effort
- **Cons:** Semantic confusion persists, `--opus` remains a lie (it's not just about Opus), new users confused
- **When to use instead:** If breaking orchestrator scripts is unacceptable

**Option C: Remove compound flags immediately (no deprecation)**
- **Pros:** Clean slate, no legacy cruft
- **Cons:** Breaks muscle memory, all CLAUDE.md examples need updating simultaneously
- **When to use instead:** If only Dylan uses the tool and he's willing to update all refs at once

**Rationale for recommendation:** Phase decomposition gives immediate bug fixes (Phase 1) while creating a clean migration path (Phase 2). Since only Dylan uses orch, the deprecation window can be short.

---

### Implementation Details

**What to implement first:**
- Fix `--headless` override (resolves `orch-go-tzdeb` immediately)
- Update stale validation (removes confusing "opus+opencode may fail" warning)
- Add `default_account` to config (eliminates daily friction)

**Things to watch out for:**
- ⚠️ Daemon spawns use `runSpawnWithSkillInternal` — ensure `--resilient` works in daemon context
- ⚠️ Claude CLI truly cannot run headless (no HTTP API, requires tty) — `--backend claude --headless` must error, not silently fall back
- ⚠️ The `opencode + opus` warning removal should be validated against actual OAuth stealth spawn success

**Areas needing further investigation:**
- Whether `--resilient` should auto-escalate display mode (e.g., force tmux for crash visibility) or leave display independent
- Whether daemon should respect `--resilient` for triage:ready issues tagged with `area:infra`
- Whether `default_account` should be per-backend (`backend_accounts`) or global

**Success criteria:**
- ✅ `orch spawn --model opus investigation "task"` spawns headless opus via opencode (no tmux)
- ✅ `orch spawn --opus investigation "task"` emits deprecation warning
- ✅ `orch spawn --resilient investigation "task"` uses crash-resistant backend without forcing tmux
- ✅ `orch spawn --backend claude --model deepseek investigation "task"` errors with clear message
- ✅ Account defaults eliminate need for `--account work` on every claude spawn

---

## Proposed Flag Semantic Model

### Three Orthogonal Axes

```
MODEL:    --model <alias>       What LLM to use
BACKEND:  --backend <name>      Where to run it (opencode/claude/docker)
DISPLAY:  --headless/--tmux/--inline  How to see it
```

### Resolution Chain (per axis)

**Model:** `--model` flag > issue `model:X` label > `skill_models[skill]` > `default_model` > backend default
**Backend:** `--backend` flag > `--resilient` flag > project config > global config > default (opencode)
**Display:** `--inline` > `--headless` > `--tmux`/`--attach` > orchestrator auto-tmux > default (headless)

### Compatibility Matrix

| Backend | headless | tmux | inline | Notes |
|---------|----------|------|--------|-------|
| opencode | ✅ default | ✅ | ✅ | All display modes supported |
| claude | ❌ error | ✅ default | ✅ | No HTTP API, requires tty |
| docker | ❌ error | ✅ default | ❌ | Container needs tmux host window |

### Flag Summary (post-decomposition)

| Flag | Axis | Meaning |
|------|------|---------|
| `--model <alias>` | model | Set model (opus, sonnet, gpt, deepseek, etc.) |
| `--backend <name>` | backend | Set runtime (opencode, claude, docker) |
| `--resilient` | backend | Prefer crash-resistant backend (claude > docker > opencode) |
| `--headless` | display | No UI, automation-friendly |
| `--tmux` | display | Tmux window, visible progress |
| `--inline` | display | Current terminal, blocking TUI |
| `--attach` | display | Tmux + auto-attach (implies --tmux) |
| `--account <name>` | auth | Override account for this spawn |
| `--opus` | **DEPRECATED** | Use `--model opus` instead |
| `--infra` | **DEPRECATED** | Use `--resilient --tmux` instead |

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go` — Flag definitions, runSpawnWithSkill entry point
- `cmd/orch/backend.go` — Backend resolution with priority chain
- `cmd/orch/spawn_pipeline.go` — Pipeline phases, dispatchSpawn routing
- `cmd/orch/spawn_account_isolation.go` — Account switching and config dir resolution
- `cmd/orch/spawn_validation.go` — Infrastructure detection, gap gating
- `cmd/orch/backend_test.go` — Backend resolution tests (all pass)
- `pkg/model/model.go` — Model alias resolution
- `pkg/userconfig/userconfig.go` — Global user config (no default_account field)
- `.kb/models/current-model-stack.md` — Current model/account/backend state
- `.orch/config.yaml` — Project config showing opencode default

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/  # OK

# Backend resolution tests
go test -run TestResolveBackend ./cmd/orch/ -count=1  # All pass

# Issue check
bd show orch-go-tzdeb  # Confirmed: --opus doesn't respect --headless
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-08-multi-model-routing-strategy-spawn-daemon.md` — Model routing strategy this investigation patches
- **Issue:** `orch-go-tzdeb` — Specific bug that triggered this review
- **Model:** `.kb/models/current-model-stack.md` — Authoritative model/backend/account state
