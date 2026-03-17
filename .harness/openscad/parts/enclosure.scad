// enclosure.scad — Parametric electronics enclosure with lid and ventilation
// Demonstrates: interior space validation, lid/base separation, vent slots

// --- Parameters (override via -D flags) ---
width = 80;            // mm — outer width
height = 50;           // mm — outer height (base + lid)
depth = 60;            // mm — outer depth
wall_thickness = 2;    // mm (minimum 0.8mm for FDM)
lid_height = 12;       // mm — height of lid portion
lip_depth = 1.5;       // mm — interlocking lip depth
lip_clearance = 0.3;   // mm — clearance for lid fit
screw_diameter = 3;    // mm — corner screw holes
vent_slot_width = 2;   // mm — ventilation slot width
vent_slot_count = 5;   // number of vent slots per side
fn = 32;               // facet count for curves

// --- Layer 1: Parameter Validation ---
use <../lib/validate.scad>

validate_part(width, height, depth, wall_thickness, fn,
              holes=[screw_diameter]);
validate_range("lid_height", lid_height, wall_thickness + 1, height - wall_thickness - 1);
validate_range("lip_depth", lip_depth, 0.5, wall_thickness - 0.5);
validate_range("lip_clearance", lip_clearance, 0.1, 1.0);
validate_positive("vent_slot_width", vent_slot_width);
validate_range("vent_slot_count", vent_slot_count, 0, 20);

// Derived dimensions
base_height = height - lid_height;

// Advisory warnings
if (wall_thickness < 1.5)
    echo("WARN: wall_thickness=", wall_thickness, "mm may be fragile for an enclosure");
if (lip_clearance > 0.5)
    echo("WARN: lip_clearance=", lip_clearance, "mm is loose — lid may rattle");

// --- Geometry ---

module screw_posts() {
    inset = wall_thickness + screw_diameter;
    positions = [
        [inset, inset],
        [width - inset, inset],
        [width - inset, depth - inset],
        [inset, depth - inset]
    ];
    for (pos = positions) {
        translate([pos[0], pos[1], 0])
            difference() {
                cylinder(d=screw_diameter * 2.5, h=base_height - wall_thickness, $fn=fn);
                cylinder(d=screw_diameter, h=base_height - wall_thickness + 1, $fn=fn);
            }
    }
}

module vent_slots(face_width, face_height) {
    if (vent_slot_count > 0) {
        slot_spacing = face_width / (vent_slot_count + 1);
        slot_height = face_height * 0.5;
        for (i = [1:vent_slot_count]) {
            translate([slot_spacing * i - vent_slot_width / 2,
                       -1,
                       face_height * 0.25])
                cube([vent_slot_width, wall_thickness + 2, slot_height]);
        }
    }
}

module enclosure_base() {
    difference() {
        // Outer shell
        cube([width, depth, base_height]);

        // Hollow interior
        translate([wall_thickness, wall_thickness, wall_thickness])
            cube([width - 2 * wall_thickness,
                  depth - 2 * wall_thickness,
                  base_height]);

        // Vent slots on front face
        vent_slots(width, base_height);

        // Vent slots on back face
        translate([0, depth - wall_thickness, 0])
            vent_slots(width, base_height);
    }

    // Screw posts
    translate([0, 0, wall_thickness])
        screw_posts();
}

module enclosure_lid() {
    translate([0, 0, base_height + 2]) {  // Offset for visibility
        difference() {
            // Outer lid
            cube([width, depth, lid_height]);

            // Hollow interior
            translate([wall_thickness, wall_thickness, 0])
                cube([width - 2 * wall_thickness,
                      depth - 2 * wall_thickness,
                      lid_height - wall_thickness]);

            // Screw holes through lid corners
            inset = wall_thickness + screw_diameter;
            positions = [
                [inset, inset],
                [width - inset, inset],
                [width - inset, depth - inset],
                [inset, depth - inset]
            ];
            for (pos = positions) {
                translate([pos[0], pos[1], -1])
                    cylinder(d=screw_diameter, h=lid_height + 2, $fn=fn);
            }
        }

        // Interlocking lip
        translate([wall_thickness + lip_clearance,
                   wall_thickness + lip_clearance, 0])
            difference() {
                cube([width - 2 * (wall_thickness + lip_clearance),
                      depth - 2 * (wall_thickness + lip_clearance),
                      lip_depth]);
                translate([lip_depth, lip_depth, -1])
                    cube([width - 2 * (wall_thickness + lip_clearance + lip_depth),
                          depth - 2 * (wall_thickness + lip_clearance + lip_depth),
                          lip_depth + 2]);
            }
    }
}

enclosure_base();
enclosure_lid();
