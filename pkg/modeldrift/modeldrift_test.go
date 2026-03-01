package modeldrift

import (
	"fmt"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// mockStore implements Store for tests.
type mockStore struct {
	ReadStalenessEventsFunc func(path string) ([]spawn.StalenessEvent, error)
	LoadMetadataFunc        func(modelPath string) (Metadata, error)
	CountCommitsFunc        func(projectDir, lastUpdated string, files []string) (int, error)
	CreateIssueFunc         func(args IssueCreateArgs) (string, error)
}

func (m *mockStore) ReadStalenessEvents(path string) ([]spawn.StalenessEvent, error) {
	if m.ReadStalenessEventsFunc != nil {
		return m.ReadStalenessEventsFunc(path)
	}
	return nil, nil
}

func (m *mockStore) LoadMetadata(modelPath string) (Metadata, error) {
	if m.LoadMetadataFunc != nil {
		return m.LoadMetadataFunc(modelPath)
	}
	return Metadata{}, nil
}

func (m *mockStore) CountCommits(projectDir, lastUpdated string, files []string) (int, error) {
	if m.CountCommitsFunc != nil {
		return m.CountCommitsFunc(projectDir, lastUpdated, files)
	}
	return 0, nil
}

func (m *mockStore) CreateIssue(args IssueCreateArgs) (string, error) {
	if m.CreateIssueFunc != nil {
		return m.CreateIssueFunc(args)
	}
	return "mock-issue-1", nil
}

// mockQuerier implements IssueQuerier for tests.
type mockQuerier struct {
	ListIssuesWithLabelFunc func(label string) ([]Issue, error)
}

func (m *mockQuerier) ListIssuesWithLabel(label string) ([]Issue, error) {
	if m.ListIssuesWithLabelFunc != nil {
		return m.ListIssuesWithLabelFunc(label)
	}
	return nil, nil
}

func TestAnalyze_NoEvents(t *testing.T) {
	store := &mockStore{
		ReadStalenessEventsFunc: func(path string) ([]spawn.StalenessEvent, error) {
			return []spawn.StalenessEvent{}, nil
		},
	}
	querier := &mockQuerier{}

	result, err := Analyze(store, querier)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "Model drift reflection: no staleness events" {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

func TestAnalyze_BelowThreshold(t *testing.T) {
	store := &mockStore{
		ReadStalenessEventsFunc: func(path string) ([]spawn.StalenessEvent, error) {
			return []spawn.StalenessEvent{
				{Model: "/path/to/.kb/models/foo/model.md", ChangedFiles: []string{"a.go"}},
				{Model: "/path/to/.kb/models/foo/model.md", ChangedFiles: []string{"a.go"}},
			}, nil
		},
	}
	querier := &mockQuerier{}

	result, err := Analyze(store, querier)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "Model drift reflection: no models exceeded threshold" {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

func TestAnalyze_CreatesIssue(t *testing.T) {
	store := &mockStore{
		ReadStalenessEventsFunc: func(path string) ([]spawn.StalenessEvent, error) {
			events := make([]spawn.StalenessEvent, SpawnThreshold)
			for i := range events {
				events[i] = spawn.StalenessEvent{
					Model:        "/home/user/project/.kb/models/test-domain/model.md",
					ChangedFiles: []string{"pkg/foo.go"},
				}
			}
			return events, nil
		},
		LoadMetadataFunc: func(modelPath string) (Metadata, error) {
			return Metadata{
				ModelPath:   modelPath,
				Domain:      "test-domain",
				DomainKey:   "test-domain",
				LastUpdated: "2026-01-01",
				ProjectDir:  "/home/user/project",
			}, nil
		},
		CountCommitsFunc: func(projectDir, lastUpdated string, files []string) (int, error) {
			return 3, nil
		},
		CreateIssueFunc: func(args IssueCreateArgs) (string, error) {
			if args.Title != "Model drift: test-domain" {
				t.Errorf("unexpected title: %s", args.Title)
			}
			return "test-issue-1", nil
		},
	}
	querier := &mockQuerier{}

	result, err := Analyze(store, querier)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Created) != 1 {
		t.Fatalf("expected 1 created, got %d", len(result.Created))
	}
	if result.Created[0] != "test-issue-1" {
		t.Errorf("unexpected issue ID: %s", result.Created[0])
	}
}

func TestAnalyze_SkipsExistingModel(t *testing.T) {
	store := &mockStore{
		ReadStalenessEventsFunc: func(path string) ([]spawn.StalenessEvent, error) {
			events := make([]spawn.StalenessEvent, SpawnThreshold)
			for i := range events {
				events[i] = spawn.StalenessEvent{
					Model:        "/home/user/project/.kb/models/test-domain/model.md",
					ChangedFiles: []string{"pkg/foo.go"},
				}
			}
			return events, nil
		},
	}
	querier := &mockQuerier{
		ListIssuesWithLabelFunc: func(label string) ([]Issue, error) {
			return []Issue{
				{Title: "Model drift: test-domain", Description: "path .kb/models/test-domain/model.md"},
			}, nil
		},
	}

	result, err := Analyze(store, querier)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// All candidates are tracked by existing issues, so no groups are created.
	// The "no new models to file" early return doesn't include skipped count.
	if result.Message != "Model drift reflection: no new models to file" {
		t.Errorf("expected 'no new models to file' message, got: %s", result.Message)
	}
	if len(result.Created) != 0 {
		t.Errorf("expected 0 created, got %d", len(result.Created))
	}
}

func TestAnalyze_CircuitBreaker(t *testing.T) {
	store := &mockStore{
		ReadStalenessEventsFunc: func(path string) ([]spawn.StalenessEvent, error) {
			events := make([]spawn.StalenessEvent, SpawnThreshold)
			for i := range events {
				events[i] = spawn.StalenessEvent{
					Model:        "/path/.kb/models/foo/model.md",
					ChangedFiles: []string{"a.go"},
				}
			}
			return events, nil
		},
	}
	querier := &mockQuerier{
		ListIssuesWithLabelFunc: func(label string) ([]Issue, error) {
			issues := make([]Issue, CircuitBreakerLimit)
			for i := range issues {
				issues[i] = Issue{Title: fmt.Sprintf("Model drift: domain-%d", i)}
			}
			return issues, nil
		},
	}

	result, err := Analyze(store, querier)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message == "" {
		t.Fatal("expected circuit breaker message")
	}
	if len(result.Created) != 0 {
		t.Errorf("expected 0 created with circuit breaker, got %d", len(result.Created))
	}
}

func TestAnalyze_Backpressure(t *testing.T) {
	store := &mockStore{
		ReadStalenessEventsFunc: func(path string) ([]spawn.StalenessEvent, error) {
			events := make([]spawn.StalenessEvent, SpawnThreshold)
			for i := range events {
				events[i] = spawn.StalenessEvent{
					Model:        "/path/.kb/models/foo/model.md",
					ChangedFiles: []string{"a.go"},
				}
			}
			return events, nil
		},
	}
	querier := &mockQuerier{
		ListIssuesWithLabelFunc: func(label string) ([]Issue, error) {
			issues := make([]Issue, BackpressureLimit)
			for i := range issues {
				issues[i] = Issue{Title: fmt.Sprintf("Model drift: domain-%d", i)}
			}
			return issues, nil
		},
	}

	result, err := Analyze(store, querier)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message == "" {
		t.Fatal("expected backpressure message")
	}
	if len(result.Created) != 0 {
		t.Errorf("expected 0 created with backpressure, got %d", len(result.Created))
	}
}

func TestNormalizeDomainKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Test Domain", "test-domain"},
		{"foo_bar", "foo-bar"},
		{"FOO/BAR", "foo-bar"},
		{"hello--world", "hello--world"},
		{"  spaces  ", "spaces"},
		{"special!@#chars", "specialchars"},
	}
	for _, tc := range tests {
		got := NormalizeDomainKey(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeDomainKey(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestDeriveDomainFromPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/home/user/.kb/models/agent-lifecycle/model.md", "model"},
		{"/home/user/.kb/models/spawn-architecture/model.md", "model"},
		{"model.md", "model"},
	}
	for _, tc := range tests {
		got := DeriveDomainFromPath(tc.input)
		if got != tc.want {
			t.Errorf("DeriveDomainFromPath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestProjectDirFromModelPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/home/user/project/.kb/models/foo/model.md", "/home/user/project"},
		{"/just/a/file.md", "/just/a"},
	}
	for _, tc := range tests {
		got := ProjectDirFromModelPath(tc.input)
		if got != tc.want {
			t.Errorf("ProjectDirFromModelPath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestPriority(t *testing.T) {
	if p := priority(candidate{Deleted: []string{"a.go"}}); p != 2 {
		t.Errorf("priority with deleted files = %d, want 2", p)
	}
	if p := priority(candidate{CommitCount: 5}); p != 2 {
		t.Errorf("priority with 5 commits = %d, want 2", p)
	}
	if p := priority(candidate{CommitCount: 2}); p != 3 {
		t.Errorf("priority with 2 commits = %d, want 3", p)
	}
}

func TestMapKeysSorted(t *testing.T) {
	input := map[string]struct{}{"c": {}, "a": {}, "b": {}}
	got := mapKeysSorted(input)
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("mapKeysSorted = %v, want [a b c]", got)
	}

	empty := mapKeysSorted(nil)
	if len(empty) != 0 {
		t.Errorf("mapKeysSorted(nil) = %v, want []", empty)
	}
}

func TestFormatLimitedList(t *testing.T) {
	if got := formatLimitedList([]string{"a", "b", "c"}, 5); got != "a, b, c" {
		t.Errorf("formatLimitedList under limit = %q", got)
	}
	if got := formatLimitedList([]string{"a", "b", "c", "d"}, 2); got != "a, b, +2 more" {
		t.Errorf("formatLimitedList over limit = %q", got)
	}
	if got := formatLimitedList(nil, 5); got != "" {
		t.Errorf("formatLimitedList(nil) = %q", got)
	}
}

func TestExtractModelField(t *testing.T) {
	content := "**Domain:** Agent Lifecycle\n**Last Updated:** 2026-01-15\nOther: stuff\n"
	if got := extractModelField(content, "Domain"); got != "Agent Lifecycle" {
		t.Errorf("extractModelField Domain = %q", got)
	}
	if got := extractModelField(content, "Last Updated"); got != "2026-01-15" {
		t.Errorf("extractModelField Last Updated = %q", got)
	}
	if got := extractModelField(content, "Missing"); got != "" {
		t.Errorf("extractModelField Missing = %q, want empty", got)
	}
}

func TestDomainKeyFromIssue(t *testing.T) {
	if got := domainKeyFromIssue(Issue{Title: "Model drift: Agent Lifecycle"}); got != "agent-lifecycle" {
		t.Errorf("domainKeyFromIssue = %q, want agent-lifecycle", got)
	}
	if got := domainKeyFromIssue(Issue{Title: "Unrelated issue"}); got != "" {
		t.Errorf("domainKeyFromIssue non-drift = %q, want empty", got)
	}
}

func TestModelKeyVariants(t *testing.T) {
	variants := modelKeyVariants("/home/user/.kb/models/foo/model.md")
	found := false
	for _, v := range variants {
		if v == "model.md" || v == "model" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected base name variant in %v", variants)
	}
}
