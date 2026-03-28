# Model: Architectural Defect Intelligence

**Domain:** Predictive codebase health measurement in orch-go
**Last Updated:** 2026-03-18
**Synthesized From:** Defect class taxonomy (7 classes, 459 fix commits), coupling hotspot analysis (2,733 commits), three-layer hotspot enforcement decision, defect class pipeline activation design, 280+ investigations

---

## Summary (30 seconds)

Orch-go has four independent measurement systems for codebase health — spatial (where problems concentrate), typological (what kind of problems recur), causal (why problems cascade), and predictive (where the next problem will appear). Each existed independently. The insight is that intersecting them transforms the system from reactive (fix bugs after they appear) to predictive (tell you what class of bug you'll hit before you touch the code). The spatial layer tells you daemon is a hotspot. The typological layer tells you it breeds duplicate-action and multi-backend-blindness bugs. The causal layer tells you it's because daemon spans 25 files across 3 architectural layers. Together: "before touching daemon code, expect Class 2 or Class 6 bugs originating from cross-layer coupling."

---

## The Four Measurement Layers

### Layer 1: Spatial — WHERE Problems Concentrate

**Source:** `orch hotspot` command (4 detection types)

| Hotspot Type | What It Measures | Threshold | Example |
|---|---|---|---|
| Fix-density | Files touched by N+ fix commits in 90 days | 15+ commits | spawn_cmd.go (42 fix commits) |
| Bloat-size | File line count | >800 yellow, >1500 CRITICAL | extraction.go (1,632 lines) |
| Investigation-cluster | Topic clusters with N+ investigations in 30 days | 5+ investigations | agent-status (7 investigations) |
| Coupling-cluster | Concepts spanning N+ architectural layers with high co-change frequency | Score >=40 | daemon (score 180, CRITICAL) |

**Key data point:** Only 6% of commits (76/1,212) are cross-surface (touching 2+ layers). But these 6% contain the highest-severity bugs because they cross architectural boundaries that individual developers don't hold in their heads.

**What spatial alone can't tell you:** A hotspot file is hot — but why? Is it accumulating stale state? Suffering from contradictory authority signals? Missing backend awareness? Spatial is necessary but not sufficient.

### Layer 2: Typological — WHAT KIND of Problems Recur

**Source:** Defect class taxonomy model (`.kb/models/defect-class-taxonomy/model.md`)

7 named structural failure patterns with 100+ total instances across 459 fix commits:

| # | Class | Instances | Signature |
|---|---|---|---|
| 0 | Scope Expansion | 8+ | Scanner widens, consumer assumptions break |
| 1 | Filter Amnesia | 15+ | Filter in path A, missing in path B |
| 2 | Multi-Backend Blindness | 15+ | Works for one backend, silent failure on other |
| 3 | Stale Artifact Accumulation | 20+ | Dead state never cleaned up |
| 4 | Cross-Project Boundary Bleed | 20+ | Single-project code in multi-project context |
| 5 | Contradictory Authority Signals | 10+ | Multiple truth sources disagree, fixes oscillate |
| 6 | Duplicate Action Without Idempotency | 12+ | Same action repeated, no dedup |
| 7 | Premature/Wrong-Target Destruction | 8+ | Resource killed on stale info |

**What typological alone can't tell you:** You know what kinds of bugs exist system-wide, but not where they'll concentrate next. A new feature touching a new file has equal risk across all classes without spatial context.

### Layer 3: Causal — WHY Problems Cascade

**Source:** Dependency graph from defect class taxonomy + coupling hotspot co-change data

Two types of causal structure:

**Between defect classes (typological causation):**
```
Stale Artifact Accumulation (3)
    | creates data that
Scope Expansion (0) finds unexpectedly
    | manifests via
Filter Amnesia (1) — missing exclusion in new consumer

Cross-Project Boundary Bleed (4)
    | multiplies exposure for
Scope Expansion (0) — cross-project data is highest-risk expansion vector

Contradictory Authority Signals (5)
    | causes
Premature/Wrong-Target Destruction (7) — wrong status -> wrong action

Duplicate Action (6) — independent, caused by missing idempotency
```

**Between architectural layers (spatial causation):**
Coupling clusters quantify why certain areas breed certain classes. daemon (coupling score 180) spans cmd/, pkg/, and web/ — any change risks multi-backend blindness (Class 2) because the concept crosses the backend boundary, and duplicate action (Class 6) because dedup logic must be consistent across all three layers.

**What causal structure enables:** Priority ordering grounded in cascading impact. Fixing stale artifact accumulation (upstream) reduces scope expansion and filter amnesia (downstream). Fixing contradictory authority (upstream) prevents premature destruction (downstream). Without causal structure, you'd prioritize by instance count, which is a weaker signal.

### Layer 4: Predictive — WHERE the NEXT Problem Will Appear

**Source:** Intersection of spatial, typological, and causal layers

Partial tooling exists: `DefectClassesForHotspots()` in `cmd/orch/hotspot_spawn.go` computes spatial x typological intersection at spawn time using keyword-based file-to-class mapping. This is a heuristic approximation, not evidence-driven prediction. The model's core claim about what becomes possible with evidence-based intersection remains valid but the gap is smaller than originally stated.

**The intersection logic:**

```
For a given code change targeting file F in area A:
  1. Spatial:     Is A a hotspot? What type? (fix-density, coupling, bloat)
  2. Typological: Which defect classes have historically appeared in A?
  3. Causal:      Which upstream classes are active in A right now?
  4. Prediction:  "Expect Class X bugs because A has property Y"
```

**Validated intersections (evidence exists):**

| Spatial Signal | Typological Signal | Why They Intersect | Evidence |
|---|---|---|---|
| daemon coupling (score 180, 3 layers) | Class 6: duplicate spawn (7 fixes) | Dedup must span all 3 layers; each layer has its own gap | 7 sequential dedup commits, each fixing a different layer's gap |
| agent-status coupling (score 67, 3 layers) | Class 5: contradictory authority (41 commits) | 14 files across 3 layers each deriving status independently | Fix oscillation: 4 contradictory fixes in 2 weeks |
| spawn coupling (score 44, 2 layers) | Class 2: multi-backend blindness (42 commits) | Spawn straddles the OpenCode/Claude CLI boundary | 15+ fixes post-backend-transition (Feb 19) |
| bloat hotspot (extraction.go, 1632 lines) | Class 1: filter amnesia | Large files accumulate consumers; each new consumer is a filter amnesia opportunity | Extraction plan designed specifically to reduce consumer count |

**Unvalidated but predicted:**
- Session coupling (score 44, 4 layers) x Class 4 (cross-project boundary bleed): session concepts span the most layers but Class 4 instances haven't been mapped to session specifically. Prediction: session management in multi-project context will produce boundary bleed bugs.

---

## How the System Acts on Signals

### Current Enforcement (Implemented)

| Mechanism | Layer Used | Action | Where |
|---|---|---|---|
| Spawn gate (hotspot) | Spatial (bloat) | Advisory warning + event emission for files >1500 lines (was blocking, converted to advisory 2026-03-17 after 100% bypass rate) | Layer 1 of three-layer enforcement |
| Daemon escalation | Spatial (bloat) | Route feature-impl -> architect for hotspot files | Layer 2 of three-layer enforcement |
| Spawn context injection | Spatial (all types) | Hotspot info in SPAWN_CONTEXT.md | Layer 3 of three-layer enforcement |
| Architect skill context | Typological | Defect class names available during review | Vocabulary injection |
| `orch doctor --defect-scan` | Typological | Detects Class 2 (multi-backend blindness) and Class 5 (contradictory authority) anti-patterns | `cmd/orch/doctor_defect_scan.go` |
| Defect class injection at spawn | Spatial + typological | `DefectClassesForHotspots()` maps hotspot files to likely defect classes in spawn warnings | `cmd/orch/hotspot_spawn.go:269` |

### Designed but Not Implemented

| Mechanism | Layers Used | Action | Source |
|---|---|---|---|
| Defect class pipeline | Typological + spatial | Auto-create architect issues when 3+ investigations share a class | Pipeline activation design |
| Full predictive spawn warning | All 4 layers | Evidence-driven (not keyword-based) prediction: "This area breeds Class X bugs because Y" | This model |

### The Gap

Current enforcement uses spatial signals plus partial typological automation. `DefectClassesForHotspots()` computes spatial x typological intersection at spawn time, but uses keyword matching (file path → defect class), not evidence-driven correlation from actual bug history. The defect scan detects 2 of 7 classes. The remaining gap is: (1) evidence-based class-to-area mapping instead of keyword heuristics, (2) causal layer validation through intervention experiments, (3) coverage of all 7 classes in automated detection.

---

## Why This Fails

### Overfitting to named classes

The 7 classes cover ~100+ of 459 fix commits. The remaining commits may contain unnamed classes, one-off bugs, or bugs that span classes. Forcing every bug into a named class creates anchoring bias — you see what you named and miss what you didn't.

**Mitigation:** The taxonomy is explicitly a living artifact. Track bugs that don't fit any class. If a cluster of misfits reaches 5+, investigate for a new class.

### Migration classes that stabilize

Multi-backend blindness (Class 2) spiked after the Feb 19 backend transition. If no third backend is added, this class may naturally reach zero instances as the codebase matures. Investing heavily in backend abstraction infrastructure for a stabilizing class is waste.

**Mitigation:** Monitor instance rate before investing in structural fixes. If Class 2 produces <1 instance per month for 3 months, downgrade from active to historical.

### Prediction without action

Telling a developer "expect Class 6 bugs" is only useful if they know what to do about it. Prediction without actionable prevention guidance is noise.

**Mitigation:** Each class has a named structural prevention pattern. Predictions should include both the class and the prevention: "Expect duplicate action — use idempotency keys for any new spawn path."

### Coupling scores as proxy, not truth

Coupling scores are computed from git co-change frequency, which is a proxy for architectural coupling. High co-change could also mean: coordinated refactoring (not real coupling), test files changing alongside source (expected), or documentation updates (irrelevant). The score needs calibration against actual bug rates.

**Mitigation:** Cross-validate coupling scores against defect class instance density. If high-coupling areas don't produce more defect class instances, the score formula needs adjustment.

---

## Constraints

### Why four layers and not fewer?

**Constraint:** Each layer answers a question the others can't.

- Spatial without typological: "daemon is hot" but you don't know what kind of hot
- Typological without spatial: "Class 6 exists" but you don't know where it'll strike next
- Both without causal: you see correlation (daemon + Class 6) but not why, so you can't prioritize fixes by cascading impact
- All three without predictive: you explain the past but don't guide the future

Dropping any layer loses a dimension of actionability.

### Why is the predictive layer not implemented yet?

**Constraint:** The predictive layer requires the other three to be reliable. Spatial (hotspot detection) is mature. Typological (defect classes) was just created. Causal (dependency graph) is a hypothesis with evidence but not validated through intervention. Building prediction on unvalidated causation risks false confidence.

**Implication:** Validate causation first — does fixing an upstream class actually reduce downstream instances? — before automating prediction.

### Why not just use generic code quality tools?

**Constraint:** Generic tools (SonarQube, CodeClimate) measure generic signals (complexity, duplication, coverage). They can't detect orch-go-specific classes like multi-backend blindness or cross-project boundary bleed because those classes are domain-specific.

**Implication:** This system complements, not replaces, generic quality tools. The defect classes capture what's unique to this codebase's architecture and history.

---

## Evolution

**2026-03-03:** Model created. Synthesized from defect class taxonomy (7 classes, 459 fix commits), coupling hotspot analysis (2,733 commits, 4 calibrated hotspots), three-layer hotspot enforcement decision, and defect class pipeline activation design. The four-layer framework (spatial, typological, causal, predictive) is proposed. Layers 1-2 are implemented, layer 3 is evidenced but unvalidated through intervention, layer 4 exists only as this model.

**Open questions:**
- Does fixing an upstream defect class (stale artifacts) actually reduce downstream class instances (scope expansion)? This validates the causal layer.
- What's the false positive rate for coupling hotspot detection? Do high-coupling areas that are NOT hotspots exist?
- Can the predictive intersection be automated in `orch spawn` warnings, or is it only useful as human-consumed context in architect reviews?
- Are these four layers sufficient, or is there a fifth (e.g., temporal — when in the development cycle do certain classes spike)?

**Probes:**
- 2026-03-18: Decay verification — 3 stale claims corrected: spawn gates now advisory (not blocking), defect-scan implemented, partial predictive tooling exists via keyword-based class injection

**Probes directory:** `probes/` — future probes should test: causal direction validation, coupling score calibration, keyword-vs-evidence prediction accuracy.

---

## References

**Models (components of this system):**
- `.kb/models/defect-class-taxonomy/model.md` — Layer 2: the 7 named classes with dependency graph
- `.kb/models/daemon-autonomous-operation/model.md` — Daemon is the highest-coupling hotspot (score 180)
- `.kb/models/completion-verification/model.md` — Verification gates interact with Classes 1, 5, 7

**Investigations (evidence base):**
- `.kb/investigations/2026-03-03-inv-catalogue-unnamed-defect-classes-orch.md` — Source data: 459 commits classified into 7 classes
- `.kb/investigations/2026-02-19-design-coupling-hotspot-analysis-system.md` — Coupling hotspot algorithm and calibrated scores
- `.kb/investigations/2026-02-26-design-defect-class-pipeline-activation.md` — Design for automated class detection in daemon

**Decisions (enforcement architecture):**
- `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md` — How spatial signals gate spawns
- `.kb/decisions/2026-02-18-two-lane-agent-discovery.md` — Domain boundaries that expose Class 4
- `.kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md` — Gates converted from blocking to advisory after 100% bypass rate

## Auto-Linked Investigations

- .kb/investigations/2026-03-27-design-automated-synthesis-ranking.md
