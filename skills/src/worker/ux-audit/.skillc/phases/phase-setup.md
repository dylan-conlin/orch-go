# Phase 1: Setup & Baseline (5-10 min)

**Purpose:** Authenticate, navigate to the target page, and capture baseline screenshots at all viewports. This establishes the visual record for the audit.

---

## Step 1: Navigate to Target Page

```
playwright-cli goto → target URL from SPAWN_CONTEXT
```

**Verify page loaded:**
- Take a snapshot to confirm content is present
- Check for auth redirects (login page instead of target)

---

## Step 2: Handle Authentication

**Auth methods (from spawn parameter):**

### storageState (default)
The session should already be authenticated via `--isolated` flag with storageState. Verify by checking the snapshot for logged-in indicators (username, avatar, logout link).

### dev-login
For local development with bypassed auth:
1. Navigate to dev login endpoint (e.g., `/auth/dev-login`)
2. Verify session established
3. Navigate back to target URL

### cdp-tab
For connecting to an existing authenticated browser tab:
1. Use `playwright-cli tab-list` to list available tabs
2. Select the tab with the target page
3. Verify auth state via snapshot

**If auth fails:**
- Report: `bd comment <beads-id> "BLOCKED: Auth failed - [method] - [error details]"`
- Do NOT proceed — audit results without auth are meaningless for authenticated pages

---

## Step 3: Capture Baseline Screenshots

**Take screenshots at all 5 standard viewports:**

1. **1280px** (Desktop default)
   ```
   playwright-cli resize → width: 1280, height: 800
   playwright-cli screenshot → baseline-1280.png
   ```

2. **1024px** (lg breakpoint)
   ```
   playwright-cli resize → width: 1024, height: 768
   playwright-cli screenshot → baseline-1024.png
   ```

3. **768px** (md breakpoint — minor tweaks only)
   ```
   playwright-cli resize → width: 768, height: 1024
   playwright-cli screenshot → baseline-768.png
   ```

4. **640px** (sm breakpoint — first structural shift)
   ```
   playwright-cli resize → width: 640, height: 1136
   playwright-cli screenshot → baseline-640.png
   ```

5. **375px** (Mobile)
   ```
   playwright-cli resize → width: 375, height: 812
   playwright-cli screenshot → baseline-375.png
   ```

**Screenshot storage:** Save to `.kb/investigations/screenshots/{date}-{page-slug}/`

---

## Step 4: Capture Accessibility Snapshot

**At desktop viewport (1280px):**
```
playwright-cli resize → width: 1280, height: 800
playwright-cli snapshot → (captures a11y tree)
```

**Record from the snapshot:**
- Page title
- Heading hierarchy (h1, h2, h3...)
- Landmark regions (banner, main, nav, contentinfo)
- Interactive element count
- Any unlabeled elements

---

## Step 5: Document Page Metadata

**Record in investigation file:**

```markdown
## Page Metadata

| Field | Value |
|-------|-------|
| URL | {target URL} |
| Page title | {from snapshot} |
| Auth method | {storageState/dev-login/cdp-tab} |
| Auth state | {authenticated/unauthenticated} |
| Data state | {loaded/empty/error} |
| Viewport tested | 1280, 1024, 768, 640, 375 |
| Audit date | {YYYY-MM-DD} |
| Audit mode | {quick/focused/full} |
```

---

## Step 6: Reset Viewport

**Return to desktop viewport for dimension audit:**
```
playwright-cli resize → width: 1280, height: 800
```

---

## Phase 1 Completion

**Report:** `bd comment <beads-id> "Phase: Dimension Audit - Setup complete, baseline captured at 5 viewports"`

**Verify before proceeding:**
- [ ] Page loaded and authenticated
- [ ] 5 baseline screenshots captured
- [ ] Accessibility snapshot captured
- [ ] Page metadata documented
- [ ] Viewport reset to 1280px

**Now proceed to Phase 2: Dimension Audit** (guided by mode — quick/focused/full)
