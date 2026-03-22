// validate-project.scad — Project-specific validators (project-owned, never overwritten by harness)
// Base validators live in validate.scad (refreshable via harness init --openscad --refresh)
//
// Add project-specific validators here. Examples:
//   validate_wire_channel(diameter)    — for cable routing parts
//   validate_bore_fit(inner, outer)    — for press-fit assemblies
//   validate_build_plate(x, y, z)     — for printer-specific size limits
//
// Call-site parameterization for material overrides:
//   Base validators already accept parameters with defaults. Override at the call site:
//     validate_wall_thickness("wall", wall, min_wall=1.2);  // PETG override
//     validate_wall_thickness("wall", wall);                  // default 0.8 (PLA)
//   No wrapper modules or config files needed.

use <validate.scad>

// --- Project-specific validators below ---
// Add your project's custom validators here.
// They can call base validators internally if needed.

// Example: Validate part fits on build plate
// module validate_build_plate(name, x, y, z, max_x=220, max_y=220, max_z=250) {
//     assert(x <= max_x, str("GATE FAIL: ", name, " x=", x, "mm > ", max_x, "mm build plate width"));
//     assert(y <= max_y, str("GATE FAIL: ", name, " y=", y, "mm > ", max_y, "mm build plate depth"));
//     assert(z <= max_z, str("GATE FAIL: ", name, " z=", z, "mm > ", max_z, "mm build plate height"));
// }
