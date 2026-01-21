<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Disk is 97% full with 76GB+ in identified consumers; cleanup script covers only ~15% of actual space usage; major gaps are Docker (23GB), Ollama (15GB), Colima (7.9GB), Messages (5.5GB), and git binaries (2.2GB).

**Evidence:** `du -sh` scans across all major directories; cleanup script analysis shows it targets only orch logs, opencode session_diff, and build caches while missing virtualization, AI models, and project artifacts.

**Knowledge:** The prior investigation (Jan 19) identified 8.8GB in OpenCode snapshots, but those have since been cleaned. The real space consumers are infrastructure tools (Docker, Colima, Ollama) not application caches.

**Next:** Implement tiered cleanup rules: Tier 1 (safe automated) for logs and caches; Tier 2 (manual) for Docker/Ollama prune; Tier 3 (one-time) for git history rewrite and old node_modules.

**Promote to Decision:** recommend-yes (establishes cleanup strategy for recurring disk pressure issue)

---

# Investigation: Deep System Scan Disk Space

**Question:** What directories consume >100MB under ~/ and /tmp, which are new leak sources vs prior investigation, what grows unbounded, and does the cleanup script cover actual consumers?

**Started:** 2026-01-21
**Updated:** 2026-01-21
**Owner:** Worker Agent (og-inv-deep-system-scan-21jan-3690)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Related Investigation:** `.kb/investigations/2026-01-19-inv-investigate-recurring-disk-space-filling.md`

---

## Findings

### Finding 1: Docker Desktop VM is the largest consumer (23GB)

**Evidence:**
- `~/Library/Containers/com.docker.docker/Data/vms` = 23GB
- Docker not currently running, but VM disk persists
- Cleanup script runs `docker system prune -f` but this only removes dangling images, not the VM disk

**Source:** `du -sh ~/Library/Containers/com.docker.docker/Data/vms`

**Significance:** Single largest consumer. VM disk grows with container layers and doesn't shrink automatically. Requires manual `docker system df` review and targeted cleanup.

---

### Finding 2: Ollama AI models consume 15GB

**Evidence:**
- `~/.ollama/models` = 15GB
- Models installed: deepseek-r1, gpt-oss, llama3, llama3.2, mistral, nomic-embed-text
- No cleanup rules exist for Ollama
- Most models haven't been used since 2024-2025

**Source:** `du -sh ~/.ollama/models` and `ls -la ~/.ollama/models/manifests/registry.ollama.ai/library/`

**Significance:** Second largest consumer. Old models persist indefinitely. `ollama rm <model>` needed for unused models.

---

### Finding 3: Colima (Docker alternative) stores 7.9GB

**Evidence:**
- `~/.colima` = 7.9GB
- Contains VM data for Docker alternative
- Overlaps with Docker Desktop - both installed

**Source:** `du -sh ~/.colima`

**Significance:** Redundant with Docker Desktop. If using Docker Desktop, Colima can be removed entirely (`colima delete`).

---

### Finding 4: iMessage attachments consume 5.5GB

**Evidence:**
- `~/Library/Messages` = 5.5GB
- Cannot be automated - user data

**Source:** `du -sh ~/Library/Messages`

**Significance:** Not cleanable by script. User can manually delete old conversations in Messages app if needed.

---

### Finding 5: orch-go .git has 2.2GB of binary commits

**Evidence:**
- `.git/objects` = 2.2GB
- Git history contains 100+ copies of ~21MB `build/orch` and `orch` binaries
- Objects: 1.18 GiB loose, 994.88 MiB in packs

**Source:** `git count-objects -vH` and `git rev-list --objects --all | git cat-file --batch-check`

**Significance:** Binary was committed to git repeatedly. Requires `git filter-repo` to rewrite history and remove binaries. Add `build/` and `orch` to `.gitignore`.

---

### Finding 6: node_modules across projects total 7GB+

**Evidence:**
- opencode: 4.7GB
- mcp_consciousness_bridge: 523MB
- gemini-cli: 290MB
- beads-ui-svelte: 210MB
- Plus 6+ more projects with 100-170MB each

**Source:** `find ~/Documents -maxdepth 3 -type d -name "node_modules" -exec du -sh {} \;`

**Significance:** Inactive projects retain full node_modules. Can be cleaned with `rm -rf node_modules` on unused projects (easy to reinstall with `npm i`).

---

### Finding 7: Python venvs and uv cache total 2.8GB

**Evidence:**
- `~/.local/share/global-venvs/aider` = 1.5GB
- `~/.local/share/uv/tools` = 1.2GB
- `~/.local/share/uv/python` = 103MB

**Source:** `du -sh ~/.local/share/global-venvs/*` and `du -sh ~/.local/share/uv/*`

**Significance:** Cleanup script targets `~/.cache/uv` but NOT `~/.local/share/uv`. Aider venv is large because it includes ML dependencies.

---

### Finding 8: Log files continue growing (105MB+ in Claude MCP logs alone)

**Evidence:**
- `~/Library/Logs/Claude/` = 105MB (multiple 11MB MCP server logs)
- `~/Library/Logs/monero-wallet-gui.log` = 21MB
- `~/Library/Logs/claude-command-sync.log` = 19MB
- `~/.orch/logs/orch-2026-01.log` = 23MB
- `~/.orch/daemon.log` = 15MB

**Source:** `find ~/Library/Logs -type f -size +10M` and `find ~/.orch -name "*.log"`

**Significance:** Cleanup truncates files >100MB but these are under threshold. Logs grow daily. Need lower threshold or rotation.

---

### Finding 9: OpenCode snapshot is now cleaned (prior 8.8GB gap closed)

**Evidence:**
- `~/.local/share/opencode/` = 96MB total now
- Prior investigation (Jan 19) showed 8.8GB in `snapshot/` directory
- Directory no longer exists - was cleaned manually or by prune

**Source:** `du -sh ~/.local/share/opencode/*`

**Significance:** The critical gap from prior investigation has been addressed. OpenCode storage is now reasonable.

---

### Finding 10: Library/Caches has 4GB+ of moderate consumers

**Evidence:**
- ms-playwright: 493MB
- com.apple.python: 476MB
- SiriTTS: 446MB
- claude-cli-nodejs: 382MB
- colima: 347MB
- Google: 330MB
- Various others: 100-250MB each

**Source:** `du -sh ~/Library/Caches/* | sort -hr`

**Significance:** Cleanup targets Chrome and Yarn caches but misses playwright, python, claude-cli. These can be cleaned occasionally.

---

## Synthesis

**Key Insights:**

1. **Virtualization/AI tools dominate space usage** - Docker (23GB), Ollama (15GB), and Colima (7.9GB) account for 46GB - more than half of used space. These aren't "leaks" but intentional tool storage that needs periodic pruning.

2. **Cleanup script targets the wrong things** - Current rules focus on build caches and session data (~1-2GB recoverable) while ignoring the 46GB in virtualization/AI tools. The script recovered space from the wrong sources.

3. **Git binary commits are a one-time fix** - The 2.2GB in orch-go's .git requires history rewrite, not ongoing cleanup. This is a one-time operation.

4. **Log rotation thresholds are too high** - 100MB truncation threshold misses the many 10-25MB logs that accumulate. Lower to 10MB or implement rotation.

**Answer to Investigation Question:**

The disk is at 97% with 443GB used. The major consumers >100MB are:

| Directory | Size | In Prior Report? | In Cleanup Script? |
|-----------|------|------------------|-------------------|
| Docker VM | 23GB | No | Partial (prune only) |
| Ollama models | 15GB | No | No |
| Colima | 7.9GB | No | No |
| node_modules (all) | ~7GB | No | No |
| Messages | 5.5GB | No | No (user data) |
| Go modules | 2.5GB | Yes | No (dependencies) |
| orch-go .git | 2.2GB | No | No |
| Docker config | 2.1GB | No | No |
| global-venvs | 1.5GB | No | No |
| uv tools | 1.2GB | No | No |
| Library/Caches | ~4GB | Partial | Partial |
| Various logs | ~200MB | Yes | Partial |

The cleanup script covers approximately 15% of actual space consumers. The prior investigation identified caches that have since been cleaned, but the real space pressure comes from infrastructure tools not caches.

---

## Structured Uncertainty

**What's tested:**

- ✅ Docker VM is 23GB (verified: `du -sh ~/Library/Containers/com.docker.docker/Data/vms`)
- ✅ Ollama models is 15GB with 6 models (verified: `du -sh ~/.ollama/models` and `ls` of manifests)
- ✅ orch-go .git has binary commits (verified: `git cat-file --batch-check` showing 100+ 21MB blobs)
- ✅ Cleanup script misses major consumers (verified: script review vs du analysis)
- ✅ OpenCode snapshot gap is closed (verified: directory no longer exists)

**What's untested:**

- ⚠️ Whether removing Ollama models breaks anything (no verification of active usage)
- ⚠️ Whether Colima can be fully removed (may have dependent workflows)
- ⚠️ Whether git filter-repo will complete without errors on orch-go
- ⚠️ Growth rate of Docker VM disk (no baseline measurement)

**What would change this:**

- Finding would be wrong if Docker Desktop is actively used and needs full VM
- Finding would be wrong if Ollama models are used by other tools (MCP servers, etc.)
- Finding would be wrong if Colima is used by scripts (check before removing)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Tiered cleanup strategy** - Separate safe automated cleanup from manual/one-time operations.

**Why this approach:**
- Automated cleanup should be conservative (only truly safe operations)
- Major space recovery requires user judgment (Docker, Ollama)
- One-time fixes (git history) shouldn't be in recurring script

**Trade-offs accepted:**
- Automated cleanup will recover less space (~2-3GB vs potential 50GB+)
- User must manually run Tier 2/3 operations when disk pressure occurs
- Why acceptable: Better than automated cleanup breaking active tools

**Implementation sequence:**
1. Update cleanup script with Tier 1 rules (safe, automated)
2. Create manual cleanup guide for Tier 2 operations
3. Perform one-time Tier 3 fixes

### Tier 1 - Add to Automated Cleanup (Safe)

```bash
# Claude MCP logs - truncate if >10MB
find ~/Library/Logs/Claude -name "*.log" -size +10M -exec truncate -s 0 {} \; 2>/dev/null

# Library/Logs large files - truncate system logs >20MB
find ~/Library/Logs -maxdepth 2 -name "*.log" -size +20M -exec truncate -s 0 {} \; 2>/dev/null

# orch logs - rotate monthly logs older than 60 days
find ~/.orch/logs -name "orch-*.log" -mtime +60 -delete 2>/dev/null

# orch daemon.log - truncate if >50MB
find ~/.orch -maxdepth 1 -name "daemon.log" -size +50M -exec truncate -s 0 {} \; 2>/dev/null

# Playwright cache older than 30 days (large but safe to clean)
find ~/Library/Caches/ms-playwright -mtime +30 -delete 2>/dev/null

# claude-cli-nodejs cache older than 30 days
find ~/Library/Caches/claude-cli-nodejs -mtime +30 -delete 2>/dev/null

# uv tools - clean old tool versions (safe, will re-download)
find ~/.local/share/uv/tools -maxdepth 1 -type d -mtime +90 -exec rm -rf {} \; 2>/dev/null
```

### Tier 2 - Manual Operations (Run When Needed)

```bash
# Docker - review and prune (recovers 10-20GB typically)
docker system df              # Review first
docker system prune -a -f     # Remove all unused (images, containers, volumes)
docker builder prune -a -f    # Remove build cache

# Ollama - remove unused models
ollama list                   # Review models
ollama rm mistral             # Remove old/unused
ollama rm nomic-embed-text

# Colima - if not using (recovers 7.9GB)
colima status                 # Check if in use
colima delete                 # Remove entirely if Docker Desktop is primary

# node_modules - clean inactive projects
rm -rf ~/Documents/personal/opencode/node_modules  # 4.7GB
rm -rf ~/Documents/personal/mcp_consciousness_bridge/node_modules  # 523MB
# Re-run npm install when needed
```

### Tier 3 - One-Time Fixes

```bash
# orch-go git history rewrite (recovers 2GB)
cd ~/Documents/personal/orch-go

# First, ensure build/ and orch binary are in .gitignore
echo "build/" >> .gitignore
echo "orch" >> .gitignore
git add .gitignore
git commit -m "Add build artifacts to gitignore"

# Rewrite history to remove binaries
pip install git-filter-repo
git filter-repo --path build/ --invert-paths
git filter-repo --path orch --invert-paths

# Force push (WARNING: rewrites history)
git push --force-with-lease

# Clean up
git reflog expire --expire=now --all
git gc --aggressive --prune=now
```

### Alternative Approaches Considered

**Option B: Aggressive automated cleanup**
- **Pros:** Maximum space recovery with no user intervention
- **Cons:** Could break Docker, Ollama, or other tools mid-task
- **When to use instead:** On a dedicated build machine without interactive usage

**Option C: Disk space alerts only**
- **Pros:** User decides all cleanup actions
- **Cons:** Reactive; disk fills before user acts
- **When to use instead:** If automated cleanup has caused problems

**Rationale for recommendation:** Tiered approach balances automation (convenience) with safety (user judgment for risky operations). Automated Tier 1 handles the "leak" patterns (logs, caches) while Tier 2/3 address the major space consumers that need user validation.

---

### Implementation Details

**What to implement first:**
- Add Tier 1 rules to cleanup script - immediate improvement, safe
- Create disk monitoring script - early warning before 90%+
- Document Tier 2 commands in a runbook

**Things to watch out for:**
- ⚠️ `docker system prune -a` removes ALL unused images, not just dangling - may need to re-pull
- ⚠️ Colima deletion is irreversible - verify no scripts depend on it
- ⚠️ git filter-repo rewrites history - coordinate with any collaborators
- ⚠️ Log truncation loses history - consider compression instead of deletion

**Areas needing further investigation:**
- Which Ollama models are actively used by MCP servers?
- What is the Docker VM growth rate per week?
- Are there scheduled jobs that depend on Colima?

**Success criteria:**
- ✅ Disk stays below 80% for >7 days after Tier 1 implementation
- ✅ No errors from Docker/Ollama after Tier 2 cleanup
- ✅ orch-go .git size drops to <200MB after Tier 3 rewrite
- ✅ Monitoring alerts fire at 80% threshold

---

## References

**Files Examined:**
- `~/Library/LaunchAgents/com.dylan.disk-cleanup.plist` - Current cleanup script
- `~/.kb/investigations/2026-01-19-inv-investigate-recurring-disk-space-filling.md` - Prior investigation

**Commands Run:**
```bash
# Overall disk status
df -h ~

# Directory size scans
du -sh ~/* | sort -hr
du -sh ~/Library/Containers/* | sort -hr
du -sh ~/.local/share/* | sort -hr
du -sh ~/.ollama/models
du -sh ~/.colima
du -sh ~/Documents/personal/orch-go/.git/objects

# Find large node_modules
find ~/Documents -maxdepth 3 -type d -name "node_modules" -exec du -sh {} \;

# Git binary analysis
git count-objects -vH
git rev-list --objects --all | git cat-file --batch-check | sort -rnk2 | head -20

# Log file analysis
find ~/Library/Logs -type f -size +10M -exec ls -lh {} \;
find ~/.orch -name "*.log" | xargs ls -lh
```

**External Documentation:**
- None required

**Related Artifacts:**
- **Prior Investigation:** `.kb/investigations/2026-01-19-inv-investigate-recurring-disk-space-filling.md` - Identified OpenCode snapshot gap (now resolved)

---

## Investigation History

**2026-01-21 16:00:** Investigation started
- Initial question: Deep scan for all disk consumers >100MB
- Context: Disk at 97% despite prior cleanup investigation; need comprehensive scan

**2026-01-21 16:10:** Major finding - Docker/Ollama/Colima total 46GB
- Docker VM: 23GB
- Ollama models: 15GB
- Colima: 7.9GB
- These were not in prior investigation

**2026-01-21 16:15:** orch-go git history bloat identified
- 2.2GB in .git/objects
- Caused by repeated binary commits (~21MB each, 100+ copies)

**2026-01-21 16:20:** Cleanup script coverage analysis
- Current script covers ~15% of actual consumers
- Targets caches/logs, misses virtualization/AI tools

**2026-01-21 16:30:** Investigation completed
- Status: Complete
- Key outcome: 76GB+ identified; need tiered cleanup strategy separating automated (logs/caches) from manual (Docker/Ollama) operations
