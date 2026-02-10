package events

import "testing"

func TestNormalizeAbandonmentReasonCode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "canonical", in: "stalled", want: AbandonmentReasonStalled},
		{name: "alias underscore", in: "context_exhaustion", want: AbandonmentReasonContextExhausted},
		{name: "alias mixed case", in: "Rate_Limit", want: AbandonmentReasonRateLimit},
		{name: "invalid", in: "foobar", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeAbandonmentReasonCode(tt.in)
			if got != tt.want {
				t.Fatalf("NormalizeAbandonmentReasonCode(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestInferAbandonmentReasonCode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "stalled", in: "Agent got stuck in loop", want: AbandonmentReasonStalled},
		{name: "infra", in: "OpenCode server crash while running", want: AbandonmentReasonInfra},
		{name: "rate limit", in: "Hit 429 rate limit", want: AbandonmentReasonRateLimit},
		{name: "scope mismatch", in: "Out of scope for this issue", want: AbandonmentReasonScopeMismatch},
		{name: "manual cleanup", in: "Manual cleanup after duplicate spawn", want: AbandonmentReasonManualCleanup},
		{name: "context exhaustion", in: "Context window exhausted", want: AbandonmentReasonContextExhausted},
		{name: "no match", in: "Needs teammate review", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferAbandonmentReasonCode(tt.in)
			if got != tt.want {
				t.Fatalf("InferAbandonmentReasonCode(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
