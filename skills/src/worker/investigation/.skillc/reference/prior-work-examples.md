# Prior Work Examples

**When to use:** Reference when filling Prior-Work table in investigations.

## Relationship Types

| Type | When to Use | Example |
|------|-------------|---------|
| **extends** | Adds to prior findings, builds on conclusions | "extends auth investigation with session timeout analysis" |
| **confirms** | Validates prior hypothesis with new evidence | "confirms database corruption hypothesis from prior investigation" |
| **contradicts** | Disproves or significantly refines prior conclusion | "contradicts claim that all restarts kill agents - only OpenCode restarts do" |
| **deepens** | Explores same question at greater depth | "deepens spawn latency investigation with profiling data" |

## Example Tables

### Novel Investigation (No Prior Work)

```markdown
## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |
```

### Extending Prior Work

```markdown
## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-01-26-inv-spawn-latency.md | extends | yes | None |
| .kb/investigations/2026-01-20-inv-claude-startup.md | confirms | yes | Prior claimed 3s startup; observed 5.2s average |
```

### Contradicting Prior Work

```markdown
## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-01-15-inv-agent-crashes.md | contradicts | yes | Prior blamed network timeouts; root cause is API rate limiting |
```

## Verification Status

| Status | Meaning |
|--------|---------|
| pending | Acknowledged but not yet tested |
| yes | Tested claim against primary source (code, test output) |

**Note:** You only need to verify claims you actually encounter and reference during your investigation. Not all prior work needs exhaustive upfront verification.
