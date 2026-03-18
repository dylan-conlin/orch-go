# Probe: Knowledge Decay Verification — Escape Hatch Visibility Architecture

**Date:** 2026-03-18
**Model:** escape-hatch-visibility-architecture
**Purpose:** Verify model claims after 999-day decay flag. Check if architectural evolution has obsoleted the model's framing.

---

## Claims Verified

### Claim 1: `--backend claude` provides independence from OpenCode server
**Verdict: CONFIRMED but framing is stale.**

The `--backend claude` flag still exists and routes to `runSpawnClaude()` in `pkg/orch/spawn_modes.go:463`. The independence mechanism is intact: Claude CLI spawns don't depend on OpenCode server for execution.

However, the model frames `--backend claude` as an *opt-in escape hatch* for critical infrastructure work. Since Feb 19, 2026 (Anthropic OAuth ban), Claude CLI is the **default backend** for all Anthropic model work. The project config confirms: `.orch/config.yaml` has `spawn_mode: claude`.

**Impact:** The "escape hatch" is now the primary path. The model's core premise — that dual-window setup is needed *specifically* for escape-hatch spawning — is weakened because tmux-based Claude spawns are now the default for all work, not just critical infrastructure.

### Claim 2: `--tmux` flag creates visible tmux windows
**Verdict: PARTIALLY STALE.**

The `--tmux` flag still exists (`spawn_cmd.go:168`), but when `spawn_mode: claude` is the default, ALL Claude backend spawns create tmux windows automatically — no `--tmux` flag needed. The `runSpawnClaude()` function always uses tmux (`spawn.SpawnClaude()` creates a tmux window). The `--tmux` flag is relevant only for the `opencode` backend path.

### Claim 3: Dual-window Ghostty setup required for visibility
**Verdict: CONFIRMED — infrastructure still exists.**

- `~/.tmux.conf.local:62` still has the `after-select-window` hook enabled
- `~/.local/bin/sync-workers-session.sh` still exists
- The auto-switch mechanism is intact

However, the model claims dual-window is "REQUIRED" for escape-hatch spawning. Since Claude-backend tmux spawns are now the default, the question becomes: is dual-window required for *all* work or just monitoring? The core visibility benefit (zero-step observation) remains valid regardless.

### Claim 4: Headless spawning is the primary path
**Verdict: STALE.**

Model states: "Primary Path (Daemon) Does NOT Require Dual-Window" implying headless/OpenCode API is the primary path. This was true in Jan 2026. Since Feb 19, 2026, Claude CLI (tmux) IS the primary path for all Anthropic models. Headless/OpenCode is now the secondary path for non-Anthropic models only.

### Claim 5: Referenced files exist at stated paths
**Verdict: PARTIALLY STALE.**

| Reference | Status |
|-----------|--------|
| `pkg/spawn/backend.go` | **MOVED** → `pkg/orch/spawn_backend.go` |
| `pkg/spawn/spawn.go` | **MOVED** → `pkg/orch/spawn_modes.go` |
| `cmd/orch/spawn.go` | **RENAMED** → `cmd/orch/spawn_cmd.go` |
| `~/.tmux.conf.local:58-61` | **Line shifted** → now line 62 |
| `~/.local/bin/sync-workers-session.sh` | **EXISTS** |
| `~/orch-knowledge/.orch/docs/orchestration-window-setup.md` | **MISSING** (orch-knowledge merged into orch-go) |

### Claim 6: Decision tree for when to use escape hatch
**Verdict: STALE.**

The decision tree says "Critical Infrastructure Work → Requires Escape Hatch" and "Feature/bug → Normal workflow (daemon + headless)." This is inverted now. Claude CLI (tmux) IS the normal workflow for Anthropic models. The escape-hatch concept is vestigial for the default path.

The `DetermineSpawnBackend()` function in `pkg/orch/spawn_backend.go` still auto-detects infrastructure work and forces claude backend, but since claude is already the configured default (`spawn_mode: claude` in project config), this auto-detection is largely redundant for orch-go itself.

---

## Overall Verdict

**Model is significantly stale.** The core architectural insight (independence + visibility + capability) remains valid, but the framing is inverted:

1. **Jan 2026 (model creation):** Claude CLI was the escape hatch; OpenCode/headless was primary
2. **Feb 2026 (OAuth ban):** Claude CLI became the default; OpenCode/headless became secondary
3. **Mar 2026 (now):** The "escape hatch" terminology is misleading — it describes the normal operating mode

**Recommended model updates:**
1. Reframe: "escape hatch" → "backend independence architecture" (the principle survives but the terminology is wrong)
2. Update primary/secondary path descriptions to match current reality
3. Fix all stale file references (pkg/spawn → pkg/orch, cmd/orch/spawn.go → spawn_cmd.go)
4. Remove reference to `~/orch-knowledge/.orch/docs/orchestration-window-setup.md` (repo merged)
5. Note that dual-window is now relevant for ALL tmux-based work, not just escape-hatch scenarios
6. Update the decision tree: the real decision is now "dual-window vs dashboard" for monitoring, not "escape hatch vs normal"
7. Consider archiving or renaming the model since its core premise (escape hatch as opt-in) no longer applies

---

## Evidence Quality

- **[direct-observation]** Code reads of `pkg/orch/spawn_backend.go`, `pkg/orch/spawn_modes.go`, `cmd/orch/spawn_cmd.go`
- **[direct-observation]** `.orch/config.yaml` showing `spawn_mode: claude`
- **[direct-observation]** `~/.tmux.conf.local` hook still present at line 62
- **[direct-observation]** File existence checks for referenced paths
- **[corroborated]** CLAUDE.md documents the Feb 19 backend shift and its history
- **[corroborated]** `.kb/guides/resilient-infrastructure-patterns.md` line 91 confirms the chronology
