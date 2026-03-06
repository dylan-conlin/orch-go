# Model: Workspace Lifecycle & Hierarchy

**Domain:** Workspaces / Persistence / Cleanup
**Last Updated:** 2026-03-06
**Synthesized From:** 13 investigations (Dec 2025 - Jan 2026) on orphaned workspaces, cleanup strategies, name collisions, and interactive session workspaces.

---

## Summary (30 seconds)

Workspaces are the **filesystem-level record** of an agent's execution. They exist in three tiers (**light**, **full**, **orchestrator**) across three primary locations based on the session type. The lifecycle follows a **Spawn → Execute → Complete → Archive** flow. Archival is currently manual via `orch clean --stale` (archiving after 7 days). Uniqueness is guaranteed for spawned workspaces by a **4-character hex suffix**, preventing the silent overwriting of session artifacts.

---

## Workspace Hierarchy

Workspaces are categorized by their role and location:

| Workspace Type | Location | Suffix | Completion Artifact |
|----------------|----------|--------|---------------------|
| **Worker** | `{project}/.orch/workspace/og-{skill}-{slug}-{date}-{hex}/` | Yes | `SYNTHESIS.md` (full only) |
| **Spawned Orchestrator** | `{project}/.orch/workspace/og-orch-{slug}-{date}-{hex}/` | Yes | `SESSION_HANDOFF.md` |
| **Interactive Session** | `~/.orch/session/{date}/` | No | `SESSION_HANDOFF.md` |

**Key Insight:** Spawned orchestrators get full, unique workspaces per goal, while interactive human-driven sessions use daily directories to maintain continuity across multiple breaks.

---

## State Indicators

The state of a workspace is determined by the presence of specific metadata files:

| File | Purpose | Authority |
|------|---------|-----------|
| `.tier` | Defines verification rules (light, full, orchestrator) | Spawn config |
| `.session_id` | Link to OpenCode conversation history | OpenCode API |
| `.beads_id` | Link to authoritative work tracking | Beads API |
| `.spawn_time` | Nanosecond timestamp for age calculation | System clock |
| `.review-state.json` | Synthesis recommendation review state | `orch review done` |
| `.spawn_mode` | How session was spawned (opencode, claude, inline) | Spawn command |

### Completion Status (File-Based)
For high-performance bulk operations (like `orch clean`), completion is inferred from the filesystem:
*   **Full Tier:** `SYNTHESIS.md` exists.
*   **Orchestrator Tier:** `SESSION_HANDOFF.md` exists.
*   **Light Tier:** `.beads_id` exists (assumed complete if no active session found).

---

## Lifecycle Flow

1.  **Spawn (Creation):** 
    *   Directory created with a 4-char random hex suffix (`crypto/rand`).
    *   Initial context (`SPAWN_CONTEXT.md` or `ORCHESTRATOR_CONTEXT.md`) is written.
    *   Metadata files (`.tier`, `.beads_id`) are initialized.
2.  **Execute (Active):**
    *   Agent performs work, appending evidence to beads comments.
    *   Full/Orchestrator agents progressively fill their synthesis/handoff artifacts.
3.  **Complete (Resolution):**
    *   `orch complete` verifies artifacts and closes the beads issue.
    *   Workspace remains in situ for immediate human review.
4.  **Archive (Cleanup):**
    *   `orch clean --stale` moves completed workspaces older than 7 days to `{project}/.orch/workspace/archived/`.
    *   Zombie sessions are detected via `orch doctor --sessions`.

5.  **Expire (TTL-based deletion of archived workspaces):**
    *   No mechanism currently exists. Archived workspaces accumulate unboundedly.
    *   As of 2026-02-28: 1,708 archived dirs, 149MB. All appear as untracked entries in `git status`.
    *   The binding constraint is `orch rework` — `FindArchivedWorkspaceByBeadsID()` in `pkg/spawn/rework.go` needs archived workspaces. Rework typically happens within days, not months, so a TTL of 30-90 days preserves the use case.
    *   The "two-tier cleanup" pattern (event-based archival on complete + periodic TTL-based expiry) should be extended here.

---

## Why This Fails

### 1. The Archival Gap (Resolved)
**Symptom:** Hundreds of active workspaces accumulate (340+ observed).
**Root Cause:** `orch complete` closes the issue but leaves the directory. Archival was an opt-in manual step (`orch clean --stale`).
**Status:** Fixed — automated archival now runs as part of `orch complete`.

### 1b. The Archived Workspace Accumulation Gap (New — Feb 2026)
**Symptom:** 1,708 archived dirs, 149MB; every `git status` shows hundreds of untracked entries obscuring real code changes. 10,448 workspace files already historically committed to git.
**Root Cause:** Model treated Archive as the terminal state. No TTL or cleanup mechanism for archived workspaces. `.gitignore` had no `.orch` entries.
**Solution:** Two steps — (1) Add `.orch/workspace/` to `.gitignore` immediately. (2) Add TTL-based expiry for archived workspaces (30-90 day window preserves `orch rework` use case). The git tracking of 10,448 historical files requires a separate `git rm --cached` cleanup pass.

### 2. Name Collisions (Fixed)
**Symptom:** New sessions overwriting previous handoffs on the same day.
**Root Cause:** Purely deterministic naming `{proj}-{skill}-{slug}-{date}`.
**Solution:** Mandatory 4-char hex suffix for all spawned workspaces.

### 3. Context Mismatch
**Symptom:** Agent thinks it's a worker but is editing orchestrator artifacts.
**Root Cause:** Tier mismatch or manual directory creation.
**Solution:** `VerifyCompletionWithTier()` enforces artifact requirements based on the `.tier` file.

---

## Constraints

### Why daily directories for interactive sessions?
**Constraint:** `orch session start` uses `~/.orch/session/{date}/`.
**Implication:** Multiple goals in one day share one `SESSION_HANDOFF.md`.
**Rationale:** Simplifies "Landing the Plane" for the human, who usually thinks in "days of work" rather than "goal-atomic sessions."

### Why local `.orch/` for workers?
**Constraint:** Worker workspaces are inside the project they are modifying.
**Implication:** Workspaces are visible to `git status` (unless ignored). **As of 2026-02-28:** they were NOT ignored — 10,448 workspace files had been committed historically, and 1,708 archived dirs were untracked noise. Mitigation: add `.orch/workspace/` to `.gitignore`.
**Rationale:** Keeps evidence close to the code; allows `orch complete` to run project-local tests easily.

---

## References

**Synthesis Investigation:**
* `.kb/investigations/2026-01-17-inv-synthesize-12-investigations-related-workspace.md` - Comprehensive synthesis of all workspace investigations

**Source Investigations (13 total):**
* `.kb/investigations/2025-12-21-inv-workspace-lifecycle-when-workspaces-created.md`
* `.kb/investigations/2025-12-21-inv-beads-kb-workspace-relationships-how.md`
* `.kb/investigations/2025-12-26-inv-add-review-state-tracking-workspace.md`
* `.kb/investigations/2026-01-05-inv-orchestrator-workspaces-clear-visual-distinction.md`
* `.kb/investigations/2026-01-05-debug-orchestrator-workspace-name-collision-bug.md`
* `.kb/investigations/2026-01-06-inv-workspace-session-architecture.md`
* `.kb/investigations/2026-01-06-inv-define-workspace-cleanup-strategy-context.md`
* `.kb/investigations/2026-01-06-inv-add-orch-attach-workspace-command.md`
* `.kb/investigations/2026-01-06-inv-extend-orch-resume-work-workspace.md`
* `.kb/investigations/2026-01-06-inv-add-orch-doctor-sessions-workspace.md`
* `.kb/investigations/2026-01-07-inv-address-340-active-workspaces-completion.md`
* `.kb/investigations/2026-01-09-inv-create-orchestrator-workspace-session-start.md`

**Source Code:**
* `pkg/spawn/config.go` - Name generation and uniqueness
* `cmd/orch/clean_cmd.go` - Archival logic
* `cmd/orch/session.go` - Interactive workspace creation
* `pkg/spawn/rework.go:19-66` - `FindArchivedWorkspaceByBeadsID()` — binding constraint on archived workspace TTL

**Primary Evidence (Verify These):**
* `pkg/spawn/config.go` - Workspace naming with 4-char hex suffix generation
* `cmd/orch/clean_cmd.go` - Stale workspace archival logic
* `cmd/orch/session.go` - Interactive session workspace creation in ~/.orch/session/
* `pkg/verify/check.go` - Tier-aware verification using .tier file
* `.orch/workspace/` - Worker workspace directory structure
* `~/.orch/session/` - Interactive orchestrator session directory structure

### Merged Probes

| Probe | Date | Verdict | Key Finding |
|-------|------|---------|-------------|
| `probes/2026-02-28-probe-archived-workspace-accumulation-git-clutter.md` | 2026-02-28 | Extends | Archival was fixed but created a new gap — 1,708 archived dirs (149MB) accumulate without TTL. 10,448 workspace files historically committed to git; `.gitignore` had no `.orch` entries. `orch rework` is the binding constraint (needs archived workspaces within days of completion). Lifecycle needs a 5th stage: Expire. |
