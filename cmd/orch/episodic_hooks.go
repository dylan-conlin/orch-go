package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/activity"
	"github.com/dylan-conlin/orch-go/pkg/episodic"
	"github.com/dylan-conlin/orch-go/pkg/events"
)

func recordEpisodicEvent(event events.Event, ctx episodic.Context) {
	entry, err := episodic.ExtractEvent(event, ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to extract episodic event: %v\n", err)
		return
	}
	if entry == nil {
		return
	}

	store := episodic.NewStore("")
	if err := store.Append(*entry); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to persist episodic event: %v\n", err)
	}
}

func recordEpisodicActivity(parts []activity.MessagePartResponse, ctx episodic.Context) {
	entries, err := episodic.ExtractActivityParts(parts, ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to extract episodic activity: %v\n", err)
		return
	}
	if len(entries) == 0 {
		return
	}

	store := episodic.NewStore("")
	if err := store.AppendMany(entries); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to persist episodic activity: %v\n", err)
	}
}

func projectFromDir(dir string) string {
	trimmed := strings.TrimSpace(dir)
	if trimmed == "" {
		return "unknown"
	}
	base := filepath.Base(trimmed)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "unknown"
	}
	return base
}

func projectFromCWD() string {
	wd, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return projectFromDir(wd)
}
