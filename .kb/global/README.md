# Global Knowledge Source

**This directory is the source for `~/.kb/`** - global knowledge shared across all projects.

## Architecture

```
~/.kb/  →  (symlink)  →  orch-knowledge/kb/
```

This means:
- Changes here propagate to all projects via `~/.kb/`
- Templates, principles, guides, and models are globally available
- `kb context` searches this when run with `--global` flag

## Why `kb/` instead of `.kb/`?

The kb CLI expects `.kb/` for **project-local** knowledge bases. This directory (`kb/`) deliberately breaks that convention because it serves a different purpose:

| Directory | Purpose | Convention |
|-----------|---------|------------|
| `.kb/` | Project-local investigations, decisions | Standard kb CLI |
| `kb/` | Global knowledge source (via symlink) | Unique to orch-knowledge |

## Contents

- `principles.md` - Foundational principles for the orchestration system
- `values.md` - Core values
- `templates/` - Templates for kb CLI commands
- `guides/` - Reusable procedural guides
- `models/` - System behavior models
- `decisions/` - Global architectural decisions
- `investigations/` - Globally-relevant investigations

## orch-knowledge Has Both Directories

orch-knowledge uniquely serves dual roles:

1. **Global Knowledge Source** (`kb/` → `~/.kb/`) - Shared with all projects
2. **Project-Local Knowledge** (`.kb/`) - Specific to orch-knowledge development

Don't confuse them:
- Edit `kb/` for content that should be globally available
- Edit `.kb/` for orch-knowledge project-specific investigations

## Related

- Investigation: `.kb/investigations/2026-01-27-inv-analyze-kb-directory-confusion-users.md`
- Ecosystem Guide: `kb/guides/orch-ecosystem.md`
