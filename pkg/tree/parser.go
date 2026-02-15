package tree

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	// Prior-Work table row pattern: | path | relationship | verified | conflicts |
	priorWorkRowRegex = regexp.MustCompile(`^\|\s*(.+?)\s*\|\s*(.+?)\s*\|\s*(.+?)\s*\|`)

	// Synthesized From header pattern
	synthesizedFromRegex = regexp.MustCompile(`(?i)^\*\*Synthesized From:\*\*\s*(.+)$`)

	// Status field pattern
	statusRegex = regexp.MustCompile(`(?i)^\*\*Status:\*\*\s*(.+)$`)

	// Date field pattern
	dateRegex = regexp.MustCompile(`(?i)^\*\*(?:Started|Date|Updated):\*\*\s*(.+)$`)
)

// ParseInvestigations parses all investigation files in a directory
func ParseInvestigations(dir string) ([]*KnowledgeNode, []Relationship, error) {
	var nodes []*KnowledgeNode
	var relationships []Relationship

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		node, rels, err := parseInvestigation(path)
		if err != nil {
			// Log error but continue parsing other files
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
			return nil
		}

		if node != nil {
			nodes = append(nodes, node)
			relationships = append(relationships, rels...)
		}

		return nil
	})

	return nodes, relationships, err
}

// parseInvestigation parses a single investigation file
func parseInvestigation(path string) (*KnowledgeNode, []Relationship, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	node := &KnowledgeNode{
		ID:       path,
		Type:     NodeTypeInvestigation,
		Path:     path,
		Status:   StatusComplete, // Default status
		Metadata: make(map[string]interface{}),
	}

	var relationships []Relationship
	scanner := bufio.NewScanner(file)
	inPriorWork := false
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Extract title from first heading
		if lineNum < 50 && strings.HasPrefix(line, "# ") && node.Title == "" {
			node.Title = strings.TrimPrefix(line, "# ")
			node.Title = strings.TrimPrefix(node.Title, "Investigation: ")
			continue
		}

		// Extract status
		if matches := statusRegex.FindStringSubmatch(line); matches != nil {
			status := strings.ToLower(strings.TrimSpace(matches[1]))
			switch status {
			case "complete", "completed":
				node.Status = StatusComplete
			case "triage", "triage:review":
				node.Status = StatusTriage
			case "in progress", "in_progress", "active":
				node.Status = StatusInProgress
			}
			continue
		}

		// Extract date
		if matches := dateRegex.FindStringSubmatch(line); matches != nil {
			if date, err := time.Parse("2006-01-02", strings.TrimSpace(matches[1])); err == nil {
				node.Date = date
			}
			continue
		}

		// Check for Prior-Work section
		if strings.Contains(line, "**Prior-Work:**") || strings.Contains(line, "## Prior Work") {
			inPriorWork = true
			continue
		}

		// End of Prior-Work section
		if inPriorWork && (strings.HasPrefix(line, "##") || strings.HasPrefix(line, "---")) {
			inPriorWork = false
			continue
		}

		// Parse Prior-Work table rows
		if inPriorWork {
			if matches := priorWorkRowRegex.FindStringSubmatch(line); matches != nil && len(matches) > 2 {
				targetPath := strings.TrimSpace(matches[1])
				relType := strings.TrimSpace(matches[2])
				verified := strings.TrimSpace(matches[3])

				// Skip table header row and separator row
				if targetPath == "Investigation" || strings.Contains(targetPath, "---") || relType == "Relationship" || strings.Contains(relType, "---") {
					continue
				}

				relationships = append(relationships, Relationship{
					From:         path,
					To:           targetPath,
					RelationType: relType,
					Verified:     strings.ToLower(verified) == "yes",
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	return node, relationships, nil
}

// ParseDecisions parses all decision files in a directory
func ParseDecisions(dir string) ([]*KnowledgeNode, []Relationship, error) {
	var nodes []*KnowledgeNode
	var relationships []Relationship

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		node, rels, err := parseDecision(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
			return nil
		}

		if node != nil {
			nodes = append(nodes, node)
			relationships = append(relationships, rels...)
		}

		return nil
	})

	return nodes, relationships, err
}

// parseDecision parses a single decision file
func parseDecision(path string) (*KnowledgeNode, []Relationship, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	node := &KnowledgeNode{
		ID:       path,
		Type:     NodeTypeDecision,
		Path:     path,
		Metadata: make(map[string]interface{}),
	}

	var relationships []Relationship
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Extract title from first heading
		if lineNum < 50 && strings.HasPrefix(line, "# ") && node.Title == "" {
			node.Title = strings.TrimPrefix(line, "# ")
			node.Title = strings.TrimPrefix(node.Title, "Decision: ")
			continue
		}

		// Extract date
		if matches := dateRegex.FindStringSubmatch(line); matches != nil {
			if date, err := time.Parse("2006-01-02", strings.TrimSpace(matches[1])); err == nil {
				node.Date = date
			}
			continue
		}

		// Extract references to investigations in Evidence section
		// Look for markdown links: [text](.kb/investigations/file.md)
		refRegex := regexp.MustCompile(`\[.+?\]\((.kb/investigations/.+?\.md)\)`)
		if matches := refRegex.FindAllStringSubmatch(line, -1); matches != nil {
			for _, match := range matches {
				if len(match) > 1 {
					relationships = append(relationships, Relationship{
						From:         path,
						To:           match[1],
						RelationType: "references",
						Verified:     true,
					})
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	return node, relationships, nil
}

// ParseModels parses all model directories
func ParseModels(dir string) ([]*KnowledgeNode, []Relationship, error) {
	var nodes []*KnowledgeNode
	var relationships []Relationship

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		modelPath := filepath.Join(dir, entry.Name())
		modelFile := filepath.Join(modelPath, entry.Name()+".md")

		// Check if the model file exists
		if _, err := os.Stat(modelFile); os.IsNotExist(err) {
			continue
		}

		node, rels, err := parseModel(modelFile, modelPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", modelFile, err)
			continue
		}

		if node != nil {
			nodes = append(nodes, node)
			relationships = append(relationships, rels...)

			// Parse probes for this model
			probesDir := filepath.Join(modelPath, "probes")
			if probes, probeRels, err := ParseProbes(probesDir, node.ID); err == nil {
				node.Children = append(node.Children, probes...)
				relationships = append(relationships, probeRels...)
			}
		}
	}

	return nodes, relationships, nil
}

// parseModel parses a single model file
func parseModel(path string, modelDir string) (*KnowledgeNode, []Relationship, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	node := &KnowledgeNode{
		ID:       modelDir,
		Type:     NodeTypeModel,
		Path:     path,
		Children: []*KnowledgeNode{},
		Metadata: make(map[string]interface{}),
	}

	var relationships []Relationship
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Extract title from first heading
		if lineNum < 50 && strings.HasPrefix(line, "# ") && node.Title == "" {
			node.Title = strings.TrimPrefix(line, "# ")
			node.Title = strings.TrimPrefix(node.Title, "Model: ")
			continue
		}

		// Extract Synthesized From header
		if matches := synthesizedFromRegex.FindStringSubmatch(line); matches != nil {
			// This indicates the model synthesizes multiple investigations
			// We could parse the investigation references here
			node.Metadata["synthesized_from"] = matches[1]
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	return node, relationships, nil
}

// ParseProbes parses all probe files for a model
func ParseProbes(dir string, modelID string) ([]*KnowledgeNode, []Relationship, error) {
	var nodes []*KnowledgeNode
	var relationships []Relationship

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nodes, relationships, nil
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		node, err := parseProbe(path, modelID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
			return nil
		}

		if node != nil {
			nodes = append(nodes, node)
			relationships = append(relationships, Relationship{
				From:         path,
				To:           modelID,
				RelationType: "probes",
				Verified:     true,
			})
		}

		return nil
	})

	return nodes, relationships, err
}

// parseProbe parses a single probe file
func parseProbe(path string, modelID string) (*KnowledgeNode, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	node := &KnowledgeNode{
		ID:       path,
		Type:     NodeTypeProbe,
		Path:     path,
		Metadata: make(map[string]interface{}),
	}
	node.Metadata["model_id"] = modelID

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Extract title from first heading
		if lineNum < 50 && strings.HasPrefix(line, "# ") && node.Title == "" {
			node.Title = strings.TrimPrefix(line, "# ")
			node.Title = strings.TrimPrefix(node.Title, "Probe: ")
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Use filename as title if no title found
	if node.Title == "" {
		node.Title = filepath.Base(path)
		node.Title = strings.TrimSuffix(node.Title, ".md")
	}

	return node, nil
}

// ParseGuides parses all guide files in a directory
func ParseGuides(dir string) ([]*KnowledgeNode, error) {
	var nodes []*KnowledgeNode

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		node, err := parseGuide(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
			return nil
		}

		if node != nil {
			nodes = append(nodes, node)
		}

		return nil
	})

	return nodes, err
}

// parseGuide parses a single guide file
func parseGuide(path string) (*KnowledgeNode, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	node := &KnowledgeNode{
		ID:       path,
		Type:     NodeTypeGuide,
		Path:     path,
		Metadata: make(map[string]interface{}),
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Extract title from first heading
		if lineNum < 50 && strings.HasPrefix(line, "# ") && node.Title == "" {
			node.Title = strings.TrimPrefix(line, "# ")
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Use filename as title if no title found
	if node.Title == "" {
		node.Title = filepath.Base(path)
		node.Title = strings.TrimSuffix(node.Title, ".md")
	}

	return node, nil
}
