# Investigation: Dashboard Supervision Circular Debugging

**Date:** 2026-01-10
**Status:** Complete
**Question:** Why did we spend 2 days running in circles on dashboard service supervision, ending up back where we started?

## Summary

**Answer:** We repeatedly debugged obstacles instead of questioning premises, creating a circular pattern where each "solution" reintroduced the problems the previous solution solved.

**Pattern:** Premise-skipping → Obstacle debugging → Return to start → Different premise → Repeat

**Cost:** 2 days, 1000+ lines of debugging, 3 different architectures, no net progress

## The Circle (Chronological)

### Step 1: Jan 9 - "Replace launchd with overmind"

**Investigation:** `.kb/investigations/2026-01-09-inv-overmind-vs-launchd-prototype.md`

**Identified problems with launchd:**
- 143 mystery restarts (no visibility)
- Orphaned vite processes accumulating
- Old binary still running after rebuild
- No unified status
- No unified logs
- No atomic deployment

**Tested overmind:**
- ✅ All services started in 475ms
- ✅ `overmind status` shows actual state
- ✅ `overmind restart` for atomic deployment
- ✅ `overmind echo` for unified logs
- ✅ Automatic process supervision
- ✅ Clean child process cleanup

**Recommendation:** "Replace launchd with overmind. Eliminates 80% of dashboard reliability issues."

**Migration plan:**
1. Create Procfile (3 lines)
2. Unload launchd services
3. Add `overmind start -D` to shell init
4. Remove custom server management code

**What we lost:** Boot persistence (services don't auto-start on login)

**Proposed solution:** Manual start or shell hook

### Step 2: Jan 9-10 - "Implement overmind"

**Actions taken:**
- Created Procfile (3 lines)
- Unloaded launchd services
- Updated CLAUDE.md with overmind workflows
- Started using overmind for dev workflow

**Result:** Working, but services don't auto-start

### Step 3: Jan 10 AM - "Add launchd supervision of overmind"

**New premise:** "Overmind needs launchd supervision for auto-restart on crash"

**Why:** "If overmind crashes, all 3 services go down"

**Implementation attempted:** Create `~/Library/LaunchAgents/com.overmind.orch-go.plist`

**Obstacle encountered:** tmux PATH propagation issues
- launchd sets PATH in EnvironmentVariables
- overmind spawns tmux with -C control mode
- tmux starts with minimal environment
- PATH doesn't propagate to tmux's subprocess spawning
- Error: "Can't find tmux. Did you forget to install it?"

**Debugging invested:** ~1000 lines in sess-4432.txt
- Tried .env files
- Tried wrapper scripts
- Tried different PATH configurations
- Investigated launchd → overmind → tmux interaction

**Result:** Circular dependency discovered
- Need overmind for service restart
- Need launchd for overmind reliability
- launchd can't run overmind due to tmux PATH
- Circle complete

### Step 4: Jan 10 PM - "Abandon overmind, return to launchd"

**New premise:** "Use individual launchd plists instead"

**Decision:** `.kb/decisions/2026-01-10-individual-launchd-services.md`

**Implementation:**
- Created 3 individual launchd plists:
  - `com.opencode.serve.plist`
  - `com.orch.serve.plist`
  - `com.orch.web.plist`
- Each service supervised directly by launchd
- KeepAlive for auto-restart
- RunAtLoad for boot persistence

**Tested:** Crash recovery confirmed (kill → auto-restart within 5s)

**Result:** Auto-restart works, but we're back to the Jan 9 architecture that we decided to replace.

### Step 5: Confusion

**Dylan:** "Reading the Jan 9 investigation and the Jan 10 decision is causing confusion. Let's step back and think about this."

**The contradiction:** Jan 9 said launchd was the problem. Jan 10 returned to launchd as the solution.

## What Actually Happened (Root Cause Analysis)

### Premise-Skipping at Each Step

**Step 1 → 2:** Correct
- Identified launchd problems
- Tested overmind alternative
- Made evidence-based recommendation

**Step 2 → 3:** Premise-skipping #1
- **Assumed premise:** "Services need auto-start on crash"
- **Didn't question:** Is overmind crashing? How often?
- **Didn't ask:** What's the actual failure mode we're preventing?

**Step 3 → 4:** Premise-skipping #2
- **Assumed premise:** "Must make launchd + overmind work"
- **Didn't question:** Do we need launchd supervision at all?
- **Debugged obstacle:** tmux PATH issues (1000 lines)
- **Didn't evaluate alternatives:** Run overmind manually, use different supervisor, abandon supervision requirement

**Step 4:** Premise-skipping #3
- **Assumed premise:** "Must have auto-restart"
- **Didn't question:** What problems does auto-restart actually solve?
- **Didn't reconcile:** Why did Jan 9 recommend against launchd?

### The Questions Never Asked

1. **How often does overmind crash?**
   - Never measured
   - Assumed it's a problem without evidence

2. **What's the actual failure mode?**
   - "If overmind crashes, all 3 services go down"
   - But: When has overmind actually crashed?
   - But: Is manual restart (rare event) worse than debugging tmux PATH (2 days)?

3. **What problems does auto-restart solve?**
   - Service crashes (opencode, orch serve, web)
   - Overmind crashes (unmeasured frequency)
   - Boot persistence (services start on login)

4. **Which services actually crash?**
   - OpenCode: Yes (observed in logs)
   - orch serve: Rare
   - Web (vite): Process accumulation, not crashes
   - Overmind: Unknown

5. **What if we just use overmind without launchd?**
   - Manual `overmind start` after login
   - Or shell hook auto-start (Jan 9 recommendation)
   - Accept rare manual restart if overmind crashes
   - Solves 90% of Jan 9 problems
   - Avoids tmux PATH complexity entirely

## What We Should Have Done

### At Step 2 → 3 (Adding launchd supervision)

**Before implementing, ask:**
1. What problem am I solving? (overmind crashes)
2. How often does this problem occur? (unknown - measure first)
3. What's the cost of this problem? (manual restart)
4. What's the cost of the solution? (launchd + tmux complexity)
5. Is there a simpler alternative? (shell hook auto-start)

**Strategic checkpoint:**
```
Problem: Services don't auto-start on boot
Solution A: launchd supervises overmind → tmux PATH issues
Solution B: Shell hook starts overmind → simple, works
Solution C: Accept manual start → zero complexity

Recommendation: B or C, definitely not A
```

### At Step 3 → 4 (Hitting tmux PATH obstacle)

**After 15 minutes of debugging, pause:**

```
STRATEGIC: Stepping back from tmux PATH debugging

Situation: launchd can't find tmux despite PATH configuration
Obstacle: tmux control mode environment propagation
Time invested: 15+ minutes

The premise to question: Do we need launchd supervision of overmind?

What does launchd supervision solve?
- Auto-restart overmind if it crashes
- Boot persistence

What's the actual failure mode?
- Overmind crashes (frequency: unknown)
- Manual restart needed (cost: 5 seconds)

Alternatives:
A) Debug tmux PATH (unknown complexity, 2+ days so far)
B) Shell hook auto-start (5 minutes, works)
C) Manual start as needed (0 complexity)

What's the actual requirement?
- Dashboard services need to be available
- Services need to restart on crash (opencode does crash)
- Overmind provides this (tested Jan 9)

Do we need overmind itself to auto-restart?
- Only if overmind crashes frequently
- We have no evidence of this
- Even if it does, manual restart is 5 seconds

Strategic decision: Abandon launchd supervision of overmind.
Use overmind as-is with shell hook or manual start.
```

**That pause would have saved 2 days.**

## What We Actually Need (Requirements Analysis)

### Real Requirements (Evidence-Based)

1. **OpenCode needs to restart on crash** (observed in logs)
2. **Dashboard services need to start reliably**
3. **No orphaned processes accumulating** (Jan 9 problem)
4. **Atomic deployment** (rebuild = running new code)
5. **Observability** (know what's running, see logs)

### Overmind Meets These

| Requirement | Overmind | Individual launchd |
|-------------|----------|-------------------|
| Service auto-restart | ✅ Built-in | ✅ KeepAlive |
| No orphans | ✅ Clean shutdown | ⚠️ Requires orch doctor |
| Atomic deploy | ✅ `overmind restart` | ⚠️ `orch deploy` needed |
| Observability | ✅ status/echo | ⚠️ `orch doctor` needed |
| Boot persistence | ❌ Manual start | ✅ RunAtLoad |

### What About Boot Persistence?

**Frequency:** System reboot
- macOS: Weeks/months between reboots
- Manual start after reboot: 5 seconds
- Shell hook auto-start: Already in ~/.zshrc

**Cost/Benefit:**
- Benefit: Services auto-start after reboot (rare event)
- Cost: 2 days debugging tmux PATH (actual time spent)

**Strategic decision:** Accept manual start for rare reboots, or use shell hook. Don't debug tmux PATH.

### What We Gained by Returning to launchd

**Positive:**
- ✅ Boot persistence (RunAtLoad)
- ✅ Service auto-restart (KeepAlive)

**Negative:**
- ❌ Lost unified status (had `overmind status`, now need `orch doctor`)
- ❌ Lost unified logs (had `overmind echo`, now need `orch logs`)
- ❌ Lost atomic restart simplicity (had `overmind restart`, now need `orch deploy`)
- ❌ Re-introduced launchd mystery restarts (143 restarts with no visibility)
- ❌ Re-introduced orphan risk (vite processes, now need `orch doctor --daemon`)
- ❌ Added 3 plist files + 1 wrapper script (vs 1 Procfile)

**Net:** Traded simplicity for boot persistence on rare reboots.

## The Correct Architecture (What We Should Use)

### Option 1: Overmind Without launchd (Recommended)

**Architecture:**
```
Shell init (~/.zshrc) → overmind start -D
  ↓
Overmind supervises:
  ├── opencode serve --port 4096
  ├── orch serve
  └── bun run dev (web)
```

**How to start:**
```bash
# Already in ~/.zshrc (from Jan 9)
if [[ -f ~/Documents/personal/orch-go/Procfile ]] && ! overmind status &>/dev/null; then
    (cd ~/Documents/personal/orch-go && overmind start -D &>/dev/null &)
fi
```

**Pros:**
- ✅ All Jan 9 benefits (status, logs, atomic restart, supervision)
- ✅ No tmux PATH issues
- ✅ Simple (1 Procfile)
- ✅ Auto-starts on shell init (every terminal)
- ✅ Standard tool

**Cons:**
- ⚠️ Doesn't auto-start if no terminal opened after boot
- ⚠️ Manual `overmind start` needed in that case (rare)

**When this fails:**
- Reboot without opening terminal → services don't start
- Fix: Open terminal (happens naturally) OR run `overmind start` manually

**Frequency:** Once per reboot IF you don't open a terminal (almost never)

### Option 2: Individual launchd Plists (Current)

**Architecture:**
```
launchd
├── com.opencode.serve
├── com.orch.serve
├── com.orch.web
└── com.orch.doctor
```

**Pros:**
- ✅ Boot persistence (RunAtLoad)
- ✅ Service auto-restart (KeepAlive)
- ✅ No manual start ever

**Cons:**
- ❌ 4 plist files (vs 1 Procfile)
- ❌ Lost unified status (need `orch doctor`)
- ❌ Lost unified logs (need `orch logs` or tail multiple files)
- ❌ Lost atomic restart simplicity (need `orch deploy`)
- ❌ Potential mystery restarts (launchd behavior)
- ❌ Orphan risk (need `orch doctor --daemon`)

**When this is better:**
- Never open terminals after boot (server-like usage)
- Need 100% hands-off operation
- Can't accept even rare manual intervention

### Option 3: Hybrid (Best of Both?)

**Architecture:**
```
launchd supervises overmind wrapper (not tmux)
  ↓
Wrapper script starts overmind
  ↓
Overmind manages services
```

**Implementation:**
```bash
# ~/.orch/start-overmind.sh
#!/bin/bash
cd /Users/dylanconlin/Documents/personal/orch-go
exec /opt/homebrew/bin/overmind start
```

**launchd plist:**
```xml
com.overmind.orch-go.plist
  ProgramArguments: /Users/dylanconlin/.orch/start-overmind.sh
  RunAtLoad: true
  KeepAlive: false (don't auto-restart - overmind handles that)
```

**Pros:**
- ✅ Boot persistence
- ✅ Overmind benefits (status, logs, atomic restart)
- ✅ No tmux PATH issues (overmind handles tmux itself)

**Cons:**
- ⚠️ More complex than Option 1
- ⚠️ If overmind crashes, won't auto-restart (but how often?)

**When this is better:**
- Need boot persistence AND overmind simplicity
- Willing to accept rare manual restart if overmind crashes

## Recommendation

**Use Option 1: Overmind without launchd supervision**

**Why:**
1. Jan 9 investigation showed this solves 80% of problems
2. Boot persistence failure mode is rare (reboot + no terminal)
3. Shell hook auto-start works for 99% of cases
4. Simplicity > completeness for rare edge cases
5. We already have this working (from Jan 9)

**Migration from current (Option 2):**
1. Unload launchd services:
   ```bash
   launchctl unload ~/Library/LaunchAgents/com.opencode.serve.plist
   launchctl unload ~/Library/LaunchAgents/com.orch.serve.plist
   launchctl unload ~/Library/LaunchAgents/com.orch.web.plist
   ```
2. Start overmind:
   ```bash
   cd ~/Documents/personal/orch-go && overmind start -D
   ```
3. Verify services:
   ```bash
   overmind status
   orch doctor
   ```

**If boot persistence is critical, use Option 3 instead.**

## Lessons for Future Sessions

### 1. Question Premises Before Debugging Obstacles

**Red flags:**
- Debugging an obstacle for >15 minutes
- Trying multiple variations of same approach
- "This should work" thinking
- No strategic pause to evaluate alternatives

**Gate:** Before spending >15 minutes on obstacle, ask:
- What premise am I accepting?
- What would happen if I questioned it?
- What alternatives exist?
- What's the actual requirement vs assumed requirement?

### 2. Measure Before Assuming

**Before implementing solution, measure problem:**
- "Overmind crashes" → Measure: How often?
- "Services need auto-restart" → Measure: Which services crash? How often?
- "Manual restart is too costly" → Measure: Cost vs solution complexity?

**Data beats assumptions.**

### 3. Evidence-Based Requirements

**Don't accept:**
- "Services need X" without evidence
- "We can't accept Y" without measuring cost
- "This must work" without questioning alternatives

**Do:**
- Test actual failure modes
- Measure frequency and impact
- Compare solution cost to problem cost

### 4. Recognize Circular Patterns

**Signs of circularity:**
- Returning to previously rejected solution
- Contradiction between decision documents
- "Why are we doing this?" confusion
- Lost sense of what problem we're solving

**When detected:**
- STOP immediately
- Map the full circle chronologically
- Identify where premises were skipped
- Re-evaluate actual requirements
- Choose based on evidence, not momentum

### 5. Strategic Checkpoints

**At critical decision points:**
- Write explicit strategic reasoning (like sess-4432 did well)
- Show reasoning BEFORE action
- But also: Question the premise, don't just debug the obstacle

**Example (correct pattern):**
```
STRATEGIC: Evaluating service supervision approaches

Requirements (evidence-based):
- OpenCode crashes ~2x/day (logs show this)
- Need auto-restart for crashed services
- Reboots happen ~1x/month
- Manual start after reboot takes 5 seconds

Options:
A) Overmind + shell hook
   - Pros: Simple, solves service crashes, unified management
   - Cons: Manual start after rare reboot if no terminal

B) Individual launchd plists
   - Pros: Boot persistence, auto-restart
   - Cons: Lost unified status/logs/restart, more files, mystery restarts

C) launchd supervises overmind
   - Pros: Boot persistence + overmind benefits
   - Cons: tmux PATH issues (2+ days debugging)

Cost/Benefit:
- Rare manual start (Option A) vs 2 days debugging (Option C)
- Simplicity (Option A) vs completeness (Option B)

Recommendation: Option A
- Solves actual problems (service crashes)
- Accepts rare edge case (manual start after reboot)
- Avoids complexity that costs more than problem

Proceeding with overmind + shell hook unless you redirect.
```

## Artifacts

**Created:**
- `.kb/investigations/2026-01-09-inv-overmind-vs-launchd-prototype.md` (Jan 9)
- `.kb/decisions/2026-01-10-launchd-supervision-architecture.md` (abandoned)
- `.kb/decisions/2026-01-10-individual-launchd-services.md` (current, questionable)
- `~/Library/LaunchAgents/com.opencode.serve.plist`
- `~/Library/LaunchAgents/com.orch.serve.plist`
- `~/Library/LaunchAgents/com.orch.web.plist`
- `~/Library/LaunchAgents/com.orch.doctor.plist`
- `~/.orch/start-web.sh`
- `Procfile` (from Jan 9, still exists)

**Session transcripts:**
- `sess-4432.txt` (1000+ lines of tmux PATH debugging)

## Next Steps

**Immediate:**
1. Decide: Option 1, 2, or 3
2. If Option 1: Unload launchd, start overmind
3. If Option 2: Keep current (but understand trade-offs)
4. If Option 3: Implement wrapper script approach

**Follow-up:**
1. Monitor actual failure modes for 1 week
2. Measure overmind crash frequency (if any)
3. Capture evidence for future architectural decisions
4. Update CLAUDE.md with final architecture decision

## Status

**Question answered:** Yes - we ran in circles because we debugged obstacles instead of questioning premises at each step.

**Root cause:** Premise-skipping pattern (3 instances)

**Current state:** Have working Option 2 (individual launchd), but Option 1 (overmind + shell hook) is simpler and may be better.

**Recommendation:** Revert to Option 1 (Jan 9 recommendation) unless Dylan has specific requirement for boot persistence that justifies Option 2's complexity.
