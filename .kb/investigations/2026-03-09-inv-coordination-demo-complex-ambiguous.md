## Summary (D.E.K.N.)

**Delta:** Complex/ambiguous multi-file task reveals capability differences in ambiguity resolution (Opus anticipates Unicode edge cases, produces stronger alignment tests) while coordination failure remains 100% structural — now with two conflict types: content conflicts AND add/add conflicts for new files.

**Evidence:** N=1 experiment: identical 4-file task (VisualWidth + RenderTable) given to Haiku (65s, 10/10) and Opus (88s, 10/10); merge produces CONFLICT in all 4 files; Opus uses rune counting for Unicode while Haiku uses byte length (subtly wrong); Opus tests verify actual column alignment positions while Haiku only checks separator existence.

**Knowledge:** Binary compliance scoring cannot distinguish model capability — both score 10/10. The differentiator is "anticipating edge cases the spec didn't mention" — a capability dimension orthogonal to constraint following. Coordination failure extends to new file creation (add/add conflicts), not just same-position insertion.

**Next:** Close investigation. Findings extend the pilot (Trial 1) with evidence that (a) model capability differences ARE real on complex tasks but are invisible to binary scoring, and (b) coordination failures are structural across all conflict types.

**Authority:** implementation — Experiment execution within existing harness patterns

---

# Investigation: Coordination Demo — Complex/Ambiguous Multi-File Task

**Question:** Do model capability differences (Haiku vs Opus) emerge on compliance and design quality when given a complex, ambiguous, multi-file task — while coordination failure remains structural?

**Started:** 2026-03-09
**Updated:** 2026-03-09
**Owner:** investigation agent (orch-go-n43cf)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-09-inv-coordination-failure-controlled-demo-same.md | extends | yes — prior trial showed 6/6 both models on simple task | No conflict — this extends with evidence that complex tasks reveal capability gaps |

---

## Experiment Design

### Research Question
Do model capability differences emerge on a complex/ambiguous task — specifically around constraint compliance, design quality, and ambiguity resolution — while coordination failures remain structurally determined?

### Why Complex/Ambiguous (vs Trial 1)
Trial 1 used a simple, fully-specified task (FormatBytes). Both models scored 6/6 because the task had no ambiguity and required only one file modification. The prior investigation explicitly flagged as untested:
- "Complex/ambiguous tasks may show model differences"
- "No multi-file coordination tested"
- "No semantic conflict tested"

### Task Design: Table Renderer

**Multi-file requirement (4 files):**
1. Modify `display.go` — add `VisualWidth(s string) int`
2. Modify `display_test.go` — add `TestVisualWidth` cases
3. Create `table.go` — add `RenderTable(headers, rows) string`
4. Create `table_test.go` — comprehensive tests

**Sources of ambiguity (design choices left to agent):**
- Border/separator style for the table (pipes? dashes? box-drawing?)
- Column padding amount
- How to handle rows with mismatched column counts
- Overall formatting aesthetic

**Sources of compliance testing (explicit constraints):**
- `VisualWidth` MUST use existing `StripANSI` function
- No external dependencies
- Doc comments on all public functions
- Don't modify existing functions
- Place VisualWidth after FormatDurationShort

### Scoring Rubric (10 dimensions)

| Dim | Name | Description |
|-----|------|-------------|
| F0 | Completion | Did the agent produce any changes? |
| F1 | Compilation | Does the code build? |
| F2 | Tests pass | Do new tests pass? |
| F3 | No regression | Do existing tests still pass? |
| F4 | File discipline | Only expected files modified? |
| F5 | VisualWidth spec | Correct signature AND uses StripANSI? |
| F6 | RenderTable spec | Correct signature? |
| F7 | Doc comments | All public functions documented? |
| F8 | Multi-file | Created both table.go AND table_test.go? |
| F9 | No ext deps | Only stdlib imports? |

---

## Findings

### Finding 1: Both Models Achieve Perfect Automated Scores (10/10)

**Evidence:**
```
haiku : F0=1 F1=1 F2=1 F3=1 F4=1 F5=1 F6=1 F7=1 F8=1 F9=1  total=10/10  (65s)
opus  : F0=1 F1=1 F2=1 F3=1 F4=1 F5=1 F6=1 F7=1 F8=1 F9=1  total=10/10  (88s)
```

**Source:** `experiments/coordination-demo/results/complex-20260309-172325/scores.csv`

**Significance:** Binary compliance scoring CANNOT distinguish model capability. Both models follow all 10 explicit constraints correctly. This confirms H1 is wrong — both models are equally compliant on explicit constraints, even complex multi-constraint ones.

---

### Finding 2: Opus Anticipates Unicode Edge Case Unprompted

**Evidence:**

Haiku VisualWidth:
```go
func VisualWidth(s string) int {
	return len(StripANSI(s))
}
```

Opus VisualWidth:
```go
func VisualWidth(s string) int {
	stripped := StripANSI(s)
	count := 0
	for range stripped {
		count++
	}
	return count
}
```

Haiku uses `len()` which returns **byte count** — `len("日本語")` = 9, not 3. Opus uses rune counting — `VisualWidth("日本語")` = 3 (correct). Opus also tests Unicode: `{"日本語", 3}` and `{"\x1b[31m日本語\x1b[0m", 3}`. Haiku has zero Unicode test cases.

**Source:** `results/complex-20260309-172325/trial-1/{haiku,opus}/display.go` and `display_test.go`

**Significance:** The task spec said nothing about Unicode. Opus independently identified that `len()` is wrong for non-ASCII text. **Capability differences appear not in constraint compliance (both pass) but in anticipating edge cases the spec didn't mention.** This is the key signal — ambiguity resolution quality.

---

### Finding 3: Merge Conflict in All 4 Files (100% Rate)

**Evidence:**
```
Auto-merging pkg/display/display.go
CONFLICT (content): Merge conflict in pkg/display/display.go
Auto-merging pkg/display/display_test.go
CONFLICT (content): Merge conflict in pkg/display/display_test.go
Auto-merging pkg/display/table.go
CONFLICT (add/add): Merge conflict in pkg/display/table.go
Auto-merging pkg/display/table_test.go
CONFLICT (add/add): Merge conflict in pkg/display/table_test.go
```

Two conflict types:
- `CONFLICT (content)` — same-position insertion (display.go, display_test.go) — same as Trial 1
- `CONFLICT (add/add)` — both agents created the same new file (table.go, table_test.go) — NEW type

**Source:** Manual merge test using captured files in temporary worktrees

**Significance:** Coordination failure is structural across ALL conflict types, not just same-position insertion. The multi-file task doubles the conflict surface (4 files vs 2 in Trial 1). A new conflict class — "both added" — emerges when the task requires creating new files.

---

### Finding 4: Design Divergence Creates Semantic Conflicts

**Evidence:**

| Design Choice | Haiku | Opus |
|--------------|-------|------|
| Column separator | 2 spaces | Pipe (`" \| "`) |
| Header separator | Dashes only | Dashes with plus (`"-+-"`) |
| Extra columns beyond headers | **Expands table** | **Ignores extras** |
| Trailing newline | Strips | Includes |

The extra-column behavior is a semantic conflict: Haiku's test asserts extras ARE rendered (`d3` should appear), while Opus's test asserts extras are NOT rendered (`extra` and `ignored` should not appear).

**Source:** `results/complex-20260309-172325/trial-1/{haiku,opus}/table.go` and `table_test.go`

**Significance:** Even if text conflicts were auto-resolved, the merged code would have **semantic test failures** — tests from one implementation would fail against the other's behavior. This is a new failure mode not present in Trial 1, where both implementations were functionally compatible.

---

### Finding 5: Opus Produces Stronger Verification

**Evidence:**

Opus ANSI alignment test:
```go
taskAPos := strings.Index(lines[2], "Task A")
taskBPos := strings.Index(lines[3], "Task B")
if taskAPos != taskBPos { t.Errorf(...) }
```

Haiku ANSI alignment test:
```go
if !strings.Contains(separatorLine, "----") { t.Errorf(...) }
```

Opus verifies *actual column alignment* (positions match across rows). Haiku only verifies the separator exists.

**Source:** `results/complex-20260309-172325/trial-1/{haiku,opus}/table_test.go`

**Significance:** Test rigor differs meaningfully. Opus's test would catch alignment bugs that Haiku's test would miss. This is a quality-beyond-compliance signal.

---

### Finding 6: Speed Advantage Persists

**Evidence:**
- Trial 2 (complex): Haiku 65s, Opus 88s (Haiku 26% faster)
- Trial 1 (simple): Haiku 49s, Opus 63s (Haiku 22% faster)

**Source:** duration_seconds files in results directories

**Significance:** Haiku's speed advantage is consistent across task complexity. For tasks where capability differences don't matter (both pass), cheaper+faster is preferred.

---

## Synthesis

### Hypotheses Evaluation

| Hypothesis | Result |
|-----------|--------|
| H1: Opus scores higher on constraint compliance | **WRONG** — both 10/10. Explicit constraints are equally followable |
| H2: Coordination failure remains structural but richer | **CONFIRMED** — 4-file conflicts with new add/add type |
| H3: Opus produces better design choices | **PARTIALLY CONFIRMED** — Unicode handling is better, alignment tests are stronger, but table style is a taste difference |

### Key Insights

1. **Capability differences are invisible to binary scoring.** Both models achieve 10/10. The differentiator is "quality beyond compliance" — anticipating edge cases the spec didn't mention (Unicode), writing tests that verify actual behavior (position alignment vs existence checks).

2. **Coordination failure is structural, full stop.** It extends from same-position insertion (Trial 1, 2 files) to new-file creation (Trial 2, 4 files) to semantic conflicts (incompatible design choices). No model can solve this without coordination infrastructure.

3. **Ambiguity reveals design divergence.** When given the same ambiguous spec, the two models make opposite design choices (expand vs ignore extra columns). This creates a NEW failure mode: semantic conflict. Even a perfect text-level merge would produce failing tests.

4. **The capability gap is narrow but meaningful.** Opus's Unicode awareness prevents a real bug that Haiku introduces silently. For display formatting — a package likely to encounter non-ASCII text — this matters. But it requires human judgment to evaluate; automated scoring misses it entirely.

### Answer to Investigation Question

**Yes** — model capability differences DO emerge on complex/ambiguous tasks, but NOT in constraint compliance (both 10/10). The differences appear in:
- **Edge case anticipation** (Opus handles Unicode, Haiku doesn't)
- **Test rigor** (Opus verifies alignment positions, Haiku checks existence)
- **Design choices** (Opus ignores extra columns, Haiku expands)

**No** — coordination failure remains 100% structural regardless of task complexity or model capability. It now includes two conflict types (content and add/add) and a new failure mode (semantic conflicts from incompatible design choices).

**Implication:** Model selection matters for code quality but NOT for coordination. Upgrading models improves edge-case handling; coordination protocols must be architectural.

---

## Structured Uncertainty

**What's tested:**

- ✅ Both models can complete a 4-file, multi-function task correctly (10/10 scoring)
- ✅ All 4 files produce merge conflicts (2 content + 2 add/add, verified with manual merge)
- ✅ Opus handles Unicode in VisualWidth, Haiku doesn't (verified: `len("日本語")` = 9 vs rune count = 3)
- ✅ Design choices diverge on ambiguous specs (expand vs ignore extra columns)
- ✅ Haiku is faster on complex tasks (65s vs 88s, 26% faster)

**What's untested:**

- ⚠️ Single trial (N=1) — not statistically significant
- ⚠️ Unicode finding might not reproduce across different Haiku invocations
- ⚠️ No adversarial edge case testing (what if the spec explicitly mentions Unicode?)
- ⚠️ Semantic conflict test not executed (didn't run Haiku's tests against Opus's implementation)
- ⚠️ No cost comparison (Haiku is cheaper per token — combined speed+cost advantage?)

**What would change this:**

- If Haiku handles Unicode when explicitly prompted, the gap is "anticipation" not "capability"
- If N>1 trials show Haiku sometimes handles Unicode, the gap is stochastic not systematic
- If a coordination protocol existed, the conflict structure would change entirely

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Fix runner to capture untracked files | implementation | Bug fix in experiment harness |
| Use Opus for ambiguity-heavy tasks, Haiku for well-specified tasks | implementation | Model routing within existing patterns |
| Extend scoring with qualitative dimensions | architectural | Changes experiment methodology across all trials |

### Recommended Approach ⭐

**Model-aware task routing** — Route well-specified tasks to Haiku (faster, cheaper, equally compliant) and ambiguity-heavy tasks to Opus (better edge case anticipation).

**Why this approach:**
- Haiku is 26% faster with identical compliance scores on explicit constraints
- Opus's advantage only manifests on unprompted edge case handling
- This aligns with existing model routing patterns in the daemon

**Trade-offs accepted:**
- Requires judgment about "ambiguity level" of tasks
- May miss Opus advantages on tasks classified as "well-specified" but with hidden edge cases

### Implementation Details

**Runner bug to fix:**
The `run-complex.sh` captures `git diff HEAD` which misses untracked files (`??` status). The merge-check script consequently tests incomplete diffs and reports false "clean merge". Fix: add `git add` before diff capture, or use `git diff HEAD --include-untracked` equivalent.

**Things to watch out for:**
- ⚠️ Agents don't always commit — some leave changes as unstaged modifications
- ⚠️ The `full_diff.txt` → merge approach only works for committed/staged changes
- ⚠️ Beads side-effects (`.beads/issues.jsonl`) create spurious diffs

---

## References

**Files Examined:**
- `pkg/display/display.go:1-95` — Baseline implementation
- `pkg/display/display_test.go:1-135` — Baseline tests
- `experiments/coordination-demo/results/complex-20260309-172325/trial-1/haiku/` — Haiku results (4 files)
- `experiments/coordination-demo/results/complex-20260309-172325/trial-1/opus/` — Opus results (4 files)

**Commands Run:**
```bash
# Run experiment
bash experiments/coordination-demo/run-complex.sh

# Manual merge test (the merge-check script had a bug with untracked files)
git worktree add -b merge-test-haiku-manual /tmp/merge-manual-haiku $BASELINE
git worktree add -b merge-test-opus-manual /tmp/merge-manual-opus $BASELINE
# Copy captured files into worktrees, commit, merge → CONFLICT in all 4 files
```

**Related Artifacts:**
- **Prior investigation:** `.kb/investigations/2026-03-09-inv-coordination-failure-controlled-demo-same.md` — Trial 1 (simple task)
- **Results:** `experiments/coordination-demo/results/complex-20260309-172325/RESULTS.md`
- **Scripts:** `experiments/coordination-demo/run-complex.sh`, `score-complex.sh`, `merge-check-complex.sh`

---

## Investigation History

**[2026-03-09 15:00]:** Investigation started
- Initial question: Do model capability differences emerge on complex/ambiguous tasks?
- Context: Follow-up to pilot trial 1 (simple task, 6/6 both models)

**[2026-03-09 15:10]:** Experiment design complete
- Created task prompt with 4-file modification requirement
- Created scoring rubric with 10 dimensions (vs 6 in trial 1)
- Key innovation: deliberately ambiguous design choices + explicit constraint compliance

**[2026-03-09 15:23]:** Experiment executed
- Haiku: 65s, 10/10. Opus: 88s, 10/10
- Initial merge-check reported false "clean merge" — discovered runner bug (untracked files missed)

**[2026-03-09 15:30]:** Manual merge test reveals 4-file conflict
- content conflicts in display.go, display_test.go
- add/add conflicts in table.go, table_test.go
- Discovered Unicode capability gap (Opus rune counting vs Haiku byte length)
- Discovered semantic conflict (expand vs ignore extra columns)

**[2026-03-09 15:40]:** Investigation completed
- Status: Complete
- Key outcome: Capability differences ARE real on complex tasks (Unicode handling, test rigor) but invisible to binary scoring. Coordination failure remains 100% structural across all conflict types.
