# Probe: LSP/Editor Auto-Restoration of sed Deletions

**Status:** Complete  
**Date:** 2026-02-14  
**Agent:** orch-go-fr3  

## Question

**Model Claim Being Tested:**  
"LSP/editor auto-restores sed deletions during agent file operations" (from agent orch-go-11z experience)

**Specific Questions:**
1. Does gopls auto-restore file deletions made by sed?
2. Does the Edit tool have the same issue?
3. What is the actual mechanism causing restoration?

## What I Tested

### Initial State Analysis
- **gopls running**: PID 85948 (confirmed active)
- **Extracted files exist**: session_handoff.go (716 lines), session_validation.go (226 lines), session_resume.go (220 lines)
- **session.go current size**: 911 lines (originally 2166 lines per issue description)

### Test Plan
1. Check for duplicate functions between session.go and extracted files
2. Test sed deletion with gopls running
3. Test Edit tool deletion with gopls running
4. Monitor file watching behavior

## What I Observed

### Investigation Timeline
1. **Agent orch-go-11z session** (11:02-11:16 on 2026-02-14):
   - Task: Decompose session.go (2166 lines) into 3 extracted files
   - Agent reported: "sed deletions being auto-restored by LSP/editor"
   - Created backup files: session.go.bak (2165 lines), session.go.bak2 (2166 lines)
   - At line 577 of session log: "Kill gopls and apply all deletions at once" → "Great! Now let me quickly build before it gets restored again"

2. **Actual git history** shows decomposition WAS completed:
   - 11:07:32: session_resume.go created (220 lines) - commit b9067948
   - 12:12:23: session_handoff.go (716 lines) + session_validation.go (226 lines) created - commit 413f3ec0
   - 12:39:56: session.go reduced by 1261 lines (2166 → 911) - commit 44c06369
   - All commits by "Test User" (agents), not Dylan

3. **Reproduction testing** (2026-02-14 with gopls PID 85948 running):
   
   **Test 1 - sed in /tmp (outside project):**
   - Created 13-line test file
   - Used `sed -i.bak '/^func deleteThis/,/^}/d'` to remove function
   - Result: 13 → 10 lines, NO restoration after 2s wait
   
   **Test 2 - sed in project directory (gopls-watched):**
   - Created cmd/orch/test_gopls_restore.go (13 lines)
   - Used same sed deletion command
   - Result: 13 → 10 lines, NO restoration after 3s wait
   
   **Test 3 - Edit tool in project directory:**
   - Created cmd/orch/test_edit_tool.go (14 lines)
   - Used Edit tool to delete function
   - Result: 14 → 9 lines, NO restoration after 3s wait

4. **Agent behavior analysis:**
   - Agent made multiple attempts to delete code
   - Agent created backups (.bak, .bak2) suggesting confusion about file state
   - Agent believed deletions were being "lost" or "restored"
   - After killing gopls, deletions "worked"
   - But this is likely coincidental - gopls was never the problem

### Root Cause Analysis

**The claimed issue does NOT exist.**

- **gopls does NOT auto-restore sed deletions**
- **The Edit tool does NOT have auto-restoration issues**
- **Both work correctly with gopls running**

**What likely happened to agent orch-go-11z:**

1. **Possible sed command errors:**
   - May have been working on backup files (.bak) instead of originals
   - May have used incorrect line ranges
   - Sed commands may have failed silently

2. **Git operations:**
   - Agent may have accidentally run `git restore` or `git checkout`
   - Looking at session log line 512: "use git to restore and then apply my changes"
   
3. **File handling confusion:**
   - Created multiple backups (session.go.bak, session.go.bak2)
   - May have been checking wrong file after edits
   - Two backup files differ by only 1 line (2165 vs 2166)

4. **Correlation ≠ Causation:**
   - Agent killed gopls → deletions worked
   - But deletions work fine WITH gopls running
   - Likely the agent fixed their sed commands or file handling, not the gopls issue

**The decomposition task WAS eventually completed successfully by other agents.**

## Model Impact

**This contradicts the reported claim** - there is NO systematic LSP/editor auto-restoration issue.

**Findings:**
1. ✅ gopls does not interfere with sed deletions
2. ✅ Edit tool works correctly with gopls running
3. ❌ Agent orch-go-11z misdiagnosed a file handling issue as an LSP problem
4. ✅ The actual decomposition work was completed successfully in subsequent commits

**Recommendation:**
- Close original issue orch-go-11z as "invalid/misdiagnosed"
- No action needed on LSP/gopls/editor
- Potential issue: Agents may jump to infrastructure blame when experiencing file operation confusion
- Could benefit from better error messages when sed/edit operations fail
