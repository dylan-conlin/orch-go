package dialogue

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	// TranscriptFileName is the markdown transcript output for dialogue sessions.
	TranscriptFileName = "DIALOGUE_TRANSCRIPT.md"
	// ArtifactReportFileName is the markdown summary of derived artifacts.
	ArtifactReportFileName = "DIALOGUE_ARTIFACTS.md"
	// ArtifactJSONFileName is the machine-readable artifact export.
	ArtifactJSONFileName = "DIALOGUE_ARTIFACTS.json"
)

var (
	headingPattern      = regexp.MustCompile(`^\s{0,3}#{1,6}\s+(.*\S)\s*$`)
	listItemPattern     = regexp.MustCompile(`^\s*(?:[-*+]\s+|\d+[.)]\s+)(.+)$`)
	cleanupTokenPattern = regexp.MustCompile(`\[\s*verdict\s*:[^\]]+\]`)
	normalizeKeyPattern = regexp.MustCompile(`[^a-z0-9]+`)
)

// TranscriptMetadata captures runtime metadata for transcript formatting.
type TranscriptMetadata struct {
	SessionID       string
	Topic           string
	QuestionerModel string
	ExpertModel     string
	StartedAt       time.Time
	CompletedAt     time.Time
}

// DecisionArtifact is a structured decision extracted from dialogue turns.
type DecisionArtifact struct {
	Summary     string `json:"summary"`
	SourceTurn  int    `json:"source_turn"`
	SourcePhase Phase  `json:"source_phase"`
}

// FollowUpIssueDraft is a structured follow-up issue draft extracted from dialogue turns.
type FollowUpIssueDraft struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	IssueType   string `json:"issue_type"`
	Priority    int    `json:"priority"`
	SourceTurn  int    `json:"source_turn"`
	SourcePhase Phase  `json:"source_phase"`
}

// ArtifactBundle is the machine-readable export for derived dialogue artifacts.
type ArtifactBundle struct {
	GeneratedAt    string               `json:"generated_at"`
	SessionID      string               `json:"session_id"`
	Topic          string               `json:"topic"`
	Approved       bool                 `json:"approved"`
	Verdict        string               `json:"verdict"`
	EndReason      string               `json:"end_reason"`
	TurnCount      int                  `json:"turn_count"`
	Decisions      []DecisionArtifact   `json:"decisions"`
	FollowUpIssues []FollowUpIssueDraft `json:"follow_up_issues"`
}

// ArtifactFiles reports where dialogue transcript artifacts were written.
type ArtifactFiles struct {
	TranscriptPath   string
	ArtifactMDPath   string
	ArtifactJSONPath string
	DecisionCount    int
	FollowUpCount    int
}

type decisionCandidate struct {
	text  string
	turn  int
	phase Phase
}

type followUpCandidate struct {
	text  string
	turn  int
	phase Phase
}

type markdownSection struct {
	heading string
	body    []string
}

// WriteArtifacts writes transcript and derived artifact files in workspacePath.
func WriteArtifacts(workspacePath string, metadata TranscriptMetadata, cfg RelayConfig, result *RelayResult) (*ArtifactFiles, error) {
	workspacePath = strings.TrimSpace(workspacePath)
	if workspacePath == "" {
		return nil, fmt.Errorf("workspace path is required")
	}
	if result == nil {
		return nil, fmt.Errorf("relay result is required")
	}
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}

	transcriptPath := filepath.Join(workspacePath, TranscriptFileName)
	if err := os.WriteFile(transcriptPath, []byte(FormatTranscript(metadata, cfg, result)), 0644); err != nil {
		return nil, fmt.Errorf("write transcript: %w", err)
	}

	decisions, followUps := DeriveArtifacts(result)
	bundle := ArtifactBundle{
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		SessionID:      metadata.SessionID,
		Topic:          metadata.Topic,
		Approved:       result.Approved,
		Verdict:        result.Verdict,
		EndReason:      result.EndReason,
		TurnCount:      len(result.Turns),
		Decisions:      decisions,
		FollowUpIssues: followUps,
	}

	artifactJSONPath := filepath.Join(workspacePath, ArtifactJSONFileName)
	payload, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal artifact bundle: %w", err)
	}
	if err := os.WriteFile(artifactJSONPath, payload, 0644); err != nil {
		return nil, fmt.Errorf("write artifact json: %w", err)
	}

	artifactMDPath := filepath.Join(workspacePath, ArtifactReportFileName)
	if err := os.WriteFile(artifactMDPath, []byte(FormatArtifactReport(bundle)), 0644); err != nil {
		return nil, fmt.Errorf("write artifact report: %w", err)
	}

	return &ArtifactFiles{
		TranscriptPath:   transcriptPath,
		ArtifactMDPath:   artifactMDPath,
		ArtifactJSONPath: artifactJSONPath,
		DecisionCount:    len(decisions),
		FollowUpCount:    len(followUps),
	}, nil
}

// FormatTranscript renders a detailed markdown transcript with session metadata.
func FormatTranscript(metadata TranscriptMetadata, cfg RelayConfig, result *RelayResult) string {
	if result == nil {
		return ""
	}

	lines := []string{
		"---",
		"type: dialogue_transcript",
		fmt.Sprintf("session_id: %q", metadata.SessionID),
		fmt.Sprintf("topic: %q", metadata.Topic),
		fmt.Sprintf("questioner_model: %q", metadata.QuestionerModel),
		fmt.Sprintf("expert_model: %q", metadata.ExpertModel),
		fmt.Sprintf("approved: %t", result.Approved),
		fmt.Sprintf("verdict: %q", result.Verdict),
		fmt.Sprintf("end_reason: %q", result.EndReason),
		fmt.Sprintf("turns: %d", len(result.Turns)),
	}

	if !metadata.StartedAt.IsZero() {
		lines = append(lines, fmt.Sprintf("started_at: %q", metadata.StartedAt.UTC().Format(time.RFC3339)))
	}
	if !metadata.CompletedAt.IsZero() {
		lines = append(lines, fmt.Sprintf("completed_at: %q", metadata.CompletedAt.UTC().Format(time.RFC3339)))
	}

	lines = append(lines,
		"---",
		"",
		"# Dialogue Transcript",
		"",
		"## Session",
		fmt.Sprintf("- Session ID: `%s`", metadata.SessionID),
		fmt.Sprintf("- Topic: %s", metadata.Topic),
		fmt.Sprintf("- Questioner model: `%s`", metadata.QuestionerModel),
		fmt.Sprintf("- Expert model: `%s`", metadata.ExpertModel),
		fmt.Sprintf("- Outcome: approved=%t, verdict=`%s`, end_reason=`%s`", result.Approved, result.Verdict, result.EndReason),
		fmt.Sprintf("- Turn limits: explore=%d, converge=%d, max=%d", cfg.ExploreTurns, cfg.ConvergeTurns, cfg.MaxTurns),
		"",
	)

	for _, turn := range result.Turns {
		lines = append(lines,
			fmt.Sprintf("## Turn %d (%s)", turn.Number, turn.Phase),
			"",
			"### Ghost Partner",
			"",
			strings.TrimSpace(turn.GhostMessage),
			"",
		)

		if turn.Usage.InputTokens > 0 || turn.Usage.OutputTokens > 0 {
			lines = append(lines, fmt.Sprintf("_Ghost usage: input=%d output=%d_", turn.Usage.InputTokens, turn.Usage.OutputTokens), "")
		}

		if turn.Verdict != "" {
			lines = append(lines, fmt.Sprintf("_Verdict token: `%s`_", turn.Verdict), "")
		}

		if strings.TrimSpace(turn.ExpertResponse) != "" {
			lines = append(lines,
				"### Expert",
				"",
				strings.TrimSpace(turn.ExpertResponse),
				"",
			)
		} else {
			lines = append(lines,
				"### Expert",
				"",
				"_No expert response (dialogue terminated on this turn)._",
				"",
			)
		}
	}

	return strings.Join(lines, "\n")
}

// FormatArtifactReport renders markdown for decisions and follow-up issue drafts.
func FormatArtifactReport(bundle ArtifactBundle) string {
	lines := []string{
		"# Dialogue Derived Artifacts",
		"",
		"## Session",
		fmt.Sprintf("- Session ID: `%s`", bundle.SessionID),
		fmt.Sprintf("- Topic: %s", bundle.Topic),
		fmt.Sprintf("- Outcome: approved=%t, verdict=`%s`, end_reason=`%s`", bundle.Approved, bundle.Verdict, bundle.EndReason),
		fmt.Sprintf("- Turns: %d", bundle.TurnCount),
		fmt.Sprintf("- Generated: %s", bundle.GeneratedAt),
		"",
		"## Decisions",
	}

	if len(bundle.Decisions) == 0 {
		lines = append(lines, "- None extracted from dialogue.", "")
	} else {
		for i, decision := range bundle.Decisions {
			lines = append(lines,
				fmt.Sprintf("### Decision %d", i+1),
				fmt.Sprintf("- Source: turn %d (%s)", decision.SourceTurn, decision.SourcePhase),
				fmt.Sprintf("- Summary: %s", decision.Summary),
				"",
			)
		}
	}

	lines = append(lines, "## Follow-Up Issue Drafts")
	if len(bundle.FollowUpIssues) == 0 {
		lines = append(lines, "- None extracted from dialogue.")
		return strings.Join(lines, "\n")
	}

	for i, followUp := range bundle.FollowUpIssues {
		lines = append(lines,
			fmt.Sprintf("### Issue %d: %s", i+1, followUp.Title),
			fmt.Sprintf("- Type: `%s`", followUp.IssueType),
			fmt.Sprintf("- Priority: `P%d`", followUp.Priority),
			fmt.Sprintf("- Source: turn %d (%s)", followUp.SourceTurn, followUp.SourcePhase),
			"- Description:",
			"",
			followUp.Description,
			"",
		)
	}

	return strings.Join(lines, "\n")
}

// DeriveArtifacts extracts decision and follow-up issue drafts from dialogue turns.
func DeriveArtifacts(result *RelayResult) ([]DecisionArtifact, []FollowUpIssueDraft) {
	if result == nil || len(result.Turns) == 0 {
		return nil, nil
	}

	decisionCandidates := make([]decisionCandidate, 0)
	followUpCandidates := make([]followUpCandidate, 0)

	for _, turn := range result.Turns {
		decisionTexts, followUpTexts := extractCandidates(turn.GhostMessage)
		for _, d := range decisionTexts {
			decisionCandidates = append(decisionCandidates, decisionCandidate{text: d, turn: turn.Number, phase: turn.Phase})
		}
		for _, f := range followUpTexts {
			followUpCandidates = append(followUpCandidates, followUpCandidate{text: f, turn: turn.Number, phase: turn.Phase})
		}
	}

	lastTurn := result.Turns[len(result.Turns)-1]
	if len(decisionCandidates) == 0 {
		if fallback := firstDecisionFallback(lastTurn.GhostMessage); fallback != "" {
			decisionCandidates = append(decisionCandidates, decisionCandidate{text: fallback, turn: lastTurn.Number, phase: lastTurn.Phase})
		}
	}
	if len(followUpCandidates) == 0 {
		for _, item := range extractListItems(strings.Split(lastTurn.GhostMessage, "\n")) {
			followUpCandidates = append(followUpCandidates, followUpCandidate{text: item, turn: lastTurn.Number, phase: lastTurn.Phase})
		}
	}

	seenDecisions := map[string]struct{}{}
	decisions := make([]DecisionArtifact, 0, len(decisionCandidates))
	for _, candidate := range decisionCandidates {
		summary := strings.TrimSpace(candidate.text)
		if summary == "" {
			continue
		}
		key := normalizeKey(summary)
		if _, exists := seenDecisions[key]; exists {
			continue
		}
		seenDecisions[key] = struct{}{}
		decisions = append(decisions, DecisionArtifact{
			Summary:     summary,
			SourceTurn:  candidate.turn,
			SourcePhase: candidate.phase,
		})
	}

	seenFollowUps := map[string]struct{}{}
	followUps := make([]FollowUpIssueDraft, 0, len(followUpCandidates))
	for _, candidate := range followUpCandidates {
		raw := strings.TrimSpace(candidate.text)
		if raw == "" {
			continue
		}
		key := normalizeKey(raw)
		if _, exists := seenFollowUps[key]; exists {
			continue
		}
		seenFollowUps[key] = struct{}{}
		followUps = append(followUps, FollowUpIssueDraft{
			Title:       titleFromAction(raw),
			Description: raw,
			IssueType:   classifyIssueType(raw),
			Priority:    2,
			SourceTurn:  candidate.turn,
			SourcePhase: candidate.phase,
		})
	}

	return decisions, followUps
}

func extractCandidates(text string) ([]string, []string) {
	sections := parseMarkdownSections(text)
	decisions := make([]string, 0)
	followUps := make([]string, 0)

	for _, section := range sections {
		heading := strings.ToLower(strings.TrimSpace(section.heading))
		bodyLines := cleanedLines(section.body)

		switch {
		case headingHasAny(heading, "decision", "recommendation", "proposal", "approach"):
			if summary := firstMeaningfulParagraph(bodyLines); summary != "" {
				decisions = append(decisions, summary)
			}
		case headingHasAny(heading, "follow-up", "follow up", "next step", "implementation", "action", "todo"):
			followUps = append(followUps, extractListItems(bodyLines)...)
		}
	}

	if len(followUps) == 0 {
		for _, section := range sections {
			if headingHasAny(strings.ToLower(section.heading), "proposal") {
				followUps = append(followUps, extractListItems(cleanedLines(section.body))...)
			}
		}
	}

	return decisions, followUps
}

func parseMarkdownSections(text string) []markdownSection {
	lines := strings.Split(text, "\n")
	sections := []markdownSection{{heading: "", body: []string{}}}

	for _, line := range lines {
		if matches := headingPattern.FindStringSubmatch(line); len(matches) == 2 {
			sections = append(sections, markdownSection{heading: strings.TrimSpace(matches[1]), body: []string{}})
			continue
		}
		sections[len(sections)-1].body = append(sections[len(sections)-1].body, line)
	}

	return sections
}

func cleanedLines(lines []string) []string {
	clean := make([]string, 0, len(lines))
	for _, line := range lines {
		line = cleanupTokenPattern.ReplaceAllString(line, "")
		clean = append(clean, strings.TrimSpace(line))
	}
	return clean
}

func firstMeaningfulParagraph(lines []string) string {
	paragraph := make([]string, 0)
	for _, line := range lines {
		if line == "" {
			if len(paragraph) > 0 {
				break
			}
			continue
		}
		if headingPattern.MatchString(line) {
			continue
		}
		if listItemPattern.MatchString(line) {
			if len(paragraph) > 0 {
				break
			}
			continue
		}
		paragraph = append(paragraph, line)
	}

	joined := strings.TrimSpace(strings.Join(paragraph, " "))
	if joined == "" {
		return ""
	}
	return joined
}

func extractListItems(lines []string) []string {
	items := make([]string, 0)
	for _, line := range lines {
		matches := listItemPattern.FindStringSubmatch(line)
		if len(matches) != 2 {
			continue
		}
		item := strings.TrimSpace(matches[1])
		item = strings.TrimPrefix(item, "[ ] ")
		item = strings.TrimPrefix(item, "[x] ")
		item = strings.TrimPrefix(item, "[X] ")
		item = strings.TrimSpace(item)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func firstDecisionFallback(text string) string {
	lines := cleanedLines(strings.Split(text, "\n"))
	for _, line := range lines {
		if line == "" || headingPattern.MatchString(line) || listItemPattern.MatchString(line) {
			continue
		}
		return line
	}
	return ""
}

func headingHasAny(heading string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(heading, keyword) {
			return true
		}
	}
	return false
}

func normalizeKey(text string) string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	normalized = normalizeKeyPattern.ReplaceAllString(normalized, " ")
	return strings.Join(strings.Fields(normalized), " ")
}

func classifyIssueType(text string) string {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "bug") || strings.Contains(lower, "fix") || strings.Contains(lower, "regression") || strings.Contains(lower, "error") {
		return "bug"
	}
	if strings.Contains(lower, "investigate") || strings.Contains(lower, "why") || strings.Contains(lower, "question") || strings.Contains(lower, "validate") {
		return "question"
	}
	if strings.Contains(lower, "add ") || strings.Contains(lower, "introduce") || strings.Contains(lower, "build") {
		return "feature"
	}
	return "task"
}

func titleFromAction(text string) string {
	title := strings.TrimSpace(text)
	title = strings.TrimSuffix(title, ".")
	title = strings.TrimPrefix(title, "TODO: ")
	title = strings.TrimPrefix(title, "Todo: ")
	title = strings.TrimPrefix(title, "todo: ")
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Follow-up from dialogue"
	}
	if len(title) > 120 {
		title = title[:117] + "..."
	}
	return title
}
