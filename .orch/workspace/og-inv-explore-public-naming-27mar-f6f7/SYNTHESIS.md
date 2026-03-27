# Session Synthesis

**Agent:** og-inv-explore-public-naming-27mar-f6f7
**Issue:** orch-go-sg39y
**Duration:** 2026-03-27
**Outcome:** success

---

## Plain-Language Summary

I checked 23 naming candidates for the public product name across three waves, searching for collisions against existing software products, GitHub repos, package registries, and domains. Every "obvious" name — Loom, Weave, Thread, Stitch, Trellis, and 12 others — is already owned by a well-funded product in a nearby space. The surviving recommendation is **Kenning**, an Old Norse word meaning "compound metaphor" (and etymologically "to know"). It's the only candidate where the meaning matches the product (composing simpler concepts into understanding), all package registries are unclaimed, multiple domains are available, and the first impression points to the right category without needing a tagline to redirect. The naming architecture recommendation is to use "Kenning" as the public product name while keeping `orch-go`/`orch` internally — no repo rename, no Go module rename — and stage a deeper rename only if v1 traction validates the name.

## Verification Contract

See `VERIFICATION_SPEC.yaml` — key outcomes are the scored shortlist (23 candidates, 17 RED / 6 YELLOW), collision evidence for each, naming architecture recommendation, and explicit rejection reasons.

---

## TLDR

Explored 23 naming candidates for the v1 product boundary. 17 are RED (fatal collisions), 6 are YELLOW. Recommend **Kenning** as product name with a layered naming architecture (product name separate from repo/CLI).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-27-inv-explore-public-naming-candidates-naming.md` — Full investigation with scored shortlist, collision evidence, and naming architecture recommendation

### Files Modified
- None

### Commits
- (pending)

---

## Evidence (What Was Observed)

- 23 web collision checks executed across GitHub, npm, PyPI, Go registries, domain WHOIS, Crunchbase/PitchBook
- Wave 1 (10 candidates): All common metaphors (Loom, Weave, Thread, Stitch, etc.) are RED — owned by well-funded products
- Wave 2 (8 candidates): Kenning emerged as strongest YELLOW — all registries clear, two domains available
- Wave 3 (5 candidates): Confirmed Kenning's position — additional candidates (Grist, Thrum, Acumen, Distill) all RED
- Antmicro's kenning project (91 stars, embedded ML) is the only collision, targeting a completely different audience (FPGA/embedded engineers)
- kenning.sh and kenning.so confirmed available via DNS/WHOIS
- npm `kenning`, PyPI `kenning`, Go `kenning` all confirmed unclaimed

### Tests Run
```bash
# Web collision checks via subagent searches for each of 23 candidates
# Domain WHOIS/DNS checks for viable candidates
# Package registry availability checks across npm, PyPI, Go, crates.io
```

---

## Architectural Choices

### Product name separate from repo/CLI name
- **What I chose:** Layered architecture — "Kenning" for public product, "orch-go"/"orch" for repo/CLI
- **What I rejected:** Monolithic rename (everything becomes "kenning" immediately)
- **Why:** Go module renames are expensive and break downstream consumers. The name isn't proven yet. Premature rename has high cost if the name changes again.
- **Risk accepted:** Cognitive split between internal and external naming until a deep rename is done

### "Kenning" over "Precis" and "Sinter"
- **What I chose:** A name that evokes knowledge/understanding through its etymology
- **What I rejected:** Precis (too narrow — suggests summarization), Sinter (too obscure — requires explanation)
- **Why:** Kenning is the only candidate where the first impression matches the product identity without tagline assistance
- **Risk accepted:** Some developers won't know the word on first contact

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-27-inv-explore-public-naming-candidates-naming.md` — Full investigation

### Decisions Made
- The single-word devtool namespace is exhausted for common English words
- Names from the understanding/knowledge semantic territory are less contested than infrastructure/process names — which aligns with the product's differentiated position

### Constraints Discovered
- The minimum-viable naming change requires only external surfaces (README, method guide, website) — zero code changes needed

---

## Next (What Should Happen)

**Recommendation:** close (this investigation is complete; the naming decision itself is strategic and requires Dylan)

### If Close
- [x] All deliverables complete (investigation with scored shortlist, collision evidence, naming architecture recommendation)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-sg39y`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Would a compound name (two words) have more room? e.g., "Thread Kenning", "Open Kenning" — not explored because compound names are harder as CLI commands
- Non-English words from knowledge/understanding traditions (Sanskrit, Japanese, Arabic) — might have cleaner namespaces but introduce pronunciation barriers
- Would a completely coined word (neologism) be better? e.g., "Synthex", "Composo" — not explored because coinages usually feel forced

**What remains unclear:**
- Whether "kenning" is too obscure for the target developer audience (needs user testing)
- Whether kenning.dev could be acquired (registered but not serving content)
- Formal trademark search for "Kenning" in software/technology classes

---

## Friction

Friction: none — smooth session. Subagent parallel execution worked well for collision checks.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-inv-explore-public-naming-27mar-f6f7/`
**Investigation:** `.kb/investigations/2026-03-27-inv-explore-public-naming-candidates-naming.md`
**Beads:** `bd show orch-go-sg39y`
