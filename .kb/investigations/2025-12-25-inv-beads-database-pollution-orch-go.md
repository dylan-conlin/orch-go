<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Cross-repo beads config (`additional: [beads]`) caused 787 bd-* and 18 kb-cli-* issues to pollute orch-go database, plus created nested `.beads/.beads/` directory.

**Evidence:** issues.jsonl had 1303 lines, only 498 were orch-go-*; git history shows config with `additional: ["/Users/.../beads"]` added in commit 38e79ef.

**Knowledge:** Beads multi-repo config imports ALL issues from additional repos into primary repo's JSONL - this is likely unintended behavior for most use cases.

**Next:** Close issue. Consider filing beads upstream issue about multi-repo config documentation/warnings.

**Confidence:** Very High (95%) - Root cause identified, fix verified, database clean.

---

# Investigation: Beads Database Pollution Orch Go

**Question:** Why does orch-go .beads/ contain 786+ bd-* prefixed issues that belong to the beads repo, and how to permanently clean it?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** debugging agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Git-tracked config.yaml had cross-repo reference

**Evidence:** 
```yaml
# Committed in 38e79ef
{repos: {primary: ".", additional: ["/Users/dylanconlin/Documents/personal/beads"]}}
```

**Source:** `git show HEAD:.beads/config.yaml` (before fix), `git log -p -S 'additional' -- .beads/config.yaml`

**Significance:** The `additional` key in beads config causes `bd sync` to import issues from all listed repos into the primary repo's database and JSONL. This was the root cause of pollution.

---

### Finding 2: Pollution included 787 bd-* and 18 kb-cli-* issues

**Evidence:**
```
$ jq -r '.id' .beads/issues.jsonl | cut -d'-' -f1-2 | sort | uniq -c | sort -rn
 498 orch-go  # legitimate
  18 kb-cli   # contamination
 787 bd-*     # contamination (individual bd-xxxx prefixes)
```

**Source:** `jq` analysis of polluted issues.jsonl (1303 total lines)

**Significance:** The pollution represented 60%+ of the database, making it difficult to work with legitimate orch-go issues. The bd-* issues came from beads repo, kb-cli-* from a previously configured additional repo.

---

### Finding 3: Nested .beads/.beads/ directory was git-tracked

**Evidence:**
```
$ ls -la .beads/.beads/
-rw-------  .gitignore
-rw-r--r--  issues.jsonl  # 1.2MB with same pollution
```

**Source:** `ls -la .beads/.beads/`, `git status .beads/`

**Significance:** This nested directory is a beads anti-pattern. It was created when the multi-repo config synced data from the beads repo (which has its own .beads/). Updated .gitignore to prevent this in future.

---

### Finding 4: Original spawn issue bd-37bw didn't survive cleanup

**Evidence:**
```
$ jq 'select(.id == "bd-37bw")' .beads/issues.jsonl.polluted
{
  "id": "bd-37bw",
  "title": "[orch-go] systematic-debugging: Beads database pollution..."
}
```

**Source:** Polluted backup file comparison, bd list after cleanup

**Significance:** The issue created to track this work was created with wrong prefix (bd-* instead of orch-go-*) due to the pollution. Created new issue orch-go-mazg with correct prefix.

---

### Finding 5: kb-cli has similar cross-repo config issue

**Evidence:**
```yaml
# kb-cli/.beads/config.yaml
repos:
  primary: "."
  additional:
    - "/Users/dylanconlin/Documents/personal/orch-go"
```

Database error: `235 orphaned dependencies (issue_id not in issues)`

**Source:** `cat ~/Documents/personal/kb-cli/.beads/config.yaml`, `bd stats` in kb-cli

**Significance:** kb-cli needs similar cleanup. The JSONL is clean (18 kb-cli issues) but the database has corruption from cross-repo references.

---

## Synthesis

**Key Insights:**

1. **Multi-repo config is dangerous without guardrails** - The beads `additional` config silently imports all issues from referenced repos. This can pollute your database with unrelated issues that have different prefixes.

2. **Nested .beads directories indicate cross-repo pollution** - When `.beads/.beads/` exists, it's a strong signal that the repo has been configured with `additional` repos pointing to other beads-enabled repos.

3. **Issue prefix indicates origin** - Issues with bd-* prefix came from beads repo, kb-cli-* from kb-cli repo. The prefix is the primary indicator of which repo owns an issue.

**Answer to Investigation Question:**

The pollution came from git-tracked `config.yaml` having `additional: ["/Users/.../beads"]` which caused `bd sync` to import 787 bd-* issues from the beads repo. The fix was:
1. Filter issues.jsonl to keep only orch-go-* issues (498 of 1303)
2. Remove nested .beads/.beads/ directory
3. Fix config.yaml to only have `primary: "."`
4. Update .gitignore to prevent nested beads directories
5. Reinitialize database with `bd init --prefix orch-go`

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Root cause clearly identified in git history, fix applied and verified, database confirmed clean with only 498 orch-go-* issues.

**What's certain:**

- ✅ The `additional: [beads]` config was the root cause (git history confirms)
- ✅ The clean database has only orch-go-* prefixed issues (verified with jq/bd stats)
- ✅ The .gitignore update prevents nested .beads/ pollution

**What's uncertain:**

- ⚠️ Whether beads upstream considers this behavior a bug or a feature
- ⚠️ Whether kb-cli cleanup will be as straightforward (different corruption type)

**What would increase confidence to Very High:**

- Upstream beads documentation clarifying multi-repo config intended behavior
- Verified kb-cli cleanup follows same pattern

---

## Implementation Recommendations

### Recommended Approach ⭐

**Filter-and-reinitialize** - Filter JSONL to keep only correct-prefix issues, remove nested directories, reinitialize database.

**Why this approach:**
- Clean separation between polluted and legitimate data
- Database rebuild ensures no orphaned references
- Git history preserved for audit

**Trade-offs accepted:**
- Comments on polluted issues are lost (acceptable - they were never ours)
- Need to recreate tracking issue with correct prefix

**Implementation sequence:**
1. Filter issues.jsonl: `jq -c 'select(.id | startswith("orch-go-"))' issues.jsonl > clean.jsonl`
2. Remove nested dirs: `rm -rf .beads/.beads/`
3. Fix config.yaml: Remove `additional` key
4. Update .gitignore: Add `.beads/` to prevent nested pollution
5. Commit all changes
6. Reinitialize: `rm .beads/beads.db* && bd init --prefix orch-go`

---

## References

**Files Examined:**
- `.beads/config.yaml` - Root cause config
- `.beads/issues.jsonl` - Polluted issue data
- `.beads/.beads/issues.jsonl` - Nested pollution

**Commands Run:**
```bash
# Count issue prefixes
jq -r '.id' .beads/issues.jsonl | cut -d'-' -f1-2 | sort | uniq -c

# Find config change in git
git log -p -S 'additional' -- .beads/config.yaml

# Filter clean issues
jq -c 'select(.id | startswith("orch-go-"))' .beads/issues.jsonl > clean.jsonl
```

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-debug-beads-database-pollution-25dec/`

---

## Investigation History

**2025-12-25 12:46:** Investigation started
- Initial question: Why does orch-go .beads/ contain 786+ bd-* prefixed issues?
- Context: Spawned from bd-37bw (which itself was polluted)

**2025-12-25 12:50:** Root cause identified
- Found `additional: [beads]` in git-tracked config.yaml
- Identified 787 bd-* + 18 kb-cli-* polluted issues

**2025-12-25 12:55:** Fix applied and verified
- Cleaned issues.jsonl to 498 orch-go-* issues
- Removed nested .beads/.beads/
- Committed fix with descriptive message
- Reinitialized database successfully

**2025-12-25 13:00:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Database cleaned, pollution mechanism understood, prevention measures added to .gitignore
