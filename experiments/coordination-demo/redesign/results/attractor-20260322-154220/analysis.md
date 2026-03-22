# Automated Attractor Discovery Experiment

## Design

Two-phase experiment testing whether merge conflict patterns contain sufficient
information to automatically generate effective structural placement constraints.

### Phase 1: Collision Collection
- No coordination instructions
- Both agents independently choose insertion points
- Merge conflicts recorded with full diff context

### Phase 2: Auto-Generated Attractors
- Collision patterns from Phase 1 parsed automatically
- Gravitational insertion point identified (where both agents insert)
- Alternative insertion points discovered from file structure
- Non-overlapping placement constraints generated and injected

## Results

### Phase 1 (No Attractors)

- **simple**: 2/3 CONFLICT, 0/3 SUCCESS

### Phase 2 (Auto-Generated Attractors)

- **simple**: 7/7 SUCCESS, 0/7 CONFLICT, 0/7 BUILD_FAIL, 0/7 SEMANTIC

### Auto-Generated Constraints

**simple:**
```json
{
  "gravitational_function": "FormatDurationShort",
  "agent_a_placement": "after FormatDurationShort",
  "agent_b_placement": "after StripANSI",
  "agent_b_before": "FormatDuration",
  "source_file": "/Users/dylanconlin/Documents/personal/orch-go/pkg/display/display.go",
  "test_file": "/Users/dylanconlin/Documents/personal/orch-go/pkg/display/display_test.go",
  "generation_method": "automated from collision analysis",
  "human_intervention": false
}
```
