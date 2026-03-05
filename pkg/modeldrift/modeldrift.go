// Package modeldrift detects stale knowledge-base model artifacts and creates
// maintenance issues. It aggregates staleness events emitted at spawn time,
// deduplicates against existing open issues, and creates grouped issues by domain.
package modeldrift

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

const (
	Label               = "model-maintenance"
	IssueType           = "task"
	SpawnThreshold      = 3
	BackpressureLimit   = 3
	CircuitBreakerLimit = 5
)

// Result contains the result of running model drift reflection.
type Result struct {
	Created []string
	Skipped int
	Message string
	Error   error
}

// IssueCreateArgs defines parameters for creating model drift issues.
type IssueCreateArgs struct {
	Title       string
	Description string
	Priority    int
	Labels      []string
	ProjectDir  string // Project directory to create the issue in (uses cwd if empty)
}

// Metadata captures model attributes used for grouping and priority.
type Metadata struct {
	ModelPath   string
	Domain      string
	DomainKey   string
	LastUpdated string
	ProjectDir  string
}

// Issue is a minimal representation of a beads issue for drift deduplication.
type Issue struct {
	Title       string
	Description string
}

// Store provides I/O for model drift analysis.
type Store interface {
	ReadStalenessEvents(path string) ([]spawn.StalenessEvent, error)
	LoadMetadata(modelPath string) (Metadata, error)
	CountCommits(projectDir, lastUpdated string, files []string) (int, error)
	CreateIssue(args IssueCreateArgs) (string, error)
}

// IssueQuerier lists issues for drift deduplication.
type IssueQuerier interface {
	ListIssuesWithLabel(label string) ([]Issue, error)
}

type aggregate struct {
	ModelPath  string
	Count      int
	ChangedSet map[string]struct{}
	DeletedSet map[string]struct{}
}

type candidate struct {
	ModelPath   string
	Count       int
	Changed     []string
	Deleted     []string
	CommitCount int
	Priority    int
	Domain      string
	DomainKey   string
	ProjectDir  string
}

type group struct {
	Domain     string
	DomainKey  string
	Priority   int
	ProjectDir string
	Models     []candidate
}

// Analyze scans staleness events and creates model-maintenance issues.
func Analyze(store Store, querier IssueQuerier) (*Result, error) {
	events, err := store.ReadStalenessEvents(spawn.DefaultStalenessEventPath())
	if err != nil {
		result := &Result{
			Error:   err,
			Message: fmt.Sprintf("Model drift reflection failed: %v", err),
		}
		return result, err
	}

	if len(events) == 0 {
		return &Result{Message: "Model drift reflection: no staleness events"}, nil
	}

	aggregates := map[string]*aggregate{}
	for _, event := range events {
		modelPath := strings.TrimSpace(event.Model)
		if modelPath == "" {
			continue
		}
		agg := aggregates[modelPath]
		if agg == nil {
			agg = &aggregate{
				ModelPath:  modelPath,
				ChangedSet: make(map[string]struct{}),
				DeletedSet: make(map[string]struct{}),
			}
			aggregates[modelPath] = agg
		}
		agg.Count++
		for _, file := range event.ChangedFiles {
			if file == "" {
				continue
			}
			agg.ChangedSet[file] = struct{}{}
		}
		for _, file := range event.DeletedFiles {
			if file == "" {
				continue
			}
			agg.DeletedSet[file] = struct{}{}
		}
	}

	var candidates []candidate
	for _, agg := range aggregates {
		if agg.Count < SpawnThreshold {
			continue
		}
		changed := mapKeysSorted(agg.ChangedSet)
		deleted := mapKeysSorted(agg.DeletedSet)
		candidates = append(candidates, candidate{
			ModelPath: agg.ModelPath,
			Count:     agg.Count,
			Changed:   changed,
			Deleted:   deleted,
		})
	}

	if len(candidates) == 0 {
		return &Result{Message: "Model drift reflection: no models exceeded threshold"}, nil
	}

	openIssues, err := querier.ListIssuesWithLabel(Label)
	if err != nil {
		result := &Result{
			Error:   err,
			Message: fmt.Sprintf("Model drift reflection failed: %v", err),
		}
		return result, err
	}

	openCount := len(openIssues)
	if openCount >= CircuitBreakerLimit {
		return &Result{
			Message: fmt.Sprintf("Model drift reflection halted: circuit breaker (%d open)", openCount),
		}, nil
	}
	if openCount >= BackpressureLimit {
		return &Result{
			Message: fmt.Sprintf("Model drift reflection paused: %d open (max %d)", openCount, BackpressureLimit),
		}, nil
	}

	existingModels := existingModelKeys(openIssues)
	existingDomains := existingDomainKeys(openIssues)

	groups := map[string]*group{}
	skipped := 0
	for _, c := range candidates {
		if isModelTracked(c.ModelPath, existingModels) {
			skipped++
			continue
		}

		metadata, metaErr := store.LoadMetadata(c.ModelPath)
		if metaErr != nil {
			metadata = fallbackMetadata(c.ModelPath)
		}
		if metadata.Domain == "" {
			metadata.Domain = DeriveDomainFromPath(c.ModelPath)
		}
		if metadata.DomainKey == "" {
			metadata.DomainKey = NormalizeDomainKey(metadata.Domain)
		}

		commitCount, err := store.CountCommits(metadata.ProjectDir, metadata.LastUpdated, c.Changed)
		if err != nil {
			commitCount = len(c.Changed)
		}
		c.CommitCount = commitCount
		c.Domain = metadata.Domain
		c.DomainKey = metadata.DomainKey
		c.ProjectDir = metadata.ProjectDir
		c.Priority = priority(c)

		g := groups[c.DomainKey]
		if g == nil {
			g = &group{
				Domain:     metadata.Domain,
				DomainKey:  metadata.DomainKey,
				Priority:   c.Priority,
				ProjectDir: metadata.ProjectDir,
			}
			groups[c.DomainKey] = g
		}
		if c.Priority < g.Priority {
			g.Priority = c.Priority
		}
		g.Models = append(g.Models, c)
	}

	if len(groups) == 0 {
		return &Result{Message: "Model drift reflection: no new models to file"}, nil
	}

	var ordered []*group
	for _, g := range groups {
		ordered = append(ordered, g)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Priority != ordered[j].Priority {
			return ordered[i].Priority < ordered[j].Priority
		}
		return ordered[i].Domain < ordered[j].Domain
	})

	allowed := BackpressureLimit - openCount
	created := []string{}
	for _, g := range ordered {
		if len(created) >= allowed {
			break
		}
		if existingDomains[g.DomainKey] {
			skipped++
			continue
		}
		issueID, err := store.CreateIssue(IssueCreateArgs{
			Title:       fmt.Sprintf("Model drift: %s", g.Domain),
			Description: formatIssueDescription(g),
			Priority:    g.Priority,
			Labels:      []string{Label},
			ProjectDir:  g.ProjectDir,
		})
		if err != nil {
			result := &Result{
				Created: created,
				Skipped: skipped,
				Error:   err,
				Message: fmt.Sprintf("Model drift issue creation failed: %v", err),
			}
			return result, err
		}
		created = append(created, issueID)
	}

	message := fmt.Sprintf("Model drift reflection: created %d issue(s), skipped %d (open: %d)", len(created), skipped, openCount)
	return &Result{
		Created: created,
		Skipped: skipped,
		Message: message,
	}, nil
}

// --- DefaultStore implementation ---

// DefaultStore is the production Store backed by filesystem and git.
type DefaultStore struct{}

// NewDefaultStore returns a new DefaultStore.
func NewDefaultStore() *DefaultStore {
	return &DefaultStore{}
}

func (s *DefaultStore) ReadStalenessEvents(path string) ([]spawn.StalenessEvent, error) {
	return readStalenessEvents(path)
}

func (s *DefaultStore) LoadMetadata(modelPath string) (Metadata, error) {
	return LoadMetadata(modelPath)
}

func (s *DefaultStore) CountCommits(projectDir, lastUpdated string, files []string) (int, error) {
	return DefaultCommitCounter(projectDir, lastUpdated, files)
}

func (s *DefaultStore) CreateIssue(args IssueCreateArgs) (string, error) {
	return DefaultCreateIssue(args)
}

// --- I/O functions ---

func readStalenessEvents(path string) ([]spawn.StalenessEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []spawn.StalenessEvent{}, nil
		}
		return nil, fmt.Errorf("failed to open staleness events file: %w", err)
	}
	defer file.Close()

	var events []spawn.StalenessEvent
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event spawn.StalenessEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read staleness events file: %w", err)
	}

	return events, nil
}

// LoadMetadata loads domain and last updated metadata from a model file.
func LoadMetadata(modelPath string) (Metadata, error) {
	normalized, err := normalizeModelPath(modelPath)
	if err != nil {
		return fallbackMetadata(modelPath), err
	}

	content, err := os.ReadFile(normalized)
	if err != nil {
		return fallbackMetadata(modelPath), err
	}

	text := string(content)
	lastUpdated := extractModelField(text, "Last Updated")
	domain := normalizeDomainName(extractModelField(text, "Domain"))
	if domain == "" {
		domain = DeriveDomainFromPath(normalized)
	}

	projectDir := ProjectDirFromModelPath(normalized)
	metadata := Metadata{
		ModelPath:   normalized,
		Domain:      domain,
		DomainKey:   NormalizeDomainKey(domain),
		LastUpdated: lastUpdated,
		ProjectDir:  projectDir,
	}
	return metadata, nil
}

// DefaultCommitCounter counts commits since last update across given files.
func DefaultCommitCounter(projectDir, lastUpdated string, files []string) (int, error) {
	if lastUpdated == "" || len(files) == 0 {
		return 0, nil
	}
	if projectDir == "" {
		projectDir = "."
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	args := []string{"log", "--since=" + lastUpdated, "--oneline", "--"}
	args = append(args, files...)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count, nil
}

// DefaultCreateIssue creates a model maintenance issue in beads.
func DefaultCreateIssue(args IssueCreateArgs) (string, error) {
	if strings.TrimSpace(args.Title) == "" {
		return "", fmt.Errorf("model drift issue title is required")
	}
	labels := args.Labels
	if len(labels) == 0 {
		labels = []string{Label}
	}
	p := args.Priority
	if p == 0 {
		p = 3
	}
	issue, err := createBeadsIssue(args.Title, args.Description, IssueType, p, labels, args.ProjectDir)
	if err != nil {
		return "", err
	}
	return issue.ID, nil
}

func createBeadsIssue(title, description, issueType string, p int, labels []string, dir string) (*beads.Issue, error) {
	socketPath, err := beads.FindSocketPath(dir)
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if dir != "" {
			opts = append(opts, beads.WithCwd(dir))
		}
		client := beads.NewClient(socketPath, opts...)
		if err := client.Connect(); err == nil {
			defer client.Close()
			issue, err := client.Create(&beads.CreateArgs{
				Title:       title,
				Description: description,
				IssueType:   issueType,
				Priority:    p,
				Labels:      labels,
			})
			if err == nil {
				return issue, nil
			}
		}
	}

	return beads.FallbackCreate(title, description, issueType, p, labels, dir)
}

// --- Helper functions ---

func priority(c candidate) int {
	if len(c.Deleted) > 0 || c.CommitCount >= 5 {
		return 2
	}
	return 3
}

func formatIssueDescription(g *group) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Model drift detected for domain \"%s\".\n\n", g.Domain))
	builder.WriteString(fmt.Sprintf("Models (>= %d stale spawns):\n", SpawnThreshold))
	for _, model := range g.Models {
		builder.WriteString(fmt.Sprintf("- %s (stale spawns: %d, commits since update: %d", model.ModelPath, model.Count, model.CommitCount))
		if len(model.Deleted) > 0 {
			builder.WriteString(fmt.Sprintf(", deleted files: %d", len(model.Deleted)))
		}
		if len(model.Changed) > 0 {
			builder.WriteString(fmt.Sprintf(", changed files: %d", len(model.Changed)))
		}
		builder.WriteString(")\n")
		if len(model.Deleted) > 0 {
			builder.WriteString(fmt.Sprintf("  Deleted: %s\n", formatLimitedList(model.Deleted, 5)))
		}
	}
	builder.WriteString("\nSource: ")
	builder.WriteString(spawn.DefaultStalenessEventPath())
	builder.WriteString("\n")
	return builder.String()
}

func mapKeysSorted(input map[string]struct{}) []string {
	if len(input) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func formatLimitedList(values []string, limit int) string {
	if len(values) == 0 {
		return ""
	}
	if limit <= 0 || len(values) <= limit {
		return strings.Join(values, ", ")
	}
	return fmt.Sprintf("%s, +%d more", strings.Join(values[:limit], ", "), len(values)-limit)
}

func existingModelKeys(issues []Issue) map[string]struct{} {
	keys := make(map[string]struct{})
	for _, issue := range issues {
		for _, modelPath := range extractModelPathsFromText(issue.Title + "\n" + issue.Description) {
			for _, key := range modelKeyVariants(modelPath) {
				keys[key] = struct{}{}
			}
		}
	}
	return keys
}

func existingDomainKeys(issues []Issue) map[string]bool {
	keys := make(map[string]bool)
	for _, issue := range issues {
		domainKey := domainKeyFromIssue(issue)
		if domainKey != "" {
			keys[domainKey] = true
		}
	}
	return keys
}

func isModelTracked(modelPath string, existing map[string]struct{}) bool {
	for _, key := range modelKeyVariants(modelPath) {
		if _, ok := existing[key]; ok {
			return true
		}
	}
	return false
}

func modelKeyVariants(modelPath string) []string {
	cleaned := filepath.Clean(modelPath)
	base := filepath.Base(cleaned)
	base = strings.TrimSpace(base)
	baseNoExt := strings.TrimSuffix(base, filepath.Ext(base))

	rel := cleaned
	marker := string(filepath.Separator) + ".kb" + string(filepath.Separator)
	if idx := strings.Index(cleaned, marker); idx != -1 {
		rel = cleaned[idx+1:]
	}

	variants := []string{
		strings.ToLower(cleaned),
		strings.ToLower(rel),
		strings.ToLower(base),
		strings.ToLower(baseNoExt),
	}
	return variants
}

func extractModelPathsFromText(text string) []string {
	pattern := regexp.MustCompile(`(?i)(?:/[^\s]+/\.kb/models/[^\s]+\.md|\.kb/models/[^\s]+\.md)`)
	matches := pattern.FindAllString(text, -1)
	seen := make(map[string]struct{})
	var results []string
	for _, match := range matches {
		cleaned := strings.TrimSpace(match)
		if cleaned == "" {
			continue
		}
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		results = append(results, cleaned)
	}
	return results
}

func domainKeyFromIssue(issue Issue) string {
	trimmed := strings.TrimSpace(issue.Title)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "model drift:") {
		domain := strings.TrimSpace(trimmed[len("model drift:"):])
		return NormalizeDomainKey(domain)
	}
	return ""
}

func normalizeModelPath(modelPath string) (string, error) {
	if modelPath == "" {
		return "", fmt.Errorf("empty model path")
	}
	if filepath.IsAbs(modelPath) {
		return filepath.Clean(modelPath), nil
	}
	return filepath.Abs(modelPath)
}

func normalizeDomainName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "/") {
		parts := strings.Split(value, "/")
		value = strings.TrimSpace(parts[0])
	}
	if strings.Contains(value, ",") {
		parts := strings.Split(value, ",")
		value = strings.TrimSpace(parts[0])
	}
	return value
}

// NormalizeDomainKey normalizes a domain name to a key suitable for deduplication.
func NormalizeDomainKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "/", "-")
	re := regexp.MustCompile(`[^a-z0-9\-]`)
	value = re.ReplaceAllString(value, "")
	value = strings.Trim(value, "-")
	return value
}

// DeriveDomainFromPath extracts a domain name from a model file path.
func DeriveDomainFromPath(modelPath string) string {
	base := filepath.Base(modelPath)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	return base
}

func fallbackMetadata(modelPath string) Metadata {
	domain := DeriveDomainFromPath(modelPath)
	return Metadata{
		ModelPath:  modelPath,
		Domain:     domain,
		DomainKey:  NormalizeDomainKey(domain),
		ProjectDir: ProjectDirFromModelPath(modelPath),
	}
}

// ProjectDirFromModelPath extracts the project directory from a model file path.
func ProjectDirFromModelPath(modelPath string) string {
	cleaned := filepath.Clean(modelPath)
	marker := string(filepath.Separator) + ".kb" + string(filepath.Separator)
	if idx := strings.Index(cleaned, marker); idx != -1 {
		return cleaned[:idx]
	}
	return filepath.Dir(cleaned)
}

func extractModelField(content, field string) string {
	lines := strings.Split(content, "\n")
	plainPrefix := field + ":"
	boldPrefix := "**" + field + ":**"
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, boldPrefix):
			return strings.TrimSpace(strings.TrimPrefix(trimmed, boldPrefix))
		case strings.HasPrefix(trimmed, plainPrefix):
			return strings.TrimSpace(strings.TrimPrefix(trimmed, plainPrefix))
		}
	}
	return ""
}
