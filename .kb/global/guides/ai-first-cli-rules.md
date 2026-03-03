# Rules for AI-First CLIs

Guidelines for building CLIs that AI agents can use effectively.

**Key insight:** AI-first ≠ JSON-first. LLM agents read prose well. Design for both human and machine consumers.

---

## The 14 Rules

### Category 1: Output Patterns (CLI → Agent)

**1.1 Dual Output Modes**
- Human-readable default (LLMs interpret prose naturally)
- `--json` flag for scripts/pipelines
- Don't force JSON on LLM agents - they read prose fine

**1.2 Consistent Error Format**
- Clear exit codes (0 = success, non-zero = failure)
- Parseable error structure

**1.3 Actionable Error Messages**
- Tell agent what went wrong AND what to do next
- Example: `"Run: git add . && git commit -m '...'"`

---

### Category 2: Input Patterns (Agent → CLI)

**2.1 TTY Detection**
- Auto-skip confirmations when `not sys.stdin.isatty()`
- No hanging on prompts when called programmatically

**2.2 Explicit Flags**
- `--yes` / `-y` to skip confirmations
- `--quiet` for minimal output
- Fallback when TTY detection insufficient

**2.3 All Args Passable**
- No required interactive prompts
- Everything specifiable via flags/args

---

### Category 3: Discovery Patterns (Agent learns CLI)

**3.1 Agent Documentation**
- `AGENTS.md` separate from README
- Focused on agent workflows, not human installation

**3.2 Self-Describing Commands**
- `--ai-help` outputs structured metadata
- Enables template resolution at runtime
- Schema: `{syntax, purpose, example}`

**3.3 Version Awareness**
- `--whats-new` shows agent-relevant changes
- Helps agents understand version differences

---

### Category 4: Workflow Patterns (Agent uses CLI over time)

**4.1 Context Injection**
- `prime` command outputs complete workflow context
- Designed for session startup injection

**4.2 Aggregated Discovery**
- `context <topic>` searches across data types
- Reduces multi-command overhead

**4.3 Task-Oriented Help**
- `help <workflow>` groups commands by task
- Better than alphabetical command lists

---

### Category 5: Observability Patterns (Coordination CLIs)

**5.1 Error Aggregation**
- `errors` command with stats over time
- Hotspot detection (which commands fail most)

**5.2 Health Checks**
- `doctor` command for self-diagnosis
- Reports problems AND how to fix them

**5.3 Time-Aware Queries**
- `--since` flag for session-scoped queries
- Enables "what happened this session?" questions
- Essential for workflow gates

---

## Implementation Tiers

### Tier 1: Essential (Day 1)
*Minimum viable AI-first CLI*

1. TTY detection for confirmations
2. `--json` flag on output commands
3. Actionable error messages with next steps

### Tier 2: Important (Week 1)
*Usable by AI agents*

4. `AGENTS.md` with agent-specific guidance
5. `--ai-help` for structured command metadata
6. Consistent exit codes

### Tier 3: Mature (Month 1+)
*Optimized for AI workflows*

7. `prime` / context injection command
8. `doctor` for self-diagnosis
9. `errors` for aggregation (coordination CLIs only)
10. `--whats-new` for version awareness
11. `--since` for time-aware queries

---

## Pattern Examples

### TTY Detection (Python/Click)
```python
should_skip = yes or not sys.stdin.isatty() or os.getenv('CLI_AUTO_CONFIRM') == '1'
if not should_skip:
    if not click.confirm('Proceed?'):
        raise click.Abort()
```

### Self-Describing Commands (--ai-help)
```json
{
  "comment": {
    "syntax": "bd comment <issue-id> <message>",
    "purpose": "Add a comment to an issue",
    "example": "bd comment ok-abc 'Phase: Planning'"
  }
}
```

### Context Injection (prime)
```markdown
# Workflow Context

## Core Rules
- Track ALL work in beads (no markdown TODOs)
- Use `bd create` to create issues

## Essential Commands
### Finding Work
- `bd ready` - Show issues ready to work
- `bd show <id>` - Detailed issue view
```

### Actionable Errors
```
❌ Completion failed:
   Git validation error: Uncommitted changes detected

Run: git add . && git commit -m "..." && git push
```

---

## When to Apply Which Patterns

| CLI Type | Essential Patterns | Additional Patterns |
|----------|-------------------|---------------------|
| Simple utility | 1.1, 2.1, 2.3 | 3.1 |
| Developer tool | 1.1-1.3, 2.1-2.3 | 3.1-3.2, 4.3 |
| Coordination CLI | All of above | 4.1-4.2, 5.1-5.3 |

---

## Lineage

**Emerged from:**
- Investigation: `.kb/investigations/2025-12-09-inv-design-rules-first-clis-extract.md` - Extracted patterns from building multiple CLIs
- Practical experience with orch-cli, beads (bd), kn, kb-cli

**Related guides:**
- `.kb/guides/ai-native-technology-choice.md` - Technology choice for AI-native development

**Last updated:** 2025-12-10
