# Principle Addition Protocol

**Purpose:** What to do when a new principle is discovered and added to `~/.kb/principles.md`.

**Last verified:** Jan 5, 2026

---

## When This Applies

A new principle has been:
- Discovered through practice (something broke, distinction emerged)
- Validated (has teeth - violation causes real problems)
- Added to `~/.kb/principles.md` with full structure

---

## Protocol

### 1. Principles.md Entry (Required)

Add the principle with:
- **The test** - How to check if you're violating it
- **What this means** - Positive framing
- **What this rejects** - What it rules out
- **Relationship to other principles** - How it connects
- **Origin** - What broke, when, evidence

Add to the **Provenance table** at the bottom:
- Date
- What broke
- Evidence (links to conversations, artifacts, incidents)

### 2. Skill Updates (Check)

Ask: Does any skill teach behavior this principle would change?

| If... | Then... |
|-------|---------|
| Skill contradicts principle | Update skill |
| Skill would benefit from principle | Add reference |
| Principle is skill-specific | Add to that skill |
| Principle is universal | Consider adding to shared/policy skills |

**Priority skills to check:**
- `orchestrator` - core coordination patterns
- `meta-orchestrator` - hierarchy and perspective
- Any skill related to the principle's domain

### 3. Guide Updates (Check)

Ask: Does any guide now have deeper grounding from this principle?

| If... | Then... |
|-------|---------|
| Guide's advice is explained by principle | Add reference |
| Guide conflicts with principle | Update guide |
| Principle implies new guide needed | Create guide |

**Guides to check:**
- `decision-authority.md` - for hierarchy/escalation principles
- `spawn.md` - for agent behavior principles
- `completion-gates.md` - for gate-related principles

### 4. CLAUDE.md Updates (Check)

Ask: Should this principle be surfaced in CLAUDE.md?

| Scope | Location |
|-------|----------|
| Applies to all projects | `~/.claude/CLAUDE.md` |
| Specific to one project | `<project>/CLAUDE.md` |
| Already covered by skill loading | May not need CLAUDE.md entry |

Most principles live in `principles.md` and are referenced when needed. Only add to CLAUDE.md if agents need it in *every* session.

### 5. Sync and Push (Required)

```bash
bd sync
git add -A && git commit -m "Add principle: <principle name>"
git push
```

Principles are high-value knowledge. Don't leave them stranded locally.

---

## Checklist

```
[ ] Principle added to ~/.kb/principles.md
[ ] Provenance table updated
[ ] Checked skills for updates needed
[ ] Checked guides for updates needed  
[ ] Checked CLAUDE.md for updates needed
[ ] Changes committed and pushed
```

---

## Example: Adding "Escalation is Information Flow"

**2026-01-05:**

1. ✓ Added to principles.md with full structure
2. ✓ Added to provenance table
3. Check skills:
   - meta-orchestrator skill - could reference it (optional, already has "Perspective is Structural")
   - orchestrator skill - could strengthen escalation guidance
4. Check guides:
   - decision-authority.md - "Uncertainty Default: Escalate" could reference principle
5. CLAUDE.md - not needed (principle is domain-specific)
6. ✓ Committed and pushed

---

## Provenance

This guide created after adding "Perspective is Structural" and "Escalation is Information Flow" on 2026-01-05. Recognized that principle addition should have a consistent protocol to ensure principles propagate to where they're needed.
