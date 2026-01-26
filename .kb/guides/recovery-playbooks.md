# Recovery Playbooks

**Purpose:** Concise recovery steps for known failure modes in orch-go.

**Last updated:** 2026-01-23

---

## Playbook: Beads SQLite Corruption

**Symptoms:** `bd list` errors, missing `beads.db`, 0-byte WAL, repeated daemon restarts.

**Likely cause:** Rapid daemon restart loop during WAL checkpoint.

**Recovery:**
- Stop daemon attempts: `pkill -f "bd daemon"`
- Remove corrupted files: `rm -f .beads/beads.db .beads/beads.db-wal .beads/beads.db-shm .beads/daemon.lock`
- Rebuild from JSONL: `bd init --prefix orch-go`
- Verify: `bd doctor` and `bd list`

**Prevention:**
- Avoid daemon auto-start in sandboxes (use direct mode).
- Investigate and fix restart loops before re-enabling daemon.

**References:** `.kb/models/beads-database-corruption.md`

---

## Playbook: Daemon Not Spawning Issues

**Symptoms:** `triage:ready` issues exist but no agents spawn.

**Likely causes:** Missing label/type, daemon not running, capacity exhausted, blockers.

**Recovery:**
- Check label and type: `bd show <id>`
- Check daemon: `launchctl list | grep orch`
- Check capacity: `orch status`
- Check blockers: `bd show <id> --deps`
- Diagnose with preview: `orch daemon preview`

**Prevention:**
- Ensure all issues have explicit types and `triage:ready` labels.

**References:** `.kb/guides/daemon.md`

---

## Playbook: Daemon Capacity Stuck at Max

**Symptoms:** Daemon reports max capacity despite no active agents.

**Likely cause:** Pool not reconciling with OpenCode sessions.

**Recovery:**
- Restart daemon: `launchctl kickstart -k gui/$(id -u)/com.orch.daemon`
- Check for stale sessions: `orch status`

**Prevention:**
- Keep daemon binary up to date (`make install-restart`).

**References:** `.kb/guides/daemon.md`

---

## Playbook: Spawn Hangs on KB Context

**Symptoms:** `orch spawn` stalls during KB context gathering.

**Likely cause:** Slow/hanging kb query.

**Recovery:**
- Re-run spawn with `--skip-artifact-check`.

**Prevention:**
- Use targeted keywords in `kb context`.

**References:** `.kb/guides/spawn.md`

---

## Playbook: OpenCode API Redirect Loop

**Symptoms:** API calls fail with "redirected too many times".

**Likely cause:** Hitting invalid endpoints (`/sessions`, `/health`).

**Recovery:**
- Use `/session` (singular) and `/event` endpoints.

**Prevention:**
- Stick to documented API endpoints.

**References:** `.kb/guides/opencode.md`

---

## Playbook: OpenCode Plugin SIGTRAP (Exit 133)

**Symptoms:** OpenCode crashes with exit code 133.

**Likely cause:** Missing `@opencode-ai/plugin` dependency in `.opencode/`.

**Recovery:**
- `cd .opencode && bun add @opencode-ai/plugin`

**Prevention:**
- Ensure plugin dependencies are installed for local dev.

**References:** `.kb/guides/opencode.md`

---

## Playbook: Dashboard Slow or Requests Pending

**Symptoms:** `/api/agents` slow (5-7s), network requests stuck pending.

**Likely causes:** Session accumulation, HTTP/1.1 connection exhaustion.

**Recovery:**
- Check session count: `curl -s http://localhost:4096/session | jq 'length'`
- If high, prune sessions: `orch clean --sessions`
- Use agentlog SSE only when needed (opt-in).

**Prevention:**
- Keep session count low and ensure caches are warm.

**References:** `.kb/guides/dashboard.md`

---

## Playbook: Dashboard Shows 0 Agents

**Symptoms:** Dashboard UI empty but API returns data.

**Likely cause:** Svelte 5 runes mixed with Svelte 4 syntax.

**Recovery:**
- Remove runes (`$state`, `$derived`, `$effect`) and use Svelte 4 reactive syntax.

**Prevention:**
- Keep dashboard codebase in Svelte 4 mode until full migration.

**References:** `.kb/guides/dashboard.md`

---

## Playbook: Status Shows Stale/Phantom Agents

**Symptoms:** `orch status` shows more agents than expected or phantom entries.

**Likely causes:** Stale OpenCode sessions, missing beads ID in session title.

**Recovery:**
- Check sessions: `curl -s http://localhost:4096/session | jq 'length'`
- Restart OpenCode server if sessions persist.
- Respawn if titles lack `[beads-id]`.

**Prevention:**
- Ensure spawns include beads ID in session titles.

**References:** `.kb/guides/status.md`

---

## Playbook: Tmux Spawn SIGKILL (Exit 137)

**Symptoms:** Tmux spawn exits with code 137.

**Likely causes:** Stale binary or launchd KeepAlive conflict.

**Recovery:**
- Rebuild: `make install`
- Verify binary source: `orch version --source`
- Check daemon conflict: `launchctl list | grep orch`

**Prevention:**
- Keep `~/bin/orch` up to date.

**References:** `.kb/guides/tmux-spawn-guide.md`

---

## Playbook: Completion Blocked by Gates

**Symptoms:** `orch complete` fails (missing Phase: Complete, tests, or visuals).

**Likely cause:** Required evidence not reported.

**Recovery:**
- Phase missing: `bd comment <id> "Phase: Complete - <summary>"`
- Test evidence: `bd comment <id> "Tests: go test ./pkg/... - ok (N tests in Xs)"`
- Visual approval: `orch complete <id> --approve`

**Prevention:**
- Agents should report phase and evidence during work.

**References:** `.kb/guides/completion-gates.md`

---

## Playbook: Workspace Accumulation

**Symptoms:** Hundreds of old workspaces in `.orch/workspace/`.

**Likely cause:** Manual archival not run.

**Recovery:**
- Preview archival: `orch clean --stale --dry-run`
- Archive: `orch clean --stale`

**Prevention:**
- Include cleanup in session-end workflow.

**References:** `.kb/guides/workspace-lifecycle.md`

---

## Playbook: Cross-Project Beads Comment Fails

**Symptoms:** `bd comment` returns "issue not found" for cross-project spawns.

**Likely cause:** Beads issue exists in orchestrator repo, agent runs in target repo.

**Recovery:**
- Use `--no-track` for cross-repo work, or create issue in target repo and use `--issue`.

**Prevention:**
- Avoid mixing beads tracking across repos without explicit `--workdir` + issue handling.

**References:** `.kb/guides/beads-integration.md`
