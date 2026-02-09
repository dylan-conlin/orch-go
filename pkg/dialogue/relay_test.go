package dialogue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

func TestRunRelayStopsOnApprovedVerdict(t *testing.T) {
	questioner := &fakeQuestioner{
		responses: []CompletionResponse{
			{Text: "What currently causes retry storms?", Usage: Usage{InputTokens: 10, OutputTokens: 4}},
			{Text: "Would centralizing retry policy remove duplication? [VERDICT: CONTINUE]", Usage: Usage{InputTokens: 12, OutputTokens: 7}},
			{Text: "[VERDICT: APPROVED]\nAdopt a single retry coordinator and delete local retry wrappers.", Usage: Usage{InputTokens: 8, OutputTokens: 11}},
		},
	}

	expert := newFakeExpert([]string{
		"Retries come from three services with slightly different backoff logic.",
		"Yes. A shared policy point would remove drift and improve observability.",
	})

	result, err := RunRelay(context.Background(), questioner, expert, "ses_test", RelayConfig{
		Topic:           "retry architecture",
		ExploreTurns:    1,
		ConvergeTurns:   2,
		MaxTurns:        5,
		PollInterval:    1 * time.Millisecond,
		ResponseTimeout: 50 * time.Millisecond,
	}, nil)
	if err != nil {
		t.Fatalf("RunRelay() error = %v", err)
	}

	if !result.Approved {
		t.Fatalf("Approved = %v, want true", result.Approved)
	}
	if result.EndReason != "ghost_approved" {
		t.Fatalf("EndReason = %q, want %q", result.EndReason, "ghost_approved")
	}
	if len(result.Turns) != 3 {
		t.Fatalf("len(Turns) = %d, want 3", len(result.Turns))
	}
	if result.Turns[0].Phase != PhaseExplore {
		t.Fatalf("turn 1 phase = %q, want %q", result.Turns[0].Phase, PhaseExplore)
	}
	if result.Turns[1].Phase != PhaseConverge {
		t.Fatalf("turn 2 phase = %q, want %q", result.Turns[1].Phase, PhaseConverge)
	}
	if result.Turns[2].Phase != PhaseTerminate {
		t.Fatalf("turn 3 phase = %q, want %q", result.Turns[2].Phase, PhaseTerminate)
	}
	if got := expert.sendCount; got != 2 {
		t.Fatalf("expert send count = %d, want 2", got)
	}
}

func TestRunRelayEndsAtMaxTurns(t *testing.T) {
	questioner := &fakeQuestioner{
		responses: []CompletionResponse{
			{Text: "Question 1"},
			{Text: "Question 2"},
			{Text: "Question 3"},
		},
	}

	expert := newFakeExpert([]string{"A1", "A2", "A3"})

	result, err := RunRelay(context.Background(), questioner, expert, "ses_test", RelayConfig{
		Topic:           "topic",
		ExploreTurns:    1,
		ConvergeTurns:   2,
		MaxTurns:        3,
		PollInterval:    1 * time.Millisecond,
		ResponseTimeout: 50 * time.Millisecond,
	}, nil)
	if err != nil {
		t.Fatalf("RunRelay() error = %v", err)
	}

	if result.Approved {
		t.Fatalf("Approved = %v, want false", result.Approved)
	}
	if result.EndReason != "max_turns" {
		t.Fatalf("EndReason = %q, want %q", result.EndReason, "max_turns")
	}
	if len(result.Turns) != 3 {
		t.Fatalf("len(Turns) = %d, want 3", len(result.Turns))
	}
}

func TestParseVerdict(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		wantOK   bool
		approved bool
	}{
		{name: "approved lowercase", input: "[verdict: approved]", want: "APPROVED", wantOK: true, approved: true},
		{name: "continue", input: "something [VERDICT: continue] else", want: "CONTINUE", wantOK: true, approved: false},
		{name: "spacing", input: "[ VERDICT : needs revision ]", want: "NEEDS_REVISION", wantOK: true, approved: false},
		{name: "missing", input: "no verdict token", want: "", wantOK: false, approved: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseVerdict(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("verdict = %q, want %q", got, tt.want)
			}
			if IsApprovedVerdict(got) != tt.approved {
				t.Fatalf("IsApprovedVerdict(%q) = %v, want %v", got, IsApprovedVerdict(got), tt.approved)
			}
		})
	}
}

type fakeQuestioner struct {
	responses []CompletionResponse
	idx       int
}

func (f *fakeQuestioner) Complete(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
	if f.idx >= len(f.responses) {
		return nil, fmt.Errorf("unexpected call %d", f.idx+1)
	}
	resp := f.responses[f.idx]
	f.idx++
	return &resp, nil
}

type fakeExpert struct {
	messages   []opencode.Message
	responses  []string
	sendCount  int
	messageSeq int
	assistSeq  int
}

func newFakeExpert(responses []string) *fakeExpert {
	return &fakeExpert{responses: responses}
}

func (f *fakeExpert) SendMessageAsync(_ string, content, _ string) error {
	f.sendCount++
	f.messageSeq++
	f.messages = append(f.messages, opencode.Message{
		Info:  opencode.MessageInfo{ID: fmt.Sprintf("u-%d", f.messageSeq), Role: "user", Time: opencode.MessageTime{Created: time.Now().UnixMilli()}},
		Parts: []opencode.MessagePart{{Type: "text", Text: content}},
	})

	if len(f.responses) == 0 {
		return nil
	}
	resp := f.responses[0]
	f.responses = f.responses[1:]
	f.assistSeq++
	f.messages = append(f.messages, opencode.Message{
		Info: opencode.MessageInfo{
			ID:     fmt.Sprintf("a-%d", f.assistSeq),
			Role:   "assistant",
			Finish: "stop",
			Time: opencode.MessageTime{
				Created:   time.Now().UnixMilli(),
				Completed: time.Now().UnixMilli(),
			},
		},
		Parts: []opencode.MessagePart{{Type: "text", Text: resp}},
	})
	return nil
}

func (f *fakeExpert) GetMessages(_ string) ([]opencode.Message, error) {
	clone := make([]opencode.Message, len(f.messages))
	copy(clone, f.messages)
	return clone, nil
}
