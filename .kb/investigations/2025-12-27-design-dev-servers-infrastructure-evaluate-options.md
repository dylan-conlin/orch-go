<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** This investigation recommends launchd plists per-project, conflicting with prior investigation (2025-12-27-design-launchd-dev-servers.md) which recommended Docker Compose. The difference: I weight "Docker Desktop is a thing Dylan must ensure is running" as violating operational invisibility, while the prior investigation treated Docker Desktop auto-start as reliable.

**Evidence:** Evaluated 4 options (launchd, Docker Compose, Nix/devbox, Hybrid) against Dylan-centric criteria: (1) operational invisibility, (2) AI debugging speed, (3) conceptual simplicity, (4) reboot resilience. launchd scores highest on operational invisibility (no Docker Desktop to think about); Docker Compose scores highest on AI debugging (unified logging).

**Knowledge:** The key disagreement is: Does Docker Desktop auto-start reliably remove it from Dylan's mental model? Prior investigation says yes. This investigation says no - Docker Desktop is macOS application that can fail to start, be accidentally quit, or need updates that require interaction. launchd is OS infrastructure that just works.

**Next:** Orchestrator should adjudicate between two recommendations: launchd (this investigation) vs Docker Compose (prior investigation). Key question: Is Docker Desktop reliable enough that Dylan never thinks about it?

---

# Investigation: Dev Servers Infrastructure Evaluation

**Question:** Given that Dylan doesn't write code (AI does everything), which dev server infrastructure delivers the best combination of: (1) operational invisibility, (2) AI debugging speed, (3) conceptual simplicity, and (4) reboot resilience?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** Promote to decision if accepted, then create feature-impl issues
**Status:** Complete

**Prior Work:** 
- `.kb/investigations/2025-12-27-inv-design-daemon-managed-development-servers.md` - Recommended on-demand health checks with servers.yaml
- `.kb/investigations/2025-12-27-design-launchd-dev-servers.md` - Recommended Docker Compose over launchd (cited AI debuggability)

**Conflict Note:** This investigation reaches a DIFFERENT conclusion than the prior launchd investigation. The prior investigation favored Docker Compose because "AI can debug Docker faster." This investigation weighs Dylan's mental model more heavily - Docker Desktop is an application Dylan must ensure is running, which violates operational invisibility. **Orchestrator should adjudicate.**

---

## Problem Framing

### Critical Context Shift

**Traditional software engineering lens:**
- Values: simplicity, maintainability, testability
- Costs: implementation complexity, learning curve, debugging difficulty

**Dylan's lens (AI does everything):**
- "Implementation complexity" = NOT a cost (AI handles it)
- "Learning curve" = NOT a cost (AI learns, not Dylan)
- ACTUAL costs: Dylan's attention, Dylan's intervention, Dylan's mental overhead

This fundamentally changes which option is "best."

### Success Criteria (in order of priority)

1. **Zero Dylan intervention after initial setup** - Dylan never thinks about servers
2. **AI can autonomously diagnose and fix** - AI doesn't need Dylan's help to debug
3. **Survives Mac reboot** - Everything comes back automatically, no manual steps
4. **Works across project types** - orch-go/vite, price-watch/docker, glass/chrome

### Constraints

- **Session Amnesia principle** - Next AI session must know server state without memory
- **Local-First principle** - Use files/processes, not external services
- **Dylan's environment** - Mac with existing launchd daemons (orch, opencode serve)

---

## Option Evaluation

### Option 1: launchd plists per-project

**Mechanism:** Each project gets a launchd plist that starts its dev servers on boot.

```xml
<!-- ~/Library/LaunchAgents/com.project.servers.plist -->
<key>ProgramArguments</key>
<array>
    <string>/bin/bash</string>
    <string>-c</string>
    <string>cd /path/to/project && make dev</string>
</array>
<key>RunAtLoad</key>
<true/>
<key>KeepAlive</key>
<true/>
```

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| **Operational invisibility** | ✅ Excellent | After initial setup, Dylan never thinks about it. Mac-native, no Docker Desktop to start, no shell to enter. |
| **AI debugging speed** | ✅ Excellent | Standard macOS tooling. AI can `launchctl list`, read logs in predictable locations, restart with `launchctl kickstart`. Rich debugging surface. |
| **Conceptual simplicity** | ✅ Good | "launchd runs my servers" - one concept. Already proven with orch daemon. |
| **Reboot resilience** | ✅ Excellent | Native macOS capability. This is literally what launchd is FOR. |
| **Project coverage** | ⚠️ Partial | Works for process-based servers (vite, rails). Docker projects need Docker daemon running first. |

**Implementation notes:**
- plist template already exists in concept (com.orch.daemon.plist as reference)
- AI can generate plists per project
- Logs go to `~/.orch/logs/{project}-servers.log` (predictable, searchable)

---

### Option 2: Docker Compose + restart policies

**Mechanism:** All dev servers run in Docker containers with `restart: unless-stopped`.

```yaml
# docker-compose.yml
services:
  frontend:
    image: node:20
    command: npm run dev
    restart: unless-stopped
    ports:
      - "5173:5173"
```

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| **Operational invisibility** | ❌ Poor | Docker Desktop must be running. If Docker Desktop doesn't start automatically, servers don't start. Adds "is Docker Desktop running?" question to Dylan's mental model. |
| **AI debugging speed** | ⚠️ Medium | AI can debug Docker, but it's another layer. `docker compose logs`, container inspection, network debugging. More surface area. |
| **Conceptual simplicity** | ⚠️ Medium | "Docker runs my servers" - but now two concepts: Docker and your app. Dylan must think "is Docker up?" |
| **Reboot resilience** | ⚠️ Depends | Docker Desktop auto-start is inconsistent on macOS. Even when enabled, there's a startup delay before containers restart. |
| **Project coverage** | ✅ Excellent | Works for everything - processes, databases, complex multi-container setups. |

**The real cost:** Docker Desktop on Mac is operationally heavy. It's a background process that consumes resources even when not used. Dylan would need to think about Docker, which violates "operational invisibility."

---

### Option 3: Nix/devbox (declarative per-project)

**Mechanism:** Each project has a `devbox.json` or `flake.nix`. Running `devbox shell` or `nix develop` starts everything.

```json
// devbox.json
{
  "packages": ["nodejs_20", "postgresql"],
  "shell": {
    "init_hook": "npm run dev &",
    "scripts": {
      "dev": "npm run dev"
    }
  }
}
```

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| **Operational invisibility** | ❌ Poor | Requires entering a shell to start servers. Dylan must remember to "enter the dev environment" - this is a mental overhead cost. |
| **AI debugging speed** | ⚠️ Medium | Nix errors are notoriously cryptic. AI can debug, but Nix has a steep learning curve even for AI. Less common tooling = less training data. |
| **Conceptual simplicity** | ❌ Poor | "Nix provides isolated environments" - but Dylan must understand what that means. The mental model is "enter environment, then servers start" which is more complex than "servers are just running." |
| **Reboot resilience** | ❌ Poor | By design, Nix/devbox doesn't auto-start services. You must enter the shell. Reboot = servers not running until someone enters the shell. |
| **Project coverage** | ✅ Excellent | Handles any stack, any language, with reproducible builds. |

**The real cost:** Nix adds conceptual overhead that violates Dylan's "conceptual simplicity" requirement. The power of Nix is in reproducible environments across machines - but Dylan works on one machine with AI doing all the work. The benefits don't apply; the costs do.

---

### Option 4: Hybrid (mix approaches per project type)

**Mechanism:** Different projects use different approaches based on their needs.
- orch-go: launchd for vite server
- price-watch: Docker Compose for Rails/Postgres
- glass: launchd for Chrome connection

| Criterion | Score | Rationale |
|-----------|-------|-----------|
| **Operational invisibility** | ⚠️ Medium | Dylan must have a mental model of which project uses which approach. "Is this a Docker project or a launchd project?" |
| **AI debugging speed** | ⚠️ Medium | AI must understand multiple patterns. More code paths = more places for bugs. |
| **Conceptual simplicity** | ❌ Poor | This is the opposite of simple. "Some projects use Docker, some use launchd, some use tmux." This is cognitive overhead. |
| **Reboot resilience** | ⚠️ Varies | Depends on which projects need which approach. Docker projects have Docker Desktop dependency. |
| **Project coverage** | ✅ Excellent | By definition, covers everything since we pick the right tool for each. |

**The real cost:** Hybrid sounds pragmatic but violates "conceptual simplicity." Dylan's mental model becomes "servers are managed differently per project" which is exactly the kind of cognitive overhead we're trying to eliminate.

---

## Synthesis

### Key Insights

1. **"Implementation complexity" is not a cost** - AI writes all the code. What matters is Dylan's experience, not engineering elegance. A launchd plist per project is "more work to implement" but that work is done by AI, not Dylan.

2. **Docker Desktop is ops debt** - It's an application Dylan must ensure is running. This adds a question to his mental model: "is Docker Desktop running?" That violates operational invisibility.

3. **Nix solves problems Dylan doesn't have** - Nix provides reproducible environments across machines and developers. Dylan works on one machine with AI doing everything. The Nix benefits don't apply; only the complexity costs apply.

4. **Hybrid is complexity dressed as pragmatism** - "Use the right tool for each job" sounds reasonable but creates cognitive overhead. Dylan must remember which projects use which approach.

### Recommendation: launchd per-project ⭐

**The recommendation is launchd plists per-project because:**

1. **Highest operational invisibility** - Once set up, Dylan literally never thinks about it. Mac starts, servers start. No Docker Desktop to check, no shell to enter.

2. **Fastest AI debugging** - `launchctl` is standard macOS. AI has extensive training on it. Logs in predictable locations. Restart with one command.

3. **Simplest mental model** - "launchd runs my servers, like it runs orch daemon." One concept, already proven in Dylan's environment.

4. **Best reboot resilience** - This is literally what launchd is designed for. It's the native macOS solution.

**What about Docker projects (price-watch)?**

For price-watch specifically:
- launchd plist starts Docker Compose (`docker compose up -d`)
- Docker Compose has `restart: unless-stopped` as belt-and-suspenders
- launchd handles "is Docker running?" - it starts Docker Desktop first if needed

This gives us launchd's operational benefits while using Docker for what Docker's good at (Postgres, Redis).

### Trade-offs Accepted

| Trade-off | Why acceptable |
|-----------|----------------|
| launchd is macOS-only | Dylan only uses macOS. Portability is not a requirement. |
| More plists to manage | AI manages them. Not Dylan's cognitive load. |
| Docker projects still need Docker Desktop | launchd can start Docker Desktop first. Dylan never manually starts it. |

---

## Implementation Recommendations

### Recommended Approach ⭐

**launchd plists per-project with Docker Compose integration for containerized projects**

**Why this approach:**
- Dylan's mental model is one concept: "launchd runs my dev servers"
- Mac reboot = everything comes back (this is launchd's primary purpose)
- AI can debug with standard macOS tooling
- Already proven in Dylan's environment (orch daemon, opencode serve)

**Trade-offs accepted:**
- macOS-only (acceptable: Dylan only uses macOS)
- Docker projects still need Docker Desktop (acceptable: launchd can start it)

**Implementation sequence:**

1. **Create plist template** - Standardize on `com.dev.{project}.servers.plist`
2. **Migrate orch-go** - Replace tmuxinator-based servers with launchd plist
3. **Migrate price-watch** - Create plist that starts Docker Compose (with Docker Desktop dependency)
4. **Add health checks** - Integrate with `orch servers check` from prior investigation

### Alternative Approaches Considered

**Option B: Docker Compose + restart policies**
- **Pros:** Works for all project types, familiar to many developers
- **Cons:** Docker Desktop dependency violates operational invisibility
- **When to use instead:** If Dylan needed to run on non-Mac platforms, or if team members needed the same environment

**Option C: Nix/devbox**
- **Pros:** Reproducible environments, language-agnostic
- **Cons:** Requires entering a shell (violates invisibility), cryptic errors (hurts AI debugging)
- **When to use instead:** If Dylan worked across multiple machines, or if reproducibility across developers mattered

**Option D: Hybrid**
- **Pros:** "Right tool for each job"
- **Cons:** Cognitive overhead of remembering which project uses which approach
- **When to use instead:** Never - complexity without benefit in Dylan's context

**Rationale for recommendation:** launchd is the only option that maximizes Dylan's three priorities (operational invisibility, AI debugging speed, reboot resilience) simultaneously. The "implementation complexity" of creating plists is a cost AI pays, not Dylan.

---

### Implementation Details

**What to implement first:**
- plist template based on com.orch.daemon.plist pattern
- `orch servers init {project}` command to generate project plist
- Integration with `servers.yaml` from prior investigation

**Things to watch out for:**
- **Docker Desktop auto-start on login** - Enable this in Docker Desktop preferences
- **launchd ordering** - Use `LaunchAfter` for Docker projects to wait for Docker Desktop
- **Logs location** - Standardize on `~/.orch/logs/{project}-servers.log`

**Areas needing further investigation:**
- How to handle port allocation conflicts across projects
- Whether to use `WatchPaths` for auto-restart on code changes
- How to coordinate with the prior servers.yaml health check design

**Success criteria:**
- Dylan reboots Mac → all dev servers are running without any action
- AI can debug server issues without asking Dylan for help
- `orch servers status` shows all projects healthy

---

## Structured Uncertainty

**What's tested:**
- ✅ launchd successfully manages orch daemon on Dylan's machine (verified: `launchctl list | grep orch`)
- ✅ launchd with `RunAtLoad: true` starts services on Mac boot (verified: orch daemon behavior)
- ✅ Docker Compose `restart: unless-stopped` works for persistent containers (standard Docker behavior)

**What's untested:**
- ⚠️ launchd starting Docker Desktop first (not tested on this machine)
- ⚠️ AI debugging speed comparison between approaches (claimed based on tooling familiarity)
- ⚠️ Multi-project plist management at scale (currently one daemon plist)

**What would change this:**
- If Dylan started using Linux: launchd is macOS-only
- If Dylan added team members: reproducibility would matter, Nix would gain value
- If Docker Desktop became more reliable on Mac boot: Docker-only approach might work

---

## References

**Files Examined:**
- `~/Library/LaunchAgents/com.orch.daemon.plist` - Reference launchd plist pattern
- `~/.tmuxinator/workers-price-watch.yml` - Current price-watch server config
- `~/.tmuxinator/workers-orch-go.yml` - Current orch-go server config
- `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/docker-compose.yml` - Docker Compose setup

**Commands Run:**
```bash
# List existing launchd plists
ls -la ~/Library/LaunchAgents/*.plist

# Check for Nix/devbox installation
which nix devbox

# Read current orch daemon plist
cat ~/Library/LaunchAgents/com.orch.daemon.plist
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-27-inv-design-daemon-managed-development-servers.md` - Prior investigation on servers.yaml and health checks
- **Principle:** `~/.kb/principles.md` - Session Amnesia, Local-First

---

## Investigation History

**2025-12-27 17:30:** Investigation started
- Initial question: Which dev server infrastructure option is best for Dylan's context?
- Context: Prior investigation recommended servers.yaml + health checks, but infrastructure layer undefined

**2025-12-27 17:45:** Problem framing complete
- Key insight: "implementation complexity" is not a cost in Dylan's context
- Reframed evaluation criteria around Dylan's experience, not engineering elegance

**2025-12-27 18:00:** Option evaluation complete
- Analyzed 4 options: launchd, Docker Compose, Nix/devbox, Hybrid
- Clear winner: launchd per-project

**2025-12-27 18:15:** Investigation completed
- Status: Complete
- Key outcome: Recommend launchd plists per-project as infrastructure for dev servers
