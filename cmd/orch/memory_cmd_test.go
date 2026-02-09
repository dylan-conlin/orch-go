package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/episodic"
)

func TestInspectEpisodesValidationStates(t *testing.T) {
	temp := t.TempDir()
	home := filepath.Join(temp, "home")
	project := filepath.Join(temp, "orch-go")
	if err := os.MkdirAll(filepath.Join(home, ".orch"), 0755); err != nil {
		t.Fatalf("mkdir home orch: %v", err)
	}
	if err := os.MkdirAll(project, 0755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	t.Setenv("HOME", home)

	eventsPath := filepath.Join(home, ".orch", "events.jsonl")
	eventsContent := []byte("{\"type\":\"session.spawned\"}\n")
	if err := os.WriteFile(eventsPath, eventsContent, 0644); err != nil {
		t.Fatalf("write events: %v", err)
	}

	storePath := filepath.Join(home, ".orch", "action-memory.jsonl")
	if err := appendEpisode(storePath, episodic.ActionMemory{
		Project: "orch-go",
		BeadsID: "orch-go-8100",
		Action:  episodic.Action{Name: "spawn"},
		Outcome: episodic.Outcome{Status: episodic.OutcomeSuccess, Summary: "Spawn succeeded."},
		Evidence: episodic.Evidence{
			Kind:      episodic.EvidenceKindEventsJSONL,
			Pointer:   "~/.orch/events.jsonl#offset=0",
			Hash:      hashValue(eventsContent),
			Timestamp: time.Now().UTC().Unix(),
		},
		Confidence: 0.95,
		ExpiresAt:  time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("append accepted episode: %v", err)
	}

	if err := appendEpisode(storePath, episodic.ActionMemory{
		Project: "orch-go",
		BeadsID: "orch-go-8100",
		Action:  episodic.Action{Name: "note"},
		Outcome: episodic.Outcome{Status: episodic.OutcomeObserved, Summary: "Weak signal."},
		Evidence: episodic.Evidence{
			Kind:    episodic.EvidenceKindEventsJSONL,
			Pointer: "~/.orch/events.jsonl#offset=0",
			Hash:    hashValue(eventsContent),
		},
		Confidence: 0.2,
		ExpiresAt:  time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("append rejected episode: %v", err)
	}

	rows, err := inspectEpisodes("orch-go-8100", 20, project)
	if err != nil {
		t.Fatalf("inspectEpisodes failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	accepted := 0
	rejected := 0
	for _, row := range rows {
		if row.State == episodic.ValidationStateAccepted {
			accepted++
		}
		if row.State == episodic.ValidationStateRejected {
			rejected++
		}
	}

	if accepted != 1 || rejected != 1 {
		t.Fatalf("expected 1 accepted and 1 rejected, got accepted=%d rejected=%d", accepted, rejected)
	}
}

func appendEpisode(path string, ep episodic.ActionMemory) error {
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

func hashValue(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
