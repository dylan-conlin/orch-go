## Probe-to-Model Merge (REQUIRED)

**Rule:** If you produced a probe file (`.kb/models/{name}/probes/*.md`) during this session, you MUST merge its findings into the parent `model.md` BEFORE reporting Phase: Complete.

**Why:** Probes that sit unmerged break the knowledge loop. The model is the authoritative synthesis — probes are evidence that feeds it. Without merging, future agents see stale models and re-investigate solved questions.

**Workflow:**

1. **Check:** Did you create or update any file in `.kb/models/*/probes/`?
   - If NO → skip this section
   - If YES → continue

2. **Read** the parent model: `.kb/models/{model-name}/model.md`

3. **Merge findings** into the model:
   - **Confirmed claims:** Strengthen language (e.g., "may" → "does")
   - **Contradicted claims:** Correct the model, note what changed
   - **New findings:** Add new sections or extend existing ones
   - **Quantitative data:** Update numbers, thresholds, measurements

4. **Add probe reference** to the model's "Evidence" or "Probes" section:
   - `- YYYY-MM-DD: [probe title] — [1-line finding summary]`

5. **Commit both** the probe and updated model together

**What NOT to do:**
- Don't just append the probe summary verbatim — synthesize into the model's structure
- Don't delete existing model content that your probe didn't examine
- Don't skip the merge because "the probe is self-contained"

---
