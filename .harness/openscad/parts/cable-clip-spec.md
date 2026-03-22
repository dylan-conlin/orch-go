# Cable Management Clip — Design Specification

## Purpose
Desk-mount cable management clip that holds USB/charging cables organized along a desk edge. Snap-fit attachment (no adhesive or screws).

## Key Dimensions
- Width: 25mm (along desk edge)
- Height: 15mm above desk surface
- Depth: 12mm front-to-back on desk surface
- Cable slots: 2 slots, 5mm diameter each, 8mm spacing
- Wall thickness: 2mm minimum

## Required Features
1. **Snap-fit desk clip**: U-shaped grip that clips onto desk edge (20mm thick desk)
2. **Cable slots**: 2 semi-circular retainer channels on top, open at front for cable insertion
3. **Parameter validation**: Assert gates for all dimensions

## Constraints
- FDM printable: wall thickness >= 0.8mm, no extreme overhangs
- Snap-fit gap: 1.5mm for desk attachment flexibility
- Cable diameter range: 2-15mm (parametric)
- Must fit 1-3 cables (configurable)

## Print Orientation
Print upright (clip opening facing down). No supports needed for snap-fit geometry.

## Verification Viewpoints

### top-down
Camera: 0,0,0,0,0,0,80
Checklist:
- Two cable slot openings visible from above
- Slots evenly spaced and centered on body
- Overall rectangular footprint with no degenerate geometry

### front
Camera: 0,0,0,90,0,0,80
Checklist:
- U-shaped clip profile visible (desk grip extending below)
- Cable retainer bumps visible on top surface
- Front opening slot for cable insertion visible

### side
Camera: 0,0,0,90,0,90,80
Checklist:
- Snap-fit gap visible between front wall and bottom grip
- Wall thickness appears adequate (not paper-thin)
- Clip depth proportional to desk thickness parameter
