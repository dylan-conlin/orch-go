import { ATTENTION_BADGE_CONFIG } from '$lib/stores/attention'
import type { AttentionBadgeType, TreeNode } from '$lib/stores/work-graph'
import { computeAgentHealth, type WIPItem } from '$lib/stores/wip'

export interface GroupHeader {
  _groupHeader: true
  key: string
  label: string
  count: number
  unlabeled: boolean
}

export type RunningAgentDetails = {
  phase?: string
  runtime?: string
  model?: string
  skill?: string
}

export function getAttentionBadge(
  badge: AttentionBadgeType | 'unverified' | 'needs_fix' | undefined,
) {
  if (!badge) return null
  return ATTENTION_BADGE_CONFIG[badge] || null
}

export function isGroupHeader(item: unknown): item is GroupHeader {
  return !!item && typeof item === 'object' && '_groupHeader' in item
}

export function isWIPItem(item: TreeNode | WIPItem | GroupHeader): item is WIPItem {
  return 'type' in item && (item.type === 'running' || item.type === 'queued')
}

export function getItemId(item: TreeNode | WIPItem | GroupHeader): string {
  if (isGroupHeader(item)) return item.key
  if (isWIPItem(item)) {
    return item.type === 'running' ? item.agent.id : item.issue.id
  }
  return item.id
}

export function getItemKey(item: TreeNode | WIPItem | GroupHeader): string {
  if (isGroupHeader(item)) return `group-${item.key}`
  if (isWIPItem(item)) {
    return item.type === 'running'
      ? `wip-running-${item.agent.id}`
      : `wip-queued-${item.issue.id}`
  }
  return `tree-${item.id}`
}

export function getRowTestId(item: TreeNode | WIPItem | GroupHeader): string {
  if (isGroupHeader(item)) return `group-header-${item.key}`
  if (isWIPItem(item)) {
    return item.type === 'running'
      ? `wip-row-${item.agent.beads_id || item.agent.id}`
      : `wip-row-${item.issue.id}`
  }
  return `issue-row-${item.id}`
}

export function flattenTree(nodes: TreeNode[], result: TreeNode[] = []): TreeNode[] {
  for (const node of nodes) {
    result.push(node)
    if (node.expanded && node.children.length > 0) {
      flattenTree(node.children, result)
    }
  }
  return result
}

export function flattenVisibleTree(
  nodes: TreeNode[],
  pinnedIds: Set<string>,
): TreeNode[] {
  return flattenTree(nodes).filter((node) => !pinnedIds.has(node.id))
}

export function findNodeById(nodes: TreeNode[], nodeId: string): TreeNode | null {
  for (const node of nodes) {
    if (node.id === nodeId) {
      return node
    }
    if (node.children.length > 0) {
      const childMatch = findNodeById(node.children, nodeId)
      if (childMatch) {
        return childMatch
      }
    }
  }
  return null
}

export function getStatusIcon(status: string): string {
  switch (status.toLowerCase()) {
    case 'in_progress':
      return '▶'
    case 'blocked':
      return '🚫'
    case 'open':
      return '○'
    case 'closed':
      return '✓'
    case 'complete':
      return '✓'
    default:
      return '•'
  }
}

export function getStatusColor(status: string): string {
  switch (status.toLowerCase()) {
    case 'in_progress':
      return 'text-blue-500'
    case 'blocked':
      return 'text-red-500'
    case 'open':
      return 'text-muted-foreground'
    case 'closed':
      return 'text-green-500'
    case 'complete':
      return 'text-green-500'
    default:
      return 'text-muted-foreground'
  }
}

export function formatStatusLabel(status: string): string {
  return status.replace(/_/g, ' ')
}

export function isDoneStatus(status: string): boolean {
  const normalized = status.toLowerCase()
  return normalized === 'closed' || normalized === 'complete' || normalized === 'accepted'
}

function normalizeDescription(description?: string): string {
  return description?.replace(/\s+/g, ' ').trim() || ''
}

function truncateText(text: string, limit: number): string {
  if (text.length <= limit) {
    return text
  }
  return `${text.slice(0, Math.max(0, limit - 3)).trimEnd()}...`
}

export function getIssueSummary(node: TreeNode): string {
  const description = normalizeDescription(node.description)
  if (description) {
    const sentenceMatch = description.match(/^(.+?[.!?])(\s|$)/)
    const sentence = sentenceMatch ? sentenceMatch[1] : description
    return truncateText(sentence, 160)
  }

  if (node.status.toLowerCase() === 'in_progress') {
    return 'Work is currently active and waiting for the next phase transition.'
  }
  if (node.blocked_by.length > 0) {
    return 'This issue is waiting on upstream work before execution can continue.'
  }
  if (node.blocks.length > 0) {
    return 'This issue is a dependency for downstream work and unblocks others when complete.'
  }
  return 'No summary provided yet. Expand related issues below for context.'
}

function collectDescendants(node: TreeNode): TreeNode[] {
  const descendants: TreeNode[] = []
  const stack = [...node.children]
  while (stack.length > 0) {
    const next = stack.shift()
    if (!next) continue
    descendants.push(next)
    if (next.children.length > 0) {
      stack.push(...next.children)
    }
  }
  return descendants
}

function countVisibleDescendants(node: TreeNode): number {
  if (!node.expanded || node.children.length === 0) {
    return 0
  }

  let visible = 0
  for (const child of node.children) {
    visible += 1
    visible += countVisibleDescendants(child)
  }
  return visible
}

export function getProgressSnapshot(
  node: TreeNode,
): { done: number; total: number; percent: number; visible: number } | null {
  const descendants = collectDescendants(node)
  if (descendants.length === 0) {
    return null
  }

  const done = descendants.filter((child) => isDoneStatus(child.status)).length
  const total = descendants.length
  const percent = Math.round((done / total) * 100)
  const visible = countVisibleDescendants(node)

  return { done, total, percent, visible }
}

export function getRelatedIssueLabel(
  issueId: string,
  treeNodeIndex: Map<string, TreeNode>,
): string {
  const relatedNode = treeNodeIndex.get(issueId)
  if (!relatedNode) {
    return issueId
  }
  return `${issueId} (${formatStatusLabel(relatedNode.status)})`
}

export function formatRelatedIssueList(
  issueIds: string[],
  treeNodeIndex: Map<string, TreeNode>,
  maxItems = 4,
): string {
  if (issueIds.length === 0) {
    return 'none'
  }

  const shown = issueIds
    .slice(0, maxItems)
    .map((issueId) => getRelatedIssueLabel(issueId, treeNodeIndex))
  const remaining = issueIds.length - shown.length
  if (remaining > 0) {
    return `${shown.join(', ')} +${remaining} more`
  }
  return shown.join(', ')
}

export function getDependencyExplanation(
  node: TreeNode,
  treeNodeIndex: Map<string, TreeNode>,
): { headline: string; detail: string; tone: string } {
  if (node.blocked_by.length > 0) {
    const blockers = formatRelatedIssueList(node.blocked_by, treeNodeIndex)
    return {
      headline: `Blocked by ${node.blocked_by.length} upstream issue${node.blocked_by.length === 1 ? '' : 's'}.`,
      detail: `This work cannot complete until these blockers resolve: ${blockers}`,
      tone: 'text-amber-500',
    }
  }

  if (node.blocks.length > 0) {
    const downstream = formatRelatedIssueList(node.blocks, treeNodeIndex)
    return {
      headline: `Ready and unblocks ${node.blocks.length} downstream issue${node.blocks.length === 1 ? '' : 's'}.`,
      detail: `Completing this item unlocks: ${downstream}`,
      tone: 'text-blue-500',
    }
  }

  return {
    headline: 'No direct blocking dependencies.',
    detail: 'This issue can be worked independently within the current graph scope.',
    tone: 'text-emerald-500',
  }
}

export function shortenModel(model?: string): string {
  if (!model) return 'model unknown'
  return model.split('/').pop()?.split('-').slice(0, 3).join('-') || model
}

function formatIssueAge(createdAt?: string): string | null {
  if (!createdAt) return null
  const created = new Date(createdAt)
  if (Number.isNaN(created.getTime())) return null

  const ageMs = Date.now() - created.getTime()
  if (ageMs < 60_000) return 'opened just now'

  const minutes = Math.floor(ageMs / 60_000)
  if (minutes < 60) return `opened ${minutes}m ago`

  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `opened ${hours}h ago`

  const days = Math.floor(hours / 24)
  return `opened ${days}d ago`
}

export function getInProgressSubline(
  node: TreeNode,
  runningAgentDetailsByIssueId: Map<string, RunningAgentDetails>,
): { text: string; tone: string } | null {
  if (node.status.toLowerCase() !== 'in_progress') {
    return null
  }

  const activeAgent = runningAgentDetailsByIssueId.get(node.id)
  const phase = node.active_agent?.phase || activeAgent?.phase
  const runtime = node.active_agent?.runtime || activeAgent?.runtime
  const model = node.active_agent?.model || activeAgent?.model

  if (phase || runtime || model) {
    const phaseText = phase || 'active'
    const runtimeText = runtime || 'runtime unknown'
    const modelText = shortenModel(model)
    return {
      text: `${phaseText} · ${runtimeText} · ${modelText}`,
      tone: 'text-blue-500/90',
    }
  }

  if (node.attentionBadge === 'verify' || node.attentionBadge === 'likely_done') {
    return {
      text: 'Awaiting review (Phase: Complete)',
      tone: 'text-emerald-500/90',
    }
  }

  const ageText = formatIssueAge(node.created_at)
  return {
    text: ageText ? `No active agent linked · ${ageText}` : 'No active agent linked',
    tone: 'text-amber-500/90',
  }
}

export function getPriorityVariant(
  priority: number,
): 'destructive' | 'secondary' | 'outline' {
  if (priority === 0) return 'destructive'
  if (priority === 1) return 'secondary'
  return 'outline'
}

export function getTypeBadge(type: string): string {
  switch (type.toLowerCase()) {
    case 'epic':
      return 'bg-purple-500/10 text-purple-500'
    case 'feature':
      return 'bg-blue-500/10 text-blue-500'
    case 'bug':
      return 'bg-red-500/10 text-red-500'
    case 'task':
      return 'bg-green-500/10 text-green-500'
    case 'question':
      return 'bg-yellow-500/10 text-yellow-500'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

export function getAgentStatusIcon(agent: any): { icon: string; color: string } {
  const health = computeAgentHealth(agent)

  if (health.status === 'critical') {
    return { icon: '🚨', color: 'text-red-500' }
  }
  if (health.status === 'warning') {
    return { icon: '⚠️', color: 'text-yellow-500' }
  }

  if (agent.is_processing) {
    return { icon: '◉', color: 'text-blue-500 animate-pulse' }
  }
  if (agent.status === 'idle') {
    return { icon: '⏸', color: 'text-muted-foreground' }
  }
  return { icon: '▶', color: 'text-blue-500' }
}
