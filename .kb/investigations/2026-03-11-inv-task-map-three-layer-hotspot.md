## Summary (D.E.K.N.)

**Delta:** Only bloat-size has blocking enforcement (spawn gate + accretion gates). Fix-density (28 hotspots), coupling-cluster (14), and investigation-cluster (85) have zero blocking gates for manual spawns — they get advisory-only treatment. Layer 1 blocking is currently dormant (0 files exceed 1500-line CRITICAL threshold).

**Evidence:** Code review of all enforcement layers: `pkg/spawn/gates/hotspot.go:76` only blocks on `HasCriticalHotspot` which requires `bloat-size > 1500`. `pkg/daemon/architect_escalation.go:111` matches ALL types but only for daemon-driven spawns. `pkg/verify/accretion.go:127` and `accretion_precommit.go:79` only check line counts. `orch hotspot --json` confirms 0 CRITICAL files.

**Knowledge:** The three-layer system was designed to cover different enforcement gaps (manual vs daemon vs advisory), but all BLOCKING gates key off file size only. 127 of 143 hotspots (89%) are non-bloat types with no blocking enforcement path for manual spawns.

**Next:** Architectural decision needed: should fix-density (the most actionable non-bloat type, 28 hotspots) get blocking enforcement? Recommend architect review — this crosses multiple components (spawn gates, daemon, completion pipeline).

**Authority:** architectural — Cross-component enforcement change affecting spawn gates, daemon, and completion pipeline.

---

# Investigation: Three-Layer Hotspot Enforcement vs Four Hotspot Measurement Types — Coverage Matrix

**Question:** Which enforcement layers target which hotspot types? Are there coverage gaps where hotspot types have no blocking enforcement?

**Started:** 2026-03-11
**Updated:** 2026-03-11
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md | extends | yes | Decision doc describes 3 layers but omits Layer 0 (precommit) and the completion advisory. Coverage gaps not explicitly addressed. |
| .kb/investigations/2026-02-24-design-architect-gate-hotspot-enforcement.md | extends | yes — confirmed implementation matches design | - |

---

## Findings

### Finding 1: Layer 1 Spawn Gate blocks ONLY on bloat-size >1500 — currently dormant

**Evidence:** In `cmd/orch/hotspot_spawn.go:182`:
```go
if h.Type == "bloat-size" && h.Score > 1500 {
    result.HasCriticalHotspot = true
    result.CriticalFiles = append(result.CriticalFiles, h.Path)
}
```

The gate in `pkg/spawn/gates/hotspot.go:76` only blocks when `result.HasCriticalHotspot && IsBlockingSkill(skillName)`. `HasCriticalHotspot` is only set for `bloat-size > 1500`.

Current `orch hotspot --json` output: **0 files** exceed 1500 lines. Maximum is `cmd/orch/harness_init.go` at 1203 lines.

**Source:** `cmd/orch/hotspot_spawn.go:182`, `pkg/spawn/gates/hotspot.go:76`, `orch hotspot --json` output

**Significance:** Layer 1's blocking enforcement is completely dormant. No file in the codebase triggers it. All 143 hotspots pass through Layer 1 without any blocking.

---

### Finding 2: Layer 2 Daemon Escalation is the ONLY blocking layer covering all 4 hotspot types

**Evidence:** In `pkg/daemon/architect_escalation.go:111`:
```go
match := FindMatchingHotspot(files, hotspots)
```

`FindMatchingHotspot` (line 43) iterates ALL hotspot types without filtering. The `GitHotspotChecker.CheckHotspots()` shells out to `orch hotspot --json` which returns all 4 types.

But this only applies to **daemon-driven spawns** (`orch daemon run`). Manual spawns (`orch spawn --bypass-triage`) skip daemon escalation entirely.

**Source:** `pkg/daemon/architect_escalation.go:43-58,78-136`, `pkg/daemon/hotspot_checker.go:31-79`

**Significance:** For the daemon workflow (the preferred path), all 4 types get blocking enforcement via architect escalation. For manual spawns, fix-density, coupling-cluster, and investigation-cluster have zero blocking gates.

---

### Finding 3: Layer 3 (Context Advisory + Completion Advisory) covers all 4 types but is non-blocking

**Evidence:**

Spawn context injection (`cmd/orch/spawn_cmd.go:505-507`) injects `HotspotArea`, `HotspotFiles`, `HotspotDefectClasses` from the full `hotspotResult`, which includes all matched types.

Completion advisory (`cmd/orch/complete_hotspot.go:27-47`) cross-references modified files against all 4 hotspot types including fix-density, investigation-cluster, coupling-cluster, and bloat-size.

Both are informational only — they display warnings but never block.

**Source:** `cmd/orch/spawn_cmd.go:505-507`, `cmd/orch/complete_hotspot.go:19-63`, `pkg/spawn/context.go:158-172`

**Significance:** Agents see hotspot context but can ignore it. Per "Gate Over Remind" principle, advisory-only enforcement is insufficient for preventing the patterns hotspots are meant to catch.

---

### Finding 4: Accretion gates (Layer 0 precommit + completion) only check file line count

**Evidence:**

Pre-commit gate (`pkg/verify/accretion_precommit.go:79`): Checks `stagedLines > AccretionCriticalThreshold` (1500). Pure line-count check.

Completion gate (`pkg/verify/accretion.go:127`): Checks `change.CurrentLines > AccretionCriticalThreshold`. Same pure line-count check.

Neither checks fix-density, coupling, or investigation clusters.

**Source:** `pkg/verify/accretion_precommit.go:38-125`, `pkg/verify/accretion.go:60-160`

**Significance:** These gates are a fourth enforcement layer (Layer 0) not mentioned in the three-layer decision doc. They cover only bloat-size, leaving 89% of hotspots (127/143) without any commit-time or completion-time blocking enforcement.

---

### Finding 5: Fix-density has the highest signal-to-noise ratio but no dedicated enforcement

**Evidence:** Fix-density (28 hotspots) identifies files with many bug-fix commits in the past 28 days. Top entries:
- `pkg/orch/extraction.go` [22 fixes]
- `cmd/orch/spawn_cmd.go` [21 fixes]
- `pkg/daemon/daemon.go` [15 fixes]

These are the files most likely to introduce new bugs when modified, yet the only blocking enforcement they receive is:
- Daemon escalation (Layer 2) — only for daemon-driven spawns
- No blocking for manual spawns

**Source:** `orch hotspot --json` output

**Significance:** Fix-density is arguably the most actionable hotspot type for preventing defects. A file with 22 fixes in 28 days is a clear signal of instability, yet a manual `orch spawn --bypass-triage feature-impl` targeting that file faces zero blocking gates.

---

## Coverage Matrix

| Hotspot Type | Count | L0: Pre-commit | L1: Spawn Gate (Block) | L1: Spawn Warning | L2: Daemon Escalation | L3: Spawn Context | L3: Completion Advisory | L3: Completion Accretion |
|---|---|---|---|---|---|---|---|---|
| **bloat-size** | 16 | ✅ Blocks >1500 | ✅ Blocks >1500 | ✅ Warning | ✅ → architect | ✅ Injected | ✅ Advisory | ✅ Blocks >1500 |
| **fix-density** | 28 | ❌ | ❌ | ✅ Warning | ✅ → architect | ✅ Injected | ✅ Advisory | ❌ |
| **coupling-cluster** | 14 | ❌ | ❌ | ✅ Warning | ✅ → architect | ✅ Injected | ✅ Advisory | ❌ |
| **investigation-cluster** | 85 | ❌ | ❌ | ✅ Warning | ✅ → architect | ✅ Injected | ✅ Advisory | ❌ |

**Legend:**
- ✅ = Active enforcement for this type
- ❌ = Not covered
- "→ architect" = Routes to architect skill instead of feature-impl

### Gap Summary

| Gap | Description | Impact |
|---|---|---|
| **Layer 1 dormant** | 0 files exceed 1500-line CRITICAL threshold | Layer 1 blocking provides zero active enforcement today |
| **Manual spawn gap for non-bloat types** | fix-density/coupling/investigation get no blocking for `orch spawn --bypass-triage` | 89% of hotspots (127/143) have advisory-only enforcement for manual spawns |
| **Completion gap for non-bloat types** | Accretion gates only check file size | A fix-density file with 22 fixes in 28 days can be modified freely at commit time |
| **Layer 0 undocumented** | Pre-commit accretion gate not in three-layer decision | Enforcement surface is actually 4 layers, not 3 |

---

## Synthesis

**Key Insights:**

1. **The enforcement system is bloat-size-centric** — All blocking gates (spawn, precommit, completion) trigger on file line count. The other 3 hotspot types were added to detection but never got blocking enforcement pathways (except through daemon escalation, which only covers daemon-driven spawns).

2. **The daemon is the only comprehensive enforcer** — Layer 2 daemon escalation is the only layer that blocks all 4 hotspot types. This means the preferred workflow (daemon-driven spawns) has full coverage, but manual spawns (which bypass the daemon) have a 3-type enforcement gap.

3. **Layer 1 blocking is aspirational, not active** — With 0 CRITICAL files, the spawn gate's blocking behavior has never triggered in the current codebase. The ongoing extraction work has kept all files below 1500 lines, making Layer 1 a safety net for future growth rather than current enforcement.

**Answer to Investigation Question:**

The three enforcement layers target hotspot types asymmetrically:
- **Blocking enforcement (spawn gate, accretion gates):** bloat-size only
- **Blocking enforcement (daemon escalation):** all 4 types, but daemon-driven spawns only
- **Advisory enforcement (spawn context, completion advisory):** all 4 types, but non-blocking

The primary coverage gap is: **manual spawns targeting fix-density, coupling-cluster, or investigation-cluster hotspots face zero blocking gates.** This affects 89% of detected hotspots.

---

## Structured Uncertainty

**What's tested:**

- ✅ Layer 1 blocking condition: confirmed `bloat-size > 1500` is the only blocking trigger (code review of `hotspot_spawn.go:182`)
- ✅ Layer 2 coverage: confirmed `FindMatchingHotspot` matches all types (code review of `architect_escalation.go:43-58`)
- ✅ Current CRITICAL count: confirmed 0 files >1500 lines via `orch hotspot --json`
- ✅ Accretion gates: confirmed pure line-count checks (code review of `accretion.go` and `accretion_precommit.go`)

**What's untested:**

- ⚠️ Whether the manual spawn gap has caused actual defects (would need to correlate fix-density hotspot files with agent-caused regressions)
- ⚠️ Whether daemon escalation actually triggers in practice (would need daemon event log analysis)
- ⚠️ Whether investigation-cluster topics match correctly in spawn hotspot scanning (topic matching uses `strings.Contains` which could false-positive)

**What would change this:**

- If fix-density hotspot files don't actually produce more defects when modified, the gap is theoretical
- If manual spawns are rare (most spawns daemon-driven), the gap affects few sessions
- If a file grows past 1500 lines, Layer 1 blocking would activate for that file

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add fix-density blocking to spawn gate | architectural | Cross-component change (spawn gates + daemon + testing), multiple valid threshold choices |
| Document Layer 0 in three-layer decision | implementation | Documentation update, no code change |
| Evaluate whether manual spawn gap causes real defects | architectural | Requires event/defect correlation across sessions |

### Recommended Approach ⭐

**Extend Layer 1 spawn gate to block on fix-density score threshold** — Add a secondary blocking condition where files with fix-density score ≥15 (top ~10% of fix-density hotspots) trigger the same architect-gated flow as bloat-size CRITICAL.

**Why this approach:**
- Fix-density is the most predictive hotspot type for future defects (files that have had many recent fixes are likely to need more)
- Reuses existing architect-gated override infrastructure (`--force-hotspot --architect-ref`)
- Addresses the highest-impact gap (28 hotspots with no blocking)

**Trade-offs accepted:**
- Adds friction for manual spawns targeting high-fix-density files
- Threshold choice (≥15) is somewhat arbitrary — needs calibration
- Coupling-cluster and investigation-cluster remain advisory-only (lower signal-to-noise)

**Implementation sequence:**
1. Update `cmd/orch/hotspot_spawn.go:checkSpawnHotspots` to set `HasCriticalHotspot` for fix-density ≥ threshold
2. Add `CriticalFiles` entries for fix-density matches
3. Update warning formatting to distinguish bloat-size vs fix-density blocking
4. Add tests in `cmd/orch/hotspot_spawn_test.go`

### Alternative Approaches Considered

**Option B: Block all 4 hotspot types in spawn gate**
- **Pros:** Complete coverage, no gaps
- **Cons:** Investigation-cluster has 85 entries with high false-positive rate — would block too many spawns
- **When to use instead:** If investigation-cluster matching accuracy improves

**Option C: Keep current system, rely on daemon enforcement**
- **Pros:** Zero implementation work, current system is functional for preferred workflow
- **Cons:** Manual spawns remain ungated for 89% of hotspots
- **When to use instead:** If manual spawns are confirmed rare and defect correlation shows no pattern

---

## References

**Files Examined:**
- `pkg/spawn/gates/hotspot.go` — Layer 1 spawn gate logic
- `pkg/daemon/architect_escalation.go` — Layer 2 daemon escalation logic
- `pkg/daemon/hotspot.go` — Daemon hotspot types and formatting
- `pkg/daemon/hotspot_checker.go` — GitHotspotChecker implementation
- `cmd/orch/hotspot_spawn.go` — Spawn-time hotspot matching (all 4 types)
- `pkg/spawn/context.go` — Layer 3 spawn context template (HOTSPOT AREA WARNING section)
- `cmd/orch/complete_hotspot.go` — Completion advisory matching (all 4 types)
- `pkg/verify/accretion.go` — Completion accretion gate (line count only)
- `pkg/verify/accretion_precommit.go` — Pre-commit accretion gate (line count only)
- `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md` — Original design decision

**Commands Run:**
```bash
# Current hotspot distribution
orch hotspot --json | python3 -c "import json, sys; ..."

# Largest Go files
find . -name "*.go" -not -path "./vendor/*" | xargs wc -l | sort -rn | head -20
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md` — Original three-layer design
- **Investigation:** `.kb/investigations/2026-02-24-design-architect-gate-hotspot-enforcement.md` — Source design investigation
