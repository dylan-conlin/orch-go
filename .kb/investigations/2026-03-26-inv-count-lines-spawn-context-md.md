TLDR: I counted every `SPAWN_CONTEXT.md` under `.orch/workspace` and found 1,446 files. The shortest file is 190 lines, the longest is 3,673 lines, and the average length is 1,113.01 lines.

## Summary (D.E.K.N.)

**Delta:** `.orch/workspace` currently contains 1,446 `SPAWN_CONTEXT.md` files with a 190/3,673 min/max range and a 1,113.01-line average.

**Evidence:** A Python filesystem scan over `.orch/workspace` counted lines in every matching file and printed aggregate statistics plus min/max paths.

**Knowledge:** The distribution is wide and the extreme files both live in `archived/`, so archived workspaces materially affect overall prompt-size averages.

**Next:** Close this investigation unless someone wants follow-up segmentation such as active-vs-archived counts.

**Authority:** implementation - This is a scoped measurement task with no architectural or strategic decision.

---

# Investigation: Count Lines in SPAWN_CONTEXT.md Files

**Question:** How many lines are in each `SPAWN_CONTEXT.md` file under `.orch/workspace`, and what are the min, max, and average counts?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** OpenCode
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## What I tried

1. Verified the working directory with `pwd`.
2. Created an orphan investigation artifact with `kb create investigation count-lines-spawn-context-md --orphan`.
3. Enumerated matching files under `.orch/workspace`.
4. Ran a Python script that recursively counted lines for every `SPAWN_CONTEXT.md` file and computed summary statistics.
5. Ran a second Python script to inspect the shortest and longest examples.

## What I observed

- Total matching files: 1,446
- Minimum line count: 190
- Minimum-path example: `.orch/workspace/archived/og-work-write-blog-post-09mar-ea0a/SPAWN_CONTEXT.md`
- Maximum line count: 3,673
- Maximum-path example: `.orch/workspace/archived/og-work-audit-orch-go-27feb-950e/SPAWN_CONTEXT.md`
- Average line count: 1,113.01
- The top 5 shortest and top 5 longest files were all in `archived/`, indicating archived workspaces dominate the extremes.

## Test performed

```bash
python3 - <<'PY'
from pathlib import Path
root = Path('/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace')
files = sorted(root.rglob('SPAWN_CONTEXT.md'))
counts = [(str(p), sum(1 for _ in p.open('r', encoding='utf-8', errors='replace'))) for p in files]
print(f'files={len(counts)}')
values = [c for _, c in counts]
print(f'min={min(values)}')
for p, c in counts:
    if c == min(values):
        print(f'min_path={p}')
print(f'max={max(values)}')
for p, c in counts:
    if c == max(values):
        print(f'max_path={p}')
print(f'avg={sum(values)/len(values):.2f}')
PY
```

Observed output:

```text
files=1446
min=190
min_path=/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/archived/og-work-write-blog-post-09mar-ea0a/SPAWN_CONTEXT.md
max=3673
max_path=/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/archived/og-work-audit-orch-go-27feb-950e/SPAWN_CONTEXT.md
avg=1113.01
```

## Conclusion

The requested count is complete: across all `.orch/workspace/**/SPAWN_CONTEXT.md` files, the minimum is 190 lines, the maximum is 3,673 lines, and the arithmetic mean is 1,113.01 lines.

## References

**Files Examined:**
- `.orch/workspace/**/SPAWN_CONTEXT.md` - measured line counts across the full workspace tree.
- `.kb/investigations/2026-03-26-inv-count-lines-spawn-context-md.md` - recorded investigation evidence and conclusion.

**Commands Run:**
```bash
pwd
kb create investigation count-lines-spawn-context-md --orphan
python3 - <<'PY'
from pathlib import Path
root = Path('/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace')
files = sorted(root.rglob('SPAWN_CONTEXT.md'))
counts = [(str(p), sum(1 for _ in p.open('r', encoding='utf-8', errors='replace'))) for p in files]
print(f'files={len(counts)}')
if counts:
    values = [c for _, c in counts]
    avg = sum(values) / len(values)
    min_count = min(values)
    max_count = max(values)
    min_paths = [p for p, c in counts if c == min_count]
    max_paths = [p for p, c in counts if c == max_count]
    print(f'min={min_count}')
    for p in min_paths:
        print(f'min_path={p}')
    print(f'max={max_count}')
    for p in max_paths:
        print(f'max_path={p}')
    print(f'avg={avg:.2f}')
PY
python3 - <<'PY'
from pathlib import Path
root = Path('/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace')
files = sorted(root.rglob('SPAWN_CONTEXT.md'))
counts = []
for p in files:
    with p.open('r', encoding='utf-8', errors='replace') as f:
        counts.append((str(p), sum(1 for _ in f)))
print('top5_longest:')
for p, c in sorted(counts, key=lambda x: (-x[1], x[0]))[:5]:
    print(f'{c}\t{p}')
print('top5_shortest:')
for p, c in sorted(counts, key=lambda x: (x[1], x[0]))[:5]:
    print(f'{c}\t{p}')
PY
```

## Investigation History

**2026-03-26 00:00:** Investigation started
- Initial question: Count line totals for all workspace `SPAWN_CONTEXT.md` files and report min/max/avg.
- Context: Requested measurement across the spawn workspace tree.

**2026-03-26 00:00:** Aggregate statistics collected
- Recursive Python scan found 1,446 files with a 190-line minimum, 3,673-line maximum, and 1,113.01-line average.

**2026-03-26 00:00:** Investigation completed
- Status: Complete
- Key outcome: Reported the full workspace min, max, and average line counts for `SPAWN_CONTEXT.md` files.
