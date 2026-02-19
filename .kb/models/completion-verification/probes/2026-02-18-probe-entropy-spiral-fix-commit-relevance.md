# Probe: Entropy Spiral Fix Commit Relevance Audit

**Model:** completion-verification
**Date:** 2026-02-18
**Status:** Complete

---

## Question

Which fix commits from `entropy-spiral-feb2026` correspond to bugs still present on current `master`, versus fixes already present or now irrelevant due to code divergence?

---

## What I Tested

```bash
git merge-base master entropy-spiral-feb2026
git log --format=%H\t%s --grep '^fix:' <merge-base>..entropy-spiral-feb2026
git cherry -v master entropy-spiral-feb2026
git show <commit> | git apply --check
git show --name-only --pretty=format: <commit>
git cat-file -e master:<path>

python3 - <<'PY'
import subprocess, json
from pathlib import Path
repo = Path('/Users/dylanconlin/Documents/personal/orch-go')
workspace = repo / '.orch/workspace/og-inv-audit-fix-commits-18feb-0583'
def run(cmd, input_text=None):
    result = subprocess.run(cmd, input=input_text, text=True, capture_output=True)
    if result.returncode != 0:
        raise RuntimeError(f"Command failed: {' '.join(cmd)}\n{result.stderr}")
    return result.stdout
base = run(['git','merge-base','master','entropy-spiral-feb2026']).strip()
log_out = run(['git','log','--format=%H\t%s','--grep','^fix:','%s..entropy-spiral-feb2026' % base])
commits = []
for line in log_out.splitlines():
    if not line.strip():
        continue
    sha, subject = line.split('\t', 1)
    commits.append((sha, subject))
cherry_out = run(['git','cherry','-v','master','entropy-spiral-feb2026'])
cherry_minus = set()
for line in cherry_out.splitlines():
    if line.startswith('- '):
        parts = line.split()
        if len(parts) >= 2:
            cherry_minus.add(parts[1])
rows = []
counts = {'still-relevant': 0,'irrelevant': 0,'already-fixed': 0}
for sha, subject in commits:
    if sha in cherry_minus:
        category = 'already-fixed'
        apply_ok = None
        exists_count = None
        total_files = None
    else:
        patch = run(['git','show', sha])
        apply = subprocess.run(['git','apply','--check'], input=patch, text=True, capture_output=True)
        apply_ok = apply.returncode == 0
        files_out = run(['git','show','--name-only','--pretty=format:', sha])
        files = [f for f in files_out.splitlines() if f.strip()]
        total_files = len(files)
        exists_count = 0
        for f in files:
            exists = subprocess.run(['git','cat-file','-e', f'master:{f}'], capture_output=True)
            if exists.returncode == 0:
                exists_count += 1
        category = 'still-relevant' if apply_ok else 'irrelevant'
    counts[category] += 1
    rows.append({'sha': sha,'subject': subject,'category': category,'patch_applies': '' if apply_ok is None else str(apply_ok).lower(),'files_on_master': '' if exists_count is None else str(exists_count),'files_total': '' if total_files is None else str(total_files)})
workspace.mkdir(parents=True, exist_ok=True)
csv_path = workspace / 'fix-commit-audit.csv'
with csv_path.open('w', encoding='utf-8') as f:
    f.write('sha,subject,category,patch_applies,files_on_master,files_total\n')
    for row in rows:
        subject = row['subject'].replace('"','""')
        f.write(f"{row['sha']},\"{subject}\",{row['category']},{row['patch_applies']},{row['files_on_master']},{row['files_total']}\n")
summary_path = workspace / 'fix-commit-audit-summary.json'
with summary_path.open('w', encoding='utf-8') as f:
    json.dump({'merge_base': base,'fix_commit_count': len(commits),'counts': counts}, f, indent=2)
print(json.dumps({'merge_base': base,'fix_commit_count': len(commits),'counts': counts,'csv_path': str(csv_path),'summary_path': str(summary_path)}, indent=2))
PY
```

---

## What I Observed

`entropy-spiral-feb2026` diverges from `master` at merge-base `0bca3dec40bf2b0f1840225b678617236944722a`. There are 161 commits matching `^fix:` in that branch range.

Patch-equivalence checks (`git cherry`) found zero fix commits already present on `master` (no patch-equivalent commits). Patch applicability checks (`git apply --check`) show 3 fixes still apply cleanly to `master`, with 158 failing to apply due to divergence.

Still-relevant fixes (patch applies cleanly):

- `8a259417c15ee61914849372ba6140035657f57d` - fix: adjust test timestamps to avoid boundary condition failures
- `6fc11e27d8cb0b9d890a64ff65f1f6092566e9e1` - fix: copy spawn artifacts to worktree so agents find SPAWN_CONTEXT.md in their CWD
- `8eee81af2dcac9029da797b3a64f94aab7a29a18` - fix: orch spawn --attach no longer steals terminal from orchestrator

Artifacts generated:

- `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-audit-fix-commits-18feb-0583/fix-commit-audit.csv`
- `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-audit-fix-commits-18feb-0583/fix-commit-audit-summary.json`

---

## Model Impact

- [ ] **Confirms** invariant: [which one]
- [ ] **Contradicts** invariant: [which one] — [what's actually true]
- [x] **Extends** model with: a concrete audit method + counts showing that most entropy-spiral fix commits no longer apply cleanly to current master (161 total fixes, 3 still-relevant by patch applicability, 0 patch-equivalent on master).

---

## Notes

Classification uses patch-equivalence (git cherry) and patch applicability (git apply --check). Fixes reimplemented differently on master could be missed by patch-equivalence, and apply failures can reflect refactors rather than true irrelevance.
