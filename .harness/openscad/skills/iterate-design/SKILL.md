# iterate-design

Systematically explore the parameter space of an existing OpenSCAD design.

## When to Use

- Optimizing an existing part (strength vs weight, material vs cost)
- Finding valid parameter ranges for a design
- Comparing configurations for different use cases

## Workflow

1. **Read existing `.scad` file** and identify all parameters with their defaults and ranges
2. **Define sweep plan**: which parameters to vary, what bounds, what step size
3. **Run parameter sweep** via `-D` flags — NEVER modify the source .scad file
4. **Collect results**: STL + summary JSON per configuration
5. **Analyze tradeoffs**: compare metrics (polygon count, bounding box, material volume)
6. **Recommend optimal configuration** with reasoning

## Running a Sweep

```bash
# Single parameter sweep
for val in 1.0 1.5 2.0 2.5 3.0; do
    openscad -o "exports/<name>-wall_${val}.stl" \
        --backend manifold --quiet \
        --summary-file "sweeps/<name>-wall_${val}.json" \
        parts/<name>.scad -D "wall_thickness=${val}"
done

# Multi-parameter sweep (2D)
for wall in 1.5 2.0 3.0; do
    for width in 30 40 50; do
        openscad -o "exports/<name>-w${wall}-W${width}.stl" \
            --backend manifold --quiet \
            --summary-file "sweeps/<name>-w${wall}-W${width}.json" \
            parts/<name>.scad -D "wall_thickness=${wall}" -D "width=${width}"
    done
done
```

## Analyzing Results

Parse `--summary-file` JSON for each configuration to extract:
- **Facet count** — proxy for geometric complexity
- **Bounding box** — physical dimensions
- **Render time** — correlates with boolean complexity

Compare across configurations:
```bash
# Quick comparison of facet counts across sweep
for f in sweeps/<name>-*.json; do
    echo "$f: $(python3 -c "import json; print(json.load(open('$f')).get('facets', 'N/A'))")"
done
```

## Required Outputs

| File | Purpose |
|------|---------|
| `sweeps/<name>-sweep-results.md` | Parameter configurations tested + metrics |
| `sweeps/<name>-recommendation.md` | Analysis, tradeoffs, and recommended config |
| `exports/<name>-optimal.stl` | Recommended configuration export |
| `exports/<name>-optimal.png` | Visual preview of recommended config |

## Constraints

- **NEVER modify the source `.scad` file** — use `-D` flags only
- Use **Manifold backend** for all sweep renders (speed matters)
- Run **CGAL validation** on the final recommended configuration only
- Record **all** configurations tested, not just the best
- Include **parameter ranges** that cause gate failures — knowing the invalid space is valuable
- Prefer `$fn=32` during sweeps, `$fn=128` for final recommended export only

## Sweep Results Format

Document results in `sweeps/<name>-sweep-results.md`:

```markdown
# Parameter Sweep: <name>

## Parameters Varied
| Parameter | Range | Step | Default |
|-----------|-------|------|---------|
| wall_thickness | 1.0 - 4.0 | 0.5 | 3.0 |

## Results
| Config | wall_thickness | Facets | Bbox (mm) | Gate Pass |
|--------|---------------|--------|-----------|-----------|
| 1 | 1.0 | 1,240 | 40x30x20 | FAIL: assert |
| 2 | 1.5 | 1,240 | 40x30x20 | PASS |
| ... | ... | ... | ... | ... |

## Gate Failures
- Config 1: GATE FAIL: wall_thickness=1.0 < 0.8mm minimum (assert gate in validate.scad)

## Recommendation
Config N because [reasoning about tradeoffs].
```

## Validation Checklist

Before marking complete:

- [ ] Source .scad file is UNMODIFIED (diff shows no changes)
- [ ] All sweep configurations documented with metrics
- [ ] Gate failures recorded (they define the valid parameter space)
- [ ] Recommended config validated with CGAL backend
- [ ] Recommended config exported as STL + PNG
