---
linked_issues:
  - orch-go-1050
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Mapped the end-to-end skillc pipeline from source templates to deploy targets and the actual spawn-time load path for orchestrator skills, including all on-disk copies with timestamps.

**Evidence:** skillc deploy preserves relative paths from the provided source root, so running deploy from a skill directory flattens output to `target/SKILL.md`; orch-go loads skills exclusively from `~/.claude/skills/**/SKILL.md` at spawn time, not from `~/.opencode/skill/`.

**Knowledge:** The canonical orchestrator skill for orch-go is `~/.claude/skills/meta/orchestrator/SKILL.md` (loaded by `pkg/skills/loader.go`). Multiple stale copies exist (`~/.claude/skills/src/**`, `~/.opencode/skill/**`, `~/Documents/personal/orch-cli/skills/**`) because deploy never cleans old directories and OpenCode scans multiple roots.

**Next:** Align deploy workflow to always run from `~/orch-knowledge/skills/src` (or pass that path explicitly), and clean stale targets that can be discovered by OpenCode or humans.

---

# Investigation: Full Skillc Pipeline Audit

**Question:** How does skillc compile and deploy orchestrator skills, where do copies land on disk, and which file is actually loaded at spawn time?

**Started:** 2026-02-18
**Updated:** 2026-02-18
**Owner:** orch-go-1050
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Pipeline Map (Source -> Compile -> Deploy -> Load)

```
~/orch-knowledge/skills/src/meta/orchestrator/.skillc/
  skill.yaml
  SKILL.md.template
  reference/
        |
        | skillc build (CompileWithStats)
        v
  ~/orch-knowledge/skills/src/meta/orchestrator/SKILL.md
        |
        | skillc deploy --target <target-dir> <source-root>
        v
  <target-dir>/<relPath>/SKILL.md
        |
        | orch spawn
        v
  ~/.claude/skills/**/SKILL.md (loaded at spawn time)
```

Key: `relPath` is computed from `absSourcePath` to the parent of `.skillc`. If you run deploy from inside a skill directory, `relPath` becomes `.`, so the output lands at `target/SKILL.md` (root) rather than `target/meta/orchestrator/SKILL.md`.

---

## Findings

### Finding 1: Source-of-truth lives in orch-knowledge `.skillc` directories

**Evidence:** `~/orch-knowledge/skills/src/meta/orchestrator/.skillc/` contains `skill.yaml` and `SKILL.md.template`. The manifest sets `output: SKILL.md` and `type: skill`, which triggers SKILL.md frontmatter emission.

**Source:** `/Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/.skillc/skill.yaml`

**Significance:** This is the only authoritative place to edit orchestrator skill content. All other copies are generated artifacts and should be treated as deploy targets or stale caches.

---

### Finding 2: Compile step determines output filename from manifest output

**Evidence:** `compiler.CompileWithOutput()` sets `outputPath` based on manifest `output` (default `CLAUDE.md`) and joins it with the base dir. The orchestrator manifest specifies `output: SKILL.md` so compilation emits `SKILL.md` in the parent directory of `.skillc`.

**Source:** `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/compiler.go`

**Significance:** This is why `SKILL.md` appears directly under `skills/src/meta/orchestrator/` after local builds.

---

### Finding 3: Deploy path preserves directory structure relative to the provided source root

**Evidence:** `handleDeploy()` computes `relPath = filepath.Rel(absSourcePath, baseDir)` where `baseDir` is the parent of `.skillc`. The target output directory is `targetDir/relPath`. If `absSourcePath` is the skill directory (e.g., `.../skills/src/meta/orchestrator`), `relPath` is `.`, so deploy writes to `targetDir/SKILL.md`. If `absSourcePath` is the higher-level root (`.../skills/src`), `relPath` becomes `meta/orchestrator`, so deploy writes to `targetDir/meta/orchestrator/SKILL.md`.

**Source:** `/Users/dylanconlin/Documents/personal/skillc/cmd/skillc/main.go` (deploy logic)

**Significance:** This explains the recurring mismatch where deploys land in `~/.opencode/skill/SKILL.md` instead of `~/.opencode/skill/meta/orchestrator/SKILL.md` when deploy is run from inside a skill directory.

---

### Finding 4: Deployed copies on disk (timestamps + staleness)

**Evidence:** The following orchestrator skill files exist with the listed modification times:

- `/Users/dylanconlin/.claude/skills/meta/orchestrator/SKILL.md` (Feb 18 14:13:33 2026) - Canonical orch-go load target
- `/Users/dylanconlin/.config/opencode/agent/orchestrator.md` (Feb 18 14:13:33 2026) - Canonical OpenCode agent file
- `/Users/dylanconlin/.opencode/skill/SKILL.md` (Feb 18 14:08:55 2026) - Flattened deploy output (depends on deploy root)
- `/Users/dylanconlin/.opencode/skill/meta/orchestrator/SKILL.md` (Feb 18 13:14:50 2026) - Older than canonical
- `/Users/dylanconlin/.claude/skills/src/meta/orchestrator/SKILL.md` (Feb 18 09:18:39 2026) - Extra deploy target (stale copy)
- `/Users/dylanconlin/.claude/skills/skills/src/meta/orchestrator/SKILL.md` (Feb 18 09:17:28 2026) - Extra deploy target (stale copy)
- `/Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/SKILL.md` (Feb 11 10:53:31 2026) - Local compile artifact
- `/Users/dylanconlin/Documents/personal/orch-cli/skills/orchestrator/SKILL.md` (Dec 14 13:18:20 2025) - Legacy repo copy

**Source:** `stat -f "%Sm %N" ...` output

**Significance:** Multiple stale copies persist because `skillc deploy` only writes new output; it never cleans old paths. These stale copies are discoverable by OpenCode skill scanners or humans, leading to accidental edits in the wrong file.

---

### Finding 5: Orch-go load path is fixed to ~/.claude/skills/

**Evidence:** `skills.DefaultLoader()` hardcodes `~/.claude/skills` as the skill root. Spawn loads skill content via `LoadSkillWithDependencies()` and embeds it into SPAWN_CONTEXT at spawn time.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/skills/loader.go`, `/Users/dylanconlin/Documents/personal/orch-go/pkg/orch/extraction.go`

**Significance:** Deploying only to `~/.opencode/skill/` will not affect orch-go spawns. The only orch-go load path that matters is under `~/.claude/skills/**/SKILL.md`.

---

### Finding 6: OpenCode agent injection uses flat agent files, not SKILL.md

**Evidence:** `skillc deploy --agent-dir` generates `~/.config/opencode/agent/{skill}.md` with frontmatter and the compiled body. `orchestrator.md` shows the deployed header and is timestamped with the latest compile.

**Source:** `/Users/dylanconlin/Documents/personal/skillc/cmd/skillc/main.go` (deployAgentFile), `/Users/dylanconlin/.config/opencode/agent/orchestrator.md`

**Significance:** For OpenCode sessions, the canonical orchestrator skill is the agent file, not the SKILL.md under `.opencode/skill/`.

---

### Finding 7: No hot reload for running sessions

**Evidence:** Spawn loads skill content once via `LoadSkillWithDependencies()` and embeds it in SPAWN_CONTEXT. There is no re-read path in spawn for already-running sessions.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/orch/extraction.go`

**Significance:** Deploying updated skills does not change any active session. The session must be restarted to pick up new skill content.

---

## Points of Confusion (Root Causes)

1. **Deploy root ambiguity:** Running `skillc deploy` from inside a skill directory flattens output to `target/SKILL.md`, which looks like a valid deploy but bypasses the expected `meta/orchestrator` path.
2. **Multiple skill roots:** Orchestrator can appear in `~/.claude/skills/`, `~/.opencode/skill/`, `~/.config/opencode/agent/`, and legacy `~/.claude/skills/src/**` locations. There is no cleanup step or warning about stale copies.
3. **Hot reload assumptions:** Deploy updates only apply on new spawns; this isn't signaled to operators, leading to confusion when a running session doesn't reflect recent changes.

---

## Recommendation: What the Pipeline Should Look Like

- **Single source of truth:** edit only `~/orch-knowledge/skills/src/**/.skillc/` files.
- **Canonical deploy command:** `cd ~/orch-knowledge/skills/src && skillc deploy --target ~/.claude/skills --agent-dir ~/.config/opencode/agent`.
- **Optional secondary deploy:** `skillc deploy --target ~/.opencode/skill` only if OpenCode skill scanning depends on it; otherwise deprecate.
- **Cleanup step:** remove legacy directories (`~/.claude/skills/src/`, `~/.claude/skills/skills/src/`, `~/Documents/personal/orch-cli/skills/`) or mark them read-only with guardrails.
- **Restart-required signal:** document in deploy output or operator checklist that active sessions must be restarted to pick up skill changes.

---

## Structured Uncertainty

**What is tested:** deploy path mechanics (from code), current on-disk copies and timestamps, and orch-go skill load path.

**What is inferred:** OpenCode's internal skill discovery behavior for `~/.opencode/skill/**/SKILL.md` (not tested here), but agent file generation is confirmed.

---

## References

**Files Examined:**
- `/Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/.skillc/skill.yaml`
- `/Users/dylanconlin/Documents/personal/skillc/cmd/skillc/main.go`
- `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/compiler.go`
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/skills/loader.go`
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/orch/extraction.go`

**Commands Run:**
```bash
pwd
which skillc
file /Users/dylanconlin/bin/skillc
stat -f "%Sm %N" /Users/dylanconlin/.claude/skills/meta/orchestrator/SKILL.md \
  /Users/dylanconlin/.opencode/skill/meta/orchestrator/SKILL.md \
  /Users/dylanconlin/.opencode/skill/SKILL.md \
  /Users/dylanconlin/.claude/skills/src/meta/orchestrator/SKILL.md \
  /Users/dylanconlin/.claude/skills/skills/src/meta/orchestrator/SKILL.md \
  /Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/SKILL.md \
  /Users/dylanconlin/Documents/personal/orch-cli/skills/orchestrator/SKILL.md \
  /Users/dylanconlin/.config/opencode/agent/orchestrator.md
```

**Related Artifacts:**
- `.kb/models/orchestrator-session-lifecycle/probes/2026-02-17-orchestrator-skill-injection-path-trace.md`
