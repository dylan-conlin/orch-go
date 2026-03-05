# Investigation Phase

**Purpose:** Understand the existing system before making changes.

**When you're in this phase:** Your SPAWN_CONTEXT specified investigation scope. Document findings progressively to inform design and implementation decisions.

---

## Deliverables

- **Investigation file:** `.kb/investigations/YYYY-MM-DD-inv-{kebab-case-description}.md`
- **Findings:** Evidence-Source-Significance pattern
- **Synthesis:** Connected insights, not just a list of facts

---

## Workflow

### 1. Create Investigation Template (Before Exploring)

**Critical:** Create template at START, not at end. Forces progressive documentation.

```bash
DATE=$(date +%Y-%m-%d)
SLUG="topic-in-kebab-case"  # From SPAWN_CONTEXT description
INVESTIGATION_FILE=${PROJECT_DIR}/.kb/investigations/${DATE}-inv-${SLUG}.md

mkdir -p ${PROJECT_DIR}/.kb/investigations
# Use investigation skill template or create from structure below
```

Template structure:
```markdown
# Investigation: [Specific Topic]

**Question:** [Precise question from SPAWN_CONTEXT]
**Started:** YYYY-MM-DD
**Updated:** YYYY-MM-DD
**Status:** In Progress

## Findings

[Add progressively as you explore]

## Synthesis

**Key Insights:**
- [Connect findings into patterns]

**Answer to Question:**
[Coherent answer based on findings]

## Structured Uncertainty

**What's tested:**
- [Verified facts with test evidence]

**What's untested:**
- [Known gaps - hypotheses not yet validated]

**What would change this:**
- [Conditions that would invalidate conclusion]
```

### 2. Fill Question and Metadata

Edit investigation file with precise question from SPAWN_CONTEXT:
- **Question:** Specific, answerable question
- **Started:** Today's date
- **Status:** In Progress

### 3. Add Findings Progressively (As You Explore)

**After each discovery**, add a finding using this pattern:

```markdown
### Finding 1: [Brief description]

**Evidence:** [Concrete observation - code snippet, output, behavior]

**Source:** [File:line reference or command that produced evidence]

**Significance:** [Why this matters for implementation]
```

**Example:**
```markdown
### Finding 1: Authentication uses JWT tokens in HTTP-only cookies

**Evidence:** Found `Set-Cookie` header with `httpOnly=true` flag in login response. Token has 3-part structure (header.payload.signature).

**Source:** `src/auth/middleware.ts:45-67` and Chrome DevTools Network tab

**Significance:** Token can't be accessed by JavaScript (XSS protection), but sent automatically with requests (CSRF risk). Must implement CSRF protection for state-changing operations.
```

**Don't wait to write everything at end** - document as you go.

### 4. Update Synthesis After Each Cluster

**Every 3-5 findings**, update synthesis section to connect patterns:

```markdown
## Synthesis

**Key Insights:**
- Auth uses JWT tokens in HTTP-only cookies (Finding 1)
- Tokens expire after 15 min, refresh tokens last 7 days (Finding 2)
- Refresh endpoint at `/auth/refresh` extends session (Finding 3)

**Answer to Question:**
[Coherent explanation connecting all findings]
```

Progressive synthesis helps spot patterns as they emerge.

### 5. Update Structured Uncertainty

As investigation progresses, update uncertainty assessment honestly:

```markdown
## Structured Uncertainty

**What's tested:**
- [Verified facts with test evidence]

**What's untested:**
- [Known gaps - hypotheses not yet validated]

**What would change this:**
- [Conditions that would invalidate conclusion]
```

**Honest uncertainty > false certainty.** State what you haven't tested.

### 6. Mark Complete and Move to Clarifying Questions Phase

When investigation answers your question:

1. Update status:
   ```markdown
   **Status:** Complete
   ```

2. Commit investigation file:
   ```bash
   cd ${PROJECT_DIR}
   git add .kb/investigations/${DATE}-inv-${SLUG}.md
   git commit -m "investigation: ${SLUG}"
   ```

3. Report phase transition:
   ```bash
   bd comments add <beads-id> "Phase: Clarifying Questions - Investigation complete, findings in [investigation file path]"
   ```

4. Output: "Investigation complete, moving to Clarifying Questions phase"

---

## Key Principles

| Principle | Application |
|-----------|-------------|
| **Progressive documentation** | Create template first, fill findings as you explore (not at end) |
| **Evidence-based** | Every finding needs concrete evidence (code, output, observation) |
| **Honest uncertainty** | State what you haven't tested - gaps are valuable information |
| **Clear sourcing** | Always include file:line or command that produced evidence |
| **Synthesis over list** | Connect findings into coherent answer, don't just list facts |

---

## Completion Criteria

Before moving to Design phase, verify:

- [ ] Investigation file created in `.kb/investigations/`
- [ ] Question answered with synthesis (not just list of findings)
- [ ] Each finding has Evidence + Source + Significance
- [ ] Uncertainty assessed honestly (tested vs untested clearly separated)
- [ ] Key architectural constraints documented
- [ ] Dependencies identified for implementation
- [ ] Investigation file committed to git
- [ ] Workspace updated: Phase -> Clarifying Questions

**If ANY box unchecked, investigation is NOT complete.**

**Routing note:** If your investigation findings recommend code changes, those changes must be routed through an architect skill before implementation. Do not implement directly from investigation findings — this bypasses architectural review and can produce code that violates existing decisions.
