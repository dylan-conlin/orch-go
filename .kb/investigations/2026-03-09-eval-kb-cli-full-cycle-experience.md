## Summary (D.E.K.N.)

**Delta:** kb-cli's full investigation→model→probe cycle works but has a critical gap: no `kb create model` command exists, forcing manual directory creation. `--extract-models` flag is broken (removes models from output), and `kb index` shows "no artifacts" in a populated project.

**Evidence:** Exercised all 10 steps of the cycle in ~/tmp/kb-test-project with real Go sync.Pool benchmarks. 7 of 11 commands worked cleanly; 3 had bugs; 1 key command is missing.

**Knowledge:** The investigation→probe→model workflow is powerful when it works — the probe genuinely improved the model by correcting a wrong claim. But the "model" artifact is a second-class citizen: no create command, not in kb init scaffold, not in kb list.

**Next:** Create `kb create model` command. Fix `--extract-models` bug. Fix `kb index` to find artifacts.

**Authority:** architectural - These are cross-component changes affecting CLI surface area and multiple command implementations.

---

# Investigation: kb-cli Full Cycle Experiential Evaluation

**Question:** What is the end-to-end experience of using kb-cli's investigation/probe/model cycle from scratch?

**Started:** 2026-03-09
**Updated:** 2026-03-09
**Owner:** Experiential eval agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| kb-cli-public-release-readiness-audit (probe) | extends | Yes | No — this is a concrete experience report vs. that probe's checklist approach |

---

## Findings

### Finding 1: `kb create model` does not exist

**Evidence:** Running `kb create model go-sync-pool` silently shows the `kb create` help text with no "unknown command" error. The `kb create` subcommands are: decision, guide, investigation, plan, probe, research, specification. Model is absent.

**Source:** `kb create --help` output; `kb create model` output (no error, just help)

**Significance:** Models are central to the kb lifecycle — investigations require `--model`, probes require `--model`. But there's no command to create a model. Users must manually `mkdir -p .kb/models/{name}/probes && cp TEMPLATE.md`. The TEMPLATE.md only exists in orch-go's .kb/, not in freshly-initialized projects.

**Score — Discoverability: 1/5 | Friction: 5/5 (high friction)**

---

### Finding 2: `kb init` doesn't scaffold models directory or template

**Evidence:** After `kb init`, `.kb/` contains only: investigations/, decisions/, guides/. No models/ directory, no TEMPLATE.md. The README.md with model instructions lives only in orch-go's existing .kb/models/.

**Source:** `ls -la ~/tmp/kb-test-project/.kb/` after init

**Significance:** New users have no path from init to model creation without reading orch-go source or documentation. The chicken-and-egg problem (investigation requires --model, but no model exists) is only escaped via `--orphan`.

**Score — Discoverability: 2/5 | Friction: 4/5**

---

### Finding 3: `--extract-models` drops models instead of extracting them

**Evidence:**
- `kb context "sync pool"` → shows Models section with model path
- `kb context "sync pool" --extract-models` → Models section disappears entirely, only shows Investigations

**Source:** Side-by-side comparison of output with and without flag

**Significance:** The flag does the opposite of what its name implies. A user expecting inlined model content gets less information than without the flag.

**Score — Discoverability: 3/5 (flag is documented) | Friction: 5/5 (broken)**

---

### Finding 4: `kb index` returns "No knowledge artifacts found" in populated project

**Evidence:** Project has 2 investigations, 1 model, 1 probe, 5 quick entries. `kb index` says "No knowledge artifacts found."

**Source:** `kb index` in ~/tmp/kb-test-project

**Significance:** The index command is useless in its current state for new projects. May have different criteria for what counts as an "artifact" or may only scan specific subdirectories.

**Score — Discoverability: 4/5 (command exists and is documented) | Friction: 4/5 (misleading output)**

---

### Finding 5: Investigation template is 246 lines — intimidating for new users

**Evidence:** Template has 246 lines with sections for: D.E.K.N. summary, Prior Work table, 3 Finding templates, Synthesis, Structured Uncertainty, Implementation Recommendations (with authority levels), References, and Investigation History.

**Source:** `wc -l` on generated investigation file

**Significance:** For a solo researcher investigating "how does sync.Pool work", the authority levels, recommendation tables, and implementation sequences are ceremony that doesn't fit. The template is designed for organizational use, not solo research. A minimal template option (`kb create investigation --minimal`?) would reduce friction.

**Score — Discoverability: 3/5 | Friction: 3/5**

---

### Finding 6: Quick entries (decide/tried/constrain/question) are excellent

**Evidence:** All 4 quick entry types worked first try. `kb quick decide`, `kb quick tried --failed`, `kb quick constrain`, `kb quick question` all have intuitive syntax. `kb quick resolve` correctly chains question→decision. `kb quick list` shows all entries cleanly.

**Source:** Full cycle through all quick subcommands

**Significance:** Quick entries are the lowest-friction part of the entire system. Capture knowledge in one command. The `resolve` lifecycle (question→decision) is elegant.

**Score — Discoverability: 5/5 | Friction: 1/5 (minimal friction)**

---

### Finding 7: Probe template auto-includes model context — great UX

**Evidence:** `kb create probe crossover-point-4kb --model go-sync-pool` automatically inlined the model's Summary section into the probe template's "Model Context" section.

**Source:** Reading generated probe file

**Significance:** This is exactly right. The probe tests a model claim, so having the model's summary right there is perfect. Zero friction for the most important part of the workflow.

**Score — Discoverability: 4/5 | Friction: 1/5**

---

### Finding 8: `kb context` works well as unified search

**Evidence:** `kb context "pool overhead"` returned all 5 relevant artifacts (1 constraint, 1 decision, 1 attempt, 2 models, 1 investigation) grouped by type with constraints and decisions first. Actionable items surface first.

**Source:** `kb context "pool overhead"` output

**Significance:** This is the command that ties everything together. The prioritized grouping (constraints > decisions > attempts > models > investigations) makes semantic sense for an agent or researcher seeking context.

**Score — Discoverability: 4/5 | Friction: 1/5**

---

### Finding 9: `kb search` provides line-level results — useful for specific lookups

**Evidence:** `kb search "crossover"` returned 3 results with matching line numbers and preview text from each file.

**Source:** `kb search "crossover"` output

**Significance:** Complements `kb context` well. Context is for "give me everything relevant to this topic"; search is for "find where I mentioned this specific term."

**Score — Discoverability: 4/5 | Friction: 1/5**

---

### Finding 10: `kb reflect` returns nothing useful in small projects

**Evidence:** All reflect types (promote, stale, synthesis, drift, open, etc.) return "No X opportunities found" for a project with 2 investigations, 1 model, 5 quick entries.

**Source:** `kb reflect` and individual `--type` flags

**Significance:** Expected — reflect needs critical mass (3+ investigations on one topic for synthesis, >7 day age for stale detection). The "No opportunities found" output is fine. But there's no guidance like "Reflection works best with 5+ investigations. You have 2." that would set expectations.

**Score — Discoverability: 4/5 | Friction: 2/5**

---

### Finding 11: `kb list` bare command shows help instead of listing everything

**Evidence:** `kb list` shows subcommand help (investigations, decisions). No `kb list all` or `kb list models` or `kb list quick`.

**Source:** `kb list` output

**Significance:** Missing `kb list models` and `kb list quick` means models and quick entries are only discoverable via `kb context` or filesystem browsing. The list command should enumerate all artifact types.

**Score — Discoverability: 2/5 | Friction: 3/5**

---

## Synthesis

**Key Insights:**

1. **Models are second-class citizens** — No create command, not in init scaffold, not in list, not in index. Every other artifact type has `kb create <type>` except models. This is the biggest gap for new users.

2. **Quick entries are first-class and excellent** — The lowest friction path in the entire system. One command captures knowledge. The question→resolve lifecycle is elegant. This should be the first thing new users try.

3. **The probe→model feedback loop works and delivers real value** — My probe genuinely corrected the model (4KB crossover → 1-2KB stack/heap boundary). This is the system's killer feature: knowledge that self-corrects through testing. But you can't get to this loop without manually creating the model directory.

**Answer to Investigation Question:**

The end-to-end experience works but has a rough onboarding. The first 10 minutes are frustrating (no model command, heavy templates, orphan escape hatch). Once past setup, the middle of the workflow (investigate, probe, update model, quick entries, context search) is smooth and genuinely valuable. The probe-to-model feedback loop produced a real correction in my understanding. Quick entries are the best entry point. The system rewards investment but has a cliff at the beginning.

---

## Structured Uncertainty

**What's tested:**

- ✅ Full cycle: init → investigate → model → probe → context → quick → reflect (all executed)
- ✅ All quick entry types (decide, tried, constrain, question, resolve)
- ✅ kb context and kb search with real data
- ✅ --extract-models behavior (confirmed broken)

**What's untested:**

- ⚠️ `kb reflect` with enough data to trigger patterns (need 3+ investigations)
- ⚠️ `kb context --global` and `--siblings` (cross-project)
- ⚠️ `kb promote` workflow (quick entry → full decision)
- ⚠️ `kb archive`, `kb supersede`, `kb learn` commands

**What would change this:**

- A project with 10+ investigations and months of history would exercise reflect and stale detection
- Multiple projects would test cross-project search

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add `kb create model` command | architectural | New CLI command affecting multiple components |
| Fix `--extract-models` | implementation | Bug fix within existing command |
| Fix `kb index` | implementation | Bug fix within existing command |
| Add `kb list models` and `kb list quick` | architectural | Extends list command surface area |
| Add `--minimal` investigation template | architectural | New template variant |

### Recommended Approach ⭐

**Prioritized fixes** — Ship `kb create model` first (blocks new user workflow), then fix --extract-models and kb index (broken features), then ergonomic improvements.

**Implementation sequence:**
1. `kb create model <name>` — creates `.kb/models/{name}/probes/` and copies template
2. Fix `--extract-models` to inline model sections instead of dropping them
3. Fix `kb index` to scan all artifact types including models and quick entries
4. Add `kb list models` and `kb list quick` subcommands
5. Consider `--minimal` flag for investigation template

---

## Command Scorecard

| Command | Discoverability (1-5) | Friction (1-5, lower=better) | Notes |
|---------|----------------------|------------------------------|-------|
| `kb init` | 5 | 1 | Clean, fast, just works |
| `kb create investigation` | 3 | 3 | --model requirement is correct but surprise for new users |
| `kb create model` | 1 | 5 | **Does not exist** |
| `kb create probe` | 4 | 1 | Model context auto-include is great |
| `kb context` | 4 | 1 | Unified search, good grouping |
| `kb context --extract-models` | 3 | 5 | **Broken** — drops models |
| `kb quick decide/tried/constrain` | 5 | 1 | Best UX in the system |
| `kb quick question/resolve` | 4 | 2 | resolve syntax is flag-based (minor friction) |
| `kb reflect` | 4 | 2 | No guidance for small projects |
| `kb search` | 4 | 1 | Line-level results, useful |
| `kb list` | 2 | 3 | Missing models, quick, bare listing |
| `kb index` | 4 | 4 | **Broken** — shows nothing |
| `kb list investigations` | 4 | 1 | Clean, shows status |

**Overall system score: 3.5/5** — Core workflow is powerful once you know the escape hatches. New user onboarding is the weakest link.

---

## References

**Test project:** ~/tmp/kb-test-project
**Commands run:** kb init, kb create investigation (×2), kb create probe, kb context (×3), kb quick (×5), kb reflect (×4), kb search, kb list (×2), kb index, kb version
