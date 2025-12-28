# Session Synthesis

**Agent:** og-arch-dev-servers-infrastructure-27dec
**Issue:** orch-go-swei
**Duration:** 2025-12-27 17:30 → 2025-12-27 18:20
**Outcome:** success

---

## TLDR

Evaluated 4 dev server infrastructure options (launchd, Docker Compose, Nix/devbox, Hybrid) through Dylan-centric lens. **Reached DIFFERENT conclusion than prior investigation** (2025-12-27-design-launchd-dev-servers.md which recommended Docker Compose). This investigation recommends **launchd plists per-project** based on higher weight for "operational invisibility" - Docker Desktop is a thing Dylan must ensure is running, launchd is invisible OS infrastructure. **Orchestrator should adjudicate the conflict.**

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-27-design-dev-servers-infrastructure-evaluate-options.md` - Full investigation with 4 options evaluated against Dylan-centric criteria

### Files Modified
- None

### Commits
- Pending (will commit this synthesis and investigation together)

---

## Evidence (What Was Observed)

- Dylan already has launchd daemons working well (`com.orch.daemon.plist`, `com.opencode.serve.plist`) - proving the pattern works in his environment
- Docker Desktop on Mac requires manual attention (auto-start is inconsistent) - violates "operational invisibility"
- Nix/devbox not installed (`which nix devbox` returned empty) - no existing tooling familiarity
- Current dev servers are managed via tmuxinator (`workers-price-watch.yml`, `workers-orch-go.yml`) - requires manual session start
- price-watch uses Docker Compose with Postgres, Redis - containerized services that benefit from Docker

### Context Gathered
```bash
# Verified launchd pattern already in use
ls ~/Library/LaunchAgents/*.plist | grep orch
# com.orch.daemon.plist exists and works

# Checked for Nix/devbox
which nix devbox
# Neither installed

# Read existing tmuxinator configs
# Servers require manual tmuxinator start - not reboot resilient
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-27-design-dev-servers-infrastructure-evaluate-options.md` - Comprehensive evaluation of 4 infrastructure options

### Decisions Made
- **Recommend launchd per-project** because: (1) Dylan already uses launchd successfully, (2) it's Mac-native with excellent reboot resilience, (3) AI can debug with standard macOS tooling, (4) no additional software (Docker Desktop) required to be running

### Constraints Discovered
- Docker Desktop auto-start on Mac is unreliable - this makes Docker-only approach unsuitable for "operational invisibility"
- "Implementation complexity" reframing: In Dylan's context where AI does all work, implementation complexity is paid by AI, not Dylan - this changes which option is "best"

### Key Insight
The critical reframe: traditional software engineering values "simplicity" meaning less code/tooling. But in Dylan's context, "simplicity" means Dylan's mental model, not engineering elegance. launchd means "servers just work after reboot" - one concept Dylan already trusts.

### Externalized via `kn`
- (Recommend orchestrator run) `kn decide "launchd per-project for dev servers" --reason "Best operational invisibility for Dylan, proven pattern, reboot resilient"`

---

## Next (What Should Happen)

**Recommendation:** escalate

### If Escalate
**Question:** Which approach for dev server infrastructure - launchd (this investigation) or Docker Compose (prior investigation)?

**Options:**
1. **launchd plists per-project** (this investigation)
   - Pros: True operational invisibility (no Docker Desktop), proven pattern (orch daemon), native macOS
   - Cons: Scattered logs, less unified debugging
   
2. **Docker Compose for everything** (prior investigation 2025-12-27-design-launchd-dev-servers.md)
   - Pros: Unified debugging (`docker compose logs`), already used for price-watch
   - Cons: Docker Desktop dependency (must be running, can fail, needs updates)

**Key disagreement:** Is Docker Desktop reliable enough that Dylan never thinks about it?
- Prior investigation: Yes, configure auto-start and it just works
- This investigation: No, Docker Desktop is macOS application that can fail/quit/need updates

**My recommendation:** launchd, because "zero Dylan intervention" means truly invisible infrastructure, not "Dylan rarely needs to think about Docker Desktop"

### Feature List Conflict
Note: `.orch/features.json` currently has Docker Compose features (feat-033 through feat-036) marked as todo, and launchd features (feat-029 through feat-032) marked as cancelled. Orchestrator should update feature list after adjudicating.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How to handle port allocation conflicts when multiple projects run simultaneously via launchd?
- Should launchd plists use `WatchPaths` to auto-restart on code changes (like nodemon behavior)?
- How to coordinate the launchd infrastructure with the servers.yaml health check design from prior investigation?

**Areas worth exploring further:**
- launchd `LaunchAfter` for sequencing (Docker Desktop → Docker projects)
- Log rotation for launchd-managed servers

**What remains unclear:**
- Whether Docker Desktop can be reliably started by launchd before Docker Compose services
- Interaction between per-project plists and orch daemon plist (potential contention)

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4
**Workspace:** `.orch/workspace/og-arch-dev-servers-infrastructure-27dec/`
**Investigation:** `.kb/investigations/2025-12-27-design-dev-servers-infrastructure-evaluate-options.md`
**Beads:** `bd show orch-go-swei`

---

## Feature List Review (Mandatory)

**Reviewed:** `.orch/features.json`

**Observation:** Feature list already contains dev server features from prior investigation:
- feat-033 through feat-036: Docker Compose approach (currently marked `todo`)
- feat-029 through feat-032: launchd approach (currently marked `cancelled`)

**Action needed:** Orchestrator should adjudicate between the two investigations and update feature list accordingly:
- If Docker Compose wins: Keep feat-033 through feat-036 as-is
- If launchd wins: Cancel feat-033 through feat-036, un-cancel feat-029 through feat-032

**Not modifying feature list directly** because the conflict needs orchestrator resolution first.
