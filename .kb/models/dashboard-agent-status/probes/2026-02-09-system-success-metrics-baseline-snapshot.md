# Probe: Can orch success be defined by measurable system-level outcomes today?

**Model:** `.kb/models/dashboard-agent-status.md`
**Date:** 2026-02-09
**Status:** Complete

---

## Question

Does current telemetry support a stable system-success metric set based on entity outcomes (completion, time-to-value, abandonment/retries, and investigation promotion), without introducing new instrumentation first?

---

## What I Tested

**Command/Code:**
```bash
python3 - <<'PY'
import json,glob
from pathlib import Path
from datetime import datetime

def parse_ts(v):
    if v is None:
        return None
    s=str(v).strip()
    try:
        x=float(s)
        if x>1e15:
            return x/1e9
        if x>1e12:
            return x/1e3
        return x
    except Exception:
        pass
    if s.endswith('Z'):
        s=s[:-1]+'+00:00'
    try:
        return datetime.fromisoformat(s).timestamp()
    except Exception:
        return None

root=Path('/Users/dylanconlin/Documents/personal/orch-go')
issues=[]
for line in (root/'.beads/issues.jsonl').read_text().splitlines():
    if line.strip():
        issues.append(json.loads(line))
issue_by_id={i.get('id'):i for i in issues}

total=len(issues)
closed_success=sum(1 for i in issues if i.get('status')=='closed' and 'abandon' not in (i.get('close_reason') or '').lower())
completion_pct=(closed_success/total*100) if total else 0

samples=[]
for ws in glob.glob(str(root/'.orch/workspace/*')):
    p=Path(ws)
    bidp=p/'.beads_id'; stp=p/'.spawn_time'
    if not (bidp.exists() and stp.exists()):
        continue
    bid=bidp.read_text().strip()
    issue=issue_by_id.get(bid)
    if not issue:
        continue
    st=parse_ts(stp.read_text())
    ct=parse_ts(issue.get('closed_at'))
    if st is None or ct is None:
        continue
    mins=(ct-st)/60
    if mins>=0:
        samples.append(mins)
samples.sort()
def q(vals,p):
    if not vals: return None
    i=(len(vals)-1)*p
    lo=int(i); hi=min(lo+1,len(vals)-1); f=i-lo
    return vals[lo]*(1-f)+vals[hi]*f

events=Path.home()/'.orch/events.jsonl'
aband=0
missing_reason=0
by_issue={}
if events.exists():
    for line in events.read_text(errors='ignore').splitlines():
        if not line.strip():
            continue
        ev=json.loads(line)
        et=ev.get('type') or ev.get('event')
        if et!='agent.abandoned':
            continue
        data=ev.get('data') or {}
        bid=ev.get('beads_id') or data.get('beads_id')
        if not str(bid).startswith('orch-go-'):
            continue
        aband+=1
        reason=(ev.get('reason') or data.get('reason') or '').strip()
        if not reason:
            missing_reason+=1
        by_issue[bid]=by_issue.get(bid,0)+1
retry_ids=sum(1 for c in by_issue.values() if c>=2)

invs=list((root/'.kb/investigations').glob('*.md'))
ref_any=0
for inv in invs:
    name=inv.name
    hit=False
    for bucket in ['models','decisions','guides']:
        for t in (root/'.kb'/bucket).rglob('*.md'):
            if name in t.read_text(errors='ignore'):
                hit=True
                break
        if hit:
            break
    if hit:
        ref_any+=1

print(f'completion_ratio={closed_success}/{total} ({completion_pct:.2f}%)')
if samples:
    print(f'time_to_value_samples={len(samples)} p50={q(samples,0.5):.2f}m p90={q(samples,0.9):.2f}m p95={q(samples,0.95):.2f}m')
else:
    print('time_to_value_samples=0')
print(f'abandonment_events={aband} missing_reason={missing_reason} missing_reason_pct={(missing_reason/aband*100 if aband else 0):.2f}% retry_issue_ids_ge2={retry_ids}')
print(f'investigation_promotion={ref_any}/{len(invs)} ({(ref_any/len(invs)*100 if invs else 0):.2f}%)')
PY
```

**Environment:**
- Repository: `/Users/dylanconlin/Documents/personal/orch-go`
- Branch/worktree: `og-arch-define-system-level-09feb-d2ba`
- Sources queried: `.beads/issues.jsonl`, `.orch/workspace/*`, `~/.orch/events.jsonl`, `.kb/investigations/`, `.kb/{models,decisions,guides}/`

---

## What I Observed

**Output:**
```text
completion_ratio=3129/3157 (99.11%)
time_to_value_samples=141 p50=7.06m p90=73.53m p95=92.76m
abandonment_events=4 missing_reason=4 missing_reason_pct=100.00% retry_issue_ids_ge2=2
investigation_promotion=9/76 (11.84%)
```

**Key observations:**
- All proposed success dimensions are directly measurable from current artifacts (entity outcomes, timing, abandonment/retries, and knowledge conversion).
- Completion and timing are stable enough for thresholding now, while abandonment reason quality is currently too sparse to be used as a hard pass/fail gate.
- Promotion throughput is measurable and low, which makes it useful as a system-success metric rather than a nice-to-have.

---

## Model Impact

**Verdict:** extends - `dashboard-agent-status` entity-level observation invariants

**Details:**
This confirms the model's direction (entity-level metrics over event counting) and extends it into an explicit system-success metric set with measurable leading/lagging indicators. It also adds a practical caveat: abandonment reason completeness must be treated as data-quality maturity, not immediate failure gating, until reason coverage improves.

**Confidence:** High - Results are generated from primary artifacts with executable measurement logic.
