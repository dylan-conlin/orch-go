package opencode

import (
	"testing"
	"time"
)

func TestClassifyTerminalOutcome(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		session  Session
		messages []Message
		want     TerminalOutcome
	}{
		{
			name: "empty execution: no messages at all",
			session: Session{
				ID:    "sess-1",
				Time:  SessionTime{Created: now.UnixMilli(), Updated: now.Add(2 * time.Second).UnixMilli()},
				Title: "test session",
			},
			messages: nil,
			want:     OutcomeEmptyExecution,
		},
		{
			name: "empty execution: only user message, no assistant response",
			session: Session{
				ID:   "sess-2",
				Time: SessionTime{Created: now.UnixMilli(), Updated: now.Add(5 * time.Second).UnixMilli()},
			},
			messages: []Message{
				{
					Info: MessageInfo{
						ID:   "msg-1",
						Role: "user",
						Time: MessageTime{Created: now.UnixMilli()},
					},
					Parts: []MessagePart{
						{Type: "text", Text: "Please implement the feature"},
					},
				},
			},
			want: OutcomeEmptyExecution,
		},
		{
			name: "empty execution: assistant responded with zero output tokens",
			session: Session{
				ID:   "sess-3",
				Time: SessionTime{Created: now.UnixMilli(), Updated: now.Add(3 * time.Second).UnixMilli()},
			},
			messages: []Message{
				{
					Info: MessageInfo{Role: "user"},
					Parts: []MessagePart{
						{Type: "text", Text: "do something"},
					},
				},
				{
					Info: MessageInfo{
						Role:   "assistant",
						Tokens: &MessageToken{Input: 500, Output: 0},
						Finish: "stop",
					},
					Parts: []MessagePart{},
				},
			},
			want: OutcomeEmptyExecution,
		},
		{
			name: "normal completion: assistant produced text output",
			session: Session{
				ID:   "sess-4",
				Time: SessionTime{Created: now.UnixMilli(), Updated: now.Add(60 * time.Second).UnixMilli()},
			},
			messages: []Message{
				{
					Info: MessageInfo{Role: "user"},
					Parts: []MessagePart{
						{Type: "text", Text: "implement feature X"},
					},
				},
				{
					Info: MessageInfo{
						Role:   "assistant",
						Tokens: &MessageToken{Input: 5000, Output: 1200},
						Finish: "stop",
					},
					Parts: []MessagePart{
						{Type: "text", Text: "I'll implement feature X for you."},
					},
				},
			},
			want: OutcomeNormalCompletion,
		},
		{
			name: "normal completion: assistant used tools and produced artifacts",
			session: Session{
				ID:      "sess-5",
				Time:    SessionTime{Created: now.UnixMilli(), Updated: now.Add(120 * time.Second).UnixMilli()},
				Summary: SessionSummary{Additions: 50, Deletions: 10, Files: 3},
			},
			messages: []Message{
				{
					Info: MessageInfo{Role: "user"},
					Parts: []MessagePart{
						{Type: "text", Text: "fix the bug"},
					},
				},
				{
					Info: MessageInfo{
						Role:   "assistant",
						Tokens: &MessageToken{Input: 8000, Output: 3000},
						Finish: "stop",
					},
					Parts: []MessagePart{
						{Type: "text", Text: "I found the bug."},
						{Type: "tool-invocation", Tool: "edit", State: &ToolState{Status: "completed"}},
					},
				},
			},
			want: OutcomeNormalCompletion,
		},
		{
			name: "error termination: assistant finish reason is error",
			session: Session{
				ID:   "sess-6",
				Time: SessionTime{Created: now.UnixMilli(), Updated: now.Add(10 * time.Second).UnixMilli()},
			},
			messages: []Message{
				{
					Info: MessageInfo{Role: "user"},
					Parts: []MessagePart{
						{Type: "text", Text: "do something"},
					},
				},
				{
					Info: MessageInfo{
						Role:   "assistant",
						Tokens: &MessageToken{Input: 500, Output: 50},
						Finish: "error",
					},
					Parts: []MessagePart{
						{Type: "text", Text: "An error occurred"},
					},
				},
			},
			want: OutcomeErrorTermination,
		},
		{
			name: "empty execution: assistant has text parts but all whitespace",
			session: Session{
				ID:   "sess-7",
				Time: SessionTime{Created: now.UnixMilli(), Updated: now.Add(3 * time.Second).UnixMilli()},
			},
			messages: []Message{
				{
					Info: MessageInfo{Role: "user"},
					Parts: []MessagePart{
						{Type: "text", Text: "do something"},
					},
				},
				{
					Info: MessageInfo{
						Role:   "assistant",
						Tokens: &MessageToken{Input: 500, Output: 5},
						Finish: "stop",
					},
					Parts: []MessagePart{
						{Type: "text", Text: "  \n  "},
					},
				},
			},
			want: OutcomeEmptyExecution,
		},
		{
			name: "empty execution: assistant message has no parts",
			session: Session{
				ID:   "sess-8",
				Time: SessionTime{Created: now.UnixMilli(), Updated: now.Add(2 * time.Second).UnixMilli()},
			},
			messages: []Message{
				{
					Info: MessageInfo{Role: "user"},
					Parts: []MessagePart{
						{Type: "text", Text: "hello"},
					},
				},
				{
					Info: MessageInfo{
						Role:   "assistant",
						Tokens: &MessageToken{Input: 200, Output: 0},
						Finish: "stop",
					},
					Parts: nil,
				},
			},
			want: OutcomeEmptyExecution,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyTerminalOutcome(tt.session, tt.messages)
			if got != tt.want {
				t.Errorf("ClassifyTerminalOutcome() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTerminalOutcomeString(t *testing.T) {
	tests := []struct {
		outcome TerminalOutcome
		want    string
	}{
		{OutcomeEmptyExecution, "empty-execution"},
		{OutcomeNormalCompletion, "normal-completion"},
		{OutcomeErrorTermination, "error-termination"},
		{TerminalOutcome("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.outcome), func(t *testing.T) {
			if got := tt.outcome.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTerminalOutcomeIsEmpty(t *testing.T) {
	if !OutcomeEmptyExecution.IsEmpty() {
		t.Error("OutcomeEmptyExecution.IsEmpty() should be true")
	}
	if OutcomeNormalCompletion.IsEmpty() {
		t.Error("OutcomeNormalCompletion.IsEmpty() should be false")
	}
	if OutcomeErrorTermination.IsEmpty() {
		t.Error("OutcomeErrorTermination.IsEmpty() should be false")
	}
}

func TestClassifyTerminalOutcomeDetail(t *testing.T) {
	now := time.Now()

	t.Run("empty execution detail includes reason", func(t *testing.T) {
		session := Session{
			ID:   "sess-1",
			Time: SessionTime{Created: now.UnixMilli(), Updated: now.Add(2 * time.Second).UnixMilli()},
		}
		detail := ClassifyTerminalOutcomeDetail(session, nil)

		if detail.Outcome != OutcomeEmptyExecution {
			t.Errorf("Outcome = %v, want %v", detail.Outcome, OutcomeEmptyExecution)
		}
		if detail.Reason == "" {
			t.Error("Reason should not be empty for empty execution")
		}
		if detail.AssistantMessages != 0 {
			t.Errorf("AssistantMessages = %d, want 0", detail.AssistantMessages)
		}
	})

	t.Run("normal completion detail includes token counts", func(t *testing.T) {
		session := Session{
			ID:      "sess-2",
			Time:    SessionTime{Created: now.UnixMilli(), Updated: now.Add(60 * time.Second).UnixMilli()},
			Summary: SessionSummary{Additions: 20, Files: 2},
		}
		messages := []Message{
			{
				Info:  MessageInfo{Role: "user"},
				Parts: []MessagePart{{Type: "text", Text: "build it"}},
			},
			{
				Info: MessageInfo{
					Role:   "assistant",
					Tokens: &MessageToken{Input: 5000, Output: 2000},
					Finish: "stop",
				},
				Parts: []MessagePart{
					{Type: "text", Text: "Done building."},
					{Type: "tool-invocation", Tool: "edit"},
				},
			},
		}

		detail := ClassifyTerminalOutcomeDetail(session, messages)

		if detail.Outcome != OutcomeNormalCompletion {
			t.Errorf("Outcome = %v, want %v", detail.Outcome, OutcomeNormalCompletion)
		}
		if detail.OutputTokens != 2000 {
			t.Errorf("OutputTokens = %d, want 2000", detail.OutputTokens)
		}
		if detail.AssistantMessages != 1 {
			t.Errorf("AssistantMessages = %d, want 1", detail.AssistantMessages)
		}
		if detail.ToolInvocations != 1 {
			t.Errorf("ToolInvocations = %d, want 1", detail.ToolInvocations)
		}
		if !detail.HasArtifacts {
			t.Error("HasArtifacts should be true when session has file changes")
		}
	})
}
