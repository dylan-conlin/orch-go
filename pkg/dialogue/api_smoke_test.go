package dialogue

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDialogueClientSmoke(t *testing.T) {
	if os.Getenv("ORCH_DIALOGUE_SMOKE") != "1" {
		t.Skip("set ORCH_DIALOGUE_SMOKE=1 to run live Anthropic Messages API smoke test")
	}

	client, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	resp, err := client.Complete(ctx, CompletionRequest{
		MaxTokens: 32,
		Messages: []Message{{
			Role:    "user",
			Content: "Reply with exactly: ok",
		}},
	})
	if err != nil {
		t.Fatalf("Complete() error: %v", err)
	}

	if strings.TrimSpace(resp.Text) != "ok" {
		t.Fatalf("resp.Text = %q, want %q", resp.Text, "ok")
	}
}
