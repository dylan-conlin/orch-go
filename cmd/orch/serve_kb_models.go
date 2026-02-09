package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type KBModelProbesResponse struct {
	Summary    KBModelProbeSummary     `json:"summary"`
	Queue      []KBModelProbeQueueItem `json:"queue"`
	Models     []KBModelProbeModel     `json:"models"`
	ProjectDir string                  `json:"project_dir,omitempty"`
	Error      string                  `json:"error,omitempty"`
}

type KBModelProbeSummary struct {
	ModelsTotal   int `json:"models_total"`
	ProbesTotal   int `json:"probes_total"`
	NeedsReview   int `json:"needs_review"`
	Stale         int `json:"stale"`
	WellValidated int `json:"well_validated"`
}

type KBModelProbeQueueItem struct {
	ProbePath string `json:"probe_path"`
	Model     string `json:"model"`
	Verdict   string `json:"verdict"`
	Date      string `json:"date"`
	Claim     string `json:"claim"`
	Merged    bool   `json:"merged"`
}

type KBModelProbeCounts struct {
	Confirms    int `json:"confirms"`
	Extends     int `json:"extends"`
	Contradicts int `json:"contradicts"`
}

type KBModelProbeModel struct {
	Name          string             `json:"name"`
	Path          string             `json:"path"`
	LastUpdated   string             `json:"last_updated,omitempty"`
	Status        string             `json:"status"`
	ProbeCounts   KBModelProbeCounts `json:"probe_counts"`
	UnmergedCount int                `json:"unmerged_count"`
	LastProbeAt   string             `json:"last_probe_at,omitempty"`
}

type kbModelProbeAggregate struct {
	Model                 KBModelProbeModel
	RecentProbeCount      int
	UnmergedContradiction int
	LastProbeTime         time.Time
}

type kbModelProbeQueueAggregate struct {
	Item KBModelProbeQueueItem
	When time.Time
}

var (
	modelLastUpdatedRe = regexp.MustCompile(`(?m)^\*\*Last Updated:\*\*\s*(\d{4}-\d{2}-\d{2})`)
	probeDateRe        = regexp.MustCompile(`(?m)^\*\*Date:\*\*\s*(\d{4}-\d{2}-\d{2})`)
	probePathRe        = regexp.MustCompile(`probes/([A-Za-z0-9._-]+)\.md`)
	probeSlugRe        = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}-[A-Za-z0-9][A-Za-z0-9-]*\b`)
)

func (s *Server) handleKBModelProbes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectDir := r.URL.Query().Get("project_dir")
	if projectDir == "" {
		projectDir, _ = s.currentProjectDir()
	}

	staleDays := 30
	if raw := strings.TrimSpace(r.URL.Query().Get("stale_days")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			resp := KBModelProbesResponse{
				Summary:    KBModelProbeSummary{},
				Queue:      []KBModelProbeQueueItem{},
				Models:     []KBModelProbeModel{},
				ProjectDir: projectDir,
				Error:      "Invalid stale_days parameter: expected positive integer",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		staleDays = parsed
	}

	models, queue, summary, err := scanKBModelProbes(projectDir, staleDays)
	if err != nil {
		resp := KBModelProbesResponse{
			Summary:    KBModelProbeSummary{},
			Queue:      []KBModelProbeQueueItem{},
			Models:     []KBModelProbeModel{},
			ProjectDir: projectDir,
			Error:      fmt.Sprintf("Failed to scan model probes: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := KBModelProbesResponse{
		Summary:    summary,
		Queue:      queue,
		Models:     models,
		ProjectDir: projectDir,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

func scanKBModelProbes(projectDir string, staleDays int) ([]KBModelProbeModel, []KBModelProbeQueueItem, KBModelProbeSummary, error) {
	modelsDir := filepath.Join(projectDir, ".kb", "models")
	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		return nil, nil, KBModelProbeSummary{}, fmt.Errorf("no .kb/models directory found")
	}

	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil, nil, KBModelProbeSummary{}, err
	}

	aggregates := []kbModelProbeAggregate{}
	queue := []kbModelProbeQueueAggregate{}
	cutoff := time.Now().AddDate(0, 0, -staleDays)
	summary := KBModelProbeSummary{}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") || skipKBModelFile(entry.Name()) {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")
		path := filepath.Join(modelsDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		rel, err := filepath.Rel(projectDir, path)
		if err != nil {
			rel = path
		}

		a := kbModelProbeAggregate{
			Model: KBModelProbeModel{
				Name:          name,
				Path:          filepath.ToSlash(rel),
				ProbeCounts:   KBModelProbeCounts{},
				UnmergedCount: 0,
			},
		}
		a.Model.LastUpdated = parseModelLastUpdated(string(content))
		merged := parseMergedProbeSlugs(string(content))

		probesDir := filepath.Join(modelsDir, name, "probes")
		probeEntries, err := os.ReadDir(probesDir)
		if err == nil {
			for _, pe := range probeEntries {
				if pe.IsDir() || !strings.HasSuffix(pe.Name(), ".md") || pe.Name() == ".gitkeep" {
					continue
				}

				probePath := filepath.Join(probesDir, pe.Name())
				probeData, err := os.ReadFile(probePath)
				if err != nil {
					continue
				}

				probeInfo, err := pe.Info()
				if err != nil {
					continue
				}

				summary.ProbesTotal++
				date, at := parseProbeDate(string(probeData), pe.Name(), probeInfo.ModTime())
				if at.After(cutoff) {
					a.RecentProbeCount++
				}
				if a.LastProbeTime.IsZero() || at.After(a.LastProbeTime) {
					a.LastProbeTime = at
					a.Model.LastProbeAt = date
				}

				verdict := parseProbeVerdictFromContent(string(probeData))
				switch verdict {
				case "confirms":
					a.Model.ProbeCounts.Confirms++
				case "extends":
					a.Model.ProbeCounts.Extends++
				case "contradicts":
					a.Model.ProbeCounts.Contradicts++
				}

				slug := strings.ToLower(strings.TrimSuffix(pe.Name(), ".md"))
				isMerged := merged[slug]
				needsReview := !isMerged && (verdict == "extends" || verdict == "contradicts")
				if needsReview {
					a.Model.UnmergedCount++
					if verdict == "contradicts" {
						a.UnmergedContradiction++
					}

					probeRel, err := filepath.Rel(projectDir, probePath)
					if err != nil {
						probeRel = probePath
					}

					queue = append(queue, kbModelProbeQueueAggregate{
						Item: KBModelProbeQueueItem{
							ProbePath: filepath.ToSlash(probeRel),
							Model:     name,
							Verdict:   verdict,
							Date:      date,
							Claim:     parseProbeClaimFromContent(string(probeData)),
							Merged:    false,
						},
						When: at,
					})
				}
			}
		}

		a.Model.Status = classifyModelProbeStatus(a)
		switch a.Model.Status {
		case "needs_review":
			summary.NeedsReview++
		case "stale":
			summary.Stale++
		case "well_validated":
			summary.WellValidated++
		}

		aggregates = append(aggregates, a)
	}

	summary.ModelsTotal = len(aggregates)

	sort.SliceStable(aggregates, func(i, j int) bool {
		if modelStatusRank(aggregates[i].Model.Status) != modelStatusRank(aggregates[j].Model.Status) {
			return modelStatusRank(aggregates[i].Model.Status) < modelStatusRank(aggregates[j].Model.Status)
		}
		if aggregates[i].LastProbeTime.Equal(aggregates[j].LastProbeTime) {
			return aggregates[i].Model.Name < aggregates[j].Model.Name
		}
		if aggregates[i].LastProbeTime.IsZero() {
			return false
		}
		if aggregates[j].LastProbeTime.IsZero() {
			return true
		}
		return aggregates[i].LastProbeTime.After(aggregates[j].LastProbeTime)
	})

	sort.SliceStable(queue, func(i, j int) bool {
		if queue[i].When.Equal(queue[j].When) {
			return queue[i].Item.ProbePath < queue[j].Item.ProbePath
		}
		return queue[i].When.After(queue[j].When)
	})

	models := make([]KBModelProbeModel, 0, len(aggregates))
	for _, item := range aggregates {
		models = append(models, item.Model)
	}

	items := make([]KBModelProbeQueueItem, 0, len(queue))
	for _, item := range queue {
		items = append(items, item.Item)
	}

	return models, items, summary, nil
}

func skipKBModelFile(name string) bool {
	if name == "README.md" || name == "_TEMPLATE.md" {
		return true
	}
	if strings.HasPrefix(name, "PHASE") && strings.HasSuffix(name, "_REVIEW.md") {
		return true
	}
	return false
}

func parseModelLastUpdated(content string) string {
	m := modelLastUpdatedRe.FindStringSubmatch(content)
	if len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func parseProbeDate(content, filename string, modTime time.Time) (string, time.Time) {
	m := probeDateRe.FindStringSubmatch(content)
	if len(m) == 2 {
		at, err := time.Parse("2006-01-02", m[1])
		if err == nil {
			return m[1], at
		}
	}

	date := extractDateFromFilename(filename)
	if date != "" {
		at, err := time.Parse("2006-01-02", date)
		if err == nil {
			return date, at
		}
	}

	return modTime.Format("2006-01-02"), modTime
}

func parseProbeVerdictFromContent(content string) string {
	impact := markdownSection(content, "Model Impact")
	if impact == "" {
		return ""
	}
	return probeVerdict(impact)
}

func parseProbeClaimFromContent(content string) string {
	question := markdownSection(content, "Question")
	if question == "" {
		return ""
	}

	for _, raw := range strings.Split(question, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		return line
	}

	return ""
}

func markdownSection(content, heading string) string {
	lines := strings.Split(content, "\n")
	collected := []string{}
	inSection := false

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "## ") {
			title := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			if strings.EqualFold(title, heading) {
				inSection = true
				continue
			}
			if inSection {
				break
			}
		}

		if inSection {
			collected = append(collected, raw)
		}
	}

	return strings.TrimSpace(strings.Join(collected, "\n"))
}

func parseMergedProbeSlugs(content string) map[string]bool {
	result := map[string]bool{}
	section := ""

	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		lower := strings.ToLower(line)

		if strings.HasPrefix(lower, "**recent probes:**") || strings.HasPrefix(lower, "## recent probes") {
			section = "recent"
			continue
		}
		if strings.HasPrefix(lower, "**merged probes:**") || strings.HasPrefix(lower, "## merged probes") {
			section = "merged"
			continue
		}
		if section == "" {
			continue
		}

		if line == "---" || strings.HasPrefix(line, "## ") {
			section = ""
			continue
		}
		if strings.HasPrefix(line, "**") && strings.HasSuffix(line, ":**") {
			section = ""
			continue
		}

		for _, match := range probePathRe.FindAllStringSubmatch(line, -1) {
			if len(match) != 2 {
				continue
			}
			result[strings.ToLower(strings.TrimSuffix(match[1], ".md"))] = true
		}

		for _, slug := range probeSlugRe.FindAllString(line, -1) {
			result[strings.ToLower(slug)] = true
		}
	}

	return result
}

func classifyModelProbeStatus(a kbModelProbeAggregate) string {
	if a.Model.UnmergedCount > 0 {
		return "needs_review"
	}
	if a.RecentProbeCount == 0 {
		return "stale"
	}
	if a.RecentProbeCount >= 3 && a.UnmergedContradiction == 0 {
		return "well_validated"
	}
	return "active"
}

func modelStatusRank(status string) int {
	switch status {
	case "needs_review":
		return 0
	case "active":
		return 1
	case "well_validated":
		return 2
	case "stale":
		return 3
	default:
		return 4
	}
}
