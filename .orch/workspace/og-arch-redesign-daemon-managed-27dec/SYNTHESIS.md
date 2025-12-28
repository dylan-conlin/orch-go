# Session Synthesis

**Agent:** og-arch-redesign-daemon-managed-27dec
**Issue:** orch-go-bmhd
**Duration:** 2025-12-27 17:30 → 2025-12-27 18:30
**Outcome:** success

---

## TLDR

Redesigned dev servers as launchd-native services (not on-demand health checks). After reboot, servers auto-start via RunAtLoad plists - zero orchestrator attention, no SessionStart gating needed.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-27-design-launchd-dev-servers.md` - New investigation with launchd-native design

### Files Modified
- None (design-only phase)

### Commits
- Pending (investigation file ready)

---

## Evidence (What Was Observed)

- `com.orch.daemon.plist` proves launchd pattern works with RunAtLoad + KeepAlive
- `com.orch-go.serve.plist` shows same pattern for orch serve
- `com.user.tmuxinator.plist` shows existing tmuxinator autoload (but not for servers)
- `workers-orch-go.yml` and `workers-price-watch.yml` have server definitions in tmuxinator format
- `~/.orch/ports.yaml` has port allocations that can inform plist generation
- price-watch docker-compose.yml lacks `restart: unless-stopped` policy

### Tests Run
```bash
# Verified existing launchd plists exist and are valid
cat ~/Library/LaunchAgents/com.orch.daemon.plist  # Valid XML
cat ~/Library/LaunchAgents/com.orch-go.serve.plist  # Valid XML
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-27-design-launchd-dev-servers.md` - launchd-native server design

### Decisions Made
- **Servers are infrastructure, not dependencies:** Frame them like orch daemon (always running), not like test preconditions (check before work)
- **Use launchd-native plists:** RunAtLoad + KeepAlive gives us auto-start on boot + auto-restart on crash
- **Supersede prior investigation:** The on-demand health check approach violated the HARD CONSTRAINT

### Constraints Discovered
- launchd requires fully-expanded PATH (doesn't inherit user shell PATH)
- launchd WorkingDirectory must be absolute path
- Docker Compose projects may benefit from native `restart: unless-stopped` instead of launchd wrapper

### Externalized via `kn`
- Not applicable (investigation documents the constraints)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation with design)
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete orch-go-bmhd`

### Implementation Work (for follow-up spawns)

**Phase 1: servers.yaml + gen-plist**
- `pkg/servers/config.go` - servers.yaml schema
- `pkg/servers/launchd.go` - plist generation
- `cmd/orch/servers.go` - gen-plist subcommand

**Phase 2: install/uninstall**
- `orch servers install <project>` - launchctl bootstrap
- `orch servers uninstall <project>` - launchctl bootout

**Phase 3: Dashboard visibility**
- `pkg/servers/health.go` - TCP/HTTP health checks
- `cmd/orch/serve.go` - enhance /api/servers

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How to handle servers that need environment variables beyond PATH (DATABASE_URL, API_KEYS)?
- Should we support server dependencies (wait for postgres before rails)?
- How to handle Docker Desktop startup (separate from launchd)?

**Areas worth exploring further:**
- Log rotation for launchd-managed services (newsyslog.d)
- Health check caching to reduce /api/servers latency

**What remains unclear:**
- Best approach for price-watch: launchd wrapper vs Docker native restart policies

---

## Session Metadata

**Skill:** architect
**Model:** Claude
**Workspace:** `.orch/workspace/og-arch-redesign-daemon-managed-27dec/`
**Investigation:** `.kb/investigations/2025-12-27-design-launchd-dev-servers.md`
**Beads:** `bd show orch-go-bmhd`
