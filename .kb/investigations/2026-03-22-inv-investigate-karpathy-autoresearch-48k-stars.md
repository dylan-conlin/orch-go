## Summary (D.E.K.N.)

**Delta:** autoresearch succeeds not because of sophisticated orchestration but because it constrains the problem so tightly (1 file, 1 metric, 5 min runs, keep/discard) that a single agent with a markdown prompt can do useful autonomous work overnight.

**Evidence:** 1,225 total lines across 3 files. No agent framework, no task decomposition, no state machine. The "orchestration" is a 114-line markdown file (`program.md`) that gives Claude/Codex a git branch + experiment loop. 60 commits, 48k stars in 16 days.

**Knowledge:** The constraint surface is the architecture. autoresearch proves that tight constraint design eliminates the need for orchestration machinery. orch-go solves a harder problem (multi-agent, open-ended tasks, governance) but should internalize the lesson: the power is in the problem formulation, not the framework.

**Next:** Strategic discussion — what does this mean for orch-go's positioning and Dylan's career narrative around agent orchestration?

**Authority:** strategic - This touches positioning and career narrative, not implementation

---

# Investigation: Karpathy's autoresearch — 48k Stars in 16 Days

**Question:** How does karpathy/autoresearch work, why did it explode in popularity, and what does it teach orch-go about agent orchestration?

**Started:** 2026-03-22
**Updated:** 2026-03-22
**Owner:** investigation agent (orch-go-h40x4)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## Findings

### Finding 1: Radical simplicity — the entire "framework" is 3 files

**Evidence:** The repo is 1,225 lines total across 3 meaningful files:
- `train.py` (630 lines) — GPT model + MuonAdamW optimizer + training loop. This is the **only file the agent edits**.
- `prepare.py` (389 lines) — Data prep, tokenizer, dataloader, evaluation. **Read-only**.
- `program.md` (114 lines) — Agent instructions. **The only "orchestration"**.

There is no agent framework. No task queue. No state machine. No coordination protocol. No daemon. No governance. The `program.md` is literally a markdown document that says "read these files, create a git branch, then loop forever: edit train.py, run it, check if val_bpb improved, keep or discard."

**Source:** All files in `~/Documents/personal/autoresearch/`. `wc -l` confirms: 630 + 389 + 114 + 92 (README) = 1,225 lines.

**Significance:** The entire orchestration strategy is *constraint design*, not machinery. By constraining the problem to 1 file, 1 metric, 1 GPU, fixed 5-minute runs, the agent doesn't need coordination — it just needs a clear loop and a git branch.

---

### Finding 2: The agent loop is a keep/discard hill-climber with git as state

**Evidence:** From `program.md` lines 94-106, the experiment loop:

```
LOOP FOREVER:
1. Look at the git state: the current branch/commit we're on
2. Tune train.py with an experimental idea by directly hacking the code
3. git commit
4. Run the experiment: uv run train.py > run.log 2>&1
5. Read out the results: grep "^val_bpb:|^peak_vram_mb:" run.log
6. If grep empty → crashed. Read tail -n 50 run.log, attempt fix
7. Record results in results.tsv
8. If val_bpb improved → keep the commit (advance the branch)
9. If val_bpb equal or worse → git reset back
```

Key design decisions:
- **Git IS the state machine.** Branch position = current best. No database, no registry, no projection.
- **Single scalar metric** (val_bpb, lower is better). No ambiguity about what "better" means.
- **Fixed time budget** (5 minutes wall clock). Makes every experiment directly comparable regardless of what the agent changed (architecture, batch size, model size).
- **Simplicity criterion** (program.md:37): "A 0.001 val_bpb improvement that adds 20 lines of hacky code? Probably not worth it. A 0.001 improvement from deleting code? Definitely keep."
- **NEVER STOP** instruction (program.md:112): "do NOT pause to ask the human... The human might be asleep."

**Source:** `program.md:94-112`, `train.py:543-604` (the training loop respects TIME_BUDGET from prepare.py)

**Significance:** This is a **single-agent, single-objective, greedy hill-climber** with git as the undo mechanism. The "orchestration" is entirely embedded in the prompt. Compare to orch-go which manages multi-agent lifecycle, skill routing, governance, and open-ended tasks — fundamentally different problem shapes.

---

### Finding 3: The "human programs in markdown" paradigm shift

**Evidence:** From README.md line 7: "The core idea is that you're not touching any of the Python files like you normally would as a researcher. Instead, you are programming the `program.md` Markdown files that provide context to the AI agents."

And: "The default `program.md` in this repo is intentionally kept as a bare bones baseline, though it's obvious how one would iterate on it over time to find the 'research org code' that achieves the fastest research progress, how you'd add more agents to the mix, etc."

The Karpathy opening quote frames this as the origin story of fully autonomous AI research: "Research is now entirely the domain of autonomous swarms of AI agents running across compute cluster megastructures in the skies."

**Source:** `README.md:6-7`, `README.md:1-5`

**Significance:** This frames AI agent orchestration as a *new programming paradigm* — you don't write code, you write prompts that tell the agent how to write and evaluate code. This is exactly what orch-go's skill system does (SKILL.md files are the "program.md" equivalent), but Karpathy packaged it as a clean, viral concept.

---

### Finding 4: 48k stars driven by Karpathy brand + "while you sleep" narrative + tangible results

**Evidence:**
- **Brand:** Karpathy (ex-Tesla AI director, OpenAI founding team, 900k+ Twitter followers) is the most trusted name in practical ML education. His repos (nanoGPT, minbpe, llm.c) routinely get 10k-40k stars.
- **"While you sleep" hook:** README and program.md both emphasize: "You wake up in the morning to a log of experiments." The program.md calculates: "~12 experiments/hour, ~100 experiments while you sleep." This is an irresistible narrative.
- **Tangible results:** The git history shows 60 commits, many of which are actual experiment results (e.g., "embedding LR 0.8 to 0.9", "muon momentum warmup 300 to 200 steps"). The `analysis.ipynb` produces a real `progress.png` chart. You can see the val_bpb curve descending.
- **Accessibility:** Single GPU, single file, `uv sync && uv run prepare.py && uv run train.py`. Three commands and you're running autonomous ML research.
- **Timing:** March 2026 — peak "AI agents" hype cycle. Coding agents (Claude Code, Codex, Cursor) are mainstream. This gives people something *concrete* to do with them.

**Source:** Git history (`git log --oneline --all`), `README.md`, `program.md`, tweet references in README

**Significance:** The viral success isn't from technical sophistication — it's from *narrative packaging* of a simple idea by a trusted voice at the right moment. The "research org code" framing turns prompt engineering into something that sounds like founding a company.

---

### Finding 5: What autoresearch does NOT have (the orch-go gap)

**Evidence:** autoresearch has zero:
- **Multi-agent coordination.** One agent, one branch, one loop. No spawn, no daemon, no agent discovery.
- **Task decomposition.** The agent decides what to try next purely from its own reasoning + results.tsv history. No issue tracker, no skill routing.
- **Quality gates / governance.** Keep/discard is based solely on a single number (val_bpb). No code review, no accretion limits, no pre-commit hooks.
- **State persistence across sessions.** If the agent crashes, the git branch preserves state, but there's no structured handoff, no knowledge base, no investigation file.
- **Open-ended task support.** The metric is val_bpb. Period. There's no way to run "investigate why loss spikes" or "design a new attention mechanism" — the system only optimizes a scalar.
- **Observability/dashboard.** results.tsv is the dashboard. `analysis.ipynb` generates a chart manually.
- **Error recovery.** "If you can't get things to work after more than a few attempts, give up" (program.md:101).

**Source:** Full repo analysis — none of these features exist in any file.

**Significance:** autoresearch succeeds by *not needing* these things. The tight constraint surface (1 file, 1 metric, fixed budget) makes orchestration machinery unnecessary. orch-go needs all of it because it handles fundamentally open-ended, multi-agent, governance-heavy workflows. These are different tool categories, not competing products.

---

### Finding 6: The train.py is serious ML engineering, not a toy

**Evidence:** train.py includes:
- **MuonAdamW optimizer** — a combined Muon (for 2D matrix params via polar express orthogonalization) + AdamW (for embeddings, scalars) optimizer with `torch.compile` fused kernels
- **Flash Attention 3** via `kernels` package, with Hopper/non-Hopper fallback
- **Value embeddings** (ResFormer-style) with input-dependent gating
- **Sliding window attention** patterns (SSSL)
- **Cautious weight decay** — only applies WD where gradient and parameter signs agree
- **NorMuon variance reduction** in the Muon optimizer
- **GC management** — disables Python GC after warmup to avoid 500ms stalls

This isn't a tutorial. It's a real, optimized LLM training setup distilled from Karpathy's nanochat project. The agent is genuinely doing ML research on a competitive codebase.

**Source:** `train.py:1-630`, especially the MuonAdamW class (lines 296-427) and model architecture (lines 32-291)

**Significance:** The quality of the substrate matters. The reason the "autonomous researcher" concept works here is that train.py is already at a level where small architectural/hyperparameter tweaks produce meaningful metric changes. A toy setup would produce noise. This is a key design insight: **the agent needs a substrate that rewards good decisions**.

---

## Synthesis

**Key Insights:**

1. **Constraints are the architecture.** autoresearch has no orchestration framework because the problem constraints (1 file, 1 metric, 5 min, keep/discard) ARE the framework. The genius is in the constraint design, not in building machinery to manage agents. This echoes a principle orch-go should internalize: before adding orchestration complexity, ask "can I constrain the problem so the complexity isn't needed?"

2. **Narrative > mechanism for adoption.** autoresearch's 48k stars aren't from technical sophistication — it's from Karpathy's brand + the "autonomous research while you sleep" narrative + the accessibility of 3 commands to start. orch-go is 100x more capable but has 0 stars. The lesson isn't "build simpler" — it's "package the narrative." Dylan's blog publication strategy should take this seriously.

3. **Single-agent hill-climbing is underrated.** For problems with a clear scalar metric, a single agent in a tight loop with git-based rollback is remarkably effective. orch-go's multi-agent approach is necessary for open-ended tasks, but for narrow optimization problems, autoresearch's pattern might be worth stealing for specific use cases (e.g., "orch optimize" that runs a tight loop against a measurable target).

4. **Git as state machine is elegant.** autoresearch uses git branch position as the canonical state. Keep = advance branch. Discard = reset. No database, no registry, no projection. This aligns perfectly with orch-go's "no local agent state" architectural constraint — git IS the authoritative source. autoresearch takes this to its logical extreme.

5. **"Program.md" = orch-go's SKILL.md.** The core insight — humans write markdown that programs AI agents — is exactly what orch-go's skill system already does. Karpathy just named it better ("research org code") and applied it to a domain (ML research) where the results are immediately impressive.

**Answer to Investigation Question:**

autoresearch works by constraining ML research to a single-file, single-metric, fixed-budget optimization problem where a single agent can hill-climb autonomously using git as its state machine. It exploded in popularity because Karpathy's brand + the "while you sleep" narrative + tangible results + accessibility created a perfect viral package at peak AI-agent hype.

The relation to orch-go is *complementary, not competitive*. autoresearch solves a narrow problem (scalar optimization) beautifully through constraint design. orch-go solves a broad problem (open-ended multi-agent orchestration with governance) through machinery. The key lessons for orch-go:

- **Constraint design before orchestration complexity** — can we define problem shapes where tight constraints eliminate the need for governance?
- **Narrative packaging matters** — the blog strategy needs a "while you sleep" hook
- **Git-as-state is a shared principle** — orch-go's "no local agent state" is vindicated
- **Steal the tight-loop pattern** for narrow optimization tasks within orch-go

---

## Structured Uncertainty

**What's tested:**

- Autoresearch architecture (read all source files, 1,225 lines total)
- Git history analysis (60 commits, clear evolution from initial commit to current)
- Constraint surface enumeration (verified: 1 file editable, 1 metric, fixed time budget)
- orch-go comparison (verified against CLAUDE.md, architecture docs, skill system)

**What's untested:**

- Actual agent performance (we don't have an H100 to run it)
- Star growth dynamics (no Twitter/HN analytics beyond the README's tweet links)
- Whether the "research org code" concept scales beyond ML hyperparameter tuning
- Whether orch-go could adopt a tight-loop mode for narrow optimization tasks

**What would change this:**

- If autoresearch's actual results (val_bpb improvements) turned out to be noise, the "tangible results" narrative would weaken
- If someone built a multi-agent version that dramatically outperformed single-agent, the "constraints are the architecture" insight would need revision
- If the star growth was primarily bot-driven, the popularity analysis would be wrong

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Internalize constraint-first design principle | strategic | Changes how Dylan frames orch-go's value proposition |
| Consider "orch optimize" tight-loop mode | architectural | New command pattern, cross-component |
| Use autoresearch as blog/career narrative anchor | strategic | Career positioning, publication strategy |

### Recommended Approach: Use autoresearch as a contrast point in the agent orchestration narrative

**Why this approach:**
- autoresearch proves that constraint design > framework complexity for narrow problems
- orch-go proves that broad, open-ended agent work requires orchestration machinery
- The contrast is the story: "When a single metric defines success, you need autoresearch. When the problem is open-ended and multi-agent, you need orch-go."
- This positions Dylan's work as solving the harder, more valuable problem

**Trade-offs accepted:**
- Not building a competing ML-research tool (different domain, Karpathy's turf)
- Accepting that orch-go will never have autoresearch's simplicity (because it solves a harder problem)

**Implementation sequence:**
1. Internalize the constraint-first principle: before adding orch-go features, ask "can I constrain the problem instead?"
2. Consider whether orch-go should have a "tight loop" mode for tasks with clear scalar metrics
3. Use the autoresearch comparison in Dylan's blog/career narrative about AI agent orchestration

### Alternative Approaches Considered

**Option B: Build an autoresearch-like mode into orch-go**
- **Pros:** Captures the "while you sleep" narrative, demonstrates versatility
- **Cons:** ML research is Karpathy's domain, doesn't leverage orch-go's strengths
- **When to use instead:** If Dylan wants to demonstrate orch-go with a tangible, measurable demo

**Option C: Fork autoresearch and add orch-go orchestration**
- **Pros:** Shows orch-go adding value on top of a viral project
- **Cons:** Unnecessary complexity for a problem that doesn't need it, would look forced
- **When to use instead:** Never, really — the whole point is that autoresearch doesn't need orchestration

**Rationale for recommendation:** The strongest move is to use autoresearch as a *foil* — it demonstrates the boundary condition where orchestration isn't needed, which makes the case for when it IS needed (orch-go's domain) more compelling.

---

## Sources

- [karpathy/autoresearch GitHub](https://github.com/karpathy/autoresearch) — Source repository, all files read directly from local clone
- [karpathy/nanochat](https://github.com/karpathy/nanochat) — Parent project referenced in README

---

## Investigation History

**2026-03-22:** Investigation started
- Initial question: How does autoresearch work, why 48k stars in 16 days, and what's the relation to orch-go?
- Context: Spawned by orchestrator to analyze a viral AI agent project

**2026-03-22:** All source files read and analyzed
- Read README.md, program.md, train.py, prepare.py, analysis.ipynb, pyproject.toml
- Analyzed git history (60 commits)
- Compared architecture patterns against orch-go

**2026-03-22:** Investigation completed
- Status: Complete
- Key outcome: autoresearch succeeds through constraint design, not orchestration machinery. The lesson for orch-go is to package narrative better and apply constraint-first thinking, not to compete on simplicity.
