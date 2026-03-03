# Design Phase

**Purpose:** Document architectural approach before implementation.

**When you're in this phase:** Investigation findings (if any) inform the design. Document approach, architecture, and testing strategy before writing code.

---

## Deliverables

- **Design document:** `docs/designs/YYYY-MM-DD-{kebab-case-feature}.md`
- **Testing strategy:** Clear plan for what needs tests
- **Architecture decision:** Chosen approach with trade-off analysis

---

## Workflow

### 1. Review Investigation Findings (If Investigation Phase Included)

If investigation phase preceded design:
- Read investigation file: `.kb/investigations/YYYY-MM-DD-*.md`
- Note key architectural constraints
- Identify integration points
- Understand dependencies

### 2. Determine if Design Exploration Needed

**Escalate to orchestrator for design exploration when:**
- ✅ Multiple viable technical approaches exist (e.g., library selection, architecture patterns)
- ✅ Significant trade-offs to evaluate (performance vs maintainability, complexity vs flexibility)
- ✅ Uncertainty about best approach based on investigation findings
- ✅ Novel problem domain without established patterns

**Proceed with design directly when:**
- ❌ Approach is obvious from investigation
- ❌ Following established patterns in codebase
- ❌ Simple/straightforward implementations
- ❌ Orchestrator already specified approach in SPAWN_CONTEXT

**If design exploration needed:**

Report via beads that you need design exploration:
```bash
bd comment <beads-id> "Status: BLOCKED - Multiple viable approaches, need design exploration before proceeding"
```

The orchestrator may spawn an interactive architect session (`orch spawn architect -i`) for collaborative design exploration. Wait for orchestrator response before proceeding.

### 3. Create Design Document

Create design file:
```bash
DATE=$(date +%Y-%m-%d)
SLUG="feature-name-in-kebab-case"
DESIGN_FILE=${PROJECT_DIR}/docs/designs/${DATE}-${SLUG}.md

mkdir -p ${PROJECT_DIR}/docs/designs
```

**Use the design document template:**

Full template available at: `~/.claude/skills/worker/feature-impl/reference/design-template.md`

**Key sections to include:**
- Problem statement with success criteria
- Approach and architectural decisions
- Data model (if applicable)
- UI/UX (if applicable)
- Testing strategy
- Security considerations
- Performance requirements
- Rollout plan
- Alternatives considered
- Open questions and references

### 4. Present Design for Orchestrator Review

**Report design summary via beads:**

```bash
bd comment <beads-id> "Design ready for review: docs/designs/YYYY-MM-DD-{slug}.md - [chosen approach]. Key decisions: [1-2 sentences]. Awaiting approval."
```

If design exploration was done (via architect session):
```bash
bd comment <beads-id> "Design ready: Evaluated 3 approaches, recommending [A] because [reasoning]. See docs/designs/... for details."
```

### 5. Get Orchestrator Approval

**Wait for orchestrator to:**
- Review design document
- Ask clarifying questions
- Approve approach OR suggest adjustments

**Do not proceed to implementation without approval.**

### 6. Move to Implementation

Once approved:

1. Update design doc status:
   ```markdown
   **Status:** Approved
   ```

2. Commit design document:
   ```bash
   cd ${PROJECT_DIR}
   git add docs/designs/${DATE}-${SLUG}.md
   git commit -m "design: ${SLUG}"
   ```

3. Report phase transition: `bd comment <beads-id> "Phase: Implementation - Design approved, beginning implementation"`

4. Output: "✅ Design approved, moving to Implementation phase"

---

## Key Principles

| Principle | Application |
|-----------|-------------|
| **Design before code** | Architecture decisions documented upfront, not discovered during implementation |
| **Trade-offs explicit** | Document why chosen approach, what alternatives were rejected |
| **Testing strategy clear** | Know what needs tests before writing code |
| **Security by design** | Security considerations integrated, not bolted on |
| **Review before implementation** | Get approval on approach before investing time in code |

---

## Completion Criteria

Before moving to Implementation phase, verify:

- [ ] Design document created in `docs/designs/`
- [ ] Problem statement clear
- [ ] Approach documented with rationale
- [ ] Data model defined (if applicable)
- [ ] Testing strategy specified
- [ ] Security considerations addressed
- [ ] Performance requirements documented
- [ ] Alternatives considered (if design exploration was done)
- [ ] Orchestrator reviewed and approved design
- [ ] Design document committed to git
- [ ] Reported via beads: `bd comment <beads-id> "Phase: Implementation - Design approved"`

**If ANY box unchecked, design is NOT complete.**
