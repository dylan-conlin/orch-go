// Package main provides the kb ask command for inline mini-investigations.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/spf13/cobra"
)

var (
	kbAskSave   bool   // Save result as investigation artifact
	kbAskModel  string // Model to use for synthesis
	kbAskLimit  int    // Maximum artifacts to read
	kbAskGlobal bool   // Search across all projects

	// kb extract flags
	kbExtractTo           string // Target project name
	kbExtractUpdateSource bool   // Add extracted-to reference in original

	// kb archive-old flags
	kbArchiveOlderThan string // Duration threshold (e.g., "60d")
	kbArchiveDryRun    bool   // Show what would be archived without moving
)

var kbCmd = &cobra.Command{
	Use:   "kb",
	Short: "Knowledge base commands for inline queries and artifact management",
	Long: `Knowledge base commands for quick inline queries and artifact management.

The kb subcommand provides fast access to knowledge synthesis without
the overhead of spawning full investigation agents.

Examples:
  orch kb ask "how should we sort the swarm map?"
  orch kb ask "what's our auth pattern?" --save
  orch kb ask "rate limiting approach" --global
  orch kb extract .kb/decisions/2025-01-01-auth-pattern.md --to skillc`,
}

var kbExtractCmd = &cobra.Command{
	Use:   "extract <artifact-path>",
	Short: "Extract artifact to another project with lineage tracking",
	Long: `Extract a knowledge artifact to another project with lineage metadata.

This command copies an artifact (investigation, decision, etc.) to another
project's .kb/ directory while preserving lineage information. The copy
includes an 'extracted-from' header, and optionally updates the source
with an 'extracted-to' reference.

The artifact is COPIED, not moved - the original remains for historical reference.

Examples:
  # Extract a decision to skillc project
  orch kb extract .kb/decisions/2025-01-01-skill-template.md --to skillc

  # Extract and update source with back-reference
  orch kb extract .kb/investigations/2025-01-01-auth-flow.md --to auth-service --update-source

  # Use absolute path
  orch kb extract /path/to/project/.kb/decisions/foo.md --to other-project`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if kbExtractTo == "" {
			return fmt.Errorf("--to flag is required: specify target project name")
		}
		return runKBExtract(args[0], kbExtractTo, kbExtractUpdateSource)
	},
}

var kbAskCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Get inline answers from knowledge base (~5-10s)",
	Long: `Get quick inline answers by synthesizing knowledge base context.

This command:
1. Runs kb context with your question keywords
2. Reads top matching artifacts (investigations, decisions, kn entries)
3. Sends to LLM with synthesis prompt
4. Returns answer inline (~5-10 seconds)

Use this for quick questions. For questions worth preserving as artifacts,
use --save or spawn a full investigation.

Examples:
  orch kb ask "how should we handle rate limiting?"
  orch kb ask "what's our auth pattern?"
  orch kb ask "spawning best practices" --save  # Save as investigation
  orch kb ask "config patterns" --global         # Search all projects
  orch kb ask "db migrations" --limit 5          # Limit artifacts read`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		question := args[0]
		return runKBAsk(question)
	},
}

var kbArchiveOldCmd = &cobra.Command{
	Use:   "archive-old",
	Short: "Archive old investigations to reduce clutter",
	Long: `Archive investigations older than a specified threshold.

Moves old investigations from .kb/investigations/ to .kb/investigations/archive/.
Files remain discoverable via kb search and kb context (which search recursively).

Examples:
  orch kb archive-old --older-than 60d
  orch kb archive-old --older-than 90d --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if kbArchiveOlderThan == "" {
			return fmt.Errorf("--older-than flag is required (e.g., --older-than 60d)")
		}
		return runKBArchiveOld(kbArchiveOlderThan, kbArchiveDryRun)
	},
}

func init() {
	kbAskCmd.Flags().BoolVar(&kbAskSave, "save", false, "Save result as investigation artifact")
	kbAskCmd.Flags().StringVar(&kbAskModel, "model", "", "Model to use (default: sonnet for speed)")
	kbAskCmd.Flags().IntVar(&kbAskLimit, "limit", 3, "Maximum artifacts to read for context")
	kbAskCmd.Flags().BoolVarP(&kbAskGlobal, "global", "g", false, "Search across all known projects")

	kbExtractCmd.Flags().StringVar(&kbExtractTo, "to", "", "Target project name (required)")
	kbExtractCmd.Flags().BoolVar(&kbExtractUpdateSource, "update-source", false, "Add extracted-to reference in original file")

	kbArchiveOldCmd.Flags().StringVar(&kbArchiveOlderThan, "older-than", "", "Archive investigations older than this duration (e.g., 60d, 90d)")
	kbArchiveOldCmd.Flags().BoolVar(&kbArchiveDryRun, "dry-run", false, "Show what would be archived without moving files")

	kbCmd.AddCommand(kbAskCmd)
	kbCmd.AddCommand(kbExtractCmd)
	kbCmd.AddCommand(kbArchiveOldCmd)
	rootCmd.AddCommand(kbCmd)
}

// KBContextResult represents the JSON output from kb context.
type KBContextResult struct {
	Constraints    []KNEntry    `json:"constraints"`
	Decisions      []KNEntry    `json:"decisions"`
	Attempts       []KNEntry    `json:"attempts"`
	Questions      []KNEntry    `json:"questions"`
	Investigations []KBArtifact `json:"investigations"`
	KBDecisions    []KBArtifact `json:"kb_decisions"`
}

// KNEntry represents a knowledge entry from kn.
type KNEntry struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	Reason    string `json:"reason"`
	Result    string `json:"result"`
	Tags      string `json:"tags"`
	Scope     string `json:"scope"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// KBArtifact represents a kb artifact (investigation or decision).
type KBArtifact struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Title   string   `json:"title"`
	Type    string   `json:"type"`
	Matches []string `json:"matches"`
}

func runKBAsk(question string) error {
	startTime := time.Now()

	// Step 1: Extract keywords from question for better kb context search
	keywords := extractKeywords(question)
	searchQuery := keywords
	if searchQuery == "" {
		// Fallback to original question if no keywords extracted
		searchQuery = question
	}

	// Step 1: Run kb context with keywords (progressive fallback)
	fmt.Printf("🔍 Searching knowledge base for: %s\n", question)
	contextResult, err := runKBContextWithFallback(question, keywords)
	if err != nil {
		return fmt.Errorf("failed to get kb context: %w", err)
	}

	// Step 2: Build context from kn entries and artifacts
	contextBuilder := &strings.Builder{}
	writeContextForSynthesis(contextBuilder, contextResult, kbAskLimit)

	contextText := contextBuilder.String()
	if contextText == "" {
		fmt.Println("❌ No matching context found in knowledge base.")
		fmt.Println("   Try a broader question or spawn an investigation:")
		fmt.Printf("   orch spawn investigation \"%s\"\n", question)
		return nil
	}

	// Debug: show context stats
	fmt.Printf("   Found: %d constraints, %d decisions, %d investigations\n",
		len(contextResult.Constraints), len(contextResult.Decisions), len(contextResult.Investigations))

	// Step 3: Send to LLM for synthesis
	fmt.Printf("🤖 Synthesizing answer...\n")
	answer, err := synthesizeAnswer(question, contextText)
	if err != nil {
		return fmt.Errorf("failed to synthesize answer: %w", err)
	}

	elapsed := time.Since(startTime)

	// Step 4: Display result
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println(answer)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("\n⏱️  Completed in %.1fs\n", elapsed.Seconds())

	// Step 5: Optionally save as investigation
	if kbAskSave {
		path, err := saveAsInvestigation(question, answer, contextResult)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save investigation: %v\n", err)
		} else {
			fmt.Printf("📝 Saved to: %s\n", path)
		}
	}

	return nil
}

// stopwords are common words to filter out from questions for better keyword extraction.
var stopwords = map[string]bool{
	// Question words
	"what": true, "how": true, "why": true, "when": true, "where": true, "which": true, "who": true,
	// Common verbs
	"is": true, "are": true, "was": true, "were": true, "be": true, "been": true, "being": true,
	"do": true, "does": true, "did": true, "doing": true, "done": true,
	"have": true, "has": true, "had": true, "having": true,
	"can": true, "could": true, "should": true, "would": true, "will": true, "shall": true,
	"may": true, "might": true, "must": true,
	// Articles and prepositions
	"a": true, "an": true, "the": true,
	"in": true, "on": true, "at": true, "to": true, "for": true, "of": true, "with": true, "by": true,
	"from": true, "about": true, "into": true, "through": true, "during": true, "before": true, "after": true,
	// Pronouns
	"i": true, "me": true, "my": true, "we": true, "our": true, "you": true, "your": true,
	"it": true, "its": true, "they": true, "them": true, "their": true, "this": true, "that": true,
	// Common adverbs
	"not": true, "no": true, "yes": true, "also": true, "just": true, "only": true, "very": true,
	// Conjunctions
	"and": true, "or": true, "but": true, "if": true, "then": true, "because": true, "so": true,
}

// extractKeywords extracts domain-relevant keywords from a natural language question.
// This improves kb context search which works better with keywords than full questions.
func extractKeywords(question string) string {
	// Lowercase and split
	words := strings.Fields(strings.ToLower(question))

	// Filter stopwords and short words
	var keywords []string
	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,?!'\"():;")
		// Skip stopwords and very short words
		if len(word) < 2 || stopwords[word] {
			continue
		}
		keywords = append(keywords, word)
	}

	return strings.Join(keywords, " ")
}

// runKBContextWithFallback tries multiple query strategies to find relevant context.
// It starts with extracted keywords, then falls back to individual terms if needed.
func runKBContextWithFallback(question, keywords string) (*KBContextResult, error) {
	// Strategy 1: Try keywords first (best for multi-word searches)
	if keywords != "" {
		result, err := runKBContext(keywords)
		if err != nil {
			return nil, err
		}
		if hasResults(result) {
			return result, nil
		}
	}

	// Strategy 2: Try individual keywords if combined search failed
	if keywords != "" && strings.Contains(keywords, " ") {
		words := strings.Fields(keywords)
		// Try each keyword individually, longest first (more specific)
		for i := range words {
			for j := i + 1; j < len(words); j++ {
				if len(words[i]) < len(words[j]) {
					words[i], words[j] = words[j], words[i]
				}
			}
		}
		for _, word := range words {
			if len(word) < 3 {
				continue
			}
			result, err := runKBContext(word)
			if err != nil {
				continue
			}
			if hasResults(result) {
				return result, nil
			}
		}
	}

	// Strategy 3: Fall back to original question (might work for exact matches)
	result, err := runKBContext(question)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// hasResults checks if a KBContextResult has any meaningful content.
func hasResults(result *KBContextResult) bool {
	return len(result.Constraints) > 0 ||
		len(result.Decisions) > 0 ||
		len(result.Investigations) > 0 ||
		len(result.Attempts) > 0 ||
		len(result.Questions) > 0 ||
		len(result.KBDecisions) > 0
}

// runKBContext executes kb context and returns parsed results.
func runKBContext(query string) (*KBContextResult, error) {
	args := []string{"context", query, "--format", "json"}
	if kbAskGlobal {
		args = append(args, "--global")
	}

	cmd := exec.Command("kb", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("kb context failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	var result KBContextResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse kb context output: %w", err)
	}

	return &result, nil
}

// writeContextForSynthesis writes formatted context for LLM synthesis.
func writeContextForSynthesis(w *strings.Builder, result *KBContextResult, limit int) {
	// Write constraints first (most actionable)
	if len(result.Constraints) > 0 {
		w.WriteString("## CONSTRAINTS (must respect)\n\n")
		for i, c := range result.Constraints {
			if i >= limit {
				break
			}
			w.WriteString(fmt.Sprintf("- %s\n", c.Content))
			if c.Reason != "" {
				w.WriteString(fmt.Sprintf("  Reason: %s\n", c.Reason))
			}
			w.WriteString("\n")
		}
	}

	// Write decisions
	if len(result.Decisions) > 0 {
		w.WriteString("## DECISIONS\n\n")
		for i, d := range result.Decisions {
			if i >= limit {
				break
			}
			w.WriteString(fmt.Sprintf("- %s\n", d.Content))
			if d.Reason != "" {
				w.WriteString(fmt.Sprintf("  Reason: %s\n", d.Reason))
			}
			w.WriteString("\n")
		}
	}

	// Read and include top investigation artifacts
	artifactsRead := 0
	if len(result.Investigations) > 0 {
		w.WriteString("## RELEVANT INVESTIGATIONS\n\n")
		for _, inv := range result.Investigations {
			if artifactsRead >= limit {
				break
			}
			content, err := readArtifactSummary(inv.Path)
			if err != nil {
				continue
			}
			w.WriteString(fmt.Sprintf("### %s\n", inv.Title))
			w.WriteString(fmt.Sprintf("Path: %s\n\n", inv.Path))
			w.WriteString(content)
			w.WriteString("\n\n")
			artifactsRead++
		}
	}

	// Include any attempts (things that didn't work)
	if len(result.Attempts) > 0 {
		w.WriteString("## FAILED ATTEMPTS (don't retry)\n\n")
		for i, a := range result.Attempts {
			if i >= limit {
				break
			}
			w.WriteString(fmt.Sprintf("- Tried: %s\n", a.Content))
			if a.Result != "" {
				w.WriteString(fmt.Sprintf("  Result: %s\n", a.Result))
			}
			w.WriteString("\n")
		}
	}
}

// readArtifactSummary reads key sections from an investigation file.
func readArtifactSummary(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	var summary strings.Builder

	// Strategy: First check for **TLDR:** at start (common format)
	// Then look for ## TLDR, ## Conclusion, ## Summary sections
	tldrFound := false
	for _, line := range lines {
		lineLower := strings.ToLower(line)
		if strings.HasPrefix(lineLower, "**tldr:**") {
			// Found inline TLDR - extract it
			summary.WriteString("**TLDR:** ")
			summary.WriteString(strings.TrimPrefix(line, "**TLDR:** "))
			summary.WriteString("\n")
			tldrFound = true
			break
		}
	}

	// If TLDR found inline, we're done
	if tldrFound {
		return summary.String(), nil
	}

	// Try section-based extraction
	inSection := false
	sectionLines := 0
	maxLinesPerSection := 20

	for _, line := range lines {
		// Detect section headers
		if strings.HasPrefix(line, "## ") {
			header := strings.TrimPrefix(line, "## ")
			header = strings.ToLower(strings.TrimSpace(header))
			if strings.Contains(header, "tldr") ||
				strings.Contains(header, "conclusion") ||
				strings.Contains(header, "summary") ||
				strings.Contains(header, "recommendation") {
				inSection = true
				sectionLines = 0
				summary.WriteString(line + "\n")
			} else {
				inSection = false
			}
		} else if inSection && sectionLines < maxLinesPerSection {
			summary.WriteString(line + "\n")
			sectionLines++
		}
	}

	result := summary.String()
	if result == "" {
		// Fallback: take first 30 lines
		maxLines := 30
		if len(lines) < maxLines {
			maxLines = len(lines)
		}
		result = strings.Join(lines[:maxLines], "\n")
	}

	return result, nil
}

// synthesizeAnswer sends context to LLM and gets synthesized answer.
func synthesizeAnswer(question, context string) (string, error) {
	// Ensure OpenCode is running
	if err := ensureOpenCodeRunning(); err != nil {
		return "", fmt.Errorf("OpenCode not available: %w", err)
	}

	// Build synthesis prompt
	prompt := buildSynthesisPrompt(question, context)

	// Resolve model - use sonnet by default for speed (cheaper, faster)
	modelSpec := model.Resolve(kbAskModel)
	if kbAskModel == "" {
		modelSpec = model.Resolve("sonnet")
	}

	// Create a temporary session for synthesis
	client := opencode.NewClient(serverURL) // entry-point: synthesizeAnswer is a self-contained operation
	projectDir, _ := currentProjectDir()

	// Create session with title indicating kb ask
	title := fmt.Sprintf("kb-ask-%d", time.Now().Unix())
	// kb ask is not a worker spawn and doesn't need extended thinking
	session, err := client.CreateSession(title, projectDir, modelSpec.Format(), "", false)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	// Send message with model specified and wait for response
	// SendMessageAsync doesn't pass model to prompt, so we use synchronous approach
	if err := client.SendMessageAsync(session.ID, prompt, modelSpec.Format()); err != nil {
		return "", fmt.Errorf("failed to send prompt: %w", err)
	}

	// Poll for response with timeout
	maxWait := 60 * time.Second
	pollInterval := 500 * time.Millisecond
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		time.Sleep(pollInterval)

		// Check session status via messages
		messages, err := client.GetMessages(session.ID)
		if err != nil {
			continue
		}

		// Look for completed assistant message
		for i := len(messages) - 1; i >= 0; i-- {
			msg := messages[i]
			if msg.Info.Role == "assistant" && msg.Info.Time.Completed > 0 {
				// Found completed assistant message - extract text
				var textParts []string
				for _, part := range msg.Parts {
					if part.Type == "text" && part.Text != "" {
						textParts = append(textParts, part.Text)
					}
				}
				if len(textParts) > 0 {
					return strings.Join(textParts, ""), nil
				}
			}
		}
	}

	return "", fmt.Errorf("timeout waiting for LLM response (session: %s)", session.ID)
}

// buildSynthesisPrompt creates the prompt for LLM synthesis.
func buildSynthesisPrompt(question, context string) string {
	return fmt.Sprintf(`You are answering a quick question based on the provided knowledge base context.

QUESTION: %s

CONTEXT FROM KNOWLEDGE BASE:
%s

INSTRUCTIONS:
1. Answer the question directly and concisely based on the context provided
2. Reference specific constraints, decisions, or investigation findings
3. If the context doesn't fully answer the question, say what's missing
4. Keep the answer brief (2-4 paragraphs max unless more detail is needed)
5. If there are constraints that must be respected, highlight them
6. Don't make things up - only use information from the context

Provide your answer:`, question, context)
}

// saveAsInvestigation saves the Q&A as an investigation artifact.
func saveAsInvestigation(question, answer string, context *KBContextResult) (string, error) {
	projectDir, err := currentProjectDir()
	if err != nil {
		return "", err
	}

	// Generate filename
	timestamp := time.Now().Format("2006-01-02")
	slug := generateSlug(question)
	filename := fmt.Sprintf("%s-inv-%s.md", timestamp, slug)

	// Determine path
	kbDir := filepath.Join(projectDir, ".kb", "investigations", "simple")
	if err := os.MkdirAll(kbDir, 0755); err != nil {
		return "", err
	}

	path := filepath.Join(kbDir, filename)

	// Build investigation content
	var content strings.Builder
	content.WriteString(fmt.Sprintf("# %s\n\n", question))
	content.WriteString(fmt.Sprintf("**Created:** %s (via kb ask)\n", time.Now().Format("2006-01-02 15:04")))
	content.WriteString("**Status:** Complete\n\n")

	content.WriteString("## TLDR\n\n")
	// Extract first paragraph of answer as TLDR
	paragraphs := strings.Split(answer, "\n\n")
	if len(paragraphs) > 0 {
		content.WriteString(paragraphs[0])
		content.WriteString("\n\n")
	}

	content.WriteString("## Full Answer\n\n")
	content.WriteString(answer)
	content.WriteString("\n\n")

	// Add sources
	content.WriteString("## Sources\n\n")
	for _, inv := range context.Investigations {
		content.WriteString(fmt.Sprintf("- %s: %s\n", inv.Title, inv.Path))
	}
	for _, d := range context.Decisions {
		content.WriteString(fmt.Sprintf("- [kn] %s\n", d.Content))
	}
	for _, c := range context.Constraints {
		content.WriteString(fmt.Sprintf("- [constraint] %s\n", c.Content))
	}

	if err := os.WriteFile(path, []byte(content.String()), 0644); err != nil {
		return "", err
	}

	return path, nil
}

// KBProjectsRegistry represents the ~/.kb/projects.json structure.
type KBProjectsRegistry struct {
	Projects []KBProject `json:"projects"`
}

// KBProject represents a single project entry.
type KBProject struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// runKBExtract extracts an artifact to another project with lineage tracking.
func runKBExtract(artifactPath, targetProject string, updateSource bool) error {
	// Resolve artifact path to absolute
	absArtifactPath, err := resolveArtifactPath(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to resolve artifact path: %w", err)
	}

	// Verify artifact exists
	if _, err := os.Stat(absArtifactPath); os.IsNotExist(err) {
		return fmt.Errorf("artifact not found: %s", absArtifactPath)
	}

	// Find target project path from registry
	targetPath, err := findProjectPath(targetProject)
	if err != nil {
		return err
	}

	// Determine artifact type and target directory
	targetDir, err := determineTargetDir(absArtifactPath, targetPath)
	if err != nil {
		return err
	}

	// Read original artifact
	content, err := os.ReadFile(absArtifactPath)
	if err != nil {
		return fmt.Errorf("failed to read artifact: %w", err)
	}

	// Get source project name for lineage
	sourceProject := getProjectName(absArtifactPath)

	// Add lineage header
	newContent := addLineageHeader(string(content), absArtifactPath, sourceProject)

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Write to target
	targetFile := filepath.Join(targetDir, filepath.Base(absArtifactPath))
	if err := os.WriteFile(targetFile, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write artifact: %w", err)
	}

	fmt.Printf("Extracted: %s\n", absArtifactPath)
	fmt.Printf("       To: %s\n", targetFile)

	// Optionally update source with extracted-to reference
	if updateSource {
		if err := addExtractedToReference(absArtifactPath, targetFile, targetProject); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update source: %v\n", err)
		} else {
			fmt.Printf("   Updated source with extracted-to reference\n")
		}
	}

	return nil
}

// resolveArtifactPath converts a path to absolute, handling relative paths.
func resolveArtifactPath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}

	cwd, err := currentProjectDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(cwd, path), nil
}

// findProjectPath looks up a project in ~/.kb/projects.json and returns its path.
func findProjectPath(projectName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	registryPath := filepath.Join(homeDir, ".kb", "projects.json")
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return "", fmt.Errorf("failed to read projects registry: %w", err)
	}

	var registry KBProjectsRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return "", fmt.Errorf("failed to parse projects registry: %w", err)
	}

	for _, project := range registry.Projects {
		if project.Name == projectName {
			return project.Path, nil
		}
	}

	return "", fmt.Errorf("project not found in registry: %s (use 'kb projects list' to see available projects)", projectName)
}

// determineTargetDir determines the appropriate .kb/ subdirectory for an artifact.
func determineTargetDir(artifactPath, targetProjectPath string) (string, error) {
	// Extract the .kb/ relative path from the artifact
	// e.g., /path/project/.kb/investigations/foo.md -> investigations
	// e.g., /path/project/.kb/decisions/bar.md -> decisions

	artifactPath = filepath.Clean(artifactPath)

	// Find .kb/ in the path
	kbIndex := strings.Index(artifactPath, "/.kb/")
	if kbIndex == -1 {
		kbIndex = strings.Index(artifactPath, "\\.kb\\") // Windows compatibility
	}

	if kbIndex == -1 {
		// Not in a .kb directory - put in .kb/extracted/
		return filepath.Join(targetProjectPath, ".kb", "extracted"), nil
	}

	// Get the relative path after .kb/
	relativePath := artifactPath[kbIndex+5:] // len("/.kb/") = 5
	relativeDir := filepath.Dir(relativePath)

	return filepath.Join(targetProjectPath, ".kb", relativeDir), nil
}

// getProjectName extracts project name from a path.
func getProjectName(path string) string {
	path = filepath.Clean(path)

	// Find .kb/ in path and get the directory before it
	kbIndex := strings.Index(path, "/.kb/")
	if kbIndex == -1 {
		kbIndex = strings.Index(path, "\\.kb\\")
	}

	if kbIndex == -1 {
		// Fallback: use directory name
		return filepath.Base(filepath.Dir(path))
	}

	projectDir := path[:kbIndex]
	return filepath.Base(projectDir)
}

// addLineageHeader adds extracted-from metadata to artifact content.
func addLineageHeader(content, originalPath, sourceProject string) string {
	timestamp := time.Now().Format("2006-01-02")

	lineageComment := fmt.Sprintf(`<!-- Lineage metadata (added by kb extract) -->
<!-- extracted-from: %s -->
<!-- source-project: %s -->
<!-- extraction-date: %s -->

`, originalPath, sourceProject, timestamp)

	// Check if content starts with YAML frontmatter (---)
	if strings.HasPrefix(strings.TrimSpace(content), "---") {
		// Find end of frontmatter
		lines := strings.SplitN(content, "\n", -1)
		frontmatterEnd := -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				frontmatterEnd = i
				break
			}
		}

		if frontmatterEnd > 0 {
			// Insert after frontmatter
			before := strings.Join(lines[:frontmatterEnd+1], "\n")
			after := strings.Join(lines[frontmatterEnd+1:], "\n")
			return before + "\n\n" + lineageComment + after
		}
	}

	// No frontmatter - prepend lineage comment
	return lineageComment + content
}

// addExtractedToReference adds a reference to the source file indicating where it was extracted to.
func addExtractedToReference(sourcePath, targetPath, targetProject string) error {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	timestamp := time.Now().Format("2006-01-02")
	extractedToComment := fmt.Sprintf("\n<!-- extracted-to: %s (project: %s, date: %s) -->\n", targetPath, targetProject, timestamp)

	// Append to end of file
	newContent := string(content) + extractedToComment

	return os.WriteFile(sourcePath, []byte(newContent), 0644)
}

// generateSlug creates a URL-safe slug from text.
func generateSlug(text string) string {
	// Lowercase and replace spaces/special chars
	slug := strings.ToLower(text)
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		if r == ' ' || r == '_' {
			return '-'
		}
		return -1
	}, slug)

	// Remove consecutive dashes
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Trim and limit length
	slug = strings.Trim(slug, "-")
	if len(slug) > 50 {
		slug = slug[:50]
	}

	return slug
}

// ArchiveResult contains the results of an archive operation
type ArchiveResult struct {
	Matched []string // Files that matched the age threshold
	Moved   []string // Files that were actually moved
	DestDir string   // Destination directory
}

// findKBDir finds the .kb directory in the project
func findKBDir(projectDir string) (string, error) {
	kbDir := filepath.Join(projectDir, ".kb")
	if _, err := os.Stat(kbDir); os.IsNotExist(err) {
		return "", fmt.Errorf("no .kb directory found in %s", projectDir)
	}
	return kbDir, nil
}

// runKBArchiveOld archives investigations older than the specified duration
func runKBArchiveOld(olderThan string, dryRun bool) error {
	projectDir, err := currentProjectDir()
	if err != nil {
		return err
	}

	// Parse duration
	threshold, err := parseArchiveDuration(olderThan)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", olderThan, err)
	}

	// Run archive
	result, err := archiveOldInvestigations(projectDir, threshold, dryRun)
	if err != nil {
		return err
	}

	// Print results
	if dryRun {
		fmt.Println("Dry run - no files moved")
		fmt.Println()
	}

	if len(result.Matched) == 0 {
		fmt.Printf("No investigations found older than %s\n", olderThan)
		return nil
	}

	if dryRun {
		fmt.Printf("Would archive %d investigation(s) to:\n", len(result.Matched))
		fmt.Printf("  %s\n\n", result.DestDir)
		fmt.Println("Files:")
		for _, f := range result.Matched {
			fmt.Printf("  - %s\n", filepath.Base(f))
		}
	} else {
		fmt.Printf("Archived %d investigation(s) to:\n", len(result.Moved))
		fmt.Printf("  %s\n\n", result.DestDir)
		fmt.Println("Files moved:")
		for _, f := range result.Moved {
			fmt.Printf("  - %s\n", filepath.Base(f))
		}
	}

	return nil
}

// parseArchiveDuration parses duration strings like "60d", "90d"
func parseArchiveDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format (expected format: 60d, 90d, etc.)")
	}

	// Check for 'd' suffix (days)
	if !strings.HasSuffix(s, "d") {
		return 0, fmt.Errorf("only day units are supported (use 'd' suffix, e.g., 60d)")
	}

	// Parse number of days
	daysStr := s[:len(s)-1]
	var days int
	if _, err := fmt.Sscanf(daysStr, "%d", &days); err != nil {
		return 0, fmt.Errorf("invalid number: %w", err)
	}

	if days < 0 {
		return 0, fmt.Errorf("duration must be positive")
	}

	return time.Duration(days) * 24 * time.Hour, nil
}

// parseInvestigationDate parses the YYYY-MM-DD date prefix from an investigation filename
func parseInvestigationDate(filename string) (time.Time, error) {
	// Extract basename if full path provided
	basename := filepath.Base(filename)

	// Check if filename starts with YYYY-MM-DD pattern
	if len(basename) < 10 {
		return time.Time{}, fmt.Errorf("filename too short to contain date prefix")
	}

	dateStr := basename[:10]
	date, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse date from filename: %w", err)
	}

	return date, nil
}

// calculateInvestigationAge calculates the age of an investigation based on filename date
func calculateInvestigationAge(filename string) (time.Duration, error) {
	date, err := parseInvestigationDate(filename)
	if err != nil {
		return 0, err
	}

	age := time.Since(date)
	return age, nil
}

// findOldInvestigations finds investigations older than the threshold
func findOldInvestigations(projectDir string, threshold time.Duration) ([]string, error) {
	kbDir, err := findKBDir(projectDir)
	if err != nil {
		return nil, err
	}

	invDir := filepath.Join(kbDir, "investigations")
	if _, err := os.Stat(invDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	var oldFiles []string

	// Read entries in investigations directory (non-recursive - only top-level files)
	entries, err := os.ReadDir(invDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		// Skip non-markdown files
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		// Calculate age
		age, err := calculateInvestigationAge(entry.Name())
		if err != nil {
			// Skip files with invalid date prefixes
			continue
		}

		// Check if older than threshold
		if age >= threshold {
			oldFiles = append(oldFiles, filepath.Join(invDir, entry.Name()))
		}
	}

	return oldFiles, nil
}

// archiveOldInvestigations archives investigations older than the threshold
func archiveOldInvestigations(projectDir string, threshold time.Duration, dryRun bool) (*ArchiveResult, error) {
	// Find old investigations
	matched, err := findOldInvestigations(projectDir, threshold)
	if err != nil {
		return nil, err
	}

	kbDir, err := findKBDir(projectDir)
	if err != nil {
		return nil, err
	}

	result := &ArchiveResult{
		Matched: matched,
		DestDir: filepath.Join(kbDir, "investigations", "archive"),
	}

	if len(matched) == 0 || dryRun {
		return result, nil
	}

	// Create archive directory
	if err := os.MkdirAll(result.DestDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Move files
	for _, srcPath := range matched {
		filename := filepath.Base(srcPath)
		destPath := filepath.Join(result.DestDir, filename)

		// Check if destination already exists
		if _, err := os.Stat(destPath); err == nil {
			// Skip if already archived
			continue
		}

		if err := os.Rename(srcPath, destPath); err != nil {
			return nil, fmt.Errorf("failed to move %s: %w", srcPath, err)
		}
		result.Moved = append(result.Moved, destPath)
	}

	return result, nil
}
