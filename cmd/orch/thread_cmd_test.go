package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThreadsDir_DefaultUsesCurrentDir(t *testing.T) {
	// Save and restore
	origWorkdir := threadWorkdir
	defer func() { threadWorkdir = origWorkdir }()

	threadWorkdir = ""
	dir, err := threadsDir()
	if err != nil {
		t.Fatalf("threadsDir() failed: %v", err)
	}

	cwd, _ := os.Getwd()
	expected := filepath.Join(cwd, ".kb", "threads")
	if dir != expected {
		t.Errorf("threadsDir() = %q, want %q", dir, expected)
	}
}

func TestThreadsDir_WorkdirOverride(t *testing.T) {
	origWorkdir := threadWorkdir
	defer func() { threadWorkdir = origWorkdir }()

	tmpDir := t.TempDir()
	threadWorkdir = tmpDir

	dir, err := threadsDir()
	if err != nil {
		t.Fatalf("threadsDir() failed: %v", err)
	}

	expected := filepath.Join(tmpDir, ".kb", "threads")
	if dir != expected {
		t.Errorf("threadsDir() = %q, want %q", dir, expected)
	}
}

func TestThreadsDir_WorkdirNotExist(t *testing.T) {
	origWorkdir := threadWorkdir
	defer func() { threadWorkdir = origWorkdir }()

	threadWorkdir = "/nonexistent/path/that/does/not/exist"

	_, err := threadsDir()
	if err == nil {
		t.Fatal("expected error for nonexistent workdir")
	}
	if !strings.Contains(err.Error(), "workdir does not exist") {
		t.Errorf("expected 'workdir does not exist' error, got: %v", err)
	}
}

func TestThreadsDir_WorkdirIsFile(t *testing.T) {
	origWorkdir := threadWorkdir
	defer func() { threadWorkdir = origWorkdir }()

	tmpFile := filepath.Join(t.TempDir(), "not-a-dir")
	os.WriteFile(tmpFile, []byte("hello"), 0644)

	threadWorkdir = tmpFile

	_, err := threadsDir()
	if err == nil {
		t.Fatal("expected error when workdir is a file")
	}
	if !strings.Contains(err.Error(), "workdir is not a directory") {
		t.Errorf("expected 'workdir is not a directory' error, got: %v", err)
	}
}
