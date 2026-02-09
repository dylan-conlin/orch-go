package spawn

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/episodic"
)

func TestGenerateContextInjectsOnlyAcceptedEpisodes(t *testing.T) {
	temp := t.TempDir()
	home := filepath.Join(temp, "home")
	if err := os.MkdirAll(filepath.Join(home, ".orch"), 0755); err != nil {
		t.Fatalf("mkdir home orch: %v", err)
	}
	t.Setenv("HOME", home)

	eventsPath := filepath.Join(home, ".orch", "events.jsonl")
	eventsContent := []byte("{\"type\":\"session.spawned\"}\n")
	if err := os.WriteFile(eventsPath, eventsContent, 0644); err != nil {
		t.Fatalf("write events: %v", err)
	}

	storePath := filepath.Join(home, ".orch", "action-memory.jsonl")
	if err := writeEpisodeLine(storePath, episodic.Episode{
		Project: "orch-go",
		BeadsID: "orch-go-7777",
		Action:  episodic.Action{Name: "spawn"},
		Outcome: episodic.Outcome{Status: "ok", Summary: "Spawned worker successfully."},
		Evidence: episodic.Evidence{
			Kind:      "events_jsonl",
			Pointer:   "~/.orch/events.jsonl#offset=0",
			Hash:      hashBytes(eventsContent),
			Timestamp: time.Now().UTC().Unix(),
		},
		Confidence: 0.9,
		ExpiresAt:  time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("write accepted episode: %v", err)
	}

	if err := writeEpisodeLine(storePath, episodic.Episode{
		Project: "orch-go",
		BeadsID: "orch-go-7777",
		Action:  episodic.Action{Name: "note"},
		Outcome: episodic.Outcome{Status: "ok", Summary: "Low confidence note."},
		Evidence: episodic.Evidence{
			Kind:    "events_jsonl",
			Pointer: "~/.orch/events.jsonl#offset=0",
			Hash:    hashBytes(eventsContent),
		},
		Confidence: 0.3,
		ExpiresAt:  time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("write rejected episode: %v", err)
	}

	cfg := &Config{
		Task:          "test episodic injection",
		Project:       "orch-go",
		ProjectDir:    temp,
		WorkspaceName: "og-feat-test-episodic-09feb",
		BeadsID:       "orch-go-7777",
		SkillName:     "feature-impl",
		Tier:          TierLight,
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		t.Fatalf("GenerateContext failed: %v", err)
	}

	if !strings.Contains(content, "## RECENT VALIDATED EPISODES") {
		t.Fatalf("expected episodic section in context")
	}
	if !strings.Contains(content, "Spawned worker successfully") {
		t.Fatalf("expected accepted episode summary in context")
	}
	if strings.Contains(content, "Low confidence note") {
		t.Fatalf("did not expect rejected episode summary in context")
	}
}

func writeEpisodeLine(path string, ep episodic.Episode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	line, err := json.Marshal(ep)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(append(line, '\n'))
	return err
}

func hashBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
