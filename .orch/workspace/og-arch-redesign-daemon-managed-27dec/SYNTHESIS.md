# Session Synthesis

**Agent:** og-arch-redesign-daemon-managed-27dec
**Issue:** orch-go-bmhd
**Duration:** 2025-12-27 17:30 → 2025-12-27 18:30
**Outcome:** success

---

## TLDR

After re-evaluation with correct lens (Dylan=zero mental load, AI=autonomous diagnosis), **Docker Compose** wins over launchd. AI debugs faster with `docker compose logs` than hunting through launchd plists/scattered logs. Add `restart: unless-stopped` + Docker Desktop auto-start = operational invisibility.

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
- **Docker Compose > launchd for AI debuggability:** Unified logging, more AI training data, simpler mental model
- **Evaluation lens matters:** "Implementation complexity" is irrelevant - AI handles that. What matters is: Dylan never thinks, AI diagnoses fast
- **Containerize everything:** Even simple dev servers benefit from Docker's unified tooling

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

**Phase 1: Docker Desktop auto-start**
- Configure Docker Desktop → Preferences → General → Start Docker Desktop when you log in
- One-time manual step (or script via defaults write)

**Phase 2: Add restart policies**
- Add `restart: unless-stopped` to price-watch docker-compose.yml services
- Verify services come up after Docker restart

**Phase 3: Containerize remaining services**
- orch-go vite → add docker-compose.yml with node container
- Any other non-Docker dev servers

**Phase 4: Dashboard visibility**
- `cmd/orch/serve.go` - run `docker compose ps --format json` for /api/servers

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
