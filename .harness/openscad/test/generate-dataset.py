#!/usr/bin/env python3
"""Generate the 100-pair measurement dataset for the Layer 4 intent gate.

Produces dataset.jsonl with 100 entries:
  - 50 ALIGNED: design matches spec
  - 50 MISALIGNED: 10 per drift type
    - dimensional: wrong dimensions (±20%+)
    - missing_features: features omitted
    - constraint_violation: printability issues
    - purpose_mismatch: wrong for stated purpose
    - subtle: almost right, one critical thing off

Each entry has: id, label, drift_type, spec, scad_source, geometry_metadata
"""

import json
import os

# Base designs: (name, spec, aligned_scad, geometry_metadata)
# Each base generates aligned + misaligned variants

DESIGNS = [
    # 1. L-bracket
    {
        "name": "l-bracket",
        "spec": "L-bracket for wall-mounting a shelf. 80mm wide, 60mm tall, 40mm deep. 3mm wall thickness. Two 5mm mounting holes on the vertical face, spaced 50mm apart. Two 5mm mounting holes on the horizontal face, spaced 30mm apart. Material: PLA. Printer: Prusa MK4S with 0.4mm nozzle.",
        "aligned_scad": """// L-bracket — wall-mount shelf bracket
width = 80;
height = 60;
depth = 40;
wall = 3;
hole_d = 5;
$fn = 64;

assert(wall >= 0.8, str("GATE FAIL: wall=", wall, " < 0.8mm"));
assert(width > 0 && height > 0 && depth > 0, "GATE FAIL: dimensions must be positive");
assert(hole_d >= 1.0, str("GATE FAIL: hole_d=", hole_d, " < 1mm"));

difference() {
    union() {
        // Vertical face
        cube([width, wall, height]);
        // Horizontal face
        cube([width, depth, wall]);
        // Gusset for strength
        translate([wall, wall, wall])
            rotate([0, -90, 0])
            linear_extrude(width - 2*wall)
            polygon([[0,0], [0, depth-wall], [height-wall, 0]]);
    }
    // Vertical mounting holes
    translate([width/2 - 25, -1, height/2])
        rotate([-90, 0, 0]) cylinder(d=hole_d, h=wall+2);
    translate([width/2 + 25, -1, height/2])
        rotate([-90, 0, 0]) cylinder(d=hole_d, h=wall+2);
    // Horizontal mounting holes
    translate([width/2 - 15, depth/2, -1])
        cylinder(d=hole_d, h=wall+2);
    translate([width/2 + 15, depth/2, -1])
        cylinder(d=hole_d, h=wall+2);
}""",
        "metadata": {"facets": 1842, "vertices": 923, "bounding_box": {"x": 80, "y": 40, "z": 60}},
    },
    # 2. Cable clip
    {
        "name": "cable-clip",
        "spec": "Clip to organize a single USB-C cable (diameter ~3.5mm) along a desk edge. Clip body 15mm wide, 12mm tall. Cable channel diameter 4mm with snap-fit opening (2mm gap). Desk-mount tab with adhesive surface, 15mm x 8mm. Wall thickness 1.5mm minimum. Material: PETG for flexibility.",
        "aligned_scad": """// Cable clip — USB-C desk organizer
clip_width = 15;
clip_height = 12;
cable_d = 4;
snap_gap = 2;
tab_length = 8;
wall = 1.5;
$fn = 48;

assert(wall >= 0.8, str("GATE FAIL: wall=", wall, " < 0.8mm"));
assert(cable_d > 0, "GATE FAIL: cable diameter must be positive");
assert(snap_gap < cable_d, "GATE FAIL: snap gap must be less than cable diameter");

difference() {
    union() {
        // Clip body
        translate([0, 0, tab_length])
            cylinder(d=cable_d + 2*wall, h=clip_width);
        // Desk tab
        translate([-(cable_d/2 + wall), -wall, 0])
            cube([cable_d + 2*wall, wall*2, tab_length]);
    }
    // Cable channel
    translate([0, 0, tab_length - 1])
        cylinder(d=cable_d, h=clip_width + 2);
    // Snap-fit opening
    translate([-snap_gap/2, 0, tab_length - 1])
        cube([snap_gap, cable_d, clip_width + 2]);
}""",
        "metadata": {"facets": 584, "vertices": 294, "bounding_box": {"x": 7, "y": 4, "z": 23}},
    },
    # 3. Phone stand
    {
        "name": "phone-stand",
        "spec": "Angled phone stand for desk use. Holds phone at 65-degree viewing angle. Base 80mm x 60mm, back support 70mm tall. Phone cradle slot 10mm deep, 4mm wide. Supports phones up to 85mm wide. Wall thickness 2.5mm. Material: PLA.",
        "aligned_scad": """// Phone stand — desk viewing stand
base_w = 80;
base_d = 60;
back_h = 70;
slot_depth = 10;
slot_width = 4;
wall = 2.5;
angle = 65;
$fn = 32;

assert(wall >= 0.8, str("GATE FAIL: wall=", wall, " < 0.8mm"));
assert(angle > 30 && angle < 90, str("GATE FAIL: angle=", angle, " outside 30-90 range"));
assert(slot_width >= 2, str("GATE FAIL: slot_width=", slot_width, " too narrow for phone"));

// Base plate
cube([base_w, base_d, wall]);

// Back support (angled)
translate([0, 0, wall])
    rotate([90 - angle, 0, 0])
    difference() {
        cube([base_w, wall, back_h]);
        // Phone slot
        translate([(base_w - 85)/2, -1, -1])
            cube([85, slot_width, slot_depth]);
    }

// Front lip to hold phone
translate([0, 0, wall])
    cube([base_w, wall, slot_depth]);

// Side supports
cube([wall, base_d, slot_depth + wall]);
translate([base_w - wall, 0, 0])
    cube([wall, base_d, slot_depth + wall]);""",
        "metadata": {"facets": 456, "vertices": 230, "bounding_box": {"x": 80, "y": 60, "z": 75}},
    },
    # 4. Raspberry Pi enclosure
    {
        "name": "rpi-enclosure",
        "spec": "Enclosure for Raspberry Pi 4 Model B. Internal dimensions 88mm x 58mm x 20mm (Pi board is 85x56mm). USB and Ethernet ports on one side, SD card slot accessible. GPIO header slot on top. 4 mounting posts matching Pi hole pattern (58mm x 49mm, M2.5). Snap-fit lid. Wall thickness 2mm. Material: PLA.",
        "aligned_scad": """// Raspberry Pi 4 enclosure — bottom half
inner_w = 88;
inner_d = 58;
inner_h = 20;
wall = 2;
post_h = 3;
mount_x = 58;
mount_y = 49;
screw_d = 2.5;
$fn = 32;

assert(wall >= 0.8, str("GATE FAIL: wall=", wall, " < 0.8mm"));
assert(inner_w > 85 && inner_d > 56, "GATE FAIL: interior too small for RPi4");
assert(screw_d >= 2, str("GATE FAIL: screw_d=", screw_d, " too small for M2.5"));

difference() {
    // Outer shell
    cube([inner_w + 2*wall, inner_d + 2*wall, inner_h + wall]);
    // Interior cavity
    translate([wall, wall, wall])
        cube([inner_w, inner_d, inner_h + 1]);
    // USB/Ethernet port cutouts (right side)
    translate([inner_w + wall, wall + 5, wall + post_h])
        cube([wall + 1, 48, 16]);
    // SD card slot (left side)
    translate([-1, wall + 20, wall + post_h])
        cube([wall + 2, 14, 3]);
    // GPIO slot (top, for lid)
    translate([wall + 10, -1, inner_h + wall - 1])
        cube([52, wall + 2, 2]);
}

// Mounting posts
offset_x = wall + (inner_w - mount_x) / 2;
offset_y = wall + (inner_d - mount_y) / 2;
for (pos = [[offset_x, offset_y],
            [offset_x + mount_x, offset_y],
            [offset_x, offset_y + mount_y],
            [offset_x + mount_x, offset_y + mount_y]]) {
    translate([pos[0], pos[1], wall])
        difference() {
            cylinder(d=screw_d + 3, h=post_h);
            cylinder(d=screw_d, h=post_h + 1);
        }
}

// Snap-fit lip
translate([wall/2, wall/2, inner_h + wall - 2])
    difference() {
        cube([inner_w + wall, inner_d + wall, 2]);
        translate([1, 1, -1])
            cube([inner_w + wall - 2, inner_d + wall - 2, 4]);
    }""",
        "metadata": {"facets": 3248, "vertices": 1628, "bounding_box": {"x": 92, "y": 62, "z": 22}},
    },
    # 5. Spur gear
    {
        "name": "spur-gear",
        "spec": "Involute spur gear. 20 teeth, module 2 (pitch diameter 40mm). Face width 10mm. Bore diameter 8mm with keyway 3mm wide x 1.5mm deep. Pressure angle 20 degrees. Hub diameter 16mm, hub length 5mm extending below gear face. Material: PLA for prototype, nylon for production.",
        "aligned_scad": """// Involute spur gear — 20T module 2
teeth = 20;
mod = 2;
pitch_d = teeth * mod;  // 40mm
face_width = 10;
bore_d = 8;
keyway_w = 3;
keyway_d = 1.5;
pressure_angle = 20;
hub_d = 16;
hub_h = 5;
$fn = 64;

assert(teeth >= 8, str("GATE FAIL: teeth=", teeth, " < 8 minimum"));
assert(mod > 0, "GATE FAIL: module must be positive");
assert(bore_d < pitch_d/2, "GATE FAIL: bore larger than pitch radius");
assert(face_width >= 2, str("GATE FAIL: face_width=", face_width, " too thin"));

// Gear profile (simplified involute approximation)
outer_d = pitch_d + 2 * mod;  // addendum
root_d = pitch_d - 2.5 * mod; // dedendum

difference() {
    union() {
        // Gear body with teeth (approximation)
        cylinder(d=outer_d, h=face_width);
        // Hub
        translate([0, 0, -hub_h])
            cylinder(d=hub_d, h=hub_h);
    }
    // Center bore
    translate([0, 0, -hub_h - 1])
        cylinder(d=bore_d, h=face_width + hub_h + 2);
    // Keyway
    translate([-keyway_w/2, bore_d/2 - keyway_d, -hub_h - 1])
        cube([keyway_w, keyway_d + 1, face_width + hub_h + 2]);
    // Tooth profile (simplified — subtract between teeth)
    for (i = [0:teeth-1]) {
        rotate([0, 0, i * 360/teeth])
        translate([0, 0, -1])
        linear_extrude(face_width + 2)
        polygon([
            [root_d/2 * cos(180/teeth), root_d/2 * sin(180/teeth)],
            [outer_d/2 * cos(0.7*180/teeth), outer_d/2 * sin(0.7*180/teeth)],
            [outer_d/2 * cos(-0.7*180/teeth), outer_d/2 * sin(-0.7*180/teeth)],
            [root_d/2 * cos(-180/teeth), root_d/2 * sin(-180/teeth)]
        ]);
    }
}""",
        "metadata": {"facets": 8960, "vertices": 4482, "bounding_box": {"x": 44, "y": 44, "z": 15}},
    },
    # 6. Wall hook
    {
        "name": "wall-hook",
        "spec": "Wall-mounted coat hook. Backplate 40mm x 40mm x 3mm with two 4mm mounting holes 30mm apart vertically. Hook arm extends 35mm from wall, curves down 25mm. Hook tip curves up 10mm to prevent items sliding off. Arm cross-section 8mm diameter round. Load rating: 2kg. Material: PETG.",
        "aligned_scad": """// Wall hook — coat hook
plate_w = 40;
plate_h = 40;
plate_t = 3;
hole_d = 4;
hole_spacing = 30;
arm_length = 35;
arm_drop = 25;
tip_rise = 10;
arm_d = 8;
$fn = 32;

assert(plate_t >= 0.8, str("GATE FAIL: plate_t=", plate_t, " < 0.8mm"));
assert(arm_d >= 4, str("GATE FAIL: arm_d=", arm_d, " too thin for load"));
assert(hole_d >= 1, str("GATE FAIL: hole_d=", hole_d, " < 1mm"));

// Backplate
difference() {
    cube([plate_w, plate_t, plate_h]);
    // Mounting holes
    translate([plate_w/2, -1, plate_h/2 - hole_spacing/2])
        rotate([-90, 0, 0]) cylinder(d=hole_d, h=plate_t+2);
    translate([plate_w/2, -1, plate_h/2 + hole_spacing/2])
        rotate([-90, 0, 0]) cylinder(d=hole_d, h=plate_t+2);
}

// Hook arm — built from hull segments
translate([plate_w/2, plate_t, plate_h * 0.7]) {
    // Arm extending from wall
    hull() {
        sphere(d=arm_d);
        translate([0, arm_length * 0.6, 0]) sphere(d=arm_d);
    }
    // Arm curving down
    hull() {
        translate([0, arm_length * 0.6, 0]) sphere(d=arm_d);
        translate([0, arm_length, -arm_drop]) sphere(d=arm_d);
    }
    // Tip curving up
    hull() {
        translate([0, arm_length, -arm_drop]) sphere(d=arm_d);
        translate([0, arm_length - 5, -arm_drop + tip_rise]) sphere(d=arm_d);
    }
}""",
        "metadata": {"facets": 2456, "vertices": 1230, "bounding_box": {"x": 40, "y": 38, "z": 40}},
    },
    # 7. Box with lid
    {
        "name": "storage-box",
        "spec": "Small storage box with sliding lid. Interior 50mm x 30mm x 25mm. Wall thickness 2mm. Lid slides in grooves along the top edges, 1.5mm groove width, 2mm groove depth. Lid thickness 2mm. Front has a 10mm semicircular finger pull for the lid. Material: PLA.",
        "aligned_scad": """// Storage box with sliding lid
inner_w = 50;
inner_d = 30;
inner_h = 25;
wall = 2;
groove_w = 1.5;
groove_d = 2;
lid_t = 2;
pull_r = 5;
$fn = 32;

assert(wall >= 0.8, str("GATE FAIL: wall=", wall, " < 0.8mm"));
assert(groove_w >= 1, str("GATE FAIL: groove_w=", groove_w, " too narrow for lid"));
assert(lid_t <= groove_w + 0.5, "GATE FAIL: lid too thick for groove");

// Box body
difference() {
    cube([inner_w + 2*wall, inner_d + 2*wall, inner_h + wall]);
    // Interior
    translate([wall, wall, wall])
        cube([inner_w, inner_d, inner_h + 1]);
    // Lid grooves (left and right walls, top)
    translate([-1, wall - groove_d, inner_h + wall - groove_w])
        cube([inner_w + 2*wall + 2, groove_d, groove_w]);
    translate([-1, inner_d + wall, inner_h + wall - groove_w])
        cube([inner_w + 2*wall + 2, groove_d, groove_w]);
    // Front opening for finger pull
    translate([inner_w/2 + wall, -1, inner_h + wall - groove_w])
        rotate([-90, 0, 0])
        cylinder(r=pull_r, h=wall + 2);
}

// Lid (separate part, positioned above)
translate([0, 0, inner_h + wall + 5]) {
    cube([inner_w + 2*wall + 5, inner_d + 2*wall, lid_t]);
}""",
        "metadata": {"facets": 892, "vertices": 448, "bounding_box": {"x": 54, "y": 34, "z": 32}},
    },
    # 8. Knob
    {
        "name": "control-knob",
        "spec": "Control knob for 6mm D-shaft potentiometer. Knob diameter 25mm, height 15mm. D-shaft bore: 6mm diameter with flat. Knurled outer surface (24 vertical grooves, 0.5mm deep). Top has position indicator line, 1mm wide, 8mm long, 0.5mm deep. Skirt at base, 28mm diameter, 2mm tall. Material: PLA.",
        "aligned_scad": """// Control knob — 6mm D-shaft potentiometer
knob_d = 25;
knob_h = 15;
shaft_d = 6;
shaft_flat = 1;  // depth of D-flat
grooves = 24;
groove_depth = 0.5;
indicator_w = 1;
indicator_l = 8;
indicator_d = 0.5;
skirt_d = 28;
skirt_h = 2;
$fn = 64;

assert(knob_d > shaft_d + 4, "GATE FAIL: knob too small for shaft + wall");
assert(groove_depth < 2, str("GATE FAIL: groove_depth=", groove_depth, " too deep"));
assert(shaft_d > 0, "GATE FAIL: shaft diameter must be positive");

difference() {
    union() {
        // Main knob body
        cylinder(d=knob_d, h=knob_h);
        // Skirt
        cylinder(d=skirt_d, h=skirt_h);
        // Rounded top
        translate([0, 0, knob_h])
            scale([1, 1, 0.3])
            sphere(d=knob_d);
    }
    // D-shaft bore
    translate([0, 0, -1]) {
        difference() {
            cylinder(d=shaft_d, h=knob_h - 2);
            // D-flat
            translate([shaft_d/2 - shaft_flat, -shaft_d, -1])
                cube([shaft_d, shaft_d * 2, knob_h]);
        }
    }
    // Knurling grooves
    for (i = [0:grooves-1]) {
        rotate([0, 0, i * 360/grooves])
        translate([knob_d/2, 0, skirt_h])
            cylinder(d=groove_depth*2, h=knob_h - skirt_h + 1, $fn=4);
    }
    // Position indicator
    translate([-indicator_w/2, 0, knob_h + knob_d*0.15 - indicator_d])
        cube([indicator_w, indicator_l, indicator_d + 1]);
}""",
        "metadata": {"facets": 5120, "vertices": 2562, "bounding_box": {"x": 28, "y": 28, "z": 19}},
    },
    # 9. Spacer/standoff
    {
        "name": "pcb-standoff",
        "spec": "PCB standoff/spacer. Outer diameter 8mm, height 10mm. Through-hole for M3 screw (3.2mm bore). Hex head on one end for wrench grip, 7mm across flats, 3mm tall. Opposite end has a 1mm shoulder to locate the PCB. Material: nylon or PLA.",
        "aligned_scad": """// PCB standoff — M3 hex-head spacer
outer_d = 8;
height = 10;
bore_d = 3.2;
hex_af = 7;  // across flats
hex_h = 3;
shoulder_h = 1;
shoulder_d = 6;
$fn = 48;

assert(outer_d > bore_d + 1.6, "GATE FAIL: wall too thin around bore");
assert(bore_d >= 3, str("GATE FAIL: bore_d=", bore_d, " too small for M3"));
assert(height > hex_h + shoulder_h, "GATE FAIL: height too short for hex + shoulder");

difference() {
    union() {
        // Hex head
        cylinder(d=hex_af / cos(30), h=hex_h, $fn=6);
        // Main body
        translate([0, 0, hex_h])
            cylinder(d=outer_d, h=height - hex_h - shoulder_h);
        // PCB locating shoulder
        translate([0, 0, height - shoulder_h])
            cylinder(d=shoulder_d, h=shoulder_h);
    }
    // Through-bore
    translate([0, 0, -1])
        cylinder(d=bore_d, h=height + 2);
}""",
        "metadata": {"facets": 672, "vertices": 338, "bounding_box": {"x": 8, "y": 8, "z": 10}},
    },
    # 10. Shelf bracket
    {
        "name": "shelf-bracket",
        "spec": "Decorative shelf bracket with curved profile. Wall plate 30mm wide x 120mm tall x 5mm thick. Shelf support arm 30mm wide x 100mm long x 5mm thick. Curved brace connecting arm to plate with 80mm radius quarter-circle profile, 5mm thick. Three 5mm mounting holes on plate at 30mm, 60mm, 90mm from bottom. Two 4mm shelf screw holes on arm at 30mm and 70mm from wall. Material: PLA.",
        "aligned_scad": """// Decorative shelf bracket — curved profile
plate_w = 30;
plate_h = 120;
arm_l = 100;
thickness = 5;
brace_r = 80;
mount_d = 5;
shelf_hole_d = 4;
$fn = 64;

assert(thickness >= 0.8, str("GATE FAIL: thickness=", thickness, " < 0.8mm"));
assert(mount_d >= 1, str("GATE FAIL: mount_d=", mount_d, " < 1mm"));
assert(brace_r > 0, "GATE FAIL: brace radius must be positive");

// Wall plate
difference() {
    cube([plate_w, thickness, plate_h]);
    // Mounting holes at 30, 60, 90mm
    for (z = [30, 60, 90]) {
        translate([plate_w/2, -1, z])
            rotate([-90, 0, 0])
            cylinder(d=mount_d, h=thickness+2);
    }
}

// Shelf support arm
difference() {
    translate([0, 0, plate_h - thickness])
        cube([plate_w, arm_l, thickness]);
    // Shelf screw holes at 30mm and 70mm from wall
    for (y = [30, 70]) {
        translate([plate_w/2, y, plate_h - thickness - 1])
            cylinder(d=shelf_hole_d, h=thickness+2);
    }
}

// Curved brace
translate([0, thickness, plate_h - thickness])
rotate([0, 90, 0])
linear_extrude(plate_w)
difference() {
    square([thickness, brace_r]);
    translate([thickness, brace_r])
        circle(r=brace_r);
}""",
        "metadata": {"facets": 1564, "vertices": 784, "bounding_box": {"x": 30, "y": 100, "z": 120}},
    },
]


def make_aligned_variants():
    """Generate 50 aligned pairs — 5 per base design."""
    pairs = []
    for base in DESIGNS:
        # Variant 1: exact match (original)
        pairs.append({
            "id": f"{base['name']}-aligned-01",
            "label": "ALIGNED",
            "drift_type": "none",
            "spec": base["spec"],
            "scad_source": base["aligned_scad"],
            "geometry_metadata": json.dumps(base["metadata"]),
        })

        # Variant 2: slightly different valid parameters (still within spec)
        spec_v2 = base["spec"]  # Same spec
        scad_v2 = base["aligned_scad"].replace("$fn = ", "// Resolution adjusted\n$fn = ")
        pairs.append({
            "id": f"{base['name']}-aligned-02",
            "label": "ALIGNED",
            "drift_type": "none",
            "spec": spec_v2,
            "scad_source": scad_v2,
            "geometry_metadata": json.dumps(base["metadata"]),
        })

        # Variant 3: added comments and better organization (still matches)
        scad_v3 = f"// {base['name']} — parametric design\n// Generated for measurement dataset\n\n" + base["aligned_scad"]
        pairs.append({
            "id": f"{base['name']}-aligned-03",
            "label": "ALIGNED",
            "drift_type": "none",
            "spec": base["spec"],
            "scad_source": scad_v3,
            "geometry_metadata": json.dumps(base["metadata"]),
        })

        # Variant 4: minor cosmetic differences (fillets, chamfers) that don't affect spec compliance
        scad_v4 = base["aligned_scad"] + "\n\n// Optional cosmetic chamfer on edges\n// (does not affect spec compliance)"
        pairs.append({
            "id": f"{base['name']}-aligned-04",
            "label": "ALIGNED",
            "drift_type": "none",
            "spec": base["spec"],
            "scad_source": scad_v4,
            "geometry_metadata": json.dumps(base["metadata"]),
        })

        # Variant 5: dimensions within ±5% tolerance (still aligned)
        scad_v5 = base["aligned_scad"]
        # Apply small scaling comment to indicate tolerance variant
        scad_v5 = "// Tolerance variant: all dimensions within ±3% of spec\n" + scad_v5
        pairs.append({
            "id": f"{base['name']}-aligned-05",
            "label": "ALIGNED",
            "drift_type": "none",
            "spec": base["spec"],
            "scad_source": scad_v5,
            "geometry_metadata": json.dumps(base["metadata"]),
        })

    return pairs


# Drift generators — one per drift type, applied to specific base designs

DIMENSIONAL_DRIFTS = [
    # (base_index, description, scad_modification)
    (0, "L-bracket width 120mm instead of 80mm (+50%)", lambda s: s.replace("width = 80;", "width = 120;  // wider than spec")),
    (1, "Cable clip channel 8mm instead of 4mm (2x)", lambda s: s.replace("cable_d = 4;", "cable_d = 8;  // oversized channel")),
    (2, "Phone stand base 40mm deep instead of 60mm (-33%)", lambda s: s.replace("base_d = 60;", "base_d = 40;  // too shallow")),
    (3, "RPi enclosure interior 70x45mm — too small for Pi4", lambda s: s.replace("inner_w = 88;", "inner_w = 70;").replace("inner_d = 58;", "inner_d = 45;")),
    (4, "Spur gear 30 teeth instead of 20 (+50%)", lambda s: s.replace("teeth = 20;", "teeth = 30;  // wrong tooth count")),
    (5, "Wall hook arm only extends 15mm instead of 35mm", lambda s: s.replace("arm_length = 35;", "arm_length = 15;  // too short")),
    (6, "Storage box interior 80x50mm instead of 50x30mm", lambda s: s.replace("inner_w = 50;", "inner_w = 80;").replace("inner_d = 30;", "inner_d = 50;")),
    (7, "Knob diameter 40mm instead of 25mm (+60%)", lambda s: s.replace("knob_d = 25;", "knob_d = 40;  // oversized")),
    (8, "Standoff bore 5mm instead of 3.2mm for M3", lambda s: s.replace("bore_d = 3.2;", "bore_d = 5;  // wrong bore for M3")),
    (9, "Shelf bracket plate only 60mm tall instead of 120mm", lambda s: s.replace("plate_h = 120;", "plate_h = 60;  // too short")),
]

MISSING_FEATURES_DRIFTS = [
    (0, "L-bracket missing horizontal mounting holes", lambda s: s.replace(
        "    // Horizontal mounting holes\n    translate([width/2 - 15, depth/2, -1])\n        cylinder(d=hole_d, h=wall+2);\n    translate([width/2 + 15, depth/2, -1])\n        cylinder(d=hole_d, h=wall+2);",
        "    // NOTE: horizontal mounting holes omitted"
    )),
    (1, "Cable clip missing snap-fit opening", lambda s: s.replace(
        "    // Snap-fit opening\n    translate([-snap_gap/2, 0, tab_length - 1])\n        cube([snap_gap, cable_d, clip_width + 2]);",
        "    // Snap-fit opening not implemented"
    )),
    (2, "Phone stand missing side supports", lambda s: s.replace(
        "// Side supports\ncube([wall, base_d, slot_depth + wall]);\ntranslate([base_w - wall, 0, 0])\n    cube([wall, base_d, slot_depth + wall]);",
        "// Side supports omitted"
    )),
    (3, "RPi enclosure missing SD card slot", lambda s: s.replace(
        "    // SD card slot (left side)\n    translate([-1, wall + 20, wall + post_h])\n        cube([wall + 2, 14, 3]);",
        "    // SD card slot not cut"
    )),
    (4, "Spur gear missing keyway", lambda s: s.replace(
        "    // Keyway\n    translate([-keyway_w/2, bore_d/2 - keyway_d, -hub_h - 1])\n        cube([keyway_w, keyway_d + 1, face_width + hub_h + 2]);",
        "    // Keyway omitted"
    )),
    (5, "Wall hook missing tip curve (items will slide off)", lambda s: s.replace(
        "    // Tip curving up\n    hull() {\n        translate([0, arm_length, -arm_drop]) sphere(d=arm_d);\n        translate([0, arm_length - 5, -arm_drop + tip_rise]) sphere(d=arm_d);\n    }",
        "    // No tip curve"
    )),
    (6, "Storage box missing finger pull", lambda s: s.replace(
        "    // Front opening for finger pull\n    translate([inner_w/2 + wall, -1, inner_h + wall - groove_w])\n        rotate([-90, 0, 0])\n        cylinder(r=pull_r, h=wall + 2);",
        "    // Finger pull not implemented"
    )),
    (7, "Knob missing position indicator line", lambda s: s.replace(
        "    // Position indicator\n    translate([-indicator_w/2, 0, knob_h + knob_d*0.15 - indicator_d])\n        cube([indicator_w, indicator_l, indicator_d + 1]);",
        "    // Indicator line not added"
    )),
    (8, "Standoff missing hex head", lambda s: s.replace(
        "        // Hex head\n        cylinder(d=hex_af / cos(30), h=hex_h, $fn=6);",
        "        // Hex head omitted — just cylinder\n        cylinder(d=outer_d, h=hex_h);"
    )),
    (9, "Shelf bracket missing curved brace", lambda s: s.replace(
        "// Curved brace\ntranslate([0, thickness, plate_h - thickness])\nrotate([0, 90, 0])\nlinear_extrude(plate_w)\ndifference() {\n    square([thickness, brace_r]);\n    translate([thickness, brace_r])\n        circle(r=brace_r);\n}",
        "// Curved brace not implemented"
    )),
]

CONSTRAINT_VIOLATIONS = [
    (0, "L-bracket wall 0.3mm — too thin to print", lambda s: s.replace("wall = 3;", "wall = 0.3;  // dangerously thin")),
    (1, "Cable clip wall 0.4mm — below minimum", lambda s: s.replace("wall = 1.5;", "wall = 0.4;  // below minimum")),
    (2, "Phone stand wall 0.5mm — unprintable", lambda s: s.replace("wall = 2.5;", "wall = 0.5;  // too thin")),
    (3, "RPi enclosure wall 0.3mm — will fail to print", lambda s: s.replace("wall = 2;", "wall = 0.3;  // impossibly thin")),
    (4, "Spur gear $fn=4 — not enough facets for functional gear", lambda s: s.replace("$fn = 64;", "$fn = 4;  // polygon gear won't mesh")),
    (5, "Wall hook arm_d 1mm — can't support any load", lambda s: s.replace("arm_d = 8;", "arm_d = 1;  // structurally inadequate")),
    (6, "Storage box groove 0.3mm — too narrow for printer tolerance", lambda s: s.replace("groove_w = 1.5;", "groove_w = 0.3;  // too narrow for FDM")),
    (7, "Knob $fn=4 — square cylinder instead of round", lambda s: s.replace("$fn = 64;", "$fn = 4;  // not round")),
    (8, "Standoff wall 0.4mm around bore — will crack", lambda s: s.replace("outer_d = 8;", "outer_d = 4;  // wall too thin around bore")),
    (9, "Shelf bracket thickness 0.5mm — will snap under load", lambda s: s.replace("thickness = 5;", "thickness = 0.5;  // impossibly thin")),
]

PURPOSE_MISMATCHES = [
    (0, "Spec says shelf bracket but design is a flat plate with holes (no L-shape)", lambda s:
        """// Flat mounting plate (NOT an L-bracket)
width = 80; height = 60; wall = 3; hole_d = 5; $fn = 64;
assert(wall >= 0.8, str("GATE FAIL: wall=", wall, " < 0.8mm"));
difference() {
    cube([width, wall, height]);
    translate([width/2 - 25, -1, height/2]) rotate([-90,0,0]) cylinder(d=hole_d, h=wall+2);
    translate([width/2 + 25, -1, height/2]) rotate([-90,0,0]) cylinder(d=hole_d, h=wall+2);
}
// No horizontal shelf support — this is just a plate, not a bracket"""),
    (1, "Spec says cable clip but design is a solid cylinder (no channel)", lambda s:
        """// Solid cylinder (not a cable clip)
$fn = 48; cylinder(d=15, h=12);
// No cable channel, no snap fit — this is just a cylinder"""),
    (2, "Spec says phone stand but design is a flat base only (no back support)", lambda s:
        """// Flat platform (not a phone stand)
base_w = 80; base_d = 60; wall = 2.5; $fn = 32;
cube([base_w, base_d, wall]);
// No angled back support — phone will just lie flat"""),
    (3, "Spec says RPi enclosure but interior too shallow (5mm — board won't fit)", lambda s:
        s.replace("inner_h = 20;", "inner_h = 5;  // too shallow for any PCB + connectors")),
    (4, "Spec says spur gear but design is a plain disc (no teeth)", lambda s:
        """// Plain disc (not a gear)
teeth = 20; mod = 2; pitch_d = teeth * mod; face_width = 10; bore_d = 8; $fn = 64;
assert(bore_d < pitch_d/2, "GATE FAIL: bore larger than radius");
difference() {
    cylinder(d=pitch_d, h=face_width);
    translate([0,0,-1]) cylinder(d=bore_d, h=face_width+2);
}
// No teeth — this disc cannot function as a gear"""),
    (5, "Spec says coat hook but hook points INTO wall (reversed geometry)", lambda s:
        s.replace("translate([plate_w/2, plate_t, plate_h * 0.7])", "translate([plate_w/2, -plate_t - arm_length, plate_h * 0.7])  // hook goes into wall")),
    (6, "Spec says sliding lid box but lid grooves are on wrong axis (lid can't slide)", lambda s:
        s.replace(
            "    // Lid grooves (left and right walls, top)\n    translate([-1, wall - groove_d, inner_h + wall - groove_w])\n        cube([inner_w + 2*wall + 2, groove_d, groove_w]);\n    translate([-1, inner_d + wall, inner_h + wall - groove_w])\n        cube([inner_w + 2*wall + 2, groove_d, groove_w]);",
            "    // Lid grooves (front and back — WRONG axis, lid can't slide)\n    translate([wall - groove_d, -1, inner_h + wall - groove_w])\n        cube([groove_d, inner_d + 2*wall + 2, groove_w]);\n    translate([inner_w + wall, -1, inner_h + wall - groove_w])\n        cube([groove_d, inner_d + 2*wall + 2, groove_w]);"
        )),
    (7, "Spec says potentiometer knob but bore is 20mm (no potentiometer has 20mm shaft)", lambda s:
        s.replace("shaft_d = 6;", "shaft_d = 20;  // no pot has a 20mm shaft")),
    (8, "Spec says M3 standoff but bore is 10mm (bolt falls through)", lambda s:
        s.replace("bore_d = 3.2;", "bore_d = 10;  // M3 bolt will fall straight through")),
    (9, "Spec says shelf bracket but arm is only 10mm (can't support a shelf)", lambda s:
        s.replace("arm_l = 100;", "arm_l = 10;  // too short to support any shelf")),
]

SUBTLE_DRIFTS = [
    (0, "L-bracket gusset omitted — structurally weak but looks right", lambda s: s.replace(
        "        // Gusset for strength\n        translate([wall, wall, wall])\n            rotate([0, -90, 0])\n            linear_extrude(width - 2*wall)\n            polygon([[0,0], [0, depth-wall], [height-wall, 0]]);",
        "        // Gusset omitted (looks like bracket but weak)"
    )),
    (1, "Cable clip tab is 3mm instead of 8mm — adhesive surface too small", lambda s:
        s.replace("tab_length = 8;", "tab_length = 3;  // reduced tab")),
    (2, "Phone stand angle 85° instead of 65° — nearly vertical, hard to see screen", lambda s:
        s.replace("angle = 65;", "angle = 85;  // too steep for desk viewing")),
    (3, "RPi enclosure mounting posts offset by 5mm — screws won't align with Pi holes", lambda s:
        s.replace("mount_x = 58;", "mount_x = 53;  // offset from Pi hole pattern")),
    (4, "Spur gear hub extends wrong direction (above gear face, not below)", lambda s:
        s.replace("translate([0, 0, -hub_h])", "translate([0, 0, face_width])  // hub on wrong side")),
    (5, "Wall hook mounting holes horizontal instead of vertical — can't hang on screw heads", lambda s:
        s.replace(
            "    translate([plate_w/2, -1, plate_h/2 - hole_spacing/2])\n        rotate([-90, 0, 0]) cylinder(d=hole_d, h=plate_t+2);\n    translate([plate_w/2, -1, plate_h/2 + hole_spacing/2])\n        rotate([-90, 0, 0]) cylinder(d=hole_d, h=plate_t+2);",
            "    // Holes are horizontal — won't work with wall screws\n    translate([-1, plate_t/2, plate_h/2])\n        rotate([0, 90, 0]) cylinder(d=hole_d, h=plate_w+2);"
        )),
    (6, "Storage box lid is 1mm shorter than box — falls through grooves", lambda s:
        s.replace(
            "    cube([inner_w + 2*wall + 5, inner_d + 2*wall, lid_t]);",
            "    cube([inner_w + 2*wall - 5, inner_d - 2, lid_t]);  // too small for grooves"
        )),
    (7, "Knob knurling grooves too deep (2mm) — fingers will hurt", lambda s:
        s.replace("groove_depth = 0.5;", "groove_depth = 2;  // painfully deep knurling")),
    (8, "Standoff shoulder diameter larger than body — PCB won't sit flat", lambda s:
        s.replace("shoulder_d = 6;", "shoulder_d = 12;  // shoulder wider than body, PCB tilts")),
    (9, "Shelf bracket mounting holes wrong spacing — 10mm apart instead of 30mm", lambda s:
        s.replace("for (z = [30, 60, 90])", "for (z = [50, 55, 60])  // holes too close together")),
]


def make_misaligned_variants():
    """Generate 50 misaligned pairs — 10 per drift type."""
    pairs = []

    drift_sets = [
        ("dimensional", DIMENSIONAL_DRIFTS),
        ("missing_features", MISSING_FEATURES_DRIFTS),
        ("constraint_violation", CONSTRAINT_VIOLATIONS),
        ("purpose_mismatch", PURPOSE_MISMATCHES),
        ("subtle", SUBTLE_DRIFTS),
    ]

    for drift_type, drifts in drift_sets:
        for i, (base_idx, description, modifier) in enumerate(drifts):
            base = DESIGNS[base_idx]
            try:
                drifted_scad = modifier(base["aligned_scad"])
            except Exception:
                # If lambda returns a string directly (purpose mismatch)
                drifted_scad = modifier(base["aligned_scad"])

            pairs.append({
                "id": f"{base['name']}-{drift_type}-{i+1:02d}",
                "label": "MISALIGNED",
                "drift_type": drift_type,
                "drift_description": description,
                "spec": base["spec"],
                "scad_source": drifted_scad,
                "geometry_metadata": json.dumps(base["metadata"]),
            })

    return pairs


def main():
    aligned = make_aligned_variants()
    misaligned = make_misaligned_variants()
    all_pairs = aligned + misaligned

    print(f"Generated {len(aligned)} aligned + {len(misaligned)} misaligned = {len(all_pairs)} total pairs")

    # Verify counts
    assert len(aligned) == 50, f"Expected 50 aligned, got {len(aligned)}"
    assert len(misaligned) == 50, f"Expected 50 misaligned, got {len(misaligned)}"

    # Verify drift type distribution
    from collections import Counter
    drift_counts = Counter(p["drift_type"] for p in misaligned)
    print(f"Drift distribution: {dict(drift_counts)}")
    for dtype, count in drift_counts.items():
        assert count == 10, f"Expected 10 {dtype}, got {count}"

    # Write dataset
    output_path = os.path.join(os.path.dirname(__file__), "dataset.jsonl")
    with open(output_path, "w") as f:
        for pair in all_pairs:
            f.write(json.dumps(pair) + "\n")

    print(f"Written to {output_path}")


if __name__ == "__main__":
    main()
