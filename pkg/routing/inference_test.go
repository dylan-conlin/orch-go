package routing

import "testing"

func TestInferSkillForIssue(t *testing.T) {
	tests := []struct {
		name        string
		issueType   string
		title       string
		description string
		labels      []string
		want        string
		wantErr     bool
	}{
		{name: "type fallback", issueType: "feature", want: "feature-impl"},
		{name: "label override", issueType: "task", labels: []string{"skill:architect"}, want: "architect"},
		{name: "title override", issueType: "task", title: "Design routing enrichment", want: "architect"},
		{name: "description override", issueType: "task", description: "Compare routing options", want: "research"},
		{name: "unknown type", issueType: "epic", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InferSkillForIssue(tt.issueType, tt.title, tt.description, tt.labels)
			if (err != nil) != tt.wantErr {
				t.Fatalf("InferSkillForIssue() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("InferSkillForIssue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInferAreaFromText(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		description string
		want        string
	}{
		{name: "spawn path", title: "Fix pkg/spawn/context.go", want: "spawn"},
		{name: "kb path", description: "Update .kb/models/routing/model.md", want: "kb"},
		{name: "dashboard keyword", title: "Polish dashboard work graph", want: "dashboard"},
		{name: "no match", title: "General cleanup", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InferAreaFromText(tt.title, tt.description); got != tt.want {
				t.Fatalf("InferAreaFromText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnrichRoutingLabels(t *testing.T) {
	tests := []struct {
		name        string
		labels      []string
		issueType   string
		title       string
		description string
		want        []string
	}{
		{
			name:      "adds skill and area",
			issueType: "task",
			title:     "Update pkg/spawn/context.go",
			want:      []string{"skill:feature-impl", "area:spawn"},
		},
		{
			name:      "preserves explicit routing labels",
			labels:    []string{"skill:architect", "area:kb", "triage:ready"},
			issueType: "task",
			title:     "Update pkg/spawn/context.go",
			want:      []string{"skill:architect", "area:kb", "triage:ready"},
		},
		{
			name:   "requires issue type",
			labels: []string{"triage:ready"},
			title:  "Update pkg/spawn/context.go",
			want:   []string{"triage:ready"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnrichRoutingLabels(tt.labels, tt.issueType, tt.title, tt.description)
			if len(got) != len(tt.want) {
				t.Fatalf("len(EnrichRoutingLabels()) = %d, want %d (%v)", len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("EnrichRoutingLabels()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
