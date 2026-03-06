# Error Recovery Guide

**When to use:** Consult this when encountering fatal errors during investigation.

## Error Documentation Procedure

**If you encounter a fatal error during exploration:**

1. **Before doing anything else**, add a finding to your investigation file:
   ```markdown
   ### Finding N: ERROR - [brief description]

   **Error:** [Full error message]

   **Context:** [What you were attempting when error occurred]

   **Significance:** [Why this blocks progress or what it reveals]
   ```

2. Commit immediately: `git add .kb/investigations/*.md && git commit -m "investigation: {slug} - error encountered"`

3. Report via beads: `bd comments add <beads-id> "ERROR: [error summary] - see investigation file"`

4. If error is recoverable, continue. If fatal, the investigation file now has a record of what happened.

## Example Errors to Document

- API rate limits or size limits (e.g., "100 PDF pages max")
- Tool failures or missing dependencies
- Context exhaustion warnings
- External service unavailability
