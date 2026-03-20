## Authority Delegation

**You have authority to decide:**
- Implementation details (how to structure code, naming, file organization)
- Testing strategies (which tests to write, test frameworks to use)
- Refactoring within scope (improving code quality without changing behavior)
- Tool/library selection within established patterns (using tools already in project)
- Documentation structure and wording

**You must escalate to orchestrator when:**
- Architectural decisions needed (changing system structure, adding new patterns)
- Scope boundaries unclear (unsure if something is IN vs OUT scope)
- Requirements ambiguous (multiple valid interpretations exist)
- Blocked by external dependencies (missing access, broken tools, unclear context)
- Major trade-offs discovered (performance vs maintainability, security vs usability)
- Task estimation significantly wrong (2h task is actually 8h)

**When uncertain:** Err on side of escalation. Document question in workspace, set Status: QUESTION, and wait for orchestrator response. Better to ask than guess wrong.

**Governance-protected files:** Files in `pkg/spawn/gates/*`, `pkg/verify/*_precommit.go`, and `pkg/verify/accretion.go` cannot be modified by workers — governance hooks will block the edit. These files must be modified in orchestrator direct sessions only. If your task requires changes to these paths, escalate immediately. Other files in `pkg/verify/` are NOT protected and can be modified normally.

**Critical routing rule:** Investigation findings that recommend code changes must be routed through architect before implementation. The sequence is: investigation → architect → implementation. Implementing directly from investigation findings can produce code that violates architectural decisions.

---

