// axe-core audit using local axe-core module
const { chromium } = require('playwright');
const { readFileSync, writeFileSync } = require('fs');
const { join } = require('path');

const ROOT = join(__dirname, '../../..');
const RESULTS_FILE = join(ROOT, '.kb/investigations/scripts/axe-results.json');
const URL = 'http://localhost:5188/';

// Read axe-core from local node_modules
const AXE_SOURCE = readFileSync(join(ROOT, 'web/node_modules/axe-core/axe.min.js'), 'utf8');

async function run() {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1280, height: 800 } });

  await page.goto(URL, { waitUntil: 'domcontentloaded', timeout: 30000 });
  await page.waitForTimeout(5000);

  // Inject axe-core from local file
  await page.evaluate(AXE_SOURCE);
  await page.waitForTimeout(1000);

  const results = await page.evaluate(async () => {
    if (typeof axe === 'undefined') return { error: 'axe not loaded' };
    const r = await axe.run({
      runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa'] }
    });
    return {
      summary: {
        violationCount: r.violations.length,
        passCount: r.passes.length,
        incompleteCount: r.incomplete.length,
        inapplicableCount: r.inapplicable.length,
      },
      violations: r.violations.map(v => ({
        id: v.id,
        impact: v.impact,
        description: v.description,
        helpUrl: v.helpUrl,
        nodeCount: v.nodes.length,
        tags: v.tags.filter(t => t.startsWith('wcag')),
        nodes: v.nodes.slice(0, 5).map(n => ({
          target: n.target,
          html: n.html.substring(0, 300),
          failureSummary: n.failureSummary,
        })),
      })),
      incomplete: r.incomplete.map(i => ({
        id: i.id,
        impact: i.impact,
        description: i.description,
        nodeCount: i.nodes.length,
      })),
    };
  });

  writeFileSync(RESULTS_FILE, JSON.stringify(results, null, 2));
  console.log('axe-core results saved to', RESULTS_FILE);
  console.log('Violations:', results.summary?.violationCount || 'error');
  console.log('Passes:', results.summary?.passCount || 'error');

  if (results.violations) {
    for (const v of results.violations) {
      console.log(`  [${v.impact}] ${v.id}: ${v.description} (${v.nodeCount} nodes)`);
    }
  }

  await browser.close();
}

run().catch(err => {
  console.error('Failed:', err);
  process.exit(1);
});
