# Investigation: Redesign Control Plane Circuit Breaker Heuristics

**Question:** The single-day commit count heuristic (MAX_COMMITS_PER_DAY) fires false positives during normal batch review (59 commits tripped threshold=20) while the entropy spiral's actual signature was 45 commits/day sustained for 26 days without human verification. What heuristics correctly distinguish supervised bursts from autonomous drift?

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** architect (orch-go-6un)
**Phase:** Complete
**Status:** Complete

**Prior-Work:**

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-02-14-inv-entropy-spiral-deep-analysis.md | informs | yes | None |
| feat: implement immutable control plane (44c06369) | extends | yes | None |

---

## Problem Framing

**Design Question:** What circuit breaker heuristics correctly detect "system is drifting without human verification" while allowing legitimate high-output days?

**Success Criteria:**
1. Feb 14 pattern (61 commits, human present) → no halt
2. Entropy spiral pattern (45/day × 26 days, no human) → halt by day 2-3
3. Implementable in pure shell (no orch binary dependency)
4. Graduated warnings before halt
5. Backward-compatible with existing config format

**Constraints:**
- Shell-only post-commit hook (design principle: no orch binary dependency)
- macOS-specific date/stat commands acceptable (primary platform)
- Must not break existing halt/resume mechanism

---

## Exploration: Decision Forks

### Fork 1: What is the primary signal?

**Options:**
- A: Single-day commit count (current)
- B: Rolling window average (3-day)
- C: Commits-without-human-interaction count
- D: Composite signal (rolling average + human interaction recency)

**Substrate says:**
- Entropy spiral analysis Implication #7: "Verification bandwidth is a real constraint — but it's the control plane's job to enforce it"
- The spiral's signature was multi-dimensional: sustained velocity + zero human interaction
- Single-day count fails on both ends: false positive on burst days, false negative on sustained moderate velocity

**Recommendation:** Option D (composite). No single metric captures both "too fast" and "nobody watching." The two dimensions are independent: velocity measures how fast the system is changing, interaction recency measures whether anyone is verifying.

### Fork 2: How to detect human presence?

**Options:**
- A: Git author email (human vs agent)
- B: Heartbeat file touched by explicit ack command
- C: Login/session detection (check if user active at machine)
- D: Track last orch command invocation

**Substrate says:**
- All commits currently use test@test.com — no author distinction
- The control plane is shell-based, must work without orch binary
- Human "presence" means "actively monitoring and acknowledging the system," not just "logged in"

**Recommendation:** Option B (heartbeat file). Simple, explicit, no false signals. Human runs `orch control ack` to say "I've reviewed the system state." `orch control resume` implicitly acks. Shell checks file mtime.

### Fork 3: What thresholds?

**Options:**
- A: Calibrate from entropy spiral data (45/day average → halt at ~40)
- B: Calibrate from Feb 14 data (61/day peak → allow up to ~80)
- C: Use both: lower threshold for unverified (no heartbeat), higher for verified

**Substrate says:**
- Entropy spiral: 45/day sustained = clearly bad
- Feb 14: 61/day with human present = clearly fine
- The difference isn't the count, it's the verification state

**Recommendation:** Option C. Two operating modes:
- **Verified mode** (heartbeat fresh): Higher thresholds — rolling avg warn at 60, halt at 80
- **Unverified mode** (heartbeat stale >2 days): Lower thresholds — daily commits >15 triggers halt
- **Hard cap** (unconditional): 150/day absolute maximum regardless of mode

### Fork 4: Graduated warnings implementation

**Options:**
- A: 70%/100% of threshold (warn/halt)
- B: Three tiers: notify at 50%, warn at 80%, halt at 100%
- C: Two tiers: warn at threshold, halt at continued excess

**Substrate says:**
- Keep it simple for shell implementation
- Desktop notifications already work (osascript in current hook)
- Too many tiers = alert fatigue

**Recommendation:** Option A. Two levels is enough. Warn at 70% gives time to react. Halt at 100% is the hard stop.

---

## Synthesis: Recommended Design

### Three-Layer Circuit Breaker

**Layer 1: Rolling Average (catches sustained velocity)**
- Compute 3-day rolling average of daily commit counts
- Warn when average exceeds `ROLLING_AVG_WARN` (default: 50)
- Halt when average exceeds `ROLLING_AVG_HALT` (default: 70)
- Why: During the entropy spiral, avg was ~45/day sustained for 26 days. A 3-day window catches sustained velocity while allowing burst days to settle.

**Layer 2: Unverified Velocity (catches autonomous drift)**
- Track `~/.orch/heartbeat` file, touched by `orch control ack` and `orch control resume`
- If heartbeat stale >2 days AND daily commits >15: halt with reason "unverified velocity"
- Why: The entropy spiral had 0 human verification for 26 days. Even 2 days of continued velocity without human acknowledgment is a warning sign that deserves investigation.

**Layer 3: Hard Cap (emergency brake)**
- Single-day hard cap of `DAILY_HARD_CAP` (default: 150)
- No graduated warning — immediate halt
- Why: No legitimate day exceeds 150 commits. This catches runaway scenarios that might slip through rolling averages.

### Config Changes

```conf
# Control Plane Configuration (v2)
# Circuit breakers — daemon halts when threshold exceeded

# Rolling window (replaces MAX_COMMITS_PER_DAY)
ROLLING_WINDOW_DAYS=3
ROLLING_AVG_WARN=50        # warn when N-day avg exceeds this
ROLLING_AVG_HALT=70        # halt when N-day avg exceeds this

# Human verification
MAX_UNVERIFIED_DAYS=2      # halt if no heartbeat AND daily commits > UNVERIFIED_MIN
UNVERIFIED_DAILY_MIN=15    # minimum daily commits to trigger unverified check

# Emergency brake
DAILY_HARD_CAP=150         # absolute single-day maximum

# Backward compat: MAX_COMMITS_PER_DAY still works as DAILY_HARD_CAP alias

# Fix:feat ratio (unchanged)
FIX_FEAT_RATIO_THRESHOLD=50

# Churn ratio (unchanged)
CHURN_RATIO_THRESHOLD=200

# Protected paths (unchanged)
PROTECTED_PATHS="cmd/orch/ pkg/daemon/ pkg/spawn/ pkg/verify/ plugins/"

# Cooldown (unchanged)
COOLDOWN_MINUTES=30
```

### New Command: `orch control ack`

Touch `~/.orch/heartbeat` to signal human is monitoring:
```bash
orch control ack          # Touch heartbeat, show current metrics
orch control resume       # Clear halt AND touch heartbeat (already exists, add heartbeat)
```

### Validation Against Historical Patterns

| Scenario | Rolling Avg | Heartbeat | Hard Cap | Result |
|----------|------------|-----------|----------|--------|
| Feb 14: 61 commits, human active | 61 (1-day, warn) | Fresh (acked) | Under 150 | **Warn only** (first day, avg settles) |
| Entropy spiral day 1: 45 commits | 45 (1-day) | Fresh or stale | Under 150 | Pass (below thresholds) |
| Entropy spiral day 3: 45/day avg | 45 avg | Stale >2d | Under 150 | **HALT** (unverified velocity) |
| Entropy spiral day 7: 45/day sustained | 45 avg | Stale >2d | Under 150 | **HALT** (already halted day 3) |
| Quiet day: 5 commits | Low | Any | Under 150 | Pass |
| Runaway: 200 commits in one day | High | Any | **Over 150** | **HALT** (hard cap) |

The design correctly:
- Allows Feb 14 burst (human present → heartbeat fresh → only warn on rolling avg)
- Catches entropy spiral by day 3 (heartbeat stale + velocity > 15)
- Provides emergency brake for extreme scenarios

### Implementation Plan

1. **Update `~/.orch/hooks/control-plane-post-commit.sh`:**
   - Add rolling average calculation
   - Add heartbeat staleness check
   - Add graduated warnings (notify at 70% of thresholds)
   - Add hard cap check
   - Keep backward compat for MAX_COMMITS_PER_DAY

2. **Update `~/.orch/control-plane.conf`:**
   - Add new config variables with defaults
   - Keep old variables as aliases

3. **Update `pkg/control/control.go`:**
   - Add new Config fields
   - Add HeartbeatPath variable
   - Add Ack() function (touch heartbeat)
   - Update Status() to include rolling average and heartbeat age

4. **Update `cmd/orch/control_cmd.go`:**
   - Add `orch control ack` subcommand
   - Update `resume` to also touch heartbeat
   - Update `status` to show new metrics
   - Update `init` with new defaults

---

## Trade-offs Accepted

1. **Heartbeat requires human action** — Human must explicitly `orch control ack` to signal presence. Could forget. But: the whole point is that passive monitoring failed (26 days unnoticed). Requiring explicit action is the feature, not a bug.

2. **Rolling average lags on burst days** — A single burst day shows as high avg for 3 days. But: with heartbeat fresh, the higher thresholds apply (warn at 50, halt at 70), and the avg normalizes as the window slides.

3. **Shell complexity increases** — More logic in the post-commit hook. But: still pure shell, no binary dependencies, and the logic is straightforward arithmetic.
