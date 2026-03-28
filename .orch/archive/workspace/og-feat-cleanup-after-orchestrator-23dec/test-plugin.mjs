#!/usr/bin/env node
/**
 * Test: Verify session-context plugin loads and has correct structure
 */

import { readFileSync } from 'fs';
import { join } from 'path';
import { homedir } from 'os';

const pluginPath = join(homedir(), '.config/opencode/plugin/session-context.js');

console.log('[test-plugin] Loading plugin from:', pluginPath);

try {
  // Read the compiled plugin
  const pluginContent = readFileSync(pluginPath, 'utf-8');
  
  // Verify key functionality is present
  const checks = [
    {
      name: 'ORCH_WORKER check exists',
      pattern: /process\.env\.ORCH_WORKER/,
      found: pluginContent.match(/process\.env\.ORCH_WORKER/) !== null
    },
    {
      name: 'Config hook exists',
      pattern: /config:\s*async/,
      found: pluginContent.match(/config:\s*async/) !== null
    },
    {
      name: 'Orchestrator skill path is correct (meta)',
      pattern: /skills.*meta.*orchestrator/,
      found: pluginContent.match(/skills.*meta.*orchestrator/) !== null
    },
    {
      name: 'findOrchDirectory function exists',
      pattern: /findOrchDirectory/,
      found: pluginContent.match(/findOrchDirectory/) !== null
    },
    {
      name: 'Plugin export exists',
      pattern: /export[\s\{]*SessionContextPlugin/,
      found: pluginContent.match(/export[\s\{]*SessionContextPlugin/) !== null
    }
  ];
  
  console.log('\n[test-plugin] Running checks...\n');
  
  let allPassed = true;
  checks.forEach(check => {
    const status = check.found ? '✓' : '✗';
    const result = check.found ? 'PASS' : 'FAIL';
    console.log(`  ${status} ${check.name}: ${result}`);
    if (!check.found) allPassed = false;
  });
  
  console.log('\n[test-plugin] Result:', allPassed ? 'ALL CHECKS PASSED ✓' : 'SOME CHECKS FAILED ✗');
  process.exit(allPassed ? 0 : 1);
  
} catch (error) {
  console.error('[test-plugin] Error:', error.message);
  process.exit(1);
}
