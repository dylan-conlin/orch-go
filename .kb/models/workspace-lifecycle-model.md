# Model: Workspace Lifecycle & Hierarchy

**Domain:** Workspaces / Persistence / Cleanup
**Last Updated:** 2026-01-17
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

---

## Why This Fails

### 1. The Archival Gap
**Symptom:** Hundreds of active workspaces accumulate (340+ observed).
**Root Cause:** `orch complete` closes the issue but leaves the directory. Archival is an opt-in manual step (`orch clean --stale`).
**Solution:** Automated archival in the daemon poll loop or as a post-completion hook.

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
**Implication:** Workspaces are visible to `git status` (unless ignored).
**Rationale:** Keeps evidence close to the code; allows `orch complete` to run project-local tests easily.

---

## References

**Synthesis Investigation:**
* `.kb/investigations/2026-01-17-inv-synthesize-12-investigations-related-workspace.md` - Comprehensive synthesis of all workspace investigations

**Source Investigations (13 total):**
* `.kb/investigations/archived/2025-12-21-inv-workspace-lifecycle-when-workspaces-created.md`
* `.kb/investigations/archived/2025-12-21-inv-beads-kb-workspace-relationships-how.md`
* `.kb/investigations/2025-12-26-inv-add-review-state-tracking-workspace.md`
* `.kb/investigations/archived/2026-01-05-inv-orchestrator-workspaces-clear-visual-distinction.md`
* `.kb/investigations/2026-01-05-debug-orchestrator-workspace-name-collision-bug.md`
* `.kb/investigations/archived/2026-01-06-inv-workspace-session-architecture.md`
* `.kb/investigations/archived/2026-01-06-inv-define-workspace-cleanup-strategy-context.md`
* `.kb/investigations/2026-01-06-inv-add-orch-attach-workspace-command.md`
* `.kb/investigations/archived/2026-01-06-inv-extend-orch-resume-work-workspace.md`
* `.kb/investigations/2026-01-06-inv-add-orch-doctor-sessions-workspace.md`
* `.kb/investigations/archived/2026-01-07-inv-address-340-active-workspaces-completion.md`
* `.kb/investigations/2026-01-09-inv-create-orchestrator-workspace-session-start.md`

**Source Code:**
* `pkg/spawn/config.go` - Name generation and uniqueness
* `cmd/orch/clean_cmd.go` - Archival logic
* `cmd/orch/session.go` - Interactive workspace creation
