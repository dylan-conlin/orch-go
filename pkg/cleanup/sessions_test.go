package cleanup

import "testing"

func TestIsUntrackedSessionTitle(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  bool
	}{
		{
			name:  "untracked beads id in title",
			title: "og-task-name [orch-go-untracked-1768090360]",
			want:  true,
		},
		{
			name:  "tracked beads id in title",
			title: "og-task-name [orch-go-abcd1]",
			want:  false,
		},
		{
			name:  "fallback untracked keyword",
			title: "manual untracked scratch session",
			want:  true,
		},
		{
			name:  "regular title",
			title: "og-task-name",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUntrackedSessionTitle(tt.title)
			if got != tt.want {
				t.Fatalf("isUntrackedSessionTitle(%q) = %v, want %v", tt.title, got, tt.want)
			}
		})
	}
}
