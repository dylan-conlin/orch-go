## Summary (D.E.K.N.)

**Delta:** Changed systematic-debugging skill from TierFull to TierLight in SkillTierDefaults map.

**Evidence:** Before: systematic-debugging mapped to TierFull. After: mapped to TierLight. feature-impl was already TierLight.

**Knowledge:** The 5-tier escalation model says code-only skills should auto-complete; this required only one map entry change.

**Next:** Close - change is complete, awaiting orchestrator verification.

**Promote to Decision:** recommend-no (targeted fix to existing tier config, not architectural change)

---

# Investigation: Fix Tier Alignment Feature Impl

**Question:** How to make feature-impl and systematic-debugging skills spawn as light tier instead of full tier?

**Started:** 2026-01-20
**Updated:** 2026-01-20
**Owner:** Worker agent (feature-impl skill)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Tier determination flow

**Evidence:**
- `determineSpawnTier()` in `cmd/orch/spawn_cmd.go:522-539` determines tier
- Priority: --light flag > --full flag > userconfig.default_tier > skill default
- Falls back to `spawn.DefaultTierForSkill(skillName)` in `pkg/spawn/config.go:43-48`

**Source:** cmd/orch/spawn_cmd.go:522-539, pkg/spawn/config.go:43-48

**Significance:** Single location to change skill defaults - the `SkillTierDefaults` map.

---

### Finding 2: feature-impl was already TierLight

**Evidence:** In `pkg/spawn/config.go:36`, feature-impl was already mapped to `TierLight`. The task description mentioned both skills, but only systematic-debugging needed changing.

**Source:** pkg/spawn/config.go:26-39

**Significance:** Only systematic-debugging needed the actual code change.

---

### Finding 3: systematic-debugging was TierFull

**Evidence:** Line 33 had `"systematic-debugging": TierFull, // Produces investigation file with findings`

**Source:** pkg/spawn/config.go:33 (before change)

**Significance:** This was the misalignment - the comment suggested it produces investigation files, but per the 5-tier escalation model, code-focused debugging work should auto-complete.

---

## Synthesis

**Key Insights:**

1. **Single source of truth for tier defaults** - The `SkillTierDefaults` map in `pkg/spawn/config.go` is the canonical location for skill tier defaults.

2. **Comment documentation matters** - The help text in `spawn_cmd.go` also documented the tiers and needed updating to stay consistent.

3. **Minimal change** - Only one map entry change was needed; the system architecture was already correct.

**Answer to Investigation Question:**

Changed `systematic-debugging` from `TierFull` to `TierLight` in the `SkillTierDefaults` map at `pkg/spawn/config.go:36`. Also updated the documentation in `cmd/orch/spawn_cmd.go:108-111` to reflect the new tier assignments. feature-impl was already correctly assigned to TierLight.

---

## Structured Uncertainty

**What's tested:**

- Code change verified by reading modified files

**What's untested:**

- Actual spawn test (go not available in environment)
- Unit test execution

**What would change this:**

- If userconfig has default_tier="full" override, that would take precedence over skill defaults

---

## Implementation Recommendations

### Recommended Approach: Direct map change

**Why this approach:**
- Single source of truth already exists
- No architectural changes needed
- Minimal risk

**Implementation sequence:**
1. Changed systematic-debugging from TierFull to TierLight in SkillTierDefaults
2. Updated documentation in spawn_cmd.go help text

---

## References

**Files Modified:**
- `pkg/spawn/config.go:33-38` - Changed systematic-debugging to TierLight, reorganized light tier section
- `cmd/orch/spawn_cmd.go:108-111` - Updated documentation to match new tier assignments

**Commands Run:**
```bash
# Searched for tier determination logic
grep -n "determineTier\|TierLight\|TierFull" cmd/orch/spawn_cmd.go pkg/spawn/

# Verified changes
cat pkg/spawn/config.go | head -50
```

---

## Investigation History

**2026-01-20:** Investigation started
- Initial question: Make feature-impl and systematic-debugging spawn as light tier
- Context: 5-tier escalation model says code-only skills should auto-complete

**2026-01-20:** Fixed tier alignment
- Changed systematic-debugging from TierFull to TierLight
- feature-impl was already TierLight (no change needed)
- Updated spawn_cmd.go documentation

**2026-01-20:** Investigation completed
- Status: Complete
- Key outcome: systematic-debugging now defaults to light tier, enabling auto-completion
