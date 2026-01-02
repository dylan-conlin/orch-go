# Session Synthesis

**Agent:** og-inv-opencode-has-nice-26dec
**Issue:** (ad-hoc spawn - no beads tracking)
**Duration:** ~30 minutes
**Outcome:** success

---

## TLDR

Investigated OpenCode's theme system to inform theme selection feature for orch-go dashboard. OpenCode has 31 built-in themes with a sophisticated JSON format supporting dark/light variants, color references, and custom themes. Recommend porting 5-10 popular themes to the dashboard using CSS custom properties.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-26-inv-opencode-theme-selection-system.md` - Full investigation with findings and recommendations

### Files Modified
- None (investigation only)

### Commits
- (pending commit)

---

## Evidence (What Was Observed)

- OpenCode theme system lives in `packages/opencode/src/cli/cmd/tui/context/theme.tsx` (1110 lines)
- 31 built-in themes available (dracula, nord, tokyonight, catppuccin, one-dark, gruvbox, etc.)
- Theme JSON format uses `defs` (color aliases) + `theme` (semantic colors) structure
- Each color can be: hex, ANSI code, reference, or dark/light variant object
- Theme selection UI uses live preview on hover, reverts on cancel
- Custom themes loadable from ~/.config/opencode/themes/ or .opencode/themes/
- orch-go dashboard already has CSS variable architecture compatible with multi-theme approach

### Tests Run
```bash
# Cloned OpenCode source
git clone --depth 1 https://github.com/sst/opencode.git /tmp/opencode-source

# Verified theme files exist
ls /tmp/opencode-source/packages/opencode/src/cli/cmd/tui/context/theme/
# Result: 31 JSON files (aura.json through zenburn.json)

# Analyzed key files
cat theme.tsx  # 1110 lines of theme resolution logic
cat opencode.json  # 245 lines, the default theme
cat dracula.json  # 219 lines, popular theme example
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-inv-opencode-theme-selection-system.md` - Complete investigation

### Decisions Made
- **JSON theme format:** OpenCode's approach with defs + theme works well, can adapt for CSS variables
- **Theme subset:** Dashboard needs ~15 colors (not OpenCode's 40+) since no syntax highlighting needed
- **Initial themes:** Recommend porting dracula, nord, tokyonight, catppuccin, one-dark, gruvbox, github, opencode

### Constraints Discovered
- OpenCode themes use RGBA values for TUI; web needs HSL for Tailwind compatibility
- Dashboard already uses shadcn-style CSS variables; theme porting should match this pattern
- No live preview needed for web (simpler than TUI implementation)

### Externalized via `kn`
- None needed - findings captured in investigation file

---

## Next (What Should Happen)

**Recommendation:** close

### Design Recommendation for Theme Selection

**Implementation approach (for future feature-impl spawn):**

1. **Theme data structure** (`lib/themes/`):
   ```typescript
   // themes/dracula.ts
   export const dracula = {
     name: 'dracula',
     dark: {
       '--background': '231 15% 18%',
       '--foreground': '60 30% 96%',
       '--card': '232 14% 13%',
       '--primary': '265 89% 78%',
       // ... ~15 total colors
     },
     light: {
       // light mode variant
     }
   }
   ```

2. **Theme store** (`lib/stores/theme.ts`):
   - Extend current store: `type Theme = 'system' | 'light' | 'dark' | ThemeName`
   - Add `setTheme(themeName)` method
   - Apply theme by setting CSS variables on `<html>`

3. **Theme selector UI**:
   - Dropdown in header (replace current sun/moon toggle)
   - Options: System, Light, Dark, then divider, then named themes
   - Each theme shown with small color preview dots

4. **CSS changes** (`app.css`):
   - Current `:root` becomes `[data-theme="light"]`
   - Current `.dark` becomes `[data-theme="dark"]`
   - Named themes applied via JS setting variables

**Priority themes to port:**
1. dracula (most popular)
2. nord (very popular, clean look)
3. tokyonight (popular neovim theme)
4. catppuccin (popular, warm)
5. one-dark (Atom classic)
6. gruvbox (retro warm)
7. github (good light mode)
8. opencode (orange accent, matches OpenCode)

**Effort estimate:** ~2-3 hours for a feature-impl agent

### If Close
- [x] All deliverables complete (investigation file)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for orchestrator review

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should dashboard themes sync with OpenCode TUI theme selection? (Could read from same config)
- Should we support custom themes via JSON files like OpenCode does?
- Could we auto-generate CSS from OpenCode theme JSONs?

**Areas worth exploring further:**
- The `system` theme in OpenCode adapts to terminal colors - could web do similar with OS preference beyond just dark/light?
- Theme preview/live editing in a future admin panel

**What remains unclear:**
- Whether users care about theme variety in a monitoring dashboard (low priority feature)

---

## Session Metadata

**Skill:** investigation
**Model:** opus (via orchestrator spawn)
**Workspace:** `.orch/workspace/og-inv-opencode-has-nice-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-inv-opencode-theme-selection-system.md`
**Beads:** (ad-hoc, no issue)
