# Model: Architect

**Created:** 2026-03-09
**Status:** Active
**Source:** Synthesized from 4 investigation(s)

## What This Is

[What phenomenon or pattern does this model describe? What makes it a coherent concept worth naming?]

---

## Core Claims (Testable)

### Claim 1: [Concise claim statement]

[Explanation of the claim. What would you observe if it's true? What would falsify it?]

**Test:** [How to test this claim]

**Status:** Hypothesis

### Claim 2: [Concise claim statement]

[Explanation of the claim.]

**Test:** [How to test this claim]

**Status:** Hypothesis

---

## Implications

[What follows from these claims? How should this model change behavior, design, or decision-making?]

---

## Boundaries

**What this model covers:**
- [Scope item 1]

**What this model does NOT cover:**
- [Exclusion 1]

---

## Evidence

| Date | Source | Finding |
|------|--------|---------|
| 2026-03-09 | Model creation | Initial synthesis from source investigations |

---

## Open Questions

- [Question that further investigation could answer]
- [Question about model boundaries or edge cases]

## Source Investigations

### 2026-02-14-inv-architect-design-accretion-gravity-enforcement.md

**Delta:** Accretion Gravity has detection infrastructure (hotspot analysis) but zero prevention/enforcement - violates "Gate Over Remind" principle.
**Evidence:** spawn_cmd.go (2,332 lines), session.go (2,166 lines), doctor.go (1,912 lines) all flagged as CRITICAL hotspots yet agents still freely add to them; hotspot check at spawn is warning-only (line 834-850); no completion gates block modifications to bloated files.
**Knowledge:** Enforcement requires four layers: (1) Spawn-time gates that block work in hotspot areas without extraction plan, (2) Real-time coaching detection when agents attempt to modify bloated files, (3) Completion verification that rejects PRs adding >50 lines to files >800 lines, (4) Explicit CLAUDE.md boundaries declaring "DO NOT MODIFY" files.
**Next:** Implement four-layer enforcement starting with spawn-time gates (highest ROI - prevents problem before it starts), then completion gates (catches violations before merge), then coaching plugin (real-time correction), finally CLAUDE.md boundaries (declarative prevention).

---

### 2026-02-25-inv-architect-skillc-deploy-silent-failures.md

**Delta:** Consolidated 5 prior probes into a single failure taxonomy. Identified 4 distinct failure modes causing agents to run with stale skills, ranked by frequency and severity. Confirmed the loader can't reach stale `src/` copies, making cleanup a hygiene issue not a correctness issue.
**Evidence:** `skillc deploy` exits 0 on partial failure (no exit code signal). 20 stale SKILL.md files persist in `~/.claude/skills/src/`, plus 22 in `~/.opencode/skill/`. Feature-impl `src/` copy has checksum `047ddb2689b3` (Jan 7) while canonical has `76a3920c0fe9` (Feb 25) — 7 weeks stale.
**Knowledge:** The silent deploy failure is not one bug but a pipeline with 4 independent failure points. Two require skillc code changes, one requires hook fixes, one is operational hygiene.
**Next:** Create 3 issues: (1) skillc exit code fix, (2) hook spawn detection fix, (3) stale copy cleanup.

---

### 2026-03-05-inv-architect-unified-reliability-design-orch.md

**Delta:** The daemon's three structural problems (6-layer dedup gauntlet, 625-line loop, operational unreliability) share a common root: internal complexity makes failure modes invisible and reasoning impossible. The fix is a 3-phase inside-out simplification: (1) collapse dedup to CAS-like gate + advisory checks, (2) extract scheduler from loop, (3) supervision gaps close naturally when the simplified daemon is launchd-managed.
**Evidence:** Code trace of spawnIssue() (daemon.go:672-928, 245 lines, 6 dedup layers); runDaemonLoop (cmd/orch/daemon.go:380-1077, 697 lines, 12 periodic subsystems); beads UpdateArgs has no ExpectedStatus field (no CAS); `orch daemon install` already exists for launchd plist generation; daemonConfigFromFlags() unified config decision (2026-02-15) partially implemented; periodic tasks already extracted to daemon_periodic.go.
**Knowledge:** Beads lacks native CAS but we can simulate CAS semantics in Go: fresh-check + update as a single atomic function (read-then-write behind a local mutex, with the existing fail-fast on update error). The dedup layers are heuristic-first because they predate the beads status update (L6). Inverting to structural-first eliminates 4 of 6 layers from the critical path. The loop extraction is partially done (periodic tasks extracted) but the main loop body still has reconciliation, verification, completion, invariants, circuit breaker, status writing, and spawn loop all inline.
**Next:** Create implementation issues for the 3 phases. Phase 1 (dedup pipeline) is the highest-value change — it reduces spawnIssue from 245 lines to ~60 and makes the dedup invariant explicit and testable.
