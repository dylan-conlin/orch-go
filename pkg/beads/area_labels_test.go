package beads

import (
	"strings"
	"testing"
)

func TestHasAreaLabel(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
		want   bool
	}{
		{
			name:   "no labels",
			labels: nil,
			want:   false,
		},
		{
			name:   "empty labels",
			labels: []string{},
			want:   false,
		},
		{
			name:   "labels without area",
			labels: []string{"triage:ready", "subtype:factual"},
			want:   false,
		},
		{
			name:   "has area label",
			labels: []string{"area:dashboard", "triage:ready"},
			want:   true,
		},
		{
			name:   "only area label",
			labels: []string{"area:spawn"},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasAreaLabel(tt.labels); got != tt.want {
				t.Errorf("HasAreaLabel(%v) = %v, want %v", tt.labels, got, tt.want)
			}
		})
	}
}

func TestGetAreaLabel(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
		want   string
	}{
		{
			name:   "no labels",
			labels: nil,
			want:   "",
		},
		{
			name:   "labels without area",
			labels: []string{"triage:ready"},
			want:   "",
		},
		{
			name:   "has area label",
			labels: []string{"triage:ready", "area:beads"},
			want:   "area:beads",
		},
		{
			name:   "multiple area labels returns first",
			labels: []string{"area:spawn", "area:dashboard"},
			want:   "area:spawn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAreaLabel(tt.labels); got != tt.want {
				t.Errorf("GetAreaLabel(%v) = %v, want %v", tt.labels, got, tt.want)
			}
		})
	}
}

func TestSuggestAreaLabel(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		description string
		want        string
	}{
		{
			name:        "empty input",
			title:       "",
			description: "",
			want:        "",
		},
		{
			name:        "dashboard keyword in title",
			title:       "Fix dashboard loading issue",
			description: "",
			want:        "area:dashboard",
		},
		{
			name:        "spawn keyword in title",
			title:       "Agent spawn context missing",
			description: "",
			want:        "area:spawn",
		},
		{
			name:        "beads keyword in description",
			title:       "Fix tracking bug",
			description: "When using bd list the issue tracking fails",
			want:        "area:beads",
		},
		{
			name:        "cli keywords",
			title:       "Add new orch command for completion",
			description: "",
			want:        "area:cli",
		},
		{
			name:        "skill keywords",
			title:       "Create skillc template for worker procedures",
			description: "",
			want:        "area:skill",
		},
		{
			name:        "kb keywords",
			title:       "Add investigation to knowledge base",
			description: "New kb investigation template",
			want:        "area:kb",
		},
		{
			name:        "opencode keywords",
			title:       "Session management in opencode fork",
			description: "",
			want:        "area:opencode",
		},
		{
			name:        "daemon keywords",
			title:       "Daemon autonomous processing bug",
			description: "Ready queue not processed",
			want:        "area:daemon",
		},
		{
			name:        "config keywords",
			title:       "Configuration yaml settings",
			description: "",
			want:        "area:config",
		},
		{
			name:        "work graph matches dashboard",
			title:       "Work graph display issue",
			description: "",
			want:        "area:dashboard",
		},
		{
			name:        "no matching keywords",
			title:       "Generic improvement",
			description: "Some description",
			want:        "",
		},
		{
			name:        "multiple areas - best scoring wins",
			title:       "Dashboard UI for work graph display",
			description: "Fix the dashboard work graph component",
			want:        "area:dashboard", // dashboard + work graph (phrase) wins
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SuggestAreaLabel(tt.title, tt.description)
			if got != tt.want {
				t.Errorf("SuggestAreaLabel(%q, %q) = %q, want %q", tt.title, tt.description, got, tt.want)
			}
		})
	}
}

func TestFormatAreaLabelWarning(t *testing.T) {
	t.Run("has area label - no warning", func(t *testing.T) {
		labels := []string{"area:dashboard", "triage:ready"}
		got := FormatAreaLabelWarning(labels, "")
		if got != "" {
			t.Errorf("expected empty warning for labels with area:, got %q", got)
		}
	})

	t.Run("no area label - warning with suggestion", func(t *testing.T) {
		labels := []string{"triage:ready"}
		got := FormatAreaLabelWarning(labels, "area:spawn")
		if !strings.Contains(got, "Warning: Issue missing area: label") {
			t.Errorf("expected warning message, got %q", got)
		}
		if !strings.Contains(got, "Suggested: area:spawn") {
			t.Errorf("expected suggestion in warning, got %q", got)
		}
	})

	t.Run("no area label - warning without suggestion", func(t *testing.T) {
		labels := []string{"triage:ready"}
		got := FormatAreaLabelWarning(labels, "")
		if !strings.Contains(got, "Warning: Issue missing area: label") {
			t.Errorf("expected warning message, got %q", got)
		}
		if strings.Contains(got, "Suggested:") {
			t.Errorf("expected no suggestion in warning, got %q", got)
		}
	})
}

func TestValidateAreaLabel(t *testing.T) {
	tests := []struct {
		area string
		want bool
	}{
		{"dashboard", true},
		{"spawn", true},
		{"beads", true},
		{"cli", true},
		{"skill", true},
		{"kb", true},
		{"opencode", true},
		{"daemon", true},
		{"config", true},
		{"area:dashboard", true}, // with prefix
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.area, func(t *testing.T) {
			if got := ValidateAreaLabel(tt.area); got != tt.want {
				t.Errorf("ValidateAreaLabel(%q) = %v, want %v", tt.area, got, tt.want)
			}
		})
	}
}

func TestListAreaLabels(t *testing.T) {
	labels := ListAreaLabels()
	if len(labels) != len(KnownAreas) {
		t.Errorf("ListAreaLabels() returned %d labels, want %d", len(labels), len(KnownAreas))
	}

	for _, label := range labels {
		if !strings.HasPrefix(label, AreaLabelPrefix) {
			t.Errorf("ListAreaLabels() label %q missing prefix %q", label, AreaLabelPrefix)
		}
	}
}

func TestFormatAreaLabelSuggestion(t *testing.T) {
	t.Run("has suggestion", func(t *testing.T) {
		got := FormatAreaLabelSuggestion("Fix dashboard bug", "")
		if !strings.Contains(got, "area:dashboard") {
			t.Errorf("expected area:dashboard in suggestion, got %q", got)
		}
	})

	t.Run("no suggestion", func(t *testing.T) {
		got := FormatAreaLabelSuggestion("Generic task", "No keywords")
		if got != "" {
			t.Errorf("expected empty suggestion, got %q", got)
		}
	})
}
