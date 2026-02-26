## Summary (D.E.K.N.)

**Delta:** Four orch ecosystem repos were missing GitHub remotes; all now configured and pushed.

**Evidence:** `git remote -v` showed no remotes for ~/.kb, ~/orch-knowledge, ~/Documents/personal/kb-cli; opencode was missing fork remote. Created repos via `gh repo create` and pushed successfully.

**Knowledge:** Private repos created by default (safer for personal knowledge bases). Opencode fork uses `dev` branch, not `main`.

**Next:** Close - all ecosystem repos now have GitHub backup.

**Promote to Decision:** recommend-no (operational fix, not architectural)

---

# Investigation: Ensure Orch Ecosystem Repos Github

**Question:** Which orch ecosystem repos are missing GitHub remotes, and how should they be configured?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Three repos had no GitHub remotes configured

**Evidence:** 
- `~/.kb`: No remote (`git remote -v` returned empty)
- `~/orch-knowledge`: No remote
- `~/Documents/personal/kb-cli`: No remote

**Source:** `git remote -v` commands run in each directory

**Significance:** These repos had local-only version control, risking data loss and preventing remote collaboration/backup.

---

### Finding 2: Opencode was missing fork remote

**Evidence:** 
- Origin pointed to upstream `https://github.com/sst/opencode`
- No `fork` remote for Dylan's modifications

**Source:** `git remote -v` in ~/Documents/personal/opencode

**Significance:** Dylan's custom fixes (cross-project session attach, etc.) were only in local commits without remote backup.

---

### Finding 3: Repos with existing remotes were properly configured

**Evidence:**
- `~/Documents/personal/orch-go`: Has origin at `git@github.com:dylan-conlin/orch-go.git`
- `~/Documents/personal/beads`: Has both `origin` (upstream) and `fork` (Dylan's) remotes

**Source:** `git remote -v` commands

**Significance:** These served as the model for how other repos should be configured.

---

## Synthesis

**Key Insights:**

1. **Private by default** - Created new repos as private since they contain personal knowledge bases and custom code.

2. **Fork pattern for upstream projects** - Opencode follows the standard fork pattern: `origin` points to upstream (sst/opencode), `fork` points to Dylan's modifications.

3. **Branch naming varies** - Most repos use `main`, but opencode uses `dev` as its primary branch.

**Answer to Investigation Question:**

Four repos needed GitHub configuration:
- `~/.kb` → Created new private repo `dylan-conlin/kb`
- `~/orch-knowledge` → Created new private repo `dylan-conlin/orch-knowledge`
- `~/Documents/personal/kb-cli` → Created new private repo `dylan-conlin/kb-cli`
- `~/Documents/personal/opencode` → Forked `sst/opencode` to `dylan-conlin/opencode`, added `fork` remote

All repos successfully pushed to their new remotes.

---

## Structured Uncertainty

**What's tested:**

- ✅ All repos pushed successfully (verified via `git push` output)
- ✅ Remotes configured correctly (verified via `git remote -v`)
- ✅ GitHub repos created (verified via `gh repo create` success messages)

**What's untested:**

- ⚠️ Clone from new remotes works (not tested, but standard GitHub behavior)
- ⚠️ CI/CD if added later (no pipelines exist yet)

**What would change this:**

- SSH key issues would prevent push/pull
- GitHub org policy changes could affect private repo creation

---

## Implementation Recommendations

**Recommended Approach ⭐**

**All ecosystem repos now have GitHub remotes** - No further implementation needed.

**Repos created:**
1. https://github.com/dylan-conlin/kb (private)
2. https://github.com/dylan-conlin/orch-knowledge (private)
3. https://github.com/dylan-conlin/kb-cli (private)
4. https://github.com/dylan-conlin/opencode (fork)

**Future consideration:** Consider making some repos public if they could benefit the community (e.g., skills in orch-knowledge).

---

## References

**Commands Run:**
```bash
# Check remotes across all repos
git remote -v  # in each repo directory

# Create private repos
gh repo create dylan-conlin/kb --private --description "Global knowledge base (~/.kb)"
gh repo create dylan-conlin/orch-knowledge --private --description "Orchestration skills, patterns, and templates"
gh repo create dylan-conlin/kb-cli --private --description "Knowledge base CLI tool"

# Fork opencode
gh repo fork sst/opencode --clone=false

# Add remotes and push
git remote add origin git@github.com:dylan-conlin/[repo].git
git push -u origin main

# For opencode (uses dev branch)
git remote add fork git@github.com:dylan-conlin/opencode.git
git push fork dev --force-with-lease
```

---

## Investigation History

**2026-01-08 17:00:** Investigation started
- Initial question: Which orch ecosystem repos need GitHub remotes?
- Context: Found ~/.kb and ~/orch-knowledge had no remotes during prior work

**2026-01-08 17:15:** Audit complete, 4 repos identified as needing remotes

**2026-01-08 17:20:** All repos created and pushed

**2026-01-08 17:25:** Investigation completed
- Status: Complete
- Key outcome: All orch ecosystem repos now have GitHub backup
