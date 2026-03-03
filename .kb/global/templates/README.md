# Project Templates Directory

**⚠️ WARNING: This directory is now DEPRECATED**

**DO NOT EDIT FILES HERE**

Templates have been moved to the build system:

```
Source:        ~/meta-orchestration/templates-src/
Build:         orch build-global
Distribution:  ~/.orch/templates/
Consumption:   Projects reference ~/.orch/templates/
```

**To update templates:**
1. Edit files in `~/meta-orchestration/templates-src/`
2. Run `orch build-global`
3. Templates sync to `~/.orch/templates/`
4. All projects see updates

**Why this change:**
- Single source of truth (templates-src/)
- Automatic distribution (no manual copying)
- Version controlled (in meta-orchestration git)
- Follows package manager pattern (source → build → distribution)

**See decision:** `.orch/decisions/2025-11-15-global-orchestration-knowledge-distribution.md`

---

**This directory will be removed after migration complete.**
