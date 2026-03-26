package opencode

import "strings"

// TerminalOutcome classifies how an OpenCode session terminated.
type TerminalOutcome string

const (
	// OutcomeEmptyExecution means the session produced no meaningful output.
	// Zero output tokens, no assistant text, or no assistant messages at all.
	OutcomeEmptyExecution TerminalOutcome = "empty-execution"

	// OutcomeNormalCompletion means the session ran and produced output.
	OutcomeNormalCompletion TerminalOutcome = "normal-completion"

	// OutcomeErrorTermination means the session ended with an error finish reason.
	OutcomeErrorTermination TerminalOutcome = "error-termination"
)

// String returns the string representation of the outcome.
func (o TerminalOutcome) String() string { return string(o) }

// IsEmpty returns true if this outcome represents an empty execution.
func (o TerminalOutcome) IsEmpty() bool { return o == OutcomeEmptyExecution }

// OutcomeDetail provides the classification along with supporting evidence.
type OutcomeDetail struct {
	Outcome           TerminalOutcome
	Reason            string // Human-readable explanation
	OutputTokens      int    // Total output tokens across assistant messages
	AssistantMessages int    // Count of assistant messages
	ToolInvocations   int    // Count of tool-invocation parts
	HasArtifacts      bool   // Session summary shows file changes
}

// ClassifyTerminalOutcome inspects an OpenCode session's messages and metadata
// to distinguish empty executions from real completions.
//
// Classification logic:
//  1. No messages at all → empty-execution
//  2. No assistant messages → empty-execution
//  3. Any assistant message with finish="error" → error-termination
//  4. Zero output tokens AND no substantive text/tool output → empty-execution
//  5. Otherwise → normal-completion
func ClassifyTerminalOutcome(session Session, messages []Message) TerminalOutcome {
	return ClassifyTerminalOutcomeDetail(session, messages).Outcome
}

// ClassifyTerminalOutcomeDetail returns the full classification with evidence.
func ClassifyTerminalOutcomeDetail(session Session, messages []Message) OutcomeDetail {
	if len(messages) == 0 {
		return OutcomeDetail{
			Outcome: OutcomeEmptyExecution,
			Reason:  "no messages in session",
		}
	}

	var (
		assistantCount  int
		outputTokens    int
		toolInvocations int
		hasSubstantive  bool
		hasError        bool
	)

	for _, msg := range messages {
		if msg.Info.Role != "assistant" {
			continue
		}
		assistantCount++

		if msg.Info.Finish == "error" {
			hasError = true
		}

		if msg.Info.Tokens != nil {
			outputTokens += msg.Info.Tokens.Output
		}

		for _, part := range msg.Parts {
			switch part.Type {
			case "text":
				if strings.TrimSpace(part.Text) != "" {
					hasSubstantive = true
				}
			case "tool-invocation":
				toolInvocations++
				hasSubstantive = true
			}
		}
	}

	if assistantCount == 0 {
		return OutcomeDetail{
			Outcome: OutcomeEmptyExecution,
			Reason:  "no assistant messages",
		}
	}

	if hasError {
		return OutcomeDetail{
			Outcome:           OutcomeErrorTermination,
			Reason:            "assistant finish reason is error",
			OutputTokens:      outputTokens,
			AssistantMessages: assistantCount,
			ToolInvocations:   toolInvocations,
			HasArtifacts:      session.Summary.Files > 0,
		}
	}

	if outputTokens == 0 && !hasSubstantive {
		return OutcomeDetail{
			Outcome:           OutcomeEmptyExecution,
			Reason:            "zero output tokens and no substantive content",
			AssistantMessages: assistantCount,
		}
	}

	if !hasSubstantive {
		return OutcomeDetail{
			Outcome:           OutcomeEmptyExecution,
			Reason:            "assistant produced no substantive text or tool output",
			OutputTokens:      outputTokens,
			AssistantMessages: assistantCount,
		}
	}

	return OutcomeDetail{
		Outcome:           OutcomeNormalCompletion,
		Reason:            "assistant produced output",
		OutputTokens:      outputTokens,
		AssistantMessages: assistantCount,
		ToolInvocations:   toolInvocations,
		HasArtifacts:      session.Summary.Files > 0,
	}
}
