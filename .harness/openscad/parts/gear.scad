// gear.scad — Parametric involute spur gear
// Demonstrates: trigonometric geometry, module-pitch system, hub/bore

// --- Parameters (override via -D flags) ---
teeth = 20;             // number of teeth
module_pitch = 2;       // mm — gear module (tooth size)
pressure_angle = 20;    // degrees — standard: 14.5 or 20
gear_thickness = 8;     // mm — face width
hub_diameter = 15;      // mm — central hub
hub_height = 4;         // mm — hub protrusion
bore_diameter = 5;      // mm — shaft bore
fn = 64;                // facet count (gears need higher resolution)

// --- Layer 1: Parameter Validation ---
use <../lib/validate.scad>

validate_positive("teeth", teeth);
validate_positive("module_pitch", module_pitch);
validate_positive("gear_thickness", gear_thickness);
validate_positive("hub_diameter", hub_diameter);
validate_positive("bore_diameter", bore_diameter);
validate_fn(fn);

// Gear-specific validations
assert(teeth >= 6,
    str("GATE FAIL: teeth=", teeth, " < 6 minimum (undercut occurs below ~12 teeth)"));
assert(module_pitch >= 0.5,
    str("GATE FAIL: module_pitch=", module_pitch, " < 0.5mm minimum"));
assert(module_pitch <= 10,
    str("GATE FAIL: module_pitch=", module_pitch, " > 10mm maximum"));
assert(pressure_angle >= 14 && pressure_angle <= 25,
    str("GATE FAIL: pressure_angle=", pressure_angle, " outside 14-25° range"));
assert(bore_diameter < hub_diameter,
    str("GATE FAIL: bore_diameter=", bore_diameter, " >= hub_diameter=", hub_diameter));

// Derived dimensions
pitch_radius = (teeth * module_pitch) / 2;
addendum = module_pitch;
dedendum = 1.25 * module_pitch;
outer_radius = pitch_radius + addendum;
root_radius = pitch_radius - dedendum;
base_radius = pitch_radius * cos(pressure_angle);

assert(root_radius > bore_diameter / 2 + 1,
    str("GATE FAIL: root_radius=", root_radius,
        "mm too close to bore_diameter/2=", bore_diameter / 2,
        "mm (need >1mm wall)"));

// Advisory
if (teeth < 12)
    echo("WARN: teeth=", teeth, " — significant undercut expected below 12 teeth");
if (fn < 64)
    echo("WARN: $fn=", fn, " — gear tooth profiles benefit from $fn >= 64");

// --- Geometry ---

// Involute curve point at parameter t (radians from base circle)
function involute_point(base_r, t) =
    [base_r * (cos(t * 180 / PI) + t * sin(t * 180 / PI)),
     base_r * (sin(t * 180 / PI) - t * cos(t * 180 / PI))];

// Generate involute curve points
function involute_curve(base_r, steps, max_r) =
    let(max_t = sqrt((max_r / base_r) * (max_r / base_r) - 1))
    [for (i = [0:steps])
        let(t = max_t * i / steps)
        involute_point(base_r, t)];

// Single tooth profile as 2D polygon
module tooth_profile() {
    // Approximate tooth with trapezoid + tip arc for printability
    tooth_angle = 360 / teeth;
    tip_width = module_pitch * 0.3;
    root_width = module_pitch * 0.5;

    // Simplified tooth profile (polygon approximation)
    half_angle = tooth_angle / 4;
    points = [
        [root_radius * cos(-half_angle), root_radius * sin(-half_angle)],
        [outer_radius * cos(-half_angle * 0.4), outer_radius * sin(-half_angle * 0.4)],
        [outer_radius * cos(half_angle * 0.4), outer_radius * sin(half_angle * 0.4)],
        [root_radius * cos(half_angle), root_radius * sin(half_angle)]
    ];
    polygon(points);
}

module gear_body() {
    difference() {
        union() {
            // Gear disc with teeth
            linear_extrude(height=gear_thickness) {
                // Root circle
                circle(r=root_radius, $fn=teeth * 4);

                // Teeth
                for (i = [0:teeth - 1]) {
                    rotate([0, 0, i * 360 / teeth])
                        tooth_profile();
                }
            }

            // Hub
            if (hub_height > 0) {
                translate([0, 0, gear_thickness])
                    cylinder(d=hub_diameter, h=hub_height, $fn=fn);
            }
        }

        // Bore hole (through entire part)
        translate([0, 0, -1])
            cylinder(d=bore_diameter, h=gear_thickness + hub_height + 2, $fn=fn);
    }
}

gear_body();
