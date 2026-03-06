# Investigation Template

**When to use:** Reference when creating investigation file structure. Use `kb create investigation {slug}` to auto-generate.

## Full Template Structure

```markdown
## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered]
**Evidence:** [Primary evidence supporting conclusion]
**Knowledge:** [What was learned]
**Next:** [Recommended action]
**Authority:** [implementation | architectural | strategic] - Classification for routing to appropriate decision-maker

---

# Investigation: [Topic]

**Question:** [What are you trying to figure out?]
**Status:** Active | Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| [path-to-prior-investigation] | extends/confirms/contradicts/deepens | pending/yes | [conflicts found or "-"] |

> If no prior work exists: Add single row with "N/A - novel investigation"

## Findings
[Evidence gathered]

## Test performed
**Test:** [What you did to validate]
**Result:** [What happened]

## Conclusion
[Only fill if you tested]
```

## Section Guidelines

| Section | When to Fill | Key Rule |
|---------|--------------|----------|
| D.E.K.N. | END of investigation | One sentence per field |
| Question | START of investigation | Be specific |
| Prior Work | START of investigation | Entries OR "N/A - novel investigation" |
| Findings | During exploration | Add progressively |
| Test performed | After testing | Real test, not "reviewed code" |
| Conclusion | After test passes | Based on test results only |

## Prior Work Guidelines

**Relationship vocabulary:**
- **Extends:** Adds to prior findings (most common)
- **Confirms:** Validates prior hypothesis with new evidence
- **Contradicts:** Disproves or refines prior conclusion
- **Deepens:** Explores same question at greater depth

**Verified column:**
- Start with "pending"
- Update to "yes" when you test a cited claim during investigation
- Only verify claims you actually reference—not exhaustive upfront verification

**Conflicts column:**
- Document contradictions between prior claims and your evidence
- Example: "Prior claimed all restarts kill agents; only OpenCode restarts matter"
