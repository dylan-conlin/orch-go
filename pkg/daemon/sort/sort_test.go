package sort

import (
	"testing"
)

func TestGet_Priority(t *testing.T) {
	s, err := Get("priority")
	if err != nil {
		t.Fatalf("Get('priority') returned error: %v", err)
	}
	if s.Name() != "priority" {
		t.Errorf("expected name 'priority', got %q", s.Name())
	}
}

func TestGet_EmptyStringDefaultsToPriority(t *testing.T) {
	s, err := Get("")
	if err != nil {
		t.Fatalf("Get('') returned error: %v", err)
	}
	if s.Name() != "priority" {
		t.Errorf("expected name 'priority', got %q", s.Name())
	}
}

func TestGet_Unblock(t *testing.T) {
	s, err := Get("unblock")
	if err != nil {
		t.Fatalf("Get('unblock') returned error: %v", err)
	}
	if s.Name() != "unblock" {
		t.Errorf("expected name 'unblock', got %q", s.Name())
	}
}

func TestGet_CaseInsensitive(t *testing.T) {
	s, err := Get("UNBLOCK")
	if err != nil {
		t.Fatalf("Get('UNBLOCK') returned error: %v", err)
	}
	if s.Name() != "unblock" {
		t.Errorf("expected name 'unblock', got %q", s.Name())
	}
}

func TestGet_UnknownMode(t *testing.T) {
	_, err := Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown mode, got nil")
	}
}

func TestModes(t *testing.T) {
	modes := Modes()
	if len(modes) < 2 {
		t.Fatalf("expected at least 2 modes, got %d", len(modes))
	}
	found := map[string]bool{}
	for _, m := range modes {
		found[m] = true
	}
	if !found["priority"] {
		t.Error("missing 'priority' mode")
	}
	if !found["unblock"] {
		t.Error("missing 'unblock' mode")
	}
}

// --- Priority Strategy Tests ---

func TestPriorityStrategy_Sort_Empty(t *testing.T) {
	s := &PriorityStrategy{}
	result := s.Sort(nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d", len(result))
	}
}

func TestPriorityStrategy_Sort_OrdersByPriority(t *testing.T) {
	s := &PriorityStrategy{}
	issues := []Issue{
		{ID: "c", Priority: 2},
		{ID: "a", Priority: 0},
		{ID: "b", Priority: 1},
	}

	result := s.Sort(issues, nil)

	if result[0].ID != "a" || result[1].ID != "b" || result[2].ID != "c" {
		t.Errorf("expected order [a, b, c], got [%s, %s, %s]",
			result[0].ID, result[1].ID, result[2].ID)
	}
}

func TestPriorityStrategy_Sort_StableForEqualPriority(t *testing.T) {
	s := &PriorityStrategy{}
	issues := []Issue{
		{ID: "first", Priority: 1},
		{ID: "second", Priority: 1},
		{ID: "third", Priority: 1},
	}

	result := s.Sort(issues, nil)

	// Stable sort should preserve original order for equal priorities
	if result[0].ID != "first" || result[1].ID != "second" || result[2].ID != "third" {
		t.Errorf("expected stable order [first, second, third], got [%s, %s, %s]",
			result[0].ID, result[1].ID, result[2].ID)
	}
}

func TestPriorityStrategy_Sort_DoesNotModifyInput(t *testing.T) {
	s := &PriorityStrategy{}
	issues := []Issue{
		{ID: "c", Priority: 2},
		{ID: "a", Priority: 0},
	}

	_ = s.Sort(issues, nil)

	// Original slice should be unmodified
	if issues[0].ID != "c" {
		t.Error("sort modified the input slice")
	}
}

func TestPriorityStrategy_Sort_IgnoresContext(t *testing.T) {
	s := &PriorityStrategy{}
	issues := []Issue{
		{ID: "b", Priority: 1},
		{ID: "a", Priority: 0},
	}
	ctx := &SortContext{
		Leverage: map[string]*LeverageInfo{
			"b": {TotalLeverage: 100},
			"a": {TotalLeverage: 0},
		},
	}

	result := s.Sort(issues, ctx)

	// Priority strategy ignores leverage — a (P0) should still be first
	if result[0].ID != "a" {
		t.Errorf("expected 'a' first (P0), got %q", result[0].ID)
	}
}

// --- Unblock Strategy Tests ---

func TestUnblockStrategy_Sort_Empty(t *testing.T) {
	s := &UnblockStrategy{}
	result := s.Sort(nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d", len(result))
	}
}

func TestUnblockStrategy_Sort_FallsToPriorityWithoutContext(t *testing.T) {
	s := &UnblockStrategy{}
	issues := []Issue{
		{ID: "c", Priority: 2},
		{ID: "a", Priority: 0},
		{ID: "b", Priority: 1},
	}

	// No context — should fall back to priority sort
	result := s.Sort(issues, nil)

	if result[0].ID != "a" || result[1].ID != "b" || result[2].ID != "c" {
		t.Errorf("expected priority fallback order [a, b, c], got [%s, %s, %s]",
			result[0].ID, result[1].ID, result[2].ID)
	}
}

func TestUnblockStrategy_Sort_FallsToPriorityWithEmptyLeverage(t *testing.T) {
	s := &UnblockStrategy{}
	issues := []Issue{
		{ID: "c", Priority: 2},
		{ID: "a", Priority: 0},
	}
	ctx := &SortContext{
		Leverage: map[string]*LeverageInfo{},
	}

	result := s.Sort(issues, ctx)

	if result[0].ID != "a" {
		t.Errorf("expected priority fallback, got %q first", result[0].ID)
	}
}

func TestUnblockStrategy_Sort_HigherLeverageFirst(t *testing.T) {
	s := &UnblockStrategy{}
	issues := []Issue{
		{ID: "low", Priority: 0, IssueType: "task"},
		{ID: "high", Priority: 2, IssueType: "task"},
	}
	ctx := &SortContext{
		Leverage: map[string]*LeverageInfo{
			"low":  {TotalLeverage: 1},
			"high": {TotalLeverage: 5},
		},
	}

	result := s.Sort(issues, ctx)

	// "high" has more leverage, should come first despite higher priority number
	if result[0].ID != "high" {
		t.Errorf("expected 'high' first (leverage 5), got %q", result[0].ID)
	}
}

func TestUnblockStrategy_Sort_AuthorityBreaksTies(t *testing.T) {
	s := &UnblockStrategy{}
	issues := []Issue{
		{ID: "feature", Priority: 1, IssueType: "feature"},   // authority=1
		{ID: "task", Priority: 1, IssueType: "task"},         // authority=0
		{ID: "question", Priority: 1, IssueType: "question"}, // authority=2
	}
	ctx := &SortContext{
		Leverage: map[string]*LeverageInfo{
			"feature":  {TotalLeverage: 3},
			"task":     {TotalLeverage: 3},
			"question": {TotalLeverage: 3},
		},
	}

	result := s.Sort(issues, ctx)

	// Same leverage, so authority breaks tie: task (0) < feature (1) < question (2)
	if result[0].ID != "task" {
		t.Errorf("expected 'task' first (authority 0), got %q", result[0].ID)
	}
	if result[1].ID != "feature" {
		t.Errorf("expected 'feature' second (authority 1), got %q", result[1].ID)
	}
	if result[2].ID != "question" {
		t.Errorf("expected 'question' third (authority 2), got %q", result[2].ID)
	}
}

func TestUnblockStrategy_Sort_PriorityTiesBreaker(t *testing.T) {
	s := &UnblockStrategy{}
	issues := []Issue{
		{ID: "p2", Priority: 2, IssueType: "task"},
		{ID: "p0", Priority: 0, IssueType: "task"},
	}
	ctx := &SortContext{
		Leverage: map[string]*LeverageInfo{
			"p2": {TotalLeverage: 3},
			"p0": {TotalLeverage: 3},
		},
	}

	result := s.Sort(issues, ctx)

	// Same leverage, same authority, priority breaks tie
	if result[0].ID != "p0" {
		t.Errorf("expected 'p0' first (P0), got %q", result[0].ID)
	}
}

func TestUnblockStrategy_Sort_MissingLeverageGetNeutralScore(t *testing.T) {
	s := &UnblockStrategy{}
	issues := []Issue{
		{ID: "has-leverage", Priority: 1, IssueType: "task"},
		{ID: "no-leverage", Priority: 0, IssueType: "task"},
	}
	ctx := &SortContext{
		Leverage: map[string]*LeverageInfo{
			"has-leverage": {TotalLeverage: 2},
			// "no-leverage" is missing — gets 0
		},
	}

	result := s.Sort(issues, ctx)

	// "has-leverage" has leverage 2, "no-leverage" has 0 — leverage wins
	if result[0].ID != "has-leverage" {
		t.Errorf("expected 'has-leverage' first (leverage 2), got %q", result[0].ID)
	}
}

func TestUnblockStrategy_Sort_FactualQuestionsDaemonTraversable(t *testing.T) {
	s := &UnblockStrategy{}
	issues := []Issue{
		{ID: "judgment", Priority: 1, IssueType: "question", Labels: []string{"subtype:judgment"}},
		{ID: "factual", Priority: 1, IssueType: "question", Labels: []string{"subtype:factual"}},
	}
	ctx := &SortContext{
		Leverage: map[string]*LeverageInfo{
			"judgment": {TotalLeverage: 3},
			"factual":  {TotalLeverage: 3},
		},
	}

	result := s.Sort(issues, ctx)

	// Same leverage, but factual has authority 0 vs judgment authority 2
	if result[0].ID != "factual" {
		t.Errorf("expected 'factual' first (daemon-traversable), got %q", result[0].ID)
	}
}

func TestUnblockStrategy_Sort_DoesNotModifyInput(t *testing.T) {
	s := &UnblockStrategy{}
	issues := []Issue{
		{ID: "b", Priority: 1, IssueType: "task"},
		{ID: "a", Priority: 0, IssueType: "task"},
	}

	_ = s.Sort(issues, nil)

	if issues[0].ID != "b" {
		t.Error("sort modified the input slice")
	}
}

// --- authorityLevel Tests ---

func TestAuthorityLevel(t *testing.T) {
	tests := []struct {
		issueType string
		labels    []string
		expected  int
	}{
		{"task", nil, 0},
		{"bug", nil, 0},
		{"feature", nil, 1},
		{"investigation", nil, 1},
		{"question", []string{"subtype:factual"}, 0},
		{"question", []string{"subtype:judgment"}, 2},
		{"question", nil, 2},
		{"unknown", nil, 1},
	}

	for _, tt := range tests {
		issue := Issue{IssueType: tt.issueType, Labels: tt.labels}
		got := authorityLevel(issue)
		if got != tt.expected {
			t.Errorf("authorityLevel(%s, %v) = %d, want %d",
				tt.issueType, tt.labels, got, tt.expected)
		}
	}
}

// --- HasLabel Tests ---

func TestIssue_HasLabel(t *testing.T) {
	issue := Issue{Labels: []string{"area:cli", "triage:ready"}}

	if !issue.HasLabel("area:cli") {
		t.Error("expected HasLabel('area:cli') to be true")
	}
	if !issue.HasLabel("AREA:CLI") {
		t.Error("expected HasLabel('AREA:CLI') to be true (case insensitive)")
	}
	if issue.HasLabel("area:daemon") {
		t.Error("expected HasLabel('area:daemon') to be false")
	}
}
