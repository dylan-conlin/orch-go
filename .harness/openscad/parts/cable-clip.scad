// cable-clip.scad — Desk-mount cable management clip
// Holds 1-3 cables against a desk edge. Snap-fit design.
// Parameters override via: openscad -D 'param=value'

// --- Parameters ---
width = 25;            // mm — clip width along desk edge
height = 15;           // mm — total height above desk
depth = 12;            // mm — front-to-back depth
wall_thickness = 2;    // mm (minimum 0.8mm for FDM)
cable_diameter = 5;    // mm — cable slot diameter
num_slots = 2;         // number of cable slots (1-3)
slot_spacing = 8;      // mm — center-to-center spacing between slots
desk_thickness = 20;   // mm — desk edge thickness for clip
clip_gap = 1.5;        // mm — snap-fit gap (spring open distance)
fn = 32;               // facet count for curves

// --- Layer 1: Parameter Validation ---
use <../lib/validate.scad>
use <../lib/validate-project.scad>

validate_part(width, height, depth, wall_thickness, fn, holes=[cable_diameter]);
validate_range("cable_diameter", cable_diameter, 2, 15);
validate_range("num_slots", num_slots, 1, 3);
validate_range("slot_spacing", slot_spacing, cable_diameter + wall_thickness, width);
validate_range("desk_thickness", desk_thickness, 5, 50);
validate_range("clip_gap", clip_gap, 0.5, 5);
validate_positive("slot_spacing", slot_spacing);

// Ensure slots fit within width
total_slot_width = (num_slots - 1) * slot_spacing + cable_diameter;
assert(total_slot_width <= width - 2 * wall_thickness,
    str("GATE FAIL: total slot width=", total_slot_width,
        "mm > available width=", width - 2 * wall_thickness, "mm"));

// Advisory warnings
if (wall_thickness < 1.2)
    echo("WARN: wall_thickness=", wall_thickness, "mm is thin for snap-fit clips");
if (clip_gap > desk_thickness * 0.15)
    echo("WARN: clip_gap=", clip_gap, "mm may be too loose for desk_thickness=", desk_thickness);
if (fn > 128)
    echo("WARN: $fn=", fn, " will produce high polygon count");

// --- Geometry ---
module cable_clip() {
    clip_depth = desk_thickness + 2 * wall_thickness;

    difference() {
        union() {
            // Main body — top platform
            cube([width, depth, wall_thickness]);

            // Back wall (desk-side)
            translate([0, depth - wall_thickness, -clip_depth])
                cube([width, wall_thickness, clip_depth + wall_thickness]);

            // Bottom grip — U-shaped clip under desk
            translate([0, 0, -clip_depth])
                cube([width, depth, wall_thickness]);

            // Front wall (partial — snap-fit with gap)
            snap_height = clip_depth - clip_gap;
            translate([0, 0, -snap_height])
                cube([width, wall_thickness, snap_height]);

            // Cable retainer bumps on top
            for (i = [0 : num_slots - 1]) {
                slot_x = width / 2 - (num_slots - 1) * slot_spacing / 2 + i * slot_spacing;
                // Semi-circular retainer walls around each slot
                translate([slot_x, depth / 2, wall_thickness])
                    difference() {
                        cylinder(d=cable_diameter + 2 * wall_thickness, h=height - wall_thickness, $fn=fn);
                        cylinder(d=cable_diameter, h=height - wall_thickness + 1, $fn=fn);
                        // Opening for cable insertion (front slot)
                        translate([0, -(cable_diameter + 2 * wall_thickness) / 2, 0])
                            cube([cable_diameter / 2, cable_diameter + 2 * wall_thickness, height], center=false);
                    }
            }
        }

        // Cable slots through top platform
        for (i = [0 : num_slots - 1]) {
            slot_x = width / 2 - (num_slots - 1) * slot_spacing / 2 + i * slot_spacing;
            translate([slot_x, depth / 2, -1])
                cylinder(d=cable_diameter, h=wall_thickness + 2, $fn=fn);
        }
    }
}

cable_clip();
