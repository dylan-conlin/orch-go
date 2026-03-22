# OpenSCAD Hook Migration — Deployment Instructions

## What This Migration Does

Moves 3 OpenSCAD-specific Claude Code hooks from project-local `.claude/hooks/`
to global `~/.orch/hooks/`. This eliminates duplication across OpenSCAD projects
and enables automatic enforcement in any project that declares `domain: openscad`.

### Hook Changes

| Hook | Type | Detection |
|------|------|-----------|
| `gate-openscad-stl-cgal.py` | Command-pattern (PreToolUse) | Fires on `openscad` in Bash command — harmless in non-OpenSCAD projects |
| `gate-openscad-post-render.py` | Command-pattern (PostToolUse) | Fires on `openscad -o` success — finds `gates/geometry-check.sh` via project root |
| `gate-architect-production-files.py` | File-pattern (PreToolUse) | Checks `.harness/config.yaml` for `domain: openscad` before enforcing |

### Key Change: Post-Render Path Resolution

The original project-local hook found `geometry-check.sh` via relative path from
`.claude/hooks/`. The global version uses `git rev-parse --show-toplevel` to find
the project root, then looks for `gates/geometry-check.sh` relative to that.

### Key Change: Architect Hook Domain Guard

The original hook was OpenSCAD-specific with hardcoded patterns. The global version
reads `.harness/config.yaml` for `domain: openscad` before applying restrictions.
Projects without this config key are not affected. The `DOMAIN_PATTERNS` dict is
extensible for future domains.

## Deployment Steps (Run as orchestrator)

### Step 1: Install global hooks

```bash
cp .orch/staging/hooks/gate-openscad-stl-cgal.py ~/.orch/hooks/
cp .orch/staging/hooks/gate-openscad-post-render.py ~/.orch/hooks/
cp .orch/staging/hooks/gate-architect-production-files.py ~/.orch/hooks/
chmod +x ~/.orch/hooks/gate-openscad-stl-cgal.py
chmod +x ~/.orch/hooks/gate-openscad-post-render.py
chmod +x ~/.orch/hooks/gate-architect-production-files.py
```

### Step 2: Add hooks to global settings.json (~/.claude/settings.json)

Add these entries to the existing hooks arrays:

**PreToolUse** — add after existing entries:
```json
{
  "hooks": [
    {
      "command": "~/.orch/hooks/gate-openscad-stl-cgal.py",
      "timeout": 10000,
      "type": "command"
    }
  ],
  "matcher": "Bash"
},
{
  "hooks": [
    {
      "command": "~/.orch/hooks/gate-architect-production-files.py",
      "timeout": 10000,
      "type": "command"
    }
  ],
  "matcher": "Edit|Write"
}
```

**PostToolUse** — add new section (doesn't exist yet in global settings):
```json
"PostToolUse": [
  {
    "hooks": [
      {
        "command": "~/.orch/hooks/gate-openscad-post-render.py",
        "timeout": 120000,
        "type": "command"
      }
    ],
    "matcher": "Bash"
  }
]
```

### Step 3: Add domain config to OpenSCAD projects

```bash
mkdir -p ~/Documents/personal/led-magnetic-letters/.harness
echo -e "domain: openscad\nthresholds:\n  warning: 600\n  critical: 1000" > ~/Documents/personal/led-magnetic-letters/.harness/config.yaml

mkdir -p ~/Documents/personal/led-totem-toppers/.harness
echo -e "domain: openscad\nthresholds:\n  warning: 600\n  critical: 1000" > ~/Documents/personal/led-totem-toppers/.harness/config.yaml
```

### Step 4: Remove project-local OpenSCAD hooks

```bash
# led-magnetic-letters
rm ~/Documents/personal/led-magnetic-letters/.claude/hooks/gate-openscad-stl-cgal.py
rm ~/Documents/personal/led-magnetic-letters/.claude/hooks/gate-openscad-post-render.py
rm -rf ~/Documents/personal/led-magnetic-letters/.claude/hooks/__pycache__
```

```bash
# led-totem-toppers
rm ~/Documents/personal/led-totem-toppers/.claude/hooks/gate-openscad-stl-cgal.py
rm ~/Documents/personal/led-totem-toppers/.claude/hooks/gate-openscad-post-render.py
rm ~/Documents/personal/led-totem-toppers/.claude/hooks/gate-architect-production-files.py
```

### Step 5: Update project settings.json to remove hook entries

**led-magnetic-letters/.claude/settings.json** — remove the openscad hook entries,
keep only gate-git-add-all:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "hooks": [
          {
            "command": "python3 .claude/hooks/gate-git-add-all.py",
            "type": "command"
          }
        ],
        "matcher": "Bash"
      }
    ]
  },
  "permissions": {
    "deny": [
      "Edit(~/.claude/settings.json)",
      "Edit(~/.claude/settings.local.json)",
      "Write(~/.claude/settings.json)",
      "Write(~/.claude/settings.local.json)"
    ]
  }
}
```

**led-totem-toppers/.claude/settings.json** — remove all openscad hook entries,
keep only gate-git-add-all:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "hooks": [
          {
            "command": "python3 .claude/hooks/gate-git-add-all.py",
            "type": "command"
          }
        ],
        "matcher": "Bash"
      }
    ]
  },
  "permissions": {
    "deny": [
      "Edit(~/.claude/settings.json)",
      "Edit(~/.claude/settings.local.json)",
      "Write(~/.claude/settings.json)",
      "Write(~/.claude/settings.local.json)"
    ]
  }
}
```

### Step 6: Clean up staging

```bash
rm -rf .orch/staging/hooks/
```

## Verification

After deployment, test each hook:

1. **CGAL gate**: In an OpenSCAD project, try `openscad -o test.stl parts/something.scad` — should be blocked
2. **Post-render**: Run a successful `openscad -o test.png --backend manifold parts/something.scad` — should see geometry check message
3. **Architect deny**: Spawn an architect agent in an OpenSCAD project with `domain: openscad` in config.yaml — should block edits to `parts/*.scad`
4. **Non-OpenSCAD project**: Verify hooks pass through silently in orch-go (no `.harness/config.yaml` with `domain: openscad`, no `openscad` commands)
