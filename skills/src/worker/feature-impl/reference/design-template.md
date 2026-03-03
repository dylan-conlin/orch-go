# Design Document Template

**Purpose:** Template for creating design documents during the Design phase of feature implementation.

**Location:** Use this template to create: `docs/designs/YYYY-MM-DD-{kebab-case-feature}.md`

---

# Design: [Feature Name]

**Date:** YYYY-MM-DD
**Status:** Draft | Under Review | Approved
**Author:** [Your name/role]

---

## Problem

**What are we solving?**
[Concise problem statement - 2-3 sentences]

**Why now?**
[Context for timing/priority]

**Success criteria:**
- [ ] [Measurable outcome 1]
- [ ] [Measurable outcome 2]

---

## Approach

**High-level solution:**
[1-2 paragraphs describing chosen approach]

**Why this approach?**
[Reasoning - reference investigation findings if applicable]

**Key architectural decisions:**
1. [Decision 1 with rationale]
2. [Decision 2 with rationale]

---

## Data Model

**Database schema changes:**
```sql
-- New tables or columns
CREATE TABLE ... ;
```

**API contracts:**
```json
// Request/response formats
{
  "endpoint": "/api/v1/resource",
  "method": "POST",
  "body": { ... }
}
```

**State management:**
[If applicable - how data flows through system]

---

## UI/UX (If Applicable)

**User flows:**
1. User action → System response
2. Edge case handling

**Mockups/wireframes:**
[ASCII art, links to Figma, or descriptions]

**Accessibility considerations:**
- [Keyboard navigation, screen readers, etc.]

---

## Testing Strategy

**What needs tests:**
- [ ] Unit tests: [Specific components/functions]
- [ ] Integration tests: [API endpoints, database interactions]
- [ ] E2E tests: [Critical user flows]

**Test boundaries:**
[What to mock vs test with real dependencies]

**Edge cases to cover:**
- [Error conditions]
- [Boundary values]
- [Concurrent operations]

---

## Security

**Authentication/Authorization:**
[Who can access? How is access controlled?]

**Input validation:**
[What validation is needed? Where?]

**Data protection:**
[Sensitive data handling, encryption, compliance]

**Attack vectors:**
[XSS, CSRF, SQL injection, etc. - how mitigated?]

---

## Performance

**Expected load:**
[Requests per second, data volume, concurrent users]

**Scalability considerations:**
[Caching, indexing, async processing]

**Performance requirements:**
- Response time: < X ms
- Throughput: X requests/sec

**Optimization strategy:**
[Profile first, optimize bottlenecks]

---

## Rollout

**Deployment strategy:**
[All-at-once, phased, feature flags]

**Migration plan (if applicable):**
[Data migration, backward compatibility]

**Rollback plan:**
[How to revert if issues discovered]

**Monitoring:**
[Metrics to track, alerts to set up]

---

## Alternatives Considered

**(If design exploration was done, document alternatives here)**

### Alternative 1: [Approach name]
**Pros:**
- [Benefit 1]

**Cons:**
- [Drawback 1]

**Why not chosen:**
[Specific reason given trade-offs]

### Alternative 2: [Approach name]
[Same structure]

---

## Open Questions

**Unresolved:**
- [Question requiring orchestrator input]

**Assumptions:**
- [Assumed X because Y]

---

## References

- Investigation: [Link if applicable]
- Related decisions: [Link to decision documents]
- External resources: [Documentation, RFCs, etc.]
