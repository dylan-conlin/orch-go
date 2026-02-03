# Work Graph Attention Surface Mockup

## Visual Specifications

**Theme:** Dark mode, minimal, Linear/Notion-inspired
**Dimensions:** 1400x900 desktop view
**Font:** Inter or SF Pro, clean sans-serif
**Colors:** 
- Background: #0a0a0b (near black)
- Cards: #141415 (dark gray)
- Borders: #2a2a2b (subtle)
- Primary text: #e5e5e5 (off-white)
- Secondary text: #888888 (muted)
- Accent: #3b82f6 (blue)
- Warning: #f59e0b (amber)
- Success: #10b981 (green)

## Layout

### Header Bar (48px)
Left: "Work Graph" title in semibold
Right: Three tab buttons - [Attention] [Issues] [Artifacts] with Attention selected (underlined)

### Main Content Area (split into sections)

#### Section 1: "NEEDS ATTENTION" (prominent, top)
Gray label "NEEDS ATTENTION" with count "(3)" 
Three attention cards, each with:
- Left icon (emoji or icon indicating type)
- Issue ID in monospace (e.g., "orch-go-21146")
- One-line description
- Right side: action hint in muted text

Cards:
1. ⚡ orch-go-21146 "Can't collapse expanded epics" → "Likely done - commits found"
2. ✓ orch-go-21173 "Fix queued filtering timing" → "Ready to complete"  
3. 🔓 orch-go-21180 "Add rate limiting" → "Unblocked"

#### Section 2: "WIP" (compact, middle)
Gray label "WIP" with "(2 running)"
Two rows showing running agents:
- Green dot, workspace name truncated, time running
- "og-arch-pressure-test-work..." 23m
- "og-inv-extended-thinking..." 12m

#### Section 3: "READY TO SPAWN" (lower section)
Gray label "READY TO SPAWN" with count "(5)"
Right side: small badge "daemon: paused" in amber

List of issues with priority badges:
- [P1] orch-go-21179 "Enable extended thinking for orchestrators"
- [P2] orch-go-21158 "Deliverables schema and tracking"
- [P2] orch-go-21157 "Issue side panel (L2)"
- [P2] orch-go-21148 "Core interaction bugs"
- [P3] orch-go-21176 "Remove duplicate scrollbars"

### Right Side Panel (collapsed state)
Thin vertical strip indicating side panel exists but is closed

## Key Visual Elements

- Clear visual hierarchy: Attention items are most prominent
- Subtle card borders, not heavy
- Monospace for issue IDs
- Priority badges are small colored pills (P1=red, P2=yellow, P3=gray)
- Green dots for running agents
- Overall feeling: calm, organized, scannable
