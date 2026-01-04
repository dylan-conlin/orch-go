# Changelog System Documentation

The changelog system aggregates git commits across Dylan's orchestration ecosystem, providing semantic analysis and integration with the orchestration workflow.

---

## Quick Reference

| Component | Location | Purpose |
|-----------|----------|---------|
| CLI command | `orch changelog` | Human-readable or JSON output |
| API endpoint | `GET /api/changelog` | Dashboard integration |
| Repo registry | `pkg/spawn/ecosystem.go` | Which repos to scan |
| Integration | `orch complete` | Surfaces notable changes at completion |

---

## CLI Usage

```bash
# Basic usage - last 7 days across all repos
orch changelog

# Extended time range
orch changelog --days 14

# Single project
orch changelog --project orch-go

# JSON output for scripting
orch changelog --json
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--days` | 7 | Number of days to include |
| `--project` | "all" | Filter to single repo or "all" for ecosystem |
| `--json` | false | Output as JSON instead of formatted text |

---

## API Endpoint

**Endpoint:** `GET /api/changelog`

**Query Parameters:**
- `days` (optional, default: 7) - Number of days to include
- `project` (optional, default: "all") - Project filter

**Response Structure:**
```json
{
  "date_range": {
    "start": "2026-01-01",
    "end": "2026-01-03"
  },
  "total_commits": 42,
  "repo_count": 5,
  "missing_repos": ["repo-not-found"],
  "commits_by_date": {
    "2026-01-03": [
      {
        "hash": "abc12345",
        "subject": "feat: add changelog command",
        "author": "Dylan Conlin",
        "date": "2026-01-03T10:00:00-08:00",
        "date_str": "2026-01-03",
        "repo": "orch-go",
        "category": "cmd",
        "files": ["cmd/orch/changelog.go"],
        "semantic_info": {
          "change_type": "behavioral",
          "blast_radius": "local",
          "is_breaking": false,
          "commit_type": "feat",
          "semantic_label": "[behavioral]"
        }
      }
    ]
  },
  "commits_by_category": {
    "cmd": 15,
    "skills": 10,
    "kb": 8
  },
  "repo_stats": {
    "orch-go": 20,
    "orch-knowledge": 12
  }
}
```

---

## Semantic Parsing Taxonomy

### Change Types

| Type | Detection | Meaning |
|------|-----------|---------|
| `documentation` | `docs:` prefix or `.md` files | Non-code documentation changes |
| `behavioral` | `feat:`/`fix:`/`perf:`/`refactor:` or code files | Changes that affect runtime behavior |
| `structural` | `build:`/`ci:`/`chore:` or config files | Build/config changes |
| `unknown` | No conventional commit prefix, mixed files | Couldn't determine type |

**Detection priority:** Conventional commit prefix > file type inference

### Blast Radius

Indicates how many components a change affects:

| Radius | Detection | Meaning |
|--------|-----------|---------|
| `local` | Single skill/file, normal changes | Contained impact |
| `cross-skill` | 2+ skills affected | Changes spanning multiple skills |
| `infrastructure` | `pkg/spawn/`, `pkg/verify/`, `skill.yaml`, `SPAWN_CONTEXT` | Affects all agents/skills |

### Categories

Based on file paths:

| Category | Path Patterns | Icon |
|----------|---------------|------|
| `skills` | `skills/`, `*/skills/` | :dart: |
| `skill-behavioral` | Skills + non-`.md` files | :dart: |
| `skill-docs` | Skills + only `.md` files | :book: |
| `kb` | `.kb/` | :books: |
| `decision-record` | `.kb/decisions/` | :scroll: |
| `investigation` | `.kb/investigations/` | :mag: |
| `cmd` | `cmd/` | :zap: |
| `pkg` | `pkg/` | :package: |
| `web` | `web/`, `src/` | :globe_with_meridians: |
| `docs` | `docs/` | :pencil: |
| `config` | `.yaml`, `.json`, `Makefile`, `go.mod` | :gear: |
| `other` | Default | :page_facing_up: |

### Breaking Changes

Detected via:
- `!` suffix on commit type: `feat!: breaking change`
- `BREAKING` prefix: `BREAKING: remove deprecated API`
- `BREAKING CHANGE` in message body

---

## Integration Points

### orch complete Integration

When running `orch complete`, the system checks for notable changes from the last 3 days that might be relevant:

**Surfaced changes:**
- **BREAKING changes** - Always shown with :rotating_light: icon
- **Behavioral changes** in skills/cmd/pkg - Shown with :pushpin: icon
- **Skill-relevant changes** - Changes affecting the agent's skill, shown with :dart: icon

**Example output:**
```
┌─────────────────────────────────────────────────────────────────────────┐
│ Notable Changes (last 3 days)                                          │
├─────────────────────────────────────────────────────────────────────────┤
│ :rotating_light: [orch-go] feat!: breaking API change (BREAKING)               │
│ :dart: [orch-knowledge] refactor: update feature-impl skill (relevant to...)   │
│ :pushpin: [orch-go] feat: add changelog command (behavioral)                   │
└─────────────────────────────────────────────────────────────────────────┘
```

**Skip with:** `orch complete --no-changelog-check`

### Skill Relevance Detection

A change is considered "relevant" to an agent's skill if it affects:
- Files in `skills/<skill-name>/` or `skills/*/<skill-name>/`
- `SPAWN_CONTEXT` templates
- `pkg/spawn/` (spawn system infrastructure)
- `pkg/verify/skill*` (skill verification)

---

## Adding New Repos to Monitoring

### Step 1: Edit the Registry

In `pkg/spawn/ecosystem.go`:

```go
var ExpandedOrchEcosystemRepos = map[string]bool{
    // Core orchestration repos
    "orch-go":        true,
    "orch-cli":       true,
    "kb-cli":         true,
    "orch-knowledge": true,
    "beads":          true,
    "kn":             true,
    // Additional ecosystem repos
    "beads-ui-svelte": true,
    "glass":           true,
    "skillc":          true,
    "agentlog":        true,
    // Add new repo here:
    "my-new-repo":     true,
}
```

### Step 2: Ensure Repo is Discoverable

The system looks in these locations (in order):
1. `~/Documents/personal/<repo-name>`
2. `~/<repo-name>`
3. `~/projects/<repo-name>`
4. `~/code/<repo-name>`

The repo must be a git repository (have a `.git` directory).

### Step 3: Rebuild

```bash
cd ~/Documents/personal/orch-go
make install
```

### Step 4: Verify

```bash
# Check that the new repo appears
orch changelog --project my-new-repo

# Or check in the full ecosystem output
orch changelog | grep my-new-repo
```

If the repo appears in "Missing repos", check that it exists at one of the discoverable paths.

---

## Architecture

### Data Flow

```
                    ┌──────────────────┐
                    │ ExpandedOrchEco- │
                    │ systemRepos map  │
                    └────────┬─────────┘
                             │
                             ▼
┌────────────────────────────────────────────────────────────┐
│                    GetChangelog()                          │
│  1. Get list of repos to scan                             │
│  2. For each repo: findRepoPath() → getGitLog()           │
│  3. Parse git log output with parseGitLog()               │
│  4. Categorize by files (categorizeCommitByFiles)         │
│  5. Parse semantic info (parseSemanticInfo)               │
│  6. Group by date, aggregate stats                         │
└───────────────────┬────────────────────────────────────────┘
                    │
        ┌───────────┴───────────┐
        ▼                       ▼
┌───────────────┐      ┌────────────────┐
│  CLI output   │      │ API endpoint   │
│ (formatted)   │      │    (JSON)      │
└───────────────┘      └────────────────┘
```

### Key Functions

| Function | Location | Purpose |
|----------|----------|---------|
| `GetChangelog(days, project)` | `changelog.go` | Core logic, returns `*ChangelogResult` |
| `getEcosystemRepos()` | `changelog.go` | Get list of repos from registry |
| `findRepoPath(name)` | `changelog.go` | Locate repo on filesystem |
| `getGitLog(path, days)` | `changelog.go` | Run `git log` and parse output |
| `parseSemanticInfo(subject, files)` | `changelog.go` | Extract semantic metadata |
| `detectNotableChangelogEntries()` | `main.go` | Filter for orch complete |

---

## Troubleshooting

### Repo shows as "Missing"

1. Check if repo exists at one of the discoverable paths
2. Ensure it has a `.git` directory
3. Verify the name matches exactly in `ExpandedOrchEcosystemRepos`

### No commits showing

1. Check `--days` parameter (default 7 days)
2. Verify the repo has commits in the time range: `git log --since="7 days ago"`

### Category seems wrong

Categories are determined by file paths. If a commit touches multiple categories, the one with most files wins (with priority tie-breaking: skills > kb > cmd > pkg > web > docs > config > other).

### Semantic info missing

Ensure commits follow conventional commit format (`type: message`) for best results. Non-conventional commits fall back to file-based inference.
