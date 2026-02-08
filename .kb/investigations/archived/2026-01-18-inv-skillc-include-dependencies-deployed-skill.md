<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** skillc DOES include dependencies in frontmatter - issue was deploy directory mismatch (skills/ vs skills/src/).

**Evidence:** Compiler code exists (compiler.go:214-219); local build includes dependencies; deploy from src/ works correctly; deployed file now has dependencies field.

**Knowledge:** Deploy preserves source directory structure via relPath calculation - deploying from parent creates nested structure, deploying from src/ flattens correctly.

**Next:** close-no-action-needed (user workflow issue, not code bug)

**Promote to Decision:** recommend-no (operational issue, not architectural)

---

# Investigation: Skillc Include Dependencies Deployed Skill

**Question:** Why doesn't skillc include the dependencies field from skill.yaml in deployed SKILL.md frontmatter?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Compiler Code Already Writes Dependencies

**Evidence:** 
- Found code at `~/Documents/personal/skillc/pkg/compiler/compiler.go` lines 214-219:
```go
if len(inlineFrontmatter.Dependencies) > 0 {
    output.WriteString("dependencies:\n")
    for _, dep := range inlineFrontmatter.Dependencies {
        output.WriteString(fmt.Sprintf("  - %s\n", dep))
    }
}
```
- Manifest struct has Dependencies field: `Dependencies []string yaml:"dependencies"` (manifest.go:87)
- inlineFrontmatter is populated from skill.Manifest.Dependencies (compiler.go:130)

**Source:** 
- `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/compiler.go:214-219`
- `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/manifest.go:87`

**Significance:** The code to write dependencies already exists and is functional - no code bug present.

---

### Finding 2: Directory Structure Mismatch in Deployment

**Evidence:**
- Deploy from `~/orch-knowledge/skills` → files go to `~/.claude/skills/src/worker/investigation/`
- Deploy from `~/orch-knowledge/skills/src` → files go to `~/.claude/skills/worker/investigation/`
- orch-go loader finds `~/.claude/skills/worker/investigation/` via symlink
- Old deployment (Jan 17) at worker/investigation/ had no dependencies
- New deployment (Jan 18) from correct directory has dependencies

**Source:**
- `~/Documents/personal/skillc/cmd/skillc/main.go:1741-1761` - relPath calculation logic
- Symlink at `~/.claude/skills/investigation -> worker/investigation`

**Significance:** Deploy preserves source directory structure. Running from wrong directory caused files to deploy to nested location, leaving old files without dependencies in place.

---

### Finding 3: Verification Shows Feature Works

**Evidence:**
- Local build: `cd ~/orch-knowledge/skills/src/worker/investigation && skillc build` produced SKILL.md with dependencies
- Correct deploy: `cd ~/orch-knowledge/skills/src && skillc deploy --target ~/.claude/skills/` updated files correctly
- Deployed file now shows:
```yaml
---
name: investigation
skill-type: procedure
description: Record what you tried, what you observed, and whether you tested. Key discipline - you cannot conclude without testing.
dependencies:
  - worker-base
---
```
- orch-go tests pass: `go test ./pkg/skills -run TestLoadSkillWithDependencies` - PASS

**Source:**
- `~/.claude/skills/worker/investigation/SKILL.md:1-7`
- Test output from orch-go test suite

**Significance:** When deployed correctly, dependencies field is included and orch-go's LoadSkillWithDependencies() works as expected.

---

## Synthesis

**Key Insights:**

1. **No code bug exists** - skillc compiler already includes dependencies in frontmatter when `dependencies` field exists in skill.yaml and skill-type is set. The code at compiler.go:214-219 is functional and tested.

2. **Deploy directory matters** - The deploy command calculates relPath from source root to skill directory. Running deploy from parent of src/ preserves the src/ prefix in target path, causing structure mismatch with where orch-go expects skills.

3. **User workflow issue** - The perceived bug was actually incorrect usage. Correct workflow is: `cd ~/orch-knowledge/skills/src && skillc deploy --target ~/.claude/skills/`, which deploys to the flat structure orch-go expects.

**Answer to Investigation Question:**

skillc DOES include dependencies in deployed SKILL.md frontmatter. The issue reported was caused by deploying from the wrong directory (~/orch-knowledge/skills instead of ~/orch-knowledge/skills/src), which caused the deploy to preserve the "src/" directory prefix and deploy files to ~/.claude/skills/src/worker/investigation/ instead of ~/.claude/skills/worker/investigation/. orch-go's loader found the old files (without dependencies) at the expected location because the new files were deployed to a different nested location. When deploy is run from the correct directory, dependencies are correctly included in the frontmatter.

---

## Structured Uncertainty

**What's tested:**

- ✅ Compiler code writes dependencies to frontmatter (verified: read source code, traced execution path)
- ✅ Local build includes dependencies (verified: ran skillc build, inspected output)
- ✅ Deploy from src/ directory works correctly (verified: ran deploy, checked deployed file)
- ✅ orch-go LoadSkillWithDependencies reads frontmatter (verified: ran test suite, tests pass)

**What's untested:**

- ⚠️ Whether other users have the same directory structure issue (not surveyed)
- ⚠️ Whether documentation explains correct deploy directory (not checked)

**What would change this:**

- Finding would be wrong if compiler code at lines 214-219 was unreachable or conditional logic prevented execution
- Finding would be wrong if deployed file from src/ directory didn't include dependencies
- Finding would be wrong if orch-go tests failed after correct deployment

---

## Implementation Recommendations

**Purpose:** NO implementation needed - this is a user workflow issue, not a code bug.

### Recommended Approach ⭐

**No Code Changes** - Close issue as user error, optionally add documentation

**Why this approach:**
- Code works correctly when used properly
- Changing deploy logic could break existing workflows
- Directory structure preservation is intentional design

**Trade-offs accepted:**
- Users may continue to make this mistake without clearer documentation
- Could add warning when deploy detects "src/" in source path

**Implementation sequence:**
1. Close beads issue with explanation
2. (Optional) Add note to skillc deploy documentation about source directory selection
3. (Optional) Update orch-knowledge CI/CD scripts to deploy from correct directory

### Alternative Approaches Considered

**Option B: Auto-strip "src/" prefix in deploy**
- **Pros:** Would make deploy "just work" from any directory
- **Cons:** Could break users who intentionally want src/ in structure; changes established behavior; complex edge cases
- **When to use instead:** If multiple users report same issue and documentation doesn't help

**Option C: Add validation warning**
- **Pros:** Non-breaking, helps users discover mistake
- **Cons:** May create false positives for intentional structures
- **When to use instead:** If we want to guide users without changing behavior

**Rationale for recommendation:** No evidence of widespread issue; code works correctly; changing behavior risks breaking existing workflows.

---

### Implementation Details

**What to implement first:**
- Close the beads issue with "no action needed" + explanation
- Verify CI/CD scripts use correct deploy directory

**Things to watch out for:**
- N/A - no implementation

**Areas needing further investigation:**
- None - issue resolved

**Success criteria:**
- ✅ User understands correct deploy workflow
- ✅ Future deploys from src/ directory include dependencies

---

## References

**Files Examined:**
- `~/Documents/personal/skillc/pkg/compiler/compiler.go` - Frontmatter generation logic
- `~/Documents/personal/skillc/pkg/compiler/manifest.go` - Manifest struct definition
- `~/Documents/personal/skillc/cmd/skillc/main.go` - Deploy command implementation
- `~/.claude/skills/worker/investigation/.skillc/skill.yaml` - Source skill definition
- `~/.claude/skills/worker/investigation/SKILL.md` - Deployed output (verified fixed)
- `~/Documents/personal/orch-go/pkg/skills/loader.go` - How orch-go loads skills with dependencies

**Commands Run:**
```bash
# Rebuild skillc
cd ~/Documents/personal/skillc && make install

# Build investigation skill locally
cd ~/orch-knowledge/skills/src/worker/investigation && skillc build

# Deploy from correct directory
cd ~/orch-knowledge/skills/src && skillc deploy --target ~/.claude/skills/

# Verify output
head -10 ~/.claude/skills/worker/investigation/SKILL.md

# Test orch-go integration
cd ~/Documents/personal/orch-go && go test ./pkg/skills -run TestLoadSkillWithDependencies -v
```

**Related Artifacts:**
- **Decision:** None - operational issue, not architectural decision
- **Investigation:** None - first investigation of this issue
- **Workspace:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-arch-skillc-include-dependencies-18jan-48cf/`

---

## Investigation History

**2026-01-18 12:20:** Investigation started
- Initial question: Why doesn't skillc include dependencies field in deployed SKILL.md?
- Context: Bug report stated deployed SKILL.md missing dependencies despite skill.yaml having them

**2026-01-18 12:30:** Found compiler code exists
- Discovered lines 214-219 already write dependencies to frontmatter
- Realized this is not a code bug

**2026-01-18 12:35:** Identified deployment directory mismatch
- Traced deploy logic showing relPath calculation preserves source structure
- Discovered old files at worker/investigation/, new files at src/worker/investigation/

**2026-01-18 12:40:** Verified fix
- Deployed from correct directory (src/)
- Confirmed dependencies appear in deployed frontmatter
- Ran orch-go tests - all pass

**2026-01-18 12:45:** Investigation completed
- Status: Complete
- Key outcome: No code bug - user workflow issue. Deploy from src/ directory works correctly.
