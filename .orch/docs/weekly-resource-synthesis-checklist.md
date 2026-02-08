# C5 Weekly Resource-Class Synthesis Checklist

Reference decision: `.kb/decisions/2026-02-07-unbounded-resource-consumption-constraints.md` (Constraint C5).

Run this every Friday until `kb reflect --type synthesis` can detect resource-class clusters automatically.

## Resource Categories

- `subprocess`
- `goroutine`
- `cache`
- `memory`
- `process`
- `connection`
- `OOM`
- `zombie`

## Friday Procedure

1. Run the count sweep for all categories:

```bash
RESOURCE_TERMS=(subprocess goroutine cache memory process connection OOM zombie)

for term in "${RESOURCE_TERMS[@]}"; do
  count=$(kb context "$term" --format json --limit 0 | python3 -c 'import json,sys; data=json.load(sys.stdin); paths={i.get("path") for i in (data.get("investigations") or []) if i.get("path")}; print(len(paths))')
  printf "%-12s %s\n" "$term" "$count"
done
```

2. For each category with count `>= 3`, validate relevance (remove false positives) by listing investigation paths:

```bash
kb context "<category>" --format json --limit 0 | python3 -c 'import json,sys; data=json.load(sys.stdin); paths=sorted({i.get("path") for i in (data.get("investigations") or []) if i.get("path")}); print("\n".join(paths))'
```

3. Escalate if a category still has `>= 3` relevant investigations:
   - Create a beads item for architectural follow-up:
     - `bd create "C5 escalation: <category> appears in <N> investigations" --type question -l triage:review`
   - Add a quick constraint signal so recurrence is searchable:
     - `kb quick constrain "Resource cluster: <category> (N=<N>)" --reason "C5 Friday synthesis <YYYY-MM-DD>"`
   - Link the top investigations in the issue comment so an architect/orchestrator can convert it into a durable constraint/decision.

4. Record this week's run in the current C5 task comment:
   - Counts table
   - Escalations created (or `none`)

5. Keep it recurring until automation lands:
   - Create next Friday task:
     - `bd create "C5 weekly resource-class investigation synthesis (<YYYY-MM-DD>)" --type task -l triage:review`
   - Include pointer to this checklist in the new task description.

## Exit Condition

Stop using this manual checklist only after `kb reflect --type synthesis` (or equivalent `kb reflect` lane) reliably reports resource-class clusters with the same `>= 3` escalation threshold.
