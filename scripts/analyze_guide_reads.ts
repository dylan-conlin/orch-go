#!/usr/bin/env bun
import fs from 'fs';
import path from 'path';
import { glob } from 'glob';

const DATA_DIR = path.join(process.env.HOME, '.local/share/opencode');
const ORCH_DIR = path.join(process.env.HOME, '.orch');

async function getAgentMap() {
  const eventsFile = path.join(ORCH_DIR, 'events.jsonl');
  const agents = new Map();
  
  if (fs.existsSync(eventsFile)) {
    const lines = fs.readFileSync(eventsFile, 'utf-8').split('\n');
    for (const line of lines) {
      if (!line) continue;
      try {
        const event = JSON.parse(line);
        if (event.type === 'session.spawned') {
          agents.set(event.session_id, {
            id: event.data.beads_id || event.data.workspace,
            task: event.data.task,
            workspace: event.data.workspace
          });
        }
      } catch (e) {}
    }
  }
  return agents;
}

async function searchGuideReads() {
  const agentMap = await getAgentMap();
  const partFiles = await glob(`${DATA_DIR}/storage/part/**/prt_*.json`);
  const reads = [];

  for (const file of partFiles) {
    try {
      const content = JSON.parse(fs.readFileSync(file, 'utf-8'));
      const filePath = content.state?.input?.filePath || content.state?.input?.path;
      
      if (filePath && filePath.includes('.kb/guides/')) {
        const agent = agentMap.get(content.sessionID);
        reads.push({
          timestamp: new Date(content.state.time?.start || 0).toISOString(),
          agentID: agent ? agent.id : 'unknown',
          task: agent ? agent.task : 'unknown',
          guide: path.basename(filePath),
          sessionID: content.sessionID
        });
      }
    } catch (e) {}
  }

  // Sort by timestamp descending
  reads.sort((a, b) => b.timestamp.localeCompare(a.timestamp));

  console.log('| Timestamp | Agent | Guide | Task |');
  console.log('|-----------|-------|-------|------|');
  for (const r of reads.slice(0, 30)) {
    const task = r.task.length > 50 ? r.task.substring(0, 47) + '...' : r.task;
    console.log(`| ${r.timestamp} | ${r.agentID} | ${r.guide} | ${task} |`);
  }
}

searchGuideReads();
