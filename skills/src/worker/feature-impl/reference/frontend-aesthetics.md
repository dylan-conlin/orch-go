# Frontend Aesthetics Guidelines

**Purpose:** Design principles for UI/frontend work. Consult when implementing visual features, dashboards, or user-facing interfaces.

**When to use:** Reference this during Design phase for UI work, or during Implementation when making visual decisions.

---

## Core Principle

**Commit to a BOLD aesthetic direction** rather than safe defaults.

Every UI decision should serve a deliberate aesthetic vision. Avoid the temptation to use "industry standard" patterns that result in forgettable, cookie-cutter interfaces.

---

## Design Dimensions

### Typography

**Avoid:**
- System font stacks (`-apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto'...`)
- Inter, Roboto, Arial, or other ubiquitous sans-serifs
- Using the "safe" choice by default

**Instead:**
- Choose distinctive fonts that establish character
- Examples: Playfair Display (elegant serif), JetBrains Mono (technical monospace), Space Grotesk (modern geometric)
- Font choice should reflect the product's personality

**Principle:** The font IS part of the brand. A generic font choice signals generic thinking.

---

### Color

**Avoid:**
- Timid, safe palettes (generic gray-900/800/700 dark themes)
- Purple gradients on white backgrounds (overused AI aesthetic)
- Neutral palettes that "work but don't delight"

**Instead:**
- Dominant color with sharp accents
- Bold, intentional color choices that create visual hierarchy
- Consider unusual combinations that feel fresh

**Principle:** Color creates emotional response. Neutral colors create neutral reactions.

---

### Motion

**Avoid:**
- Scattered micro-interactions everywhere
- Motion for motion's sake
- Jarring or distracting animations

**Instead:**
- One well-orchestrated page load animation
- Purposeful motion that guides attention
- Staggered reveals, hover surprises, smooth transitions
- Motion should feel intentional and cohesive

**Principle:** Less motion, better choreographed. Quality over quantity.

---

### Backgrounds & Spatial Composition

**Avoid:**
- Solid, flat background colors
- Blank canvas with floating elements
- Generic whitespace usage

**Instead:**
- Atmosphere and depth (subtle gradients, texture, layering)
- Intentional spatial relationships
- Background treatments that create context

**Principle:** The background is not "nothing" - it's negative space that shapes the positive.

---

## Aesthetic Directions (Choose One)

When starting UI work, commit to an aesthetic direction:

- **Brutally Minimal:** Pure function, no decoration, stark contrasts
- **Maximalist Chaos:** Dense information, bold colors, layered complexity
- **Retro-Futuristic:** Neon, grids, synthwave aesthetics, technological nostalgia
- **Luxury/Refined:** Serif fonts, muted colors, generous whitespace, premium feel
- **Technical/Developer:** Monospace fonts, terminal aesthetics, data-dense layouts
- **Organic/Human:** Rounded shapes, warm colors, hand-crafted feel

**The wrong choice is no choice.** Pick a direction and commit to it consistently.

---

## Anti-Patterns Checklist

Before finalizing any UI, verify you haven't fallen into these traps:

- [ ] **Generic font stack** - Using system fonts or Inter/Roboto without justification
- [ ] **Safe gray palette** - Gray-900/800/700 without intentional color choices
- [ ] **No motion strategy** - Either zero motion or scattered random animations
- [ ] **Flat backgrounds** - Solid colors without depth or atmosphere
- [ ] **Purple gradient syndrome** - The default "AI product" look
- [ ] **No aesthetic commitment** - Looks like "every other dashboard"

---

## Integration with Design Phase

When using this reference during the Design phase:

1. **Choose aesthetic direction** before wireframing
2. **Document the choice** in your design document (e.g., "Aesthetic: Technical/Developer with JetBrains Mono, dark theme with amber accents")
3. **Apply consistently** across all components
4. **Validate against anti-patterns** before implementation

---

## References

- Source: Claude Code frontend-design skill principles
- Investigation: `.kb/investigations/simple/2025-11-29-study-claude-code-frontend-design.md`
- External: Anthropic Frontend Aesthetics Cookbook (github.com/anthropics/claude-cookbooks)
