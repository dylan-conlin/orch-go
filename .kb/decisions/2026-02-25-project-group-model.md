# Decision: Project Group Model for orch-go

**Status:** Accepted
**Date:** 2026-02-25
**Affects:** spawn kb context, daemon polling, account routing, dashboard filtering
**Investigation:** `.kb/investigations/2026-02-25-design-project-group-model.md`
**Supersedes:** `.kb/decisions/2026-01-16-single-daemon-orchestration-home.md`

---

## Decision

Introduce **project groups** via `~/.orch/groups.yaml` to model project relationships. Two group types: explicit (list members) and parent-inferred (name a parent project, children auto-discovered from directory paths). Groups cascade to kb context scope, daemon polling, account routing, and dashboard filtering.

---

## Config: `~/.orch/groups.yaml`

```yaml
groups:
  orch:
    account: personal
    projects:
      - orch-go
      - orch-cli
      - kb-cli
      - orch-knowledge
      - beads
      - kn
      - opencode

  scs:
    account: work
    parent: scs-special-projects
    # Children auto-discovered: toolshed, price-watch, specs-platform, sendassist, scs-slack
```

## Resolution Algorithm

1. Check explicit groups: is project in any group's `projects` list?
2. Check parent groups: is project's path a subdirectory of any group's parent project path?
3. Parent project itself is always a member of its own group.
4. Return all matching groups (empty if ungrouped).

## How Groups Cascade

| Consumer | Change | File |
|----------|--------|------|
| **KB context** | Replace `OrchEcosystemRepos` hardcode with `GroupsForProject()` filter | `pkg/spawn/kbcontext.go` |
| **Daemon polling** | `--group` flag scopes `ListReadyIssuesMultiProject` to group members | `daemon.go`, `project_resolution.go` |
| **Account routing** | Before spawning, look up target project's group account, switch if needed | `pkg/daemon/daemon.go` |
| **Dashboard** | `?group=scs` filter parameter, `GET /api/groups` endpoint | `serve_filter.go` |

## Trade-offs Accepted

1. **Two resolution mechanisms** (explicit + path inference) — justified because pure-explicit redundantly lists SCS projects the filesystem already encodes, and pure-inference can't handle orch ecosystem (scattered paths)
2. **Global config at ~/.orch/** — not per-project, so single view of all groups, aligns with daemon's home
3. **Ungrouped projects work as before** — no breaking changes

## Migration

1. Ship groups.yaml alongside existing `OrchEcosystemRepos` hardcode (backward compat)
2. If `~/.orch/groups.yaml` exists, use group-based filtering
3. If not, fall back to hardcode
4. Remove hardcode after groups.yaml is in place

## References

- Cross-repo orchestration decision: `scs-special-projects/.kb/decisions/2026-02-25-cross-repo-orchestration-from-parent.md`
- Investigation: `.kb/investigations/2026-02-25-design-project-group-model.md`
- Code: `pkg/spawn/kbcontext.go:15-22` (OrchEcosystemRepos), `pkg/daemon/project_resolution.go` (ProjectRegistry)
