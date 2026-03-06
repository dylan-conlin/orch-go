## Friction Capture

**Log friction when it happens, not at the end.** When you hit something that slows you down — a tool that fails, a hook that blocks, a missing capability, ceremony that takes longer than the fix — log it immediately:

```bash
bd comments add {{.BeadsID}} "Friction: <category>: <what happened>"
```

Categories: `bug` (broken behavior), `gap` (missing capability), `ceremony` (disproportionate process), `tooling` (tool UX issues)

Examples of friction worth logging:
- Hook blocked a valid action and you had to work around it
- A tool failed and you retried or switched approaches
- Process steps took longer than the actual work
- You needed a capability that doesn't exist

**Don't overthink it.** One sentence, in the moment. This data drives system improvements.

---

## Session Complete Protocol

**Git Staging Rule:** NEVER use `git add -A` or `git add .` — the workspace often has unrelated changes (.autorebuild.lock, .beads/, build/). Stage ONLY the specific files you created or modified for your task, by name.

**When your work is done (all deliverables ready), complete in this EXACT order:**

{{if eq .Tier "light"}}
1. Author/update `VERIFICATION_SPEC.yaml` in the workspace root.
   - Fill the pre-populated skeleton with the exact commands you ran, expectations you verified, and any manual steps still required.
2. **Probe-to-Model Merge:** If you created any probe files (`.kb/models/*/probes/*.md`), verify you merged findings into the parent `model.md` per the Probe-to-Model Merge section above. Completion will be REJECTED if probes exist without model updates.
3. If you didn't log any friction during the session: `bd comments add {{.BeadsID}} "Friction: none"`
4. Run: `bd comments add {{.BeadsID}} "Phase: Complete - "[1-2 sentence summary of deliverables]"` (report phase FIRST - before commit)
5. **Verify all .kb/ files are committed:**
   - Run: `git status --porcelain` and check for any .kb/ files (investigations, probes, decisions, etc.)
   - If uncommitted .kb/ files exist: `git add .kb/ && git commit -m "knowledge artifacts from session"`
   - This ensures probe files in .kb/models/{name}/probes/ are not left behind
6. Commit any remaining changes (including `VERIFICATION_SPEC.yaml`)
7. Run: `/exit` to close the agent session

**Light Tier:** SYNTHESIS.md is NOT required for this spawn.
{{else}}
1. Author/update `VERIFICATION_SPEC.yaml` in the workspace root.
   - Fill the pre-populated skeleton with the exact commands you ran, expectations you verified, and any manual steps still required.
2. **Probe-to-Model Merge:** If you created any probe files (`.kb/models/*/probes/*.md`), verify you merged findings into the parent `model.md` per the Probe-to-Model Merge section above. Completion will be REJECTED if probes exist without model updates.
3. If you didn't log any friction during the session: `bd comments add {{.BeadsID}} "Friction: none"`
4. Run: `bd comments add {{.BeadsID}} "Phase: Complete - "[1-2 sentence summary of deliverables]"` (report phase FIRST - before commit)
5. Ensure SYNTHESIS.md is created with these required sections:
   - **`Plain-Language Summary`** (REQUIRED): 2-4 sentences in plain language describing what you built/found/decided and why it matters. This is the scaffolding the orchestrator uses during completion review — write it for a human who hasn't read your code. No jargon without explanation. No "implemented X" without saying what X does.
   - **`Verification Contract`**: Link to `VERIFICATION_SPEC.yaml` and key outcomes
6. **Verify all .kb/ files are committed:**
   - Run: `git status --porcelain` and check for any .kb/ files (investigations, probes, decisions, etc.)
   - If uncommitted .kb/ files exist: `git add .kb/ && git commit -m "knowledge artifacts from session"`
   - This ensures probe files in .kb/models/{name}/probes/ are not left behind
7. Commit all remaining changes (including SYNTHESIS.md and `VERIFICATION_SPEC.yaml`)
8. Run: `/exit` to close the agent session
{{end}}

**Why this order matters:** If the agent dies after commit but before reporting Phase: Complete, the orchestrator cannot detect completion. Reporting phase first ensures visibility even if the agent dies before committing.

**Work is NOT complete until Phase: Complete is reported.**
The orchestrator cannot close this issue until you report Phase: Complete.

---
