# Probe: Context-scoping reducibility via spawn-source and skill-divergence rates

**Model:** `.kb/models/decidability-graph.md`
**Date:** 2026-02-09
**Status:** Complete

---

## Question

Is orchestrator context-scoping mostly reducible to a deterministic pipeline (daemon issue-type skill inference + kb auto-context), or does observed spawn behavior show non-trivial judgment edges?

---

## What I Tested

**Command/Code:**

```bash
python3 - <<'PY'
import json
path='/Users/dylanconlin/Documents/personal/orch-go/.beads/issues.jsonl'
rows=0
with_source=0
for line in open(path,encoding='utf-8',errors='ignore'):
    line=line.strip()
    if not line.startswith('{'): continue
    try:o=json.loads(line)
    except: continue
    rows+=1
    if 'source' in o:
        with_source+=1
print(f'issues_rows={rows}')
print(f'issues_with_source_field={with_source}')
PY

python3 - <<'PY'
import json,glob,os,datetime
from collections import defaultdict
root='/Users/dylanconlin/Documents/personal/orch-go'
records={}
for mp in glob.glob(os.path.join(root,'.orch','workspace','**','AGENT_MANIFEST.json'),recursive=True):
    sp=os.path.join(os.path.dirname(mp),'SPAWN_CONTEXT.md')
    if not os.path.exists(sp):
        continue
    try:m=json.load(open(mp))
    except: continue
    ws=m.get('workspace_name')
    if ws and ws not in records:
        records[ws]={'beads_id':m.get('beads_id'),'skill':m.get('skill'),'spawn_context':sp}
issues={}
for line in open(os.path.join(root,'.beads','issues.jsonl'),encoding='utf-8',errors='ignore'):
    line=line.strip()
    if not line.startswith('{'): continue
    try:o=json.loads(line)
    except: continue
    iid=o.get('id')
    if iid and iid not in issues: issues[iid]=o
session=[]; daemon=[]; bypass=[]
for line in open('/Users/dylanconlin/.orch/events.jsonl',encoding='utf-8',errors='ignore'):
    line=line.strip()
    if not line.startswith('{'): continue
    try:e=json.loads(line)
    except: continue
    t=e.get('type'); d=e.get('data') or {}; ts=e.get('timestamp')
    if not isinstance(ts,(int,float)): continue
    ts=int(ts)
    if t=='session.spawned' and d.get('workspace') in records:
        session.append((d.get('workspace'),d.get('beads_id'),d.get('skill'),ts))
    elif t=='daemon.spawn':
        daemon.append((d.get('beads_id'),d.get('skill'),ts))
    elif t=='spawn.triage_bypassed':
        bypass.append((d.get('skill'),ts))
bybid=defaultdict(list)
for bid,sk,ts in daemon:
    if bid: bybid[bid].append((sk,ts))
mapping={'bug':'systematic-debugging','feature':'feature-impl','task':'feature-impl','investigation':'investigation','question':'investigation'}
counts={'daemon':0,'manual':0}
explicit=0; explicit_type=0; explicit_div=0
custom=0; custom_daemon=0; custom_manual=0
for ws,bid,skill,ts in session:
    lines=open(records[ws]['spawn_context'],encoding='utf-8',errors='ignore').read().splitlines()
    ti=next((i for i,l in enumerate(lines) if l.startswith('SPAWN TIER:')),None)
    pre=lines[:ti] if ti is not None else lines[:20]
    has_custom=any(l.strip() for l in pre[1:])
    if has_custom: custom+=1
    src='daemon' if bid and any(abs(t-ts)<=300 for _,t in bybid.get(bid,[])) else 'manual'
    counts[src]+=1
    if src=='daemon' and has_custom: custom_daemon+=1
    if src=='manual' and has_custom: custom_manual+=1
    is_explicit=src=='manual' and any(abs(t-ts)<=300 and (not s or not skill or s==skill) for s,t in bypass)
    if is_explicit:
        explicit+=1
        it=(issues.get(bid) or {}).get('issue_type')
        inf=mapping.get(it)
        if inf:
            explicit_type+=1
            if skill and skill!=inf:
                explicit_div+=1
n=len(session)
print(f'classified_spawns={n}')
print(f'daemon={counts["daemon"]} ({counts["daemon"]/n:.2%})')
print(f'manual={counts["manual"]} ({counts["manual"]/n:.2%})')
print(f'explicit_manual={explicit} ({explicit/counts["manual"]:.2%} of manual)')
print(f'explicit_with_type={explicit_type}')
print(f'explicit_divergence={explicit_div} ({explicit_div/explicit_type:.2%})')
print(f'custom_context={custom} ({custom/n:.2%})')
print(f'custom_context_daemon={custom_daemon} ({custom_daemon/counts["daemon"]:.2%} of daemon)')
print(f'custom_context_manual={custom_manual} ({custom_manual/counts["manual"]:.2%} of manual)')
PY
```

**Environment:**

- Repo: `/Users/dylanconlin/Documents/personal/orch-go`
- Data sources: `.beads/issues.jsonl`, `.orch/workspace/**/AGENT_MANIFEST.json`, `.orch/workspace/**/SPAWN_CONTEXT.md`, `~/.orch/events.jsonl`
- Daemon inference mapping validated against `pkg/daemon/skill_inference.go`

---

## What I Observed

**Output:**

```text
issues_rows=3085
issues_with_source_field=0

classified_spawns=1015
daemon=429 (42.27%)
manual=586 (57.73%)
explicit_manual=558 (95.22% of manual)
explicit_with_type=420
explicit_divergence=206 (49.05%)
custom_context=525 (51.72%)
custom_context_daemon=382 (89.04% of daemon)
custom_context_manual=143 (24.40% of manual)
```

**Key observations:**

- Beads issues do not currently contain a `source` field (`0/3085`), so daemon/manual had to be inferred from event correlation (`daemon.spawn` near `session.spawned`).
- Daemon-driven spawns are a minority in this dataset (`42.27%`), with manual spawns dominant (`57.73%`).
- For explicit manual spawns with known issue type, selected skill diverges from type-based daemon inference about half the time (`49.05%`).
- More than half of spawn contexts include extra pre-KB task context (`51.72%`), indicating non-KB context shaping remains common.

---

## Model Impact

**Verdict:** extends — "context-scoping is the irreducible orchestrator function"

**Details:**
The data does not support a "90%+ deterministic" regime: daemon-driven inference is only ~42% of classified spawns, and explicit manual selections diverge from type inference ~49% of the time. This extends the model by separating two mechanisms: (1) reducible routing for a significant but minority path (daemon), and (2) high-judgment routing on the manual path where skill/type mismatch is frequent. Also, instrumentation gap exists: beads issue schema lacks `source`, forcing indirect reconstruction from events.

**Confidence:** Medium — rates are computed from 1,015 classified workspace-linked spawns, but source classification is inferred from event timing correlation because issues lack an explicit source field.
