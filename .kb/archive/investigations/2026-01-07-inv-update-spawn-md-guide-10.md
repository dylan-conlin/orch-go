<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** spawn.md guide was missing ~14 flags and 3 major behavior sections (triage bypass, rate limits, duplicate prevention).

**Evidence:** Compared spawn_cmd.go (1940 lines) against spawn.md (198 lines) - found extensive undocumented functionality.

**Knowledge:** The spawn command has evolved significantly with safety features (rate limits, duplicate prevention, triage bypass friction) that weren't documented.

**Next:** Close - spawn.md now documents all flags and behaviors matching the implementation.

**Promote to Decision:** recommend-no (documentation update, not architectural)

---

# Investigation: Update Spawn Md Guide

**Question:** What flags and behaviors are missing from the spawn.md guide?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: 14 flags missing from documentation

**Evidence:** spawn_cmd.go lines 38-63 define these flags, spawn.md only documented 6.

**Source:** 
- `cmd/orch/spawn_cmd.go:38-63` - flag declarations
- `.kb/guides/spawn.md:72-81` - original "Key Flags" section

**Significance:** Users and agents couldn't discover functionality like tier flags (--light/--full), feature-impl config (--phases/--mode/--validation), or safety flags (--max-agents).

---

### Finding 2: Rate limit monitoring implemented but undocumented

**Evidence:** `checkUsageBeforeSpawn()` function (lines 520-636) implements proactive monitoring with warn at 80%, block at 95%, auto-switch behavior.

**Source:**
- `cmd/orch/spawn_cmd.go:482-636` - UsageThresholds, checkUsageBeforeSpawn
- Environment variables: `ORCH_USAGE_WARN_THRESHOLD`, `ORCH_USAGE_BLOCK_THRESHOLD`, `ORCH_AUTO_SWITCH_DISABLED`

**Significance:** This is a major safety feature preventing rate limit exhaustion, but users had no documentation on thresholds or override options.

---

### Finding 3: Triage bypass and duplicate prevention undocumented

**Evidence:** 
- Manual spawns blocked without `--bypass-triage` (lines 727-732)
- Duplicate detection checks issue status and active sessions (lines 873-905)

**Source:**
- `cmd/orch/spawn_cmd.go:727-732` - triage bypass check
- `cmd/orch/spawn_cmd.go:873-905` - duplicate prevention
- `cmd/orch/spawn_cmd.go:1898-1922` - showTriageBypassRequired function

**Significance:** These behaviors are intentional friction to encourage daemon-driven workflow and prevent wasted work from duplicate spawns.

---

## Synthesis

**Key Insights:**

1. **Documentation drift** - spawn.md was last verified Jan 4, 2026 but the implementation had evolved with many new features.

2. **Safety features prominent in code** - Rate limits, duplicate prevention, and triage bypass are substantial features (400+ lines of code) that were completely undocumented.

3. **Flag categories help discoverability** - Grouped flags by purpose (Required, Core, Mode, Tier, Feature-impl, Safety, Context Quality) makes the 14 flags manageable.

**Answer to Investigation Question:**

The spawn.md guide was missing 14 flags and 3 major behavior sections. All have been documented with examples, environment variables for customization, and explanations of the behaviors' purposes.

---

## Structured Uncertainty

**What's tested:**

- ✅ Flag list matches spawn_cmd.go flag declarations (verified: lines 38-63)
- ✅ Rate limit thresholds match code constants (verified: DefaultUsageThresholds function)
- ✅ Duplicate prevention logic documented correctly (verified: lines 873-905)

**What's untested:**

- ⚠️ All examples in documentation work as written (not executed)

**What would change this:**

- If spawn_cmd.go changes, spawn.md will need updates
- If thresholds change, documentation needs updating

---

## Implementation Recommendations

### Recommended Approach ⭐

**Documentation updates complete** - No further implementation needed.

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go` - Full spawn implementation (1940 lines)
- `.kb/guides/spawn.md` - Original guide (198 lines)
- `docs/cli/orch-go_spawn.md` - Auto-generated CLI docs (out of date)

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-feat-update-spawn-md-07jan-b818/`
- **SYNTHESIS.md:** Created in workspace

---

## Investigation History

**2026-01-07:** Investigation started
- Initial question: What flags and behaviors are missing from spawn.md?
- Context: Orchestrator identified ~10 missing flags and 3 behaviors

**2026-01-07:** Investigation completed
- Status: Complete
- Key outcome: spawn.md updated with 14 flags and 3 behavior sections
