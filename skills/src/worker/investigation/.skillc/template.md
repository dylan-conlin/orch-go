## Template

Choose template based on SPAWN_CONTEXT mode detection.

### Probe Mode (default when injected model markers are present)

Use `.orch/templates/PROBE.md`. Write to `.kb/models/{model-name}/probes/{date}-{slug}.md`.

Required frontmatter:

- **claim:** `XX-NN` — the claim ID being probed (e.g., `AE-08`, `KA-03`). Extract from the beads issue label `claim:{id}` or description `Claim: ...`. **Without this, the completion pipeline cannot close the loop.**
- **verdict:** `confirms` | `contradicts` | `extends` — your evidence-based conclusion about the claim.

Required sections:

- **Question**
- **What I Tested** (actual command/code executed)
- **What I Observed** (concrete output)
- **Model Impact** (confirms | contradicts | extends)

### Investigation Mode (fallback when model markers are absent)

Use `kb create investigation {slug} --model <model-name>` (or `--orphan` if no model applies). Required sections:

- **D.E.K.N. Summary** (Delta, Evidence, Knowledge, Next)
- **Prior Work** table (entries OR "N/A - novel investigation")
- **Question** and **Status**
- **Findings** (add progressively)
- **Test performed** (not "reviewed code" - actual test)
- **Conclusion** (only if you tested)

### Prior-Work Table Structure

```markdown
## Prior Work

| Investigation                          | Relationship | Verified | Conflicts |
| -------------------------------------- | ------------ | -------- | --------- |
| .kb/investigations/2026-01-26-inv-X.md | extends      | pending  | -         |
| N/A - novel investigation              | -            | -        | -         |
```

**Relationship vocabulary:**

- **Extends:** Adds to prior findings (most common)
- **Confirms:** Validates prior hypothesis with new evidence
- **Contradicts:** Disproves or refines prior conclusion
- **Deepens:** Explores same question at greater depth

**Verified column:** Start with "pending", update to "yes" when you test a cited claim during investigation.

**Conflicts column:** Document contradictions found during verification.

**Reference:** See `~/.claude/skills/worker/investigation/reference/template.md` for full structure and `reference/examples.md` for common failures.

### Sources Section

If your investigation references external URLs (documentation, APIs, articles, papers), add a `## Sources` section at the end. **All URLs must use markdown hyperlink syntax** — never bare URLs or plain-text descriptions.

```markdown
## Sources

- [Claude Code CLI Reference](https://code.claude.com/docs/en/cli-usage)
- [IAEA INSAG-10: Defence in Depth](https://www-pub.iaea.org/MTCD/Publications/PDF/Pub1013e_web.pdf)
```

**Wrong** (plain text, not clickable in exports):
```
- claude.ai/pricing — Claude pricing page
- https://code.claude.com/docs/en/cli-usage
```

**Why:** Plain-text URLs become dead text in Google Docs and other exports. Markdown hyperlinks convert to clickable links.

---

## When Not to Use

- **Fixing bugs** → Use `systematic-debugging`
- **Trivial questions** → Just answer them
- **Documentation** → Use `capture-knowledge`
