// bracket.scad — Sample parametric L-bracket with Layer 1 assert gates
// Demonstrates: parameter validation, GATE FAIL pattern, -D flag overrides

// --- Parameters (override via -D flags) ---
width = 40;           // mm
height = 30;          // mm
depth = 20;           // mm
wall_thickness = 3;   // mm (minimum 0.8mm for FDM)
hole_diameter = 5;    // mm (mounting holes)
fillet_radius = 2;    // mm (edge fillets)
fn = 32;              // facet count for curves

// --- Layer 1: Parameter Validation ---
use <../lib/validate.scad>
use <../lib/validate-project.scad>

validate_part(width, height, depth, wall_thickness, fn,
              holes=[hole_diameter]);
validate_range("fillet_radius", fillet_radius, 0, min(wall_thickness, 5));
validate_positive("hole_diameter", hole_diameter);

// Advisory warnings (non-blocking, echo to stderr)
if (wall_thickness < 1.2)
    echo("WARN: wall_thickness=", wall_thickness, "mm is thin for structural parts");
if (fn > 128)
    echo("WARN: $fn=", fn, " will produce high polygon count");

// --- Geometry ---
module bracket() {
    difference() {
        union() {
            // Vertical plate
            cube([width, wall_thickness, height]);
            // Horizontal plate
            cube([width, depth, wall_thickness]);
            // Fillet (structural reinforcement)
            if (fillet_radius > 0) {
                translate([0, wall_thickness, wall_thickness])
                    rotate([0, 90, 0])
                        difference() {
                            cube([fillet_radius, fillet_radius, width]);
                            translate([fillet_radius, fillet_radius, -1])
                                cylinder(r=fillet_radius, h=width+2, $fn=fn);
                        }
            }
        }

        // Mounting holes in vertical plate
        for (x = [width * 0.25, width * 0.75]) {
            translate([x, -1, height * 0.6])
                rotate([-90, 0, 0])
                    cylinder(d=hole_diameter, h=wall_thickness+2, $fn=fn);
        }

        // Mounting holes in horizontal plate
        for (x = [width * 0.25, width * 0.75]) {
            translate([x, depth * 0.6, -1])
                cylinder(d=hole_diameter, h=wall_thickness+2, $fn=fn);
        }
    }
}

bracket();
