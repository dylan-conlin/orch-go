// Dashboard UX Audit Script - Playwright CLI
// Usage: npx playwright test --config=playwright.config.ts dashboard-audit.mjs
// Or: node --experimental-modules dashboard-audit.mjs (with playwright installed)

import { chromium } from 'playwright';
import { writeFileSync, mkdirSync } from 'fs';
import { join } from 'path';

const SCREENSHOT_DIR = join(process.cwd(), '.kb/investigations/screenshots/2026-02-28-dashboard-audit');
const RESULTS_FILE = join(process.cwd(), '.kb/investigations/scripts/dashboard-audit-results.json');
const URL = 'http://localhost:5188/';

const VIEWPORTS = [
  { name: '1280', width: 1280, height: 800 },
  { name: '1024', width: 1024, height: 768 },
  { name: '768', width: 768, height: 1024 },
  { name: '640', width: 640, height: 1136 },
  { name: '375', width: 375, height: 812 },
];

mkdirSync(SCREENSHOT_DIR, { recursive: true });

const results = {
  timestamp: new Date().toISOString(),
  url: URL,
  viewports: {},
  visualConsistency: {},
  accessibility: {},
  dataPresentation: {},
  navigation: {},
  interactive: {},
  consoleErrors: [],
};

async function run() {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ viewport: VIEWPORTS[0] });
  const page = await context.newPage();

  // Collect console errors
  page.on('console', msg => {
    if (msg.type() === 'error') {
      results.consoleErrors.push({
        text: msg.text().substring(0, 200),
        url: msg.location()?.url || '',
      });
    }
  });

  // Navigate and wait for network idle
  await page.goto(URL, { waitUntil: 'networkidle', timeout: 30000 });
  // Extra wait for SSE and deferred fetches
  await page.waitForTimeout(3000);

  // === PHASE 1: Baseline screenshots at all viewports ===
  console.log('Phase 1: Baseline screenshots...');
  for (const vp of VIEWPORTS) {
    await page.setViewportSize({ width: vp.width, height: vp.height });
    await page.waitForTimeout(500);
    await page.screenshot({
      path: join(SCREENSHOT_DIR, `baseline-${vp.name}.png`),
      fullPage: true,
    });
    console.log(`  Screenshot: baseline-${vp.name}.png`);
  }

  // Reset to desktop
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.waitForTimeout(500);

  // === PHASE 2: Visual Consistency ===
  console.log('Phase 2: Visual consistency...');
  results.visualConsistency = await page.evaluate(() => {
    const body = getComputedStyle(document.body);
    const main = document.querySelector('main');
    const mainStyle = main ? getComputedStyle(main) : null;
    const header = document.querySelector('header');
    const headerStyle = header ? getComputedStyle(header) : null;

    // Cards
    const cardSelectors = '[class*="card"], [class*="rounded-lg"]';
    const cards = document.querySelectorAll(cardSelectors);

    // Typography
    const typeSamples = {};
    const elements = {
      h1: document.querySelector('h1'),
      h2: document.querySelector('h2'),
      h3: document.querySelector('h3'),
      bodyText: document.querySelector('main p, main span'),
      link: document.querySelector('nav a'),
      badge: document.querySelector('[class*="badge"]'),
      code: document.querySelector('code'),
    };
    for (const [name, el] of Object.entries(elements)) {
      if (el) {
        const s = getComputedStyle(el);
        typeSamples[name] = {
          fontFamily: s.fontFamily.substring(0, 80),
          fontSize: s.fontSize,
          fontWeight: s.fontWeight,
          color: s.color,
          lineHeight: s.lineHeight,
        };
      }
    }

    // Shadows vs borders
    const shadowElements = [];
    const allEls = document.querySelectorAll('*');
    allEls.forEach(el => {
      const s = getComputedStyle(el);
      if (s.boxShadow && s.boxShadow !== 'none') {
        shadowElements.push({
          tag: el.tagName,
          class: el.className.toString().substring(0, 60),
          boxShadow: s.boxShadow.substring(0, 100),
        });
      }
    });

    return {
      foundation: {
        bodyBackground: body.backgroundColor,
        bodyFont: body.fontFamily.substring(0, 80),
        bodyColor: body.color,
      },
      header: headerStyle ? {
        background: headerStyle.backgroundColor,
        height: headerStyle.height,
        borderBottom: headerStyle.borderBottom,
      } : null,
      main: mainStyle ? {
        background: mainStyle.backgroundColor,
        padding: mainStyle.padding,
      } : null,
      cards: {
        count: cards.length,
        samples: Array.from(cards).slice(0, 8).map(c => {
          const s = getComputedStyle(c);
          return {
            class: c.className.toString().substring(0, 80),
            background: s.backgroundColor,
            border: s.border,
            borderRadius: s.borderRadius,
            boxShadow: s.boxShadow !== 'none' ? s.boxShadow.substring(0, 80) : 'none',
            padding: s.padding,
          };
        }),
      },
      typography: typeSamples,
      shadowCount: shadowElements.length,
      shadows: shadowElements.slice(0, 10),
    };
  });

  // === PHASE 3: Accessibility ===
  console.log('Phase 3: Accessibility...');
  results.accessibility.structure = await page.evaluate(() => {
    const headings = [];
    document.querySelectorAll('h1, h2, h3, h4, h5, h6').forEach(h => {
      headings.push({
        level: parseInt(h.tagName[1]),
        text: h.textContent.trim().substring(0, 60),
      });
    });

    const landmarks = [];
    document.querySelectorAll('main, nav, header, footer, aside, [role="main"], [role="navigation"], [role="banner"], [role="contentinfo"]').forEach(el => {
      landmarks.push({
        tag: el.tagName,
        role: el.getAttribute('role') || el.tagName.toLowerCase(),
        ariaLabel: el.getAttribute('aria-label') || '',
      });
    });

    // Unlabeled buttons
    const unlabeledButtons = [];
    document.querySelectorAll('button, [role="button"]').forEach(el => {
      const text = el.textContent.trim();
      const label = el.getAttribute('aria-label');
      const title = el.getAttribute('title');
      if (!text && !label && !title) {
        unlabeledButtons.push({
          tag: el.tagName,
          class: el.className.toString().substring(0, 60),
          html: el.outerHTML.substring(0, 150),
        });
      }
    });

    // Unlabeled links
    const unlabeledLinks = [];
    document.querySelectorAll('a').forEach(el => {
      const text = el.textContent.trim();
      const label = el.getAttribute('aria-label');
      if (!text && !label) {
        unlabeledLinks.push({
          href: el.getAttribute('href'),
          html: el.outerHTML.substring(0, 150),
        });
      }
    });

    // Images without alt
    const imagesNoAlt = [];
    document.querySelectorAll('img').forEach(el => {
      if (!el.hasAttribute('alt')) {
        imagesNoAlt.push({ src: el.src.substring(0, 80) });
      }
    });

    // aria-expanded usage
    const ariaExpanded = [];
    document.querySelectorAll('[aria-expanded]').forEach(el => {
      ariaExpanded.push({
        tag: el.tagName,
        text: el.textContent.trim().substring(0, 40),
        expanded: el.getAttribute('aria-expanded'),
      });
    });

    return {
      headings,
      landmarks,
      unlabeledButtons,
      unlabeledLinks,
      imagesNoAlt,
      ariaExpanded,
      h1Count: document.querySelectorAll('h1').length,
      hasMain: !!document.querySelector('main'),
      hasNav: !!document.querySelector('nav'),
    };
  });

  // axe-core
  console.log('  Running axe-core...');
  try {
    await page.addScriptTag({ url: 'https://cdnjs.cloudflare.com/ajax/libs/axe-core/4.11.1/axe.min.js' });
    await page.waitForTimeout(2000);
    results.accessibility.axeCore = await page.evaluate(async () => {
      if (typeof axe === 'undefined') return { error: 'axe-core not loaded' };
      const r = await axe.run({ runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa'] } });
      return {
        summary: {
          violationCount: r.violations.length,
          passCount: r.passes.length,
          incompleteCount: r.incomplete.length,
        },
        violations: r.violations.map(v => ({
          id: v.id,
          impact: v.impact,
          description: v.description,
          nodeCount: v.nodes.length,
          tags: v.tags.filter(t => t.startsWith('wcag')),
          nodes: v.nodes.slice(0, 3).map(n => ({
            target: n.target,
            html: n.html.substring(0, 200),
            failureSummary: n.failureSummary,
          })),
        })),
      };
    });
  } catch (e) {
    results.accessibility.axeCore = { error: e.message };
  }

  // === PHASE 4: Responsive checks ===
  console.log('Phase 4: Responsive checks...');
  results.responsive = {};
  for (const vp of VIEWPORTS) {
    await page.setViewportSize({ width: vp.width, height: vp.height });
    await page.waitForTimeout(500);
    results.responsive[vp.name] = await page.evaluate(() => {
      const body = document.body;
      const overflowElements = [];
      document.querySelectorAll('*').forEach(el => {
        if (el.scrollWidth > el.clientWidth + 1 && el.clientWidth > 0) {
          const rect = el.getBoundingClientRect();
          if (rect.width > 0 && rect.height > 0 && el.tagName !== 'HTML' && el.tagName !== 'BODY') {
            overflowElements.push({
              tag: el.tagName,
              class: el.className.toString().substring(0, 60),
              scrollWidth: el.scrollWidth,
              clientWidth: el.clientWidth,
            });
          }
        }
      });

      // Small touch targets
      const smallTargets = [];
      document.querySelectorAll('a, button, input, select, textarea, [role="button"], [tabindex]').forEach(el => {
        const rect = el.getBoundingClientRect();
        if (rect.width > 0 && rect.height > 0 && (rect.width < 44 || rect.height < 44)) {
          smallTargets.push({
            tag: el.tagName,
            text: (el.textContent || el.getAttribute('aria-label') || '').substring(0, 40).trim(),
            width: Math.round(rect.width),
            height: Math.round(rect.height),
          });
        }
      });

      return {
        viewport: { width: window.innerWidth, height: window.innerHeight },
        hasHorizontalOverflow: body.scrollWidth > window.innerWidth,
        bodyScrollWidth: body.scrollWidth,
        overflowElements: overflowElements.slice(0, 10),
        smallTouchTargets: smallTargets.slice(0, 15),
      };
    });
  }

  // Reset to desktop
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.waitForTimeout(500);

  // === PHASE 5: Data Presentation ===
  console.log('Phase 5: Data presentation...');
  results.dataPresentation = await page.evaluate(() => {
    const suspicious = [];
    const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    let node;
    while (node = walker.nextNode()) {
      const text = node.textContent.trim();
      if (['null', 'undefined', 'NaN', 'None', '[object Object]'].includes(text)) {
        const parent = node.parentElement;
        suspicious.push({
          value: text,
          tag: parent.tagName,
          class: parent.className.toString().substring(0, 60),
          context: parent.parentElement ? parent.parentElement.textContent.trim().substring(0, 80) : '',
        });
      }
    }

    // Raw database values (snake_case, ALL_CAPS)
    const rawValues = [];
    const walker2 = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    while (node = walker2.nextNode()) {
      const text = node.textContent.trim();
      if (!text || text.length > 50 || text.length < 3) continue;
      if (/^[a-z][a-z_]+[a-z]$/.test(text) && text.includes('_')) {
        rawValues.push(text);
      }
    }

    // Tables
    const tables = [];
    document.querySelectorAll('table').forEach(t => {
      const headers = Array.from(t.querySelectorAll('th')).map(th => th.textContent.trim());
      tables.push({
        headers: headers.slice(0, 10),
        rows: t.querySelectorAll('tbody tr').length,
        hasHeader: t.querySelector('thead') !== null,
      });
    });

    return {
      suspiciousValues: suspicious.slice(0, 10),
      rawDatabaseValues: [...new Set(rawValues)].slice(0, 20),
      tables,
    };
  });

  // === PHASE 6: Navigation ===
  console.log('Phase 6: Navigation...');
  results.navigation = await page.evaluate(() => {
    const navLinks = [];
    document.querySelectorAll('nav a, [role="navigation"] a').forEach(a => {
      const href = a.getAttribute('href');
      const computed = getComputedStyle(a);
      navLinks.push({
        text: (a.textContent.trim() || a.getAttribute('aria-label') || '').substring(0, 40),
        href,
        ariaCurrent: a.getAttribute('aria-current'),
        hasActiveClass: /active|current|selected/.test(a.className),
        color: computed.color,
        fontWeight: computed.fontWeight,
      });
    });

    return {
      pageTitle: document.title,
      h1s: Array.from(document.querySelectorAll('h1')).map(h => h.textContent.trim()),
      navLinks,
      currentPath: window.location.pathname,
    };
  });

  // === PHASE 7: Interactive States ===
  console.log('Phase 7: Interactive states...');
  results.interactive = await page.evaluate(() => {
    const buttons = [];
    document.querySelectorAll('button, [role="button"]').forEach(el => {
      const computed = getComputedStyle(el);
      buttons.push({
        text: (el.textContent.trim() || el.getAttribute('aria-label') || '').substring(0, 40),
        disabled: el.disabled || el.getAttribute('aria-disabled') === 'true',
        cursor: computed.cursor,
        opacity: computed.opacity,
        hasHoverClass: /hover/.test(el.className),
      });
    });

    const formInputs = [];
    document.querySelectorAll('input:not([type="hidden"]), textarea, select').forEach(el => {
      formInputs.push({
        type: el.type || el.tagName.toLowerCase(),
        name: el.name || el.id || '',
        placeholder: el.placeholder || '',
        disabled: el.disabled,
        hasLabel: !!document.querySelector(`label[for="${el.id}"]`) || !!el.closest('label'),
      });
    });

    return {
      buttonCount: buttons.length,
      buttons: buttons.slice(0, 20),
      formInputs: formInputs.slice(0, 10),
    };
  });

  // === PHASE 8: Contrast check ===
  console.log('Phase 8: Contrast check...');
  results.contrast = await page.evaluate(() => {
    function luminance(r, g, b) {
      const a = [r, g, b].map(v => {
        v /= 255;
        return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
      });
      return a[0] * 0.2126 + a[1] * 0.7152 + a[2] * 0.0722;
    }
    function parseColor(str) {
      const m = str.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
      return m ? [parseInt(m[1]), parseInt(m[2]), parseInt(m[3])] : [0, 0, 0];
    }
    function getContrastRatio(fg, bg) {
      const fgRGB = parseColor(fg);
      const bgRGB = parseColor(bg);
      const l1 = luminance(...fgRGB) + 0.05;
      const l2 = luminance(...bgRGB) + 0.05;
      return Math.round((Math.max(l1, l2) / Math.min(l1, l2)) * 100) / 100;
    }

    const samples = [];
    // Body text
    const bodyP = document.querySelector('main p, main span');
    if (bodyP) {
      const s = getComputedStyle(bodyP);
      const bg = s.backgroundColor === 'rgba(0, 0, 0, 0)' ? getComputedStyle(document.body).backgroundColor : s.backgroundColor;
      samples.push({
        element: 'body text',
        color: s.color,
        background: bg,
        fontSize: s.fontSize,
        ratio: getContrastRatio(s.color, bg),
      });
    }

    // Muted text
    const muted = document.querySelector('.text-muted-foreground');
    if (muted) {
      const s = getComputedStyle(muted);
      const bg = s.backgroundColor === 'rgba(0, 0, 0, 0)' ? getComputedStyle(document.body).backgroundColor : s.backgroundColor;
      samples.push({
        element: 'muted text',
        color: s.color,
        background: bg,
        fontSize: s.fontSize,
        ratio: getContrastRatio(s.color, bg),
      });
    }

    // Nav links
    const navLink = document.querySelector('nav a');
    if (navLink) {
      const s = getComputedStyle(navLink);
      const bg = s.backgroundColor === 'rgba(0, 0, 0, 0)' ? getComputedStyle(document.body).backgroundColor : s.backgroundColor;
      samples.push({
        element: 'nav link',
        color: s.color,
        background: bg,
        fontSize: s.fontSize,
        ratio: getContrastRatio(s.color, bg),
      });
    }

    // Badge text
    const badge = document.querySelector('[class*="badge"]');
    if (badge) {
      const s = getComputedStyle(badge);
      samples.push({
        element: 'badge',
        color: s.color,
        background: s.backgroundColor,
        fontSize: s.fontSize,
        ratio: getContrastRatio(s.color, s.backgroundColor),
      });
    }

    return { contrastSamples: samples };
  });

  // === PHASE 9: Additional screenshots for specific states ===
  console.log('Phase 9: Interactive screenshots...');

  // Header area close-up
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.screenshot({
    path: join(SCREENSHOT_DIR, 'header-desktop.png'),
    clip: { x: 0, y: 0, width: 1280, height: 60 },
  });

  // Try to click a collapsible section if visible
  const collapseBtn = await page.$('[aria-expanded]');
  if (collapseBtn) {
    await collapseBtn.click();
    await page.waitForTimeout(300);
    await page.screenshot({
      path: join(SCREENSHOT_DIR, 'interactive-collapse.png'),
      fullPage: false,
    });
  }

  // Save results
  writeFileSync(RESULTS_FILE, JSON.stringify(results, null, 2));
  console.log(`\nResults saved to ${RESULTS_FILE}`);
  console.log(`Screenshots saved to ${SCREENSHOT_DIR}`);

  await browser.close();
}

run().catch(err => {
  console.error('Audit script failed:', err);
  process.exit(1);
});
