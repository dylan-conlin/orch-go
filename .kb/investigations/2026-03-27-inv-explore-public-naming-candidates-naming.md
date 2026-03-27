## Summary (D.E.K.N.)

**Delta:** 23 naming candidates assessed across 3 waves; 17 RED (fatal collision), 6 YELLOW (viable with friction), 0 GREEN. The single-word devtool namespace is brutally crowded. Top candidate is **Kenning** — all package registries clear, two domains available, strong semantic fit for comprehension/composition, only conflict is a niche embedded ML framework.

**Evidence:** Web search collision checks for each candidate across domains, GitHub, npm/PyPI/Go registries, and funded startups. Every "obvious" metaphor name (Loom, Weave, Thread, Stitch, Trellis, etc.) is owned by a well-funded product in an overlapping space.

**Knowledge:** The naming landscape teaches something about the product: names that evoke weaving, accumulation, or data processing are saturated because those are infrastructure concepts. The product's actual identity (comprehension, named incompleteness, understanding that compounds) lives in a semantic space that most dev tools don't target — which means less-obvious words from knowledge/language/understanding traditions have more room.

**Next:** Strategic — Dylan chooses the product name. Recommendation: adopt "Kenning" as working product name, keep `orch`/`orch-go` as repo/CLI, stage a rename only after v1 traction proves the name.

**Authority:** strategic — Irreversible public positioning, value judgment, affects all external-facing surfaces.

---

# Investigation: Public Naming Candidates and Naming Architecture for v1 Product Boundary

**Question:** What public naming candidates and naming architecture best fit the v1 product boundary (thread/comprehension/knowledge-composition layer)?

**Started:** 2026-03-27
**Updated:** 2026-03-27
**Owner:** orch-go-sg39y
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` | extends | yes | - |
| `.kb/models/knowledge-accretion/probes/2026-03-26-probe-minimum-open-release-bundle-definition.md` | extends | yes | - |
| `.kb/investigations/2026-03-26-design-thread-first-home-surface.md` | extends | yes | - |
| `.kb/investigations/2026-03-11-inv-investigation-wedge-candidate-inventory-orch.md` | extends | yes | - |
| `.kb/threads/2026-03-22-narrative-packaging-orch-go-s.md` | extends | yes | confirms: mechanism descriptions don't land, need a story |

---

## Findings

### Finding 1: Every "Obvious" Metaphor Name Is RED

**Evidence:** Checked 17 candidates in textile/accumulation/synthesis metaphor families. Results:

| Name | Collision | Fatal Conflict |
|------|-----------|----------------|
| Loom | RED | Atlassian Loom ($975M acquisition), Java Project Loom |
| Weave | RED | W&B Weave (AI devtool), Figma Weave, Weave YC startup, Google Service Weaver |
| Thread | RED | Meta Threads (275M+ MAU), Thread AI (Index Ventures), Thread Group trademark |
| Stitch | RED | Google Stitch (AI UI design), Stitch Data (3K+ companies) |
| Trellis | RED | mindfold-ai/Trellis (4.3K stars, "best agent harness", same space) |
| Gist | RED | GitHub Gists (core feature), Gist AI (summarization) |
| Thesis | RED | Thesis Labs (YC F25, AI dev tooling), owns GitHub org |
| Patina | RED | Patina Systems (VC-funded devtools, "post-personal computing") |
| Sediment | RED | rendro/sediment ("local-first semantic memory for AI agents") |
| Residue | RED | residue.dev (AI agent conversation capture for git) |
| Kindle | RED | Amazon Kindle (trademark risk) |
| Corpus | RED | Corpus AI ("personal AI memory"), NLP term saturation |
| Grist | RED | Grist Labs (10.8K stars, data tool) |
| Thrum | RED | leonletto/thrum (Go AI agent messaging, same space) |
| Distill | RED | distill.pub (ML journal), ML term "knowledge distillation" |
| Acumen | RED | acumen.io (engineering intelligence for software teams) |
| Compendium | YELLOW | Too long for CLI (10 chars), generic word |

**Source:** Web searches across product databases, GitHub, npm/PyPI/Go registries, domain WHOIS, Crunchbase/PitchBook for each candidate.

**Significance:** The single-word metaphor namespace for devtools is exhausted. Any name that a product person would brainstorm in 30 seconds is already taken. This means viable names must come from less-trafficked semantic territories.

---

### Finding 2: Six YELLOW Candidates Survive — Kenning Is Strongest

**Evidence:** Scored each YELLOW candidate on four axes:

| Candidate | Collision Risk | Semantic Fit | Domain/Registry | CLI Usability | Overall |
|-----------|---------------|-------------|-----------------|--------------|---------|
| **Kenning** | YELLOW (Antmicro embedded ML, different audience) | EXCELLENT (compound metaphor + "to know") | npm/PyPI/Go ALL CLEAR; kenning.sh, kenning.so available | `kenning` (7 chars) | **#1** |
| **Sinter** | YELLOW (Trail of Bits security agent, different space) | GOOD metaphor, TOO OBSCURE for most | npm, Go clear; getsinter.com available | `sinter` (6 chars) | **#2** |
| **Precis** | YELLOW (no dominant product in space) | DIRECT (summary/abstract) but too narrow | Go clear; precis.sh available; npm/PyPI squatted | `precis` (6 chars) | **#3** |
| **Anneal** | YELLOW (getanneal.com "Engineering Orchestration") | GOOD metaphor but evokes process, not understanding | Go clear; anneal.sh available | `anneal` (6 chars) | **#4** |
| **Skein** | YELLOW (no product in same space) | GOOD (threads composing) | ALL registries taken, ALL domains taken | `skein` (5 chars) | **#5** |
| **Accrue** | YELLOW (PyPI LLM pipeline tool, .com is active) | DIRECT but too financial | Go clear; accrue.sh maybe available | `accrue` (6 chars) | **#6** |

**Kenning detail:** A kenning is an Old Norse poetic device — a compound metaphor that creates understanding through composition of simpler concepts ("whale-road" = sea, "battle-sweat" = blood). The word etymologically derives from Old Norse *kenna* ("to know, to perceive"). It evokes exactly what the product does: composing simpler observations into richer understanding.

Only conflict: Antmicro's `kenning` — an embedded ML deployment framework for edge hardware (91 GitHub stars). This is a niche product targeting FPGA/embedded engineers, not devtools/knowledge workers. The audiences do not overlap. Antmicro owns kenning.ai but not kenning.sh, kenning.so, or the npm/PyPI/Go package names.

**Source:** Collision check agents across web search, GitHub, package registries, domain WHOIS for each candidate.

**Significance:** Kenning is the only candidate that scores well on ALL four axes simultaneously. The others each have at least one significant weakness (obscurity, pronunciation, registry availability, or misleading category association).

---

### Finding 3: Semantic Fit — Names Must Evoke Comprehension, Not Infrastructure

**Evidence:** Categorized all 23 candidates by what they evoke on first contact:

| Category | Names | Product Fit |
|----------|-------|-------------|
| **Weaving/composition** | Loom, Weave, Stitch, Skein, Braid | Evokes pipeline/ETL — WRONG |
| **Accumulation/geology** | Sediment, Accrue, Accrete, Residue | Evokes data storage — PARTIAL |
| **Process/metallurgy** | Anneal, Sinter, Forge | Evokes CI/optimization — WRONG |
| **Summary/distillation** | Precis, Distill, Gist | Evokes summarization tool — TOO NARROW |
| **Understanding/knowledge** | Kenning, Thesis, Acumen, Corpus | Evokes comprehension — CORRECT |
| **Structure/support** | Trellis, Lattice, Scaffold | Evokes infrastructure — WRONG |

The product's identity per the decision document: "turns agent work into durable, legible understanding." Names from the understanding/knowledge category point toward this. Names from other categories point toward infrastructure, data processing, or workflow — which is explicitly the substrate, not the product.

Of the understanding/knowledge names, only Kenning survives collision checks. Thesis (YC startup), Acumen (funded eng intelligence), and Corpus (NLP term saturation) are all RED.

**Significance:** The naming landscape contains a useful product signal: the comprehension/understanding semantic territory is less contested than the infrastructure/process territory. This aligns with the product decision — the differentiated value IS in the less-crowded space.

---

### Finding 4: First-Impression Test for Top Candidates

**Evidence:** Paired each viable name with a homepage headline and assessed category perception:

**Kenning — "Durable understanding from AI agent work"**
- First impression: knowledge/comprehension tool for developers
- Category guess: AI knowledge management ✓
- Bonus: the name itself teaches something (kennings are compound metaphors from Norse poetry) — it sticks once learned and creates a natural conversation starter
- Risk: some people won't know the word, but "ken" (to know/understand) is familiar in many English dialects

**Sinter — "Fusing knowledge into understanding"**
- First impression: unclear — could be manufacturing, a game, anything
- Category guess: requires the tagline to understand ✗
- Risk: the name doesn't carry its own weight

**Precis — "The essential understanding, nothing more"**
- First impression: summarization/documentation tool
- Category guess: too narrow — suggests compression, not composition ✗
- Risk: pronunciation ambiguity (PRAY-see vs PRESS-ee)

**Anneal — "Knowledge that gets stronger the more you test it"**
- First impression: testing/CI tool
- Category guess: misleading — suggests iteration, not understanding ✗
- Risk: collision with getanneal.com "Engineering Orchestration System" reinforces wrong category

**Source:** Synthetic evaluation based on word etymology, common associations, and product positioning alignment.

**Significance:** Kenning is the only name where the first impression matches the product identity WITHOUT relying on the tagline to redirect. Every other candidate either misdirects or requires explanation.

---

### Finding 5: Naming Architecture — Separate Product Name from Repo/CLI

**Evidence:** Evaluated three architecture options:

**Option A: One name everywhere (rename repo, CLI, package)**
- Cost: Rename `orch-go` → `kenning`, all Go import paths, CLI binary, every script reference, every CLAUDE.md mention, every skill reference
- Go module rename = all downstream consumers break (if any external users exist)
- Risk: Premature — if the name doesn't stick, you rename twice

**Option B: Separate product name, keep orch-go repo and orch CLI**
- Cost: Zero code changes. Product name appears only in: README, website, blog posts, method guide, artifact format docs
- The CLI `orch` continues to work internally. A public binary can be named `kenning` as an alias.
- Risk: Cognitive split between internal (orch) and external (product name) naming. Manageable with an alias.

**Option C: Temporary working name → deeper rename later**
- Cost: Option B now, Option A later when name is proven
- Risk: Two transitions, but the first one is cheap and the second only happens if v1 gets traction

**Source:** Analysis of Go module rename costs, codebase grep for `orch` references, assessment of current external surface area.

**Significance:** The correct architecture is B→C: adopt the product name for external surfaces now, keep orch internally, and stage a deep rename only if the name proves itself through adoption. The minimum open release bundle (artifact formats + composition docs + thread surface) can use the product name without touching a single line of Go code.

---

## Synthesis

**Key Insights:**

1. **The namespace teaches you about your product.** Names that evoke infrastructure are saturated because that's where most devtools live. The understanding/comprehension space is less contested — which is exactly the product's thesis. The difficulty of finding a name IS evidence that the product positioning is differentiated.

2. **Kenning is the strongest candidate by a significant margin.** It's the only name that scores well on all four axes: collision risk, semantic fit, registry/domain availability, and first-impression accuracy. The word itself embodies the product's core operation — composing simpler concepts into richer understanding.

3. **The naming architecture should be layered, not monolithic.** Product name ≠ repo name ≠ CLI name at v1. The public product name appears in docs, website, and positioning. The internal `orch` name stays in code. A deep rename happens only when adoption validates the name.

**Answer to Investigation Question:**

The recommended product name is **Kenning**. It evokes comprehension and composition (what the product does), avoids infrastructure/pipeline/workflow associations (what the product is not), has clean package registry availability across npm/PyPI/Go, has available domains (kenning.sh, kenning.so), and creates a natural "what does that mean?" conversation that becomes a feature of the brand. The naming architecture should separate product name (Kenning) from repo/CLI (orch-go/orch), with a deep rename staged for post-v1 traction.

---

## Structured Uncertainty

**What's tested:**

- ✅ 23 candidates collision-checked via web search, GitHub, npm/PyPI/Go registries, domain WHOIS
- ✅ Antmicro's kenning (91 stars) is embedded ML deployment, not devtools/knowledge (verified via GitHub repo description and documentation)
- ✅ kenning.sh and kenning.so domains are available (verified via DNS/WHOIS)
- ✅ npm `kenning`, PyPI `kenning`, Go `kenning` packages do not exist (verified via registry queries)
- ✅ Product decision document confirms thread/comprehension layer is primary identity (verified by reading .kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md)

**What's untested:**

- ⚠️ Whether "kenning" is too obscure for the target audience (requires user research / first-contact testing)
- ⚠️ Whether the Old Norse poetry association helps or hurts developer adoption (cultural reception)
- ⚠️ Whether kenning.dev could be acquired (registered but not serving content)
- ⚠️ How the name sounds in non-English-speaking developer communities
- ⚠️ Whether Antmicro's use of "kenning" in the AI space creates more confusion than the 91-star count suggests

**What would change this:**

- If user testing shows >50% of developers cannot pronounce or remember "kenning," try Precis or Sinter
- If Antmicro's kenning project grows significantly (>1K stars) or expands into devtools, reassess
- If a strong GREEN candidate emerges that this search missed (compound words, neologisms, non-English)
- If Dylan decides the name should evoke process/iteration rather than understanding/composition, Anneal becomes the top pick

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Choose product name | strategic | Irreversible public positioning, affects all external surfaces, value judgment |
| Naming architecture (product vs repo vs CLI) | strategic | Determines migration cost path and public identity structure |
| Register domains and package names | implementation | Tactical execution once name is chosen |

### Recommended Approach ⭐

**Adopt "Kenning" as working product name with layered naming architecture**

**Why this approach:**
- Only candidate that scores well on all four evaluation axes simultaneously
- Semantic alignment: "kenning" = composing simpler concepts into richer understanding = exactly what the product does
- Clean registries: can claim `kenning` on npm, PyPI, and Go immediately
- Available domains: kenning.sh and kenning.so are unregistered
- The name creates brand conversation: "What's a kenning?" → natural explanation of the product's thesis

**Trade-offs accepted:**
- Some developers won't know the word on first contact (mitigated by memorable pronunciation and natural curiosity)
- kenning.dev is parked (can use kenning.sh or kenning.so; can attempt to acquire .dev later)
- Antmicro's embedded ML framework creates minor search noise at 91 stars

**Implementation sequence:**
1. **Register domains and package names** — claim kenning.sh, kenning.so, npm `kenning`, PyPI `kenning` before someone else does
2. **Use the name in external surfaces** — README rewrite, method guide, blog post, v1 artifact format docs
3. **Keep orch-go/orch internally** — no code changes, no Go module rename
4. **Add CLI alias** — `kenning` binary that wraps or symlinks to `orch`, for public-facing use
5. **Stage deep rename for post-v1** — only if adoption validates the name and the cognitive split becomes costly

### Alternative Approaches Considered

**Option B: Use "Precis" as product name**
- **Pros:** Direct semantic fit (summary/abstract), pronounceable
- **Cons:** Too narrow (suggests summarization, not composition); pronunciation ambiguity; npm/PyPI squatted; precis.dev parked
- **When to use instead:** If user testing shows "kenning" is too obscure for the audience

**Option C: Use "Sinter" as product name**
- **Pros:** Strong metaphor (fusing particles into solid through process); npm and Go clear; getsinter.com available
- **Cons:** Too obscure — most developers will think it's a typo or arbitrary word; doesn't carry its own weight
- **When to use instead:** If the product evolves toward a more process-oriented identity (how knowledge is forged)

**Option D: Use "Anneal" as product name**
- **Pros:** Good metaphor (strengthening through controlled cycles); anneal.sh available
- **Cons:** Evokes iteration/optimization rather than understanding; getanneal.com collision with "Engineering Orchestration System"
- **When to use instead:** If the product's primary value proposition shifts toward iterative improvement rather than comprehension

**Rationale for recommendation:** Kenning is the only candidate where the name's meaning matches the product's core operation, the namespace is clear, and the first impression points to the right category. Every alternative either misdirects, is unavailable, or requires explanation to reach the same understanding that Kenning provides naturally.

---

### Implementation Details

**What to implement first:**
- Domain registration (kenning.sh, kenning.so) — perishable asset
- Package name registration (npm, PyPI, Go) — first-come-first-served
- README rewrite using "Kenning" as product name (already planned per product decision follow-through)

**Things to watch out for:**
- ⚠️ Do NOT rename the repo or Go module at this stage — the cost is high and the name isn't proven
- ⚠️ Do NOT claim kenning.ai — Antmicro owns it and attempting to register a conflicting .ai domain invites confusion
- ⚠️ Package name claims should be minimal placeholder packages, not empty squats — publish a v0.0.1 with a README pointing to the product
- ⚠️ The method guide (must-ship from release bundle) should use "Kenning" consistently — this is the first external artifact to carry the name

**Areas needing further investigation:**
- User testing: show 5-10 developers the name + tagline and assess comprehension/recall
- kenning.dev acquisition: the domain is registered but not serving content — may be acquirable
- Trademark search: formal trademark search for "Kenning" in software/technology classes

**Success criteria:**
- ✅ Domains registered and resolving
- ✅ Package names claimed on npm, PyPI, Go
- ✅ README and method guide use "Kenning" consistently
- ✅ First-contact users can recall the name and associate it with "knowledge/understanding tool" after one exposure

---

## Explicit Rejections

Names rejected for structural reasons (not just collision):

| Name | Rejection Reason |
|------|-----------------|
| **Orch / Orchestrate** | Explicitly wrong identity — "orchestration" is substrate, not product |
| **Any textile metaphor** (Loom, Weave, Stitch, Braid) | Points toward pipeline/ETL/data-flow — wrong category |
| **Any data/NLP term** (Corpus, Distill, Gist) | Pre-loaded association with data processing, not understanding |
| **Any infrastructure name** (Substrate, Pipeline, Conduit, Mesh) | Wrong direction by definition |
| **Any process/optimization name** (Forge, Crucible, Catalyst) | Evokes CI/CD or iteration, not comprehension |
| **Compound** | Evokes chemical compounds or financial compounding, not knowledge |
| **Compendium** | Too long for CLI (10 chars), too generic for SEO |

---

## References

**Files Examined:**
- `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` — Product boundary decision
- `.kb/models/knowledge-accretion/probes/2026-03-26-probe-minimum-open-release-bundle-definition.md` — Release bundle definition
- `.kb/investigations/2026-03-26-design-thread-first-home-surface.md` — Thread-first UI design
- `.kb/threads/2026-03-27-this-product-called-does-name.md` — Naming thread
- `.kb/threads/2026-03-27-generative-systems-are-organized-around.md` — Named incompleteness principle
- `.kb/threads/2026-03-22-narrative-packaging-orch-go-s.md` — Narrative packaging thread
- `.kb/investigations/2026-03-11-inv-investigation-wedge-candidate-inventory-orch.md` — Wedge candidate inventory

**Commands Run:**
```bash
# Web collision checks for 23 naming candidates
# (executed via subagent web searches across domains, GitHub, registries)
```

**External Documentation:**
- Collision data sourced via web search agents checking GitHub, npm, PyPI, pkg.go.dev, domain WHOIS/DNS, Crunchbase, PitchBook for each candidate

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` — Product identity this serves
- **Thread:** `.kb/threads/2026-03-27-this-product-called-does-name.md` — Naming question thread

---

## Investigation History

**2026-03-27:** Investigation started
- Initial question: What public naming candidates and naming architecture best fit the v1 product boundary?
- Context: Product decision established thread/comprehension layer as primary product; naming is the next strategic question

**2026-03-27:** Wave 1 — 10 candidates checked (Loom, Weave, Accrue, Thread, Gist, Kindle, Thesis, Stitch, Sediment, Residue)
- All "obvious" metaphor names RED except Accrue (YELLOW)
- Key finding: single-word devtool namespace is exhausted for common English words

**2026-03-27:** Wave 2 — 8 candidates checked (Trellis, Skein, Precis, Anneal, Patina, Kenning, Sinter, Corpus)
- Trellis, Patina, Corpus RED; Skein, Precis, Anneal, Kenning, Sinter YELLOW
- Key finding: Kenning stands out — all registries clear, strong semantic fit

**2026-03-27:** Wave 3 — 5 candidates checked (Grist, Thrum, Acumen, Distill, Compendium)
- Grist, Thrum, Acumen, Distill RED; Compendium YELLOW but too long
- Confirmed: Kenning is the strongest surviving candidate

**2026-03-27:** Investigation completed
- Status: Complete
- Key outcome: Kenning recommended as product name with layered naming architecture (product name separate from repo/CLI)
