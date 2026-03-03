# Integration Phase

**Purpose:** Combine multiple validated phases into cohesive feature.

**When to use:** Multi-phase features after all phases validated individually

**Prerequisites:** All dependent phases must be validated and approved by orchestrator.

---

## When Integration is Needed

**Use when:**
- ✅ Feature split into multiple phases (A, B, C...)
- ✅ Each phase validated independently
- ✅ Phases need to work together
- ✅ E2E testing required across phase boundaries

**Skip when:**
- ❌ Single-phase feature
- ❌ Phases completely independent
- ❌ Each phase already tested integration

---

## Workflow

### 1. Review Completed Phases

Review each phase via beads comments (`bd show <beads-id>`) to understand:
- What was implemented
- What was tested
- Open questions/concerns
- Integration points mentioned

### 2. Identify Integration Points

Map where phases interact:
- Data flow between components
- Shared state/configuration
- API contracts between modules
- Database schema dependencies
- Event handling across boundaries

### 3. Integration Testing

Write integration tests for cross-phase scenarios:
- Test interactions between phases
- Verify data flows correctly
- Confirm shared contracts work
- Check error handling across boundaries

### 4. E2E Verification

Test complete user flows across all phases:
1. Define end-to-end scenarios
2. Execute flows manually or automated
3. Verify all phases work together
4. Document test results

### 5. Performance Testing (If Applicable)

- Measure end-to-end performance
- Verify meets requirements
- Document metrics

### 6. Regression Testing

1. Run full test suite
2. Verify existing features still work
3. Check for unintended side effects

### 7. Document Results

Report via beads:
```bash
bd comment <beads-id> "Integration results: Phases [A,B,C] integrated. Integration tests: [X passing]. E2E: [verified]. Performance: [metrics if applicable]"
```

### 8. Final Smoke Test

Perform manual verification of complete feature:
- Use as end user would
- Test all flows across phases
- Verify UI polish
- Check error messages
- Confirm performance

### 9. Move to Validation

Report via `bd comment <beads-id> "Phase: Validation - Integration complete"`

---

## Completion Criteria

- [ ] All phases reviewed (via beads comment history)
- [ ] Integration points identified and documented
- [ ] Integration tests written and passing
- [ ] E2E tests cover complete user flows
- [ ] Performance requirements met (if applicable)
- [ ] No regressions (full test suite passing)
- [ ] Final smoke test passed
- [ ] Integration results reported via beads
- [ ] Reported via beads: `bd comment <beads-id> "Phase: Validation"`

---

## Integration Deliverables

**Required:**
- Integration tests (cross-phase coverage)
- E2E tests (complete user flows)
- Integration documentation

**Optional:**
- Performance metrics
- Architecture diagram
- API documentation
- Deployment guide
