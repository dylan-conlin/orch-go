package beadsutil

import "testing"

func TestExtractIDFromTitle(t *testing.T) {
	tests := []struct {
		name   string
		title  string
		wantID string
	}{
		{
			name:   "beads ID in brackets at end",
			title:  "og-feat-add-feature-19dec [orch-go-abc12]",
			wantID: "orch-go-abc12",
		},
		{
			name:   "beads ID in brackets with extra spaces",
			title:  "og-inv-something [ proj-xyz ]",
			wantID: "proj-xyz",
		},
		{
			name:   "no brackets",
			title:  "og-feat-add-feature-19dec",
			wantID: "",
		},
		{
			name:   "empty title",
			title:  "",
			wantID: "",
		},
		{
			name:   "unclosed bracket",
			title:  "og-feat-test [incomplete",
			wantID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractIDFromTitle(tt.title)
			if got != tt.wantID {
				t.Errorf("ExtractIDFromTitle(%q) = %q, want %q", tt.title, got, tt.wantID)
			}
		})
	}
}

func TestExtractIDFromWindowName(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantID string
	}{
		{
			name:   "standard window name",
			input:  "🔧 og-feat-test [orch-go-abc12]",
			wantID: "orch-go-abc12",
		},
		{
			name:   "no brackets",
			input:  "og-feat-test",
			wantID: "",
		},
		{
			name:   "empty",
			input:  "",
			wantID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractIDFromWindowName(tt.input)
			if got != tt.wantID {
				t.Errorf("ExtractIDFromWindowName(%q) = %q, want %q", tt.input, got, tt.wantID)
			}
		})
	}
}

func TestExtractProjectFromID(t *testing.T) {
	tests := []struct {
		name    string
		beadsID string
		want    string
	}{
		{
			name:    "simple two-part ID",
			beadsID: "orch-go-abc1",
			want:    "orch-go",
		},
		{
			name:    "three-part project name",
			beadsID: "kb-cli-xyz9",
			want:    "kb-cli",
		},
		{
			name:    "single-word project",
			beadsID: "beads-12ab",
			want:    "beads",
		},
		{
			name:    "multi-hyphen project name",
			beadsID: "some-long-project-name-a1b2",
			want:    "some-long-project-name",
		},
		{
			name:    "empty beads ID",
			beadsID: "",
			want:    "",
		},
		{
			name:    "single part (no hyphen)",
			beadsID: "abc1",
			want:    "abc1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractProjectFromID(tt.beadsID)
			if got != tt.want {
				t.Errorf("ExtractProjectFromID(%q) = %q, want %q", tt.beadsID, got, tt.want)
			}
		})
	}
}

func TestResolveShortIDSimple(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantHas string // substring that result should contain
		wantErr bool
	}{
		{
			name:    "already full ID",
			id:      "orch-go-abc1",
			wantHas: "orch-go-abc1",
		},
		{
			name:    "short ID gets project prefix",
			id:      "abc1",
			wantHas: "-abc1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveShortIDSimple(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveShortIDSimple(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
				return
			}
			if err == nil && !contains(got, tt.wantHas) {
				t.Errorf("ResolveShortIDSimple(%q) = %q, want to contain %q", tt.id, got, tt.wantHas)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && s[len(s)-len(substr):] == substr || s == substr
}
