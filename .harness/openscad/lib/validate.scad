// validate.scad — BASE Layer 1 parameter validation library (refreshable)
// Refreshable via: harness init --openscad --refresh
// DO NOT add project-specific validators here — use validate-project.scad instead.
//
// Pattern: assert() gates with GATE FAIL: prefix for grep-able error extraction
// Usage: use <../lib/validate.scad> then call validate_* modules

// Validate a numeric parameter is within bounds
// Fails with GATE FAIL: prefix and interpolated values
module validate_range(name, value, min_val, max_val) {
    assert(value >= min_val,
        str("GATE FAIL: ", name, "=", value, " < ", min_val, " minimum"));
    assert(value <= max_val,
        str("GATE FAIL: ", name, "=", value, " > ", max_val, " maximum"));
}

// Validate a dimension is positive
module validate_positive(name, value) {
    assert(value > 0,
        str("GATE FAIL: ", name, "=", value, " must be positive"));
}

// Validate wall thickness meets printability minimum
module validate_wall_thickness(name, value, min_wall=0.8) {
    assert(value >= min_wall,
        str("GATE FAIL: ", name, "=", value, "mm < ", min_wall, "mm minimum wall thickness"));
}

// Validate $fn is reasonable for printable curves
module validate_fn(fn_val, min_fn=16, max_fn=256) {
    assert(fn_val >= min_fn,
        str("GATE FAIL: $fn=", fn_val, " < ", min_fn, " (too coarse for printable curves)"));
    assert(fn_val <= max_fn,
        str("GATE FAIL: $fn=", fn_val, " > ", max_fn, " (excessive polygon count)"));
}

// Validate hole diameter is printable
module validate_hole(name, diameter, min_diameter=1.0) {
    assert(diameter >= min_diameter,
        str("GATE FAIL: ", name, "=", diameter, "mm < ", min_diameter, "mm minimum hole diameter"));
}

// Validate that interior space exists (wall doesn't consume entire dimension)
module validate_interior(dim_name, dim_value, wall_name, wall_value) {
    assert(wall_value < dim_value / 2,
        str("GATE FAIL: ", wall_name, "=", wall_value,
            " >= ", dim_name, "/2=", dim_value/2, " (no interior space)"));
}

// Batch validate common bracket/enclosure parameters
// This is the recommended entry point for simple parts
module validate_part(width, height, depth, wall_thickness,
                     fn=32, holes=[], min_wall=0.8) {
    validate_positive("width", width);
    validate_positive("height", height);
    validate_positive("depth", depth);
    validate_wall_thickness("wall_thickness", wall_thickness, min_wall);
    validate_fn(fn);
    validate_interior("width", width, "wall_thickness", wall_thickness);
    validate_interior("depth", depth, "wall_thickness", wall_thickness);

    // Validate each hole diameter if provided
    for (i = [0:len(holes)-1]) {
        validate_hole(str("hole[", i, "]"), holes[i]);
    }
}
