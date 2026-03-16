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
)

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
	if err := opencode.EnsureRunning(serverURL); err != nil {
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
	client := opencode.NewClient(serverURL)
	projectDir, _ := os.Getwd()

	// Create session with title indicating kb ask
	title := fmt.Sprintf("kb-ask-%d", time.Now().Unix())
	// kb ask sessions don't have beads tracking, so metadata is empty
	// Set 4-hour TTL for automatic cleanup of temporary kb ask sessions
	timeTTL := 4 * 60 * 60 // 4 hours in seconds
	session, err := client.CreateSession(title, projectDir, modelSpec.Format(), nil, timeTTL)
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
	projectDir, err := os.Getwd()
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
