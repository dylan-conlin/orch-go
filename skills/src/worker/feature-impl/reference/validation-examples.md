# Validation Examples and Templates

**Purpose:** Reference templates and examples for documenting validation results during the Validation phase.

---

## Validation: tests - Template

```markdown
## Validation Results (Tests)

**Test Command:** npm test
**Run At:** YYYY-MM-DD HH:MM

**Results:**
- Total tests: 245
- Passed: 245
- Failed: 0
- Skipped: 0

**Coverage (if applicable):**
- Overall: 87%
- New files: 92%

**Status:** ✅ All tests passing
```

---

## Validation: smoke-test - Template

### Test Commands by Technology

**JavaScript/TypeScript:**
```bash
npm test
npm run test:integration  # if applicable
```

**Python:**
```bash
pytest
pytest --cov  # with coverage
```

**Ruby:**
```bash
rspec
bundle exec rspec
```

**Rust:**
```bash
cargo test
```

**Go:**
```bash
go test ./...
```

### Loading Feature for Manual Testing

**Web UI:**
```bash
# Start development server
npm run dev
# or
rails server

# Open browser to feature URL
open http://localhost:3000/feature-path
```

**API:**
```bash
# Test with curl
curl -X POST http://localhost:3000/api/feature \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'

# Or use Postman/HTTPie/Insomnia
```

**CLI:**
```bash
# Run actual command
./bin/my-tool feature-command --arg value
```

### Visual/Integration Verification Checklists

**For web UI:**
- [ ] Page loads without errors
- [ ] Styling renders correctly (CSS loaded, layout correct)
- [ ] Data displays as expected (no $0.00, no blank cells, no placeholder text)
- [ ] Interactive elements work (buttons click, forms submit, selectors change)
- [ ] JavaScript executes without errors (check browser console)
- [ ] Network requests succeed (check Network tab)

**For API:**
- [ ] Endpoint responds with correct status code
- [ ] Response body matches expected format
- [ ] Data is correct (not mock/placeholder data)
- [ ] Error cases handled properly

**For CLI:**
- [ ] Command executes successfully
- [ ] Output format is correct
- [ ] Help text displays properly (if applicable)
- [ ] Error messages are clear

### Smoke Test Results Template

```markdown
## Validation Results (Smoke Test)

**Test Suite:** ✅ All tests passing (245/245)

**Smoke Test:**

**URL/Endpoint:** http://localhost:3000/quotes/comparison
**Browser/Tool:** Chrome 120
**Tested At:** 2025-11-18 14:35

**Visual Verification:**
- [x] Page loads without errors
- [x] Styling renders correctly (CSS loaded)
- [x] Data displays as expected (prices, deltas, run selector all working)
- [x] Interactive elements work (run selector changes quotes, buttons respond)
- [x] No console errors (checked DevTools)

**Screenshot:**
[Description of what's shown: "Comparison view showing 2 runs side-by-side with pricing data, green/red deltas, and run selector dropdown working correctly"]

**Issues Found:** None - all working as expected

**Status:** ✅ Smoke test passed
```

### Browser Automation with playwright-cli

**For automated smoke tests (UI features):**

1. Open browser and navigate to feature URL:
   ```bash
   playwright-cli open http://localhost:3000/feature
   ```

2. Get page state (accessibility tree):
   ```bash
   playwright-cli snapshot
   ```

3. Interact with elements using refs from snapshot:
   ```bash
   playwright-cli click e12
   ```

4. Capture screenshot evidence:
   ```bash
   playwright-cli screenshot
   ```

5. Clean up:
   ```bash
   playwright-cli close
   ```

**Key points:**
- Use `playwright-cli snapshot` for semantic element refs (stable)
- Close browser when done to free resources

---

## Validation: multi-phase - Template

```markdown
## Multi-Phase Validation

**Phase ID:** phase-a (from SPAWN_CONTEXT)
**Depends On:** (none) | phase-x (from SPAWN_CONTEXT)

**What Was Implemented:**
[Concise summary of what this phase delivered - 2-3 sentences]

Example: "Phase A implements the foundational time-series data model and comparison logic. Added Run and Quote models with associations, comparison calculation methods, and database migrations. Backend logic complete but no UI yet."

**Evidence It Works:**

**Test Suite:**
- Total tests: 245
- Passed: 245
- Coverage: 87%
- All tests green ✅

**Smoke Test (if applicable):**
- URL: /quotes/comparison
- Browser: Chrome 120
- Verified: Data loads, calculations correct, no errors
- Screenshot: [description or link]
- Status: Working ✅

**Open Questions/Concerns:**
[Any uncertainties, trade-offs made, or issues discovered]

Example:
- None - implementation straightforward
OR
- Performance: Comparison calculation is O(n²) for n quotes. Acceptable for current data size (<100 quotes) but may need optimization if scale increases.
- UI decisions: Deferred styling to Phase B per design doc

**Recommendation:**
[ ] ✅ PROCEED - Ready for next phase
[ ] ⚠️ ADJUST - Issues found, need changes before proceeding
[ ] ❌ BLOCK - Critical issues, cannot proceed

**Reasoning for recommendation:**
[Why you're recommending proceed/adjust/block]

Example: "Recommend PROCEED - all tests passing, smoke test validated, no blocking issues found. Performance concern noted but acceptable for current requirements."
```

---

## Common Issues - Reference

### Tests pass but feature doesn't work

**Symptom:** Automated tests all green, but feature broken in browser/production

**Cause:** Tests verify logic, not integration/UI

**Why this matters:**
- Stylesheets may not load
- JavaScript may have runtime errors
- Data may not render in HTML
- Visual layout may be broken
- User may not be able to use the feature

**Real example:**
- Agent: "240 tests passing" ✅
- Reality: No stylesheets loaded ❌
- Reality: All prices showing $0.00 ❌
- Reality: Interactive elements broken ❌

**Solution:** Always perform smoke test for UI features

### Smoke test reveals issues

**What to do:**
1. Return to Implementation phase
2. Fix issues
3. Re-run validation (tests + smoke test)
4. Don't mark complete with known issues (or document as accepted limitations)

### Multi-phase: orchestrator finds issues

**What to do:**
1. Don't argue or defend implementation
2. Fix issues immediately
3. Re-run validation
4. Update workspace with fixes
5. Request re-validation
