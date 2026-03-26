# Session Synthesis

**Agent:** og-inv-inventory-current-system-26mar-8333
**Issue:** orch-go-wgkj4
**Duration:** 2026-03-26
**Outcome:** success

---

## TLDR

Classified all 55 packages, 100+ cmd files, 5 web routes, and the skills directory into core/substrate/adjacent. The codebase is ~16% core (thread, comprehension, knowledge), ~72% substrate (daemon, spawn, verify, backends), and ~2% adjacent (research, benchmarks). Three packages (daemon 39K, spawn 30K, verify 23K) contain 48% of all package code and represent the heaviest substrate investment. The web UI's default landing page is execution-centric, directly contradicting the product boundary.

## Plain-Language Summary

This investigation counted every line of Go in the project and asked: which pieces are the product we want to be, which pieces are the plumbing that makes it run, and which pieces are side projects? The answer is stark — for every line of code that makes orch-go a comprehension and knowledge system, there are five lines of execution plumbing. The thread package (the conceptual spine) is 2,300 lines. The daemon (autonomous task runner) is 39,000 lines. That ratio isn't wrong per se — plumbing is necessary — but it explains why the project still feels like an orchestration tool. The most actionable finding is about the web UI: the default page a user sees is an agent execution dashboard, not a comprehension surface. Fixing that is the most visible way to make the product match the decision.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-26-inv-inventory-current-system-into-core.md` - Complete subsystem classification with line counts, rationale, and boundary cases

### Files Modified
- None

### Commits
- (pending)

---

## Evidence (What Was Observed)

- Total Go codebase: ~283K lines (cmd/orch: 90K, pkg/: 193K)
- Core packages total: ~22.6K lines (12% of pkg/)
- Substrate packages total: ~163K lines (84% of pkg/)
- Adjacent packages total: ~4.2K lines (2% of pkg/)
- Three largest packages: daemon (39K), spawn (30K), verify (23K) = 48% of pkg/
- Core cmd files: ~25K lines; Substrate cmd files: ~39K lines
- Web UI: 5 routes; 2 core-aligned (briefs, knowledge-tree), 2 substrate (root, work-graph), 1 adjacent (harness)
- Default web page (+page.svelte) is 32KB execution dashboard
- pkg/verify/ straddles boundary: ~6.6K lines are core-aligned (knowledge quality), ~16.5K are substrate (completion mechanics)
- Skills directory: 210 files, ~58K lines of markdown

### Tests Run
```bash
# Line count analysis across entire codebase
wc -l cmd/orch/*.go  # 90,232 total
find pkg/* -name '*.go' -exec wc -l  # per-package counts
ls web/src/routes/*/  # UI surface inventory
```

---

## Architectural Choices

### Classifying verify as split rather than pure substrate
- **What I chose:** Acknowledge verify's dual identity — 6.6K lines serve knowledge quality (core), 16.5K lines serve completion mechanics (substrate)
- **What I rejected:** Classifying all of verify as substrate
- **Why:** Probe-model-merge, consequence sensors, and confidence gates directly improve "trust in understanding" which is the core deletion criterion. Calling them substrate would undervalue them.
- **Risk accepted:** The split makes the numbers messier but more honest

### Classifying orient and attention as "substrate with core role"
- **What I chose:** Keep them in substrate but flag them as bridge packages
- **What I rejected:** Moving them entirely to core
- **Why:** Their infrastructure is substrate (OODA, work graph), but they surface information that feeds comprehension (thread state, knowledge decay). The bridge classification captures this.
- **Risk accepted:** Bridge packages may get under-invested if treated as "just substrate"

---

## Knowledge (What Was Learned)

### Decisions Made
- The verify package should be acknowledged as straddling core and substrate
- Orient, attention, and parts of events are bridge packages connecting layers
- The skills content (58K lines markdown) deserves its own classification pass

### Constraints Discovered
- The web UI default page directly contradicts the product decision — most impactful Phase 3 target
- Three packages (daemon, spawn, verify) contain 48% of all package code — any refactoring there has outsized impact

---

## Verification Contract

See `.kb/investigations/2026-03-26-inv-inventory-current-system-into-core.md` for the complete inventory with line counts, classification rationale, and boundary cases.

Key verification: The investigation includes a "Structured Uncertainty" section acknowledging what was measured vs. hypothesized, and "What would change this" falsifiability criteria.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete — subsystem map with classification, line counts, rationale
- [x] Investigation file has Phase: Complete
- [x] Ready for `orch complete orch-go-wgkj4`

---

## Unexplored Questions

- How much of the daemon's 39K lines is actively needed vs accumulated? A deletion audit would be high-impact.
- Should skills content (58K lines markdown) be classified separately? It defines agent behavior but isn't Go code.
- What's the actual dependency flow between verify's core-aligned and substrate parts? They may be more coupled than the line-count split suggests.

---

## Friction

No friction — smooth session. Data was straightforward to collect and classify.

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-inventory-current-system-26mar-8333/`
**Investigation:** `.kb/investigations/2026-03-26-inv-inventory-current-system-into-core.md`
**Beads:** `bd show orch-go-wgkj4`
