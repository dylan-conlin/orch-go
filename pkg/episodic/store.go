package episodic

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Store struct {
	Path string
}

type Filter struct {
	BeadsID string
	Limit   int
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".orch", "action-memory.jsonl")
	}
	return filepath.Join(home, ".orch", "action-memory.jsonl")
}

func NewStore(path string) *Store {
	if path == "" {
		path = DefaultPath()
	}
	return &Store{Path: path}
}

func (s *Store) Append(entry ActionMemory) error {
	return s.AppendMany([]ActionMemory{entry})
}

func (s *Store) AppendMany(entries []ActionMemory) error {
	if len(entries) == 0 {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(s.Path), 0755); err != nil {
		return fmt.Errorf("failed to create episodic store directory: %w", err)
	}

	f, err := os.OpenFile(s.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open episodic store: %w", err)
	}
	defer f.Close()

	for _, entry := range entries {
		line, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal episodic entry: %w", err)
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			return fmt.Errorf("failed to append episodic entry: %w", err)
		}
	}

	return nil
}

func (s *Store) Read(filter Filter) ([]ActionMemory, error) {
	entries, err := s.readRaw()
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}

	now := time.Now().UTC()
	active := make([]ActionMemory, 0, len(entries))
	for _, entry := range entries {
		if !entry.ExpiresAt.IsZero() && !now.Before(entry.ExpiresAt) {
			continue
		}
		if filter.BeadsID != "" && entry.BeadsID != filter.BeadsID {
			continue
		}
		active = append(active, entry)
	}

	if filter.Limit > 0 && len(active) > filter.Limit {
		active = active[:filter.Limit]
	}

	if err := s.rewriteActive(entries, now); err != nil {
		return active, err
	}

	return active, nil
}

func (s *Store) QueryByBeadsID(beadsID string, limit int) ([]ActionMemory, error) {
	entries, err := s.Read(Filter{BeadsID: beadsID})
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, nil
	}

	out := make([]ActionMemory, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		out = append(out, entries[i])
		if limit > 0 && len(out) >= limit {
			break
		}
	}

	return out, nil
}

func (s *Store) readRaw() ([]ActionMemory, error) {
	f, err := os.Open(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open episodic store: %w", err)
	}
	defer f.Close()

	entries := []ActionMemory{}
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 128*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry ActionMemory
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return entries, fmt.Errorf("failed reading episodic store: %w", err)
	}

	return entries, nil
}

func (s *Store) rewriteActive(entries []ActionMemory, now time.Time) error {
	kept := make([]ActionMemory, 0, len(entries))
	for _, entry := range entries {
		if !entry.ExpiresAt.IsZero() && !now.Before(entry.ExpiresAt) {
			continue
		}
		kept = append(kept, entry)
	}

	if err := os.MkdirAll(filepath.Dir(s.Path), 0755); err != nil {
		return fmt.Errorf("failed to create episodic store directory: %w", err)
	}

	tmp := s.Path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("failed to create temp episodic store: %w", err)
	}

	for _, entry := range kept {
		line, err := json.Marshal(entry)
		if err != nil {
			f.Close()
			os.Remove(tmp)
			return fmt.Errorf("failed to marshal episodic entry: %w", err)
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			f.Close()
			os.Remove(tmp)
			return fmt.Errorf("failed writing temp episodic store: %w", err)
		}
	}

	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to close temp episodic store: %w", err)
	}

	if err := os.Rename(tmp, s.Path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to swap episodic store: %w", err)
	}

	return nil
}
