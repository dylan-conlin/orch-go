# Heuristic vs Structural Approaches

When building systems that need to extract or understand information, choose between **inferring** it (heuristic) or **requiring declaration** (structural).

**Key insight:** Heuristics accumulate edge cases. Structural approaches require upfront investment but don't degrade over time.

---

## The Distinction

### Structural

Information is explicitly declared in a defined location. The system reads the value directly.

```yaml
# Structural: topic is declared in frontmatter
---
topic: Server Crash Patterns
type: investigation
---
# Investigation: Server Crash Patterns
```

**Characteristics:**
- Always correct (if declared correctly)
- Machine-readable by design
- Explicit contract between author and system
- Requires upfront declaration
- Migration cost for existing artifacts

### Heuristic

Information is inferred from patterns, conventions, or rules of thumb.

```go
// Heuristic: strip known prefixes, extract remaining words
func ExtractTopic(title string) string {
    title = strings.TrimPrefix(title, "Investigation:")
    title = strings.TrimPrefix(title, "Design:")
    // ... accumulating special cases
    return strings.TrimSpace(title)
}
```

**Characteristics:**
- Works now, no migration needed
- Flexible, handles variation
- Can fail silently on edge cases
- Accumulates special cases over time
- Maintenance burden grows

---

## Decision Framework

### Choose Structural When:

1. **The information is critical for tooling** - If wrong extraction causes silent failures (wrong search results, missed matches), structural prevents this class of error entirely.

2. **The domain is closed** - You control all artifact creation and can enforce the schema.

3. **Edge cases are already accumulating** - If you're adding the 3rd or 4th special case to a heuristic, it's time to go structural.

4. **Migration is feasible** - You can update existing artifacts or grandfather them in.

### Choose Heuristic When:

1. **The information is advisory, not critical** - Wrong extraction is annoying but not harmful.

2. **The domain is open** - You can't control all inputs (external data, user-generated content).

3. **Patterns are stable and well-defined** - The heuristic won't need constant updates.

4. **Structural would require impractical migration** - 10,000 files with no automation path.

### The Hybrid Path

Often the right answer is: **heuristic now, structural later**.

1. Ship heuristic fix to solve immediate problem
2. Track edge cases that require special handling
3. When edge cases reach threshold (3+), design structural solution
4. Migrate incrementally (new artifacts use structural, old artifacts use heuristic fallback)

---

## Examples in This System

| Domain | Current | Approach | Notes |
|--------|---------|----------|-------|
| Investigation topic | Heuristic | Title parsing with prefix stripping | Issue 21129 adds more prefixes |
| Artifact type | Structural | Frontmatter `type:` field | Explicit declaration required |
| Issue priority | Structural | `--priority P2` flag | Explicit at creation |
| Skill dependencies | Structural | `dependencies:` in skill.yaml | Machine-readable list |
| Prior work | Heuristic → Structural | In-text citations → Prior-Work table | Migration in progress |

---

## Warning Signs

### Heuristic Degradation

You're on the wrong path if:
- Adding special cases every few weeks
- "Works for most cases" keeps shrinking
- Silent failures discovered after the fact
- Different parts of system disagree on extracted value

### Structural Overhead

You're over-engineering if:
- Requiring declaration for rarely-used information
- Migration blocks shipping for weeks
- Schema changes require touching every file
- Enforcing structure on inherently unstructured data

---

## The Accumulation Problem

Heuristics don't fail suddenly—they degrade gradually.

```
Month 1: Strip "Investigation:" prefix - works perfectly
Month 2: Add "Design:" prefix - still works
Month 3: Add "## Investigation:" for markdown headers - getting messy
Month 4: User writes "Investigate:" instead of "Investigation:" - silent failure
Month 5: Add "Investigate:", "Investigating:", "inv:" - heuristic is now 20 lines
```

Each addition is locally reasonable. The accumulated result is fragile.

**The trigger for structural migration:** When you're about to add the 3rd special case, stop and evaluate structural alternative.

---

## Implementation Notes

### Structural with Fallback

Best of both worlds for migration:

```go
func GetTopic(artifact Artifact) string {
    // Structural: check frontmatter first
    if artifact.Frontmatter.Topic != "" {
        return artifact.Frontmatter.Topic
    }
    // Heuristic fallback: parse title for legacy artifacts
    return extractTopicFromTitle(artifact.Title)
}
```

New artifacts use structural. Old artifacts get heuristic fallback. No forced migration.

### Validation at Creation

If going structural, validate at creation time:

```go
func CreateInvestigation(title, topic string) error {
    if topic == "" {
        return errors.New("topic required: use --topic flag or add topic: to frontmatter")
    }
    // ...
}
```

Don't let invalid artifacts enter the system.

---

## Lineage

**Emerged from:**
- Issue orch-go-21129: Topic extraction uses heuristic (prefix stripping), noted as needing structural alternative
- Decision: Investigation Lineage Enforcement - Prior-Work table is structural replacement for informal in-text citations (heuristic)

**Related:**
- Principle: Infrastructure Over Instruction - Structural approaches are infrastructure; heuristics are instruction-dependent
- Guide: AI-First CLI Rules - `--ai-help` is structural metadata vs parsing help text (heuristic)

**Last updated:** 2026-01-31
