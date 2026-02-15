package tree

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// Icon constants for different node types
const (
	IconInvestigationComplete = "◉"
	IconInvestigationTriage   = "◇"
	IconInvestigationProgress = "●"
	IconDecision              = "★"
	IconModel                 = "◆"
	IconGuide                 = "◈"
	IconIssue                 = "●"
	IconCluster               = "◉"
)

// getNodeIcon returns the icon for a node based on its type and status
func getNodeIcon(node *KnowledgeNode) string {
	switch node.Type {
	case NodeTypeDecision:
		return IconDecision
	case NodeTypeModel:
		return IconModel
	case NodeTypeGuide:
		return IconGuide
	case NodeTypeIssue:
		return IconIssue
	case NodeTypeCluster:
		return IconCluster
	case NodeTypeInvestigation:
		switch node.Status {
		case StatusComplete:
			return IconInvestigationComplete
		case StatusTriage:
			return IconInvestigationTriage
		case StatusInProgress:
			return IconInvestigationProgress
		default:
			return IconInvestigationComplete
		}
	default:
		return "◉"
	}
}

// getNodeTitle returns the display title for a node
func getNodeTitle(node *KnowledgeNode) string {
	if node.Title != "" {
		return node.Title
	}
	// Fallback to filename without extension
	base := filepath.Base(node.Path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// getNodeStatus returns the status label for display
func getNodeStatus(node *KnowledgeNode) string {
	switch node.Status {
	case StatusComplete:
		return "CLOSED"
	case StatusTriage:
		return "triage:review"
	case StatusInProgress:
		return "IN PROGRESS"
	case StatusClosed:
		return "CLOSED"
	case StatusOpen:
		return "OPEN"
	default:
		return ""
	}
}

// RenderTree renders the tree as ASCII text
func RenderTree(root *KnowledgeNode, opts TreeOptions, clusters []*Cluster) (string, error) {
	if opts.Format == "json" {
		return renderJSON(root)
	}

	if opts.Format == "summary" {
		return RenderSummary(root, opts, clusters)
	}

	var sb strings.Builder
	sb.WriteString("orch-go knowledge tree\n")
	sb.WriteString("│\n")

	// Build cluster map for smell lookup
	clusterMap := make(map[string]*Cluster)
	for _, cluster := range clusters {
		clusterMap[cluster.Name] = cluster
	}

	renderNode(&sb, root, "", true, 0, opts.Depth, clusterMap)

	return sb.String(), nil
}

// renderJSON renders the tree as JSON
func renderJSON(root *KnowledgeNode) (string, error) {
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// renderNode renders a single node and its children recursively
func renderNode(sb *strings.Builder, node *KnowledgeNode, prefix string, isLast bool, currentDepth int, maxDepth int, clusterMap map[string]*Cluster) {
	if maxDepth > 0 && currentDepth >= maxDepth {
		return
	}

	// Skip root node
	if node.Type == NodeTypeCluster || currentDepth == 0 {
		// Render cluster header or root
		if currentDepth > 0 {
			connector := "├─"
			if isLast {
				connector = "└─"
			}
			sb.WriteString(prefix)
			sb.WriteString(connector)
			sb.WriteString(getNodeIcon(node))
			sb.WriteString(" ")
			sb.WriteString(getNodeTitle(node))

			// Add cluster stats and smell badges for depth 2 (cluster level)
			if currentDepth == 1 {
				if cluster, ok := clusterMap[node.Title]; ok {
					sb.WriteString(renderClusterStats(cluster))
				}
			}

			sb.WriteString("\n")
		}

		// Render children
		for i, child := range node.Children {
			childIsLast := i == len(node.Children)-1
			childPrefix := prefix
			if currentDepth > 0 {
				if isLast {
					childPrefix += "  "
				} else {
					childPrefix += "│ "
				}
			}
			renderNode(sb, child, childPrefix, childIsLast, currentDepth+1, maxDepth, clusterMap)
		}
		return
	}

	// Render regular node
	connector := "├─"
	if isLast {
		connector = "└─"
	}

	sb.WriteString(prefix)
	sb.WriteString(connector)
	sb.WriteString(getNodeIcon(node))
	sb.WriteString(" ")

	// For investigations/decisions, show path relative to .kb/
	if node.Type == NodeTypeInvestigation || node.Type == NodeTypeDecision || node.Type == NodeTypeModel {
		// Show relative path
		if strings.Contains(node.Path, "/.kb/") {
			parts := strings.Split(node.Path, "/.kb/")
			if len(parts) > 1 {
				sb.WriteString(parts[1])
			} else {
				sb.WriteString(filepath.Base(node.Path))
			}
		} else {
			sb.WriteString(filepath.Base(node.Path))
		}
	} else if node.Type == NodeTypeIssue {
		// For issues, show ID and title
		sb.WriteString(node.ID)
		sb.WriteString("  ")
		sb.WriteString(getNodeTitle(node))

		// Add status if present
		if status := getNodeStatus(node); status != "" {
			// Pad to align status (approximate)
			titleLen := len(getNodeTitle(node))
			padding := 40 - titleLen
			if padding < 1 {
				padding = 1
			}
			sb.WriteString(strings.Repeat(" ", padding))
			sb.WriteString(status)
		}
	} else {
		sb.WriteString(getNodeTitle(node))
	}

	sb.WriteString("\n")

	// Render children
	for i, child := range node.Children {
		childIsLast := i == len(node.Children)-1
		childPrefix := prefix
		if isLast {
			childPrefix += "  "
		} else {
			childPrefix += "│ "
		}
		renderNode(sb, child, childPrefix, childIsLast, currentDepth+1, maxDepth, clusterMap)
	}
}

// renderClusterStats renders statistics and health smells for a cluster
func renderClusterStats(cluster *Cluster) string {
	// Count node types
	var invCount, decCount, modelCount, issueCount int
	for _, node := range cluster.Nodes {
		switch node.Type {
		case NodeTypeInvestigation:
			invCount++
		case NodeTypeDecision:
			decCount++
		case NodeTypeModel:
			modelCount++
		case NodeTypeIssue:
			issueCount++
		}
	}

	var parts []string
	if invCount > 0 {
		parts = append(parts, fmt.Sprintf("%d investigations", invCount))
	}
	if decCount > 0 {
		parts = append(parts, fmt.Sprintf("%d decisions", decCount))
	}
	if modelCount > 0 {
		parts = append(parts, fmt.Sprintf("%d models", modelCount))
	}
	if issueCount > 0 {
		parts = append(parts, fmt.Sprintf("%d issues", issueCount))
	}

	statsStr := ""
	if len(parts) > 0 {
		statsStr = " (" + strings.Join(parts, ", ") + ")"
	}

	// Add health smells
	if len(cluster.Smells) > 0 {
		for _, smell := range cluster.Smells {
			statsStr += "  " + FormatSmellDescription(smell)
		}
	}

	return statsStr
}

// RenderWorkView renders the tree in work view (issues as primary nodes)
func RenderWorkView(issues []*KnowledgeNode, opts TreeOptions) (string, error) {
	if opts.Format == "json" {
		return renderJSON(&KnowledgeNode{
			Type:     NodeTypeCluster,
			Title:    "work-view",
			Children: issues,
		})
	}

	var sb strings.Builder
	sb.WriteString("orch-go work tree\n")
	sb.WriteString("│\n")

	// Group issues by status
	statusGroups := make(map[NodeStatus][]*KnowledgeNode)
	for _, issue := range issues {
		statusGroups[issue.Status] = append(statusGroups[issue.Status], issue)
	}

	// Render each status group
	statusOrder := []NodeStatus{StatusInProgress, StatusTriage, StatusOpen, StatusClosed}
	groupIndex := 0
	totalGroups := 0
	for _, status := range statusOrder {
		if len(statusGroups[status]) > 0 {
			totalGroups++
		}
	}

	for _, status := range statusOrder {
		group := statusGroups[status]
		if len(group) == 0 {
			continue
		}

		groupIndex++
		isLastGroup := groupIndex == totalGroups

		// Render group header
		connector := "├─"
		if isLastGroup {
			connector = "└─"
		}

		statusLabel := getStatusLabel(status)
		sb.WriteString(connector)
		sb.WriteString(statusLabel)
		sb.WriteString(fmt.Sprintf(" (%d)\n", len(group)))

		// Render issues in group
		for i, issue := range group {
			isLast := i == len(group)-1
			prefix := ""
			if !isLastGroup {
				prefix = "│ "
			} else {
				prefix = "  "
			}

			issueConnector := "├─"
			if isLast {
				issueConnector = "└─"
			}

			sb.WriteString(prefix)
			sb.WriteString(issueConnector)
			sb.WriteString(getNodeIcon(issue))
			sb.WriteString(" ")
			sb.WriteString(issue.ID)
			sb.WriteString("  ")
			sb.WriteString(getNodeTitle(issue))
			sb.WriteString("\n")

			// Render knowledge artifacts as children
			for j, child := range issue.Children {
				childIsLast := j == len(issue.Children)-1
				childPrefix := prefix
				if isLast {
					childPrefix += "  "
				} else {
					childPrefix += "│ "
				}

				childConnector := "├─"
				if childIsLast {
					childConnector = "└─"
				}

				sb.WriteString(childPrefix)
				sb.WriteString(childConnector)
				sb.WriteString(" from ")
				sb.WriteString(getNodeIcon(child))
				sb.WriteString(" ")
				sb.WriteString(filepath.Base(child.Path))
				sb.WriteString("\n")
			}
		}
	}

	return sb.String(), nil
}

// getStatusLabel returns the display label for a status
func getStatusLabel(status NodeStatus) string {
	switch status {
	case StatusInProgress:
		return "● IN PROGRESS"
	case StatusTriage:
		return "◇ TRIAGE:REVIEW"
	case StatusOpen:
		return "◐ QUEUED"
	case StatusClosed:
		return "◓ COMPLETED"
	default:
		return "● " + string(status)
	}
}

// RenderSummary renders a concise 3-5 line area briefing for a cluster.
// Format:
//   - Line 1: Cluster name and artifact counts (N investigations, M decisions, K models, J open issues)
//   - Line 2+: Health smells if any
//   - Last line: Most recent artifact date
func RenderSummary(root *KnowledgeNode, opts TreeOptions, clusters []*Cluster) (string, error) {
	// If no cluster filter specified, return error
	if opts.ClusterFilter == "" {
		return "", fmt.Errorf("--format summary requires --cluster <name> to specify which area to summarize")
	}

	// Find the target cluster
	var targetCluster *Cluster
	for _, cluster := range clusters {
		if cluster.Name == opts.ClusterFilter {
			targetCluster = cluster
			break
		}
	}

	if targetCluster == nil {
		return "", fmt.Errorf("cluster %q not found", opts.ClusterFilter)
	}

	var sb strings.Builder

	// Line 1: Cluster name and artifact counts
	sb.WriteString(fmt.Sprintf("## %s\n", targetCluster.Name))

	// Count node types
	var invCount, decCount, modelCount, probeCount, guideCount int
	var mostRecentDate string

	for _, node := range targetCluster.Nodes {
		switch node.Type {
		case NodeTypeInvestigation:
			invCount++
		case NodeTypeDecision:
			decCount++
		case NodeTypeModel:
			modelCount++
		case NodeTypeProbe:
			probeCount++
		case NodeTypeGuide:
			guideCount++
		}

		// Track most recent date
		if !node.Date.IsZero() {
			dateStr := node.Date.Format("2006-01-02")
			if mostRecentDate == "" || dateStr > mostRecentDate {
				mostRecentDate = dateStr
			}
		}
	}

	// Build stats line
	var stats []string
	if invCount > 0 {
		stats = append(stats, fmt.Sprintf("%d investigations", invCount))
	}
	if decCount > 0 {
		stats = append(stats, fmt.Sprintf("%d decisions", decCount))
	}
	if modelCount > 0 {
		stats = append(stats, fmt.Sprintf("%d models", modelCount))
	}
	if probeCount > 0 {
		stats = append(stats, fmt.Sprintf("%d probes", probeCount))
	}
	if guideCount > 0 {
		stats = append(stats, fmt.Sprintf("%d guides", guideCount))
	}

	if len(stats) > 0 {
		sb.WriteString(fmt.Sprintf("**Artifacts:** %s\n", strings.Join(stats, ", ")))
	} else {
		sb.WriteString("**Artifacts:** none\n")
	}

	// Health smells (if any)
	if len(targetCluster.Smells) > 0 {
		sb.WriteString("**Health:**")
		for i, smell := range targetCluster.Smells {
			if i == 0 {
				sb.WriteString(" ")
			} else {
				sb.WriteString(", ")
			}
			sb.WriteString(FormatSmellDescription(smell))
		}
		sb.WriteString("\n")
	}

	// Most recent artifact date
	if mostRecentDate != "" {
		sb.WriteString(fmt.Sprintf("**Last updated:** %s\n", mostRecentDate))
	}

	return sb.String(), nil
}
