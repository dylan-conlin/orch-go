package dialogue

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

const (
	defaultExploreTurns    = 6
	defaultConvergeTurns   = 12
	defaultMaxTurns        = 15
	defaultPollInterval    = 2 * time.Second
	defaultResponseTimeout = 2 * time.Minute

	maxStablePolls = 2
)

var verdictPattern = regexp.MustCompile(`(?is)\[\s*verdict\s*:\s*([^\]]+)\]`)

// QuestionerClient is the API surface required from the Ghost questioner.
type QuestionerClient interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}

// ExpertClient is the API surface required from the running OpenCode session.
type ExpertClient interface {
	SendMessageAsync(sessionID, content, model string) error
	GetMessages(sessionID string) ([]opencode.Message, error)
}

// Observer receives per-turn relay updates.
type Observer interface {
	OnGhostTurn(turn int, phase Phase, text string)
	OnExpertTurn(turn int, text string)
}

// NoopObserver ignores all turn updates.
type NoopObserver struct{}

// OnGhostTurn implements Observer.
func (NoopObserver) OnGhostTurn(int, Phase, string) {}

// OnExpertTurn implements Observer.
func (NoopObserver) OnExpertTurn(int, string) {}

// RelayConfig configures the Ghost Partner relay loop.
type RelayConfig struct {
	Topic           string
	QuestionerModel string
	ExpertModel     string
	ExploreTurns    int
	ConvergeTurns   int
	MaxTurns        int
	MaxTokens       int
	PollInterval    time.Duration
	ResponseTimeout time.Duration
}

// Turn records one ghost/expert exchange.
type Turn struct {
	Number         int
	Phase          Phase
	GhostMessage   string
	ExpertResponse string
	Verdict        string
	Usage          Usage
}

// RelayResult captures dialogue loop output.
type RelayResult struct {
	Turns     []Turn
	Approved  bool
	Verdict   string
	EndReason string
}

// RunRelay executes the Ghost Partner relay loop.
func RunRelay(ctx context.Context, questioner QuestionerClient, expert ExpertClient, sessionID string, cfg RelayConfig, observer Observer) (*RelayResult, error) {
	cfg = withDefaults(cfg)
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	if observer == nil {
		observer = NoopObserver{}
	}

	history := []Message{initialGhostUserMessage(cfg.Topic)}
	result := &RelayResult{}

	for turnNumber := 1; turnNumber <= cfg.MaxTurns; turnNumber++ {
		phase := phaseForTurn(turnNumber, cfg.ExploreTurns, cfg.ConvergeTurns)

		ghostResp, err := questioner.Complete(ctx, CompletionRequest{
			Model:        cfg.QuestionerModel,
			MaxTokens:    cfg.MaxTokens,
			SystemPrompt: buildGhostSystemPrompt(cfg.Topic, phase, turnNumber, cfg.MaxTurns),
			Messages:     history,
		})
		if err != nil {
			return nil, fmt.Errorf("ghost turn %d failed: %w", turnNumber, err)
		}

		ghostText := strings.TrimSpace(ghostResp.Text)
		if ghostText == "" {
			return nil, fmt.Errorf("ghost turn %d returned empty response", turnNumber)
		}

		observer.OnGhostTurn(turnNumber, phase, ghostText)
		history = append(history, Message{Role: "assistant", Content: ghostText})

		turn := Turn{
			Number:       turnNumber,
			Phase:        phase,
			GhostMessage: ghostText,
			Usage:        ghostResp.Usage,
		}

		if verdict, ok := ParseVerdict(ghostText); ok {
			turn.Verdict = verdict
			if IsApprovedVerdict(verdict) {
				result.Turns = append(result.Turns, turn)
				result.Approved = true
				result.Verdict = verdict
				result.EndReason = "ghost_approved"
				return result, nil
			}
		}

		expertText, err := relayToExpert(ctx, expert, sessionID, ghostText, cfg.ExpertModel, cfg.PollInterval, cfg.ResponseTimeout)
		if err != nil {
			return nil, fmt.Errorf("expert turn %d failed: %w", turnNumber, err)
		}

		observer.OnExpertTurn(turnNumber, expertText)
		history = append(history, Message{Role: "user", Content: expertText})
		turn.ExpertResponse = expertText
		result.Turns = append(result.Turns, turn)
	}

	result.EndReason = "max_turns"
	return result, nil
}

func withDefaults(cfg RelayConfig) RelayConfig {
	if cfg.ExploreTurns <= 0 {
		cfg.ExploreTurns = defaultExploreTurns
	}
	if cfg.ConvergeTurns <= 0 {
		cfg.ConvergeTurns = defaultConvergeTurns
	}
	if cfg.MaxTurns <= 0 {
		cfg.MaxTurns = defaultMaxTurns
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = DefaultMaxTokens
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaultPollInterval
	}
	if cfg.ResponseTimeout <= 0 {
		cfg.ResponseTimeout = defaultResponseTimeout
	}
	return cfg
}

func validateConfig(cfg RelayConfig) error {
	if strings.TrimSpace(cfg.Topic) == "" {
		return fmt.Errorf("topic is required")
	}
	if cfg.ExploreTurns <= 0 {
		return fmt.Errorf("explore turns must be > 0")
	}
	if cfg.ConvergeTurns < cfg.ExploreTurns {
		return fmt.Errorf("converge turns (%d) must be >= explore turns (%d)", cfg.ConvergeTurns, cfg.ExploreTurns)
	}
	if cfg.MaxTurns < cfg.ConvergeTurns {
		return fmt.Errorf("max turns (%d) must be >= converge turns (%d)", cfg.MaxTurns, cfg.ConvergeTurns)
	}
	return nil
}

// ParseVerdict extracts [VERDICT: ...] token from text.
func ParseVerdict(text string) (string, bool) {
	matches := verdictPattern.FindStringSubmatch(text)
	if len(matches) < 2 {
		return "", false
	}
	verdict := normalizeVerdict(matches[1])
	if verdict == "" {
		return "", false
	}
	return verdict, true
}

// IsApprovedVerdict returns true when verdict indicates approval.
func IsApprovedVerdict(verdict string) bool {
	verdict = normalizeVerdict(verdict)
	return verdict == "APPROVED" || verdict == "APPROVE"
}

func normalizeVerdict(raw string) string {
	v := strings.ToUpper(strings.TrimSpace(raw))
	v = strings.ReplaceAll(v, "-", "_")
	v = strings.Join(strings.Fields(v), "_")
	return v
}

func relayToExpert(ctx context.Context, expert ExpertClient, sessionID, prompt, model string, pollInterval, timeout time.Duration) (string, error) {
	baselineMessages, err := expert.GetMessages(sessionID)
	if err != nil {
		return "", fmt.Errorf("fetch baseline messages: %w", err)
	}
	baselineIDs := make(map[string]struct{}, len(baselineMessages))
	for _, msg := range baselineMessages {
		if msg.Info.ID != "" {
			baselineIDs[msg.Info.ID] = struct{}{}
		}
	}

	if err := expert.SendMessageAsync(sessionID, prompt, model); err != nil {
		return "", fmt.Errorf("send message: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var lastMessageID string
	var lastText string
	stablePolls := 0

	for {
		select {
		case <-ctx.Done():
			if strings.TrimSpace(lastText) != "" {
				return strings.TrimSpace(lastText), nil
			}
			return "", fmt.Errorf("timed out waiting for expert response")
		case <-ticker.C:
			messages, err := expert.GetMessages(sessionID)
			if err != nil {
				continue
			}
			msg, text, done := latestAssistantReply(messages, baselineIDs)
			if msg == nil || strings.TrimSpace(text) == "" {
				continue
			}

			if done {
				return strings.TrimSpace(text), nil
			}

			if msg.Info.ID == lastMessageID && text == lastText {
				stablePolls++
			} else {
				lastMessageID = msg.Info.ID
				lastText = text
				stablePolls = 0
			}

			if stablePolls >= maxStablePolls {
				return strings.TrimSpace(text), nil
			}
		}
	}
}

func latestAssistantReply(messages []opencode.Message, baselineIDs map[string]struct{}) (*opencode.Message, string, bool) {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := &messages[i]
		if msg.Info.Role != "assistant" {
			continue
		}
		if _, exists := baselineIDs[msg.Info.ID]; exists {
			continue
		}
		text := extractAssistantText(*msg)
		done := msg.Info.Time.Completed > 0 || msg.Info.Finish != ""
		return msg, text, done
	}
	return nil, "", false
}

func extractAssistantText(msg opencode.Message) string {
	parts := make([]string, 0, len(msg.Parts))
	for _, part := range msg.Parts {
		if part.Type == "text" {
			text := strings.TrimSpace(part.Text)
			if text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "\n")
}
