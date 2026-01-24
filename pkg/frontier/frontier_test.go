package frontier

import "testing"

func TestIsBlockingDependency(t *testing.T) {
	tests := []struct {
		name     string
		dep      Dependency
		expected bool
	}{
		{
			name: "parent-child never blocks",
			dep: Dependency{
				ID:             "test-1",
				Status:         "open",
				DependencyType: "parent-child",
			},
			expected: false,
		},
		{
			name: "blocks type with open status blocks",
			dep: Dependency{
				ID:             "test-2",
				Status:         "open",
				DependencyType: "blocks",
			},
			expected: true,
		},
		{
			name: "blocks type with closed status does not block",
			dep: Dependency{
				ID:             "test-3",
				Status:         "closed",
				DependencyType: "blocks",
			},
			expected: false,
		},
		{
			name: "question answered does not block",
			dep: Dependency{
				ID:             "test-4",
				Status:         "answered",
				IssueType:      "question",
				DependencyType: "blocks",
			},
			expected: false,
		},
		{
			name: "question open blocks",
			dep: Dependency{
				ID:             "test-5",
				Status:         "open",
				IssueType:      "question",
				DependencyType: "blocks",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBlockingDependency(tt.dep)
			if result != tt.expected {
				t.Errorf("isBlockingDependency(%+v) = %v, expected %v", tt.dep, result, tt.expected)
			}
		})
	}
}

func TestIsResolved(t *testing.T) {
	tests := []struct {
		name     string
		dep      Dependency
		expected bool
	}{
		{
			name:     "closed status is resolved",
			dep:      Dependency{Status: "closed"},
			expected: true,
		},
		{
			name:     "open status is not resolved",
			dep:      Dependency{Status: "open"},
			expected: false,
		},
		{
			name:     "question with answered status is resolved",
			dep:      Dependency{Status: "answered", IssueType: "question"},
			expected: true,
		},
		{
			name:     "non-question with answered status is not resolved",
			dep:      Dependency{Status: "answered", IssueType: "task"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isResolved(tt.dep)
			if result != tt.expected {
				t.Errorf("isResolved(%+v) = %v, expected %v", tt.dep, result, tt.expected)
			}
		})
	}
}

func TestFormatLeverage(t *testing.T) {
	tests := []struct {
		name     string
		bi       *BlockedIssue
		expected string
	}{
		{
			name: "no leverage",
			bi: &BlockedIssue{
				TotalLeverage: 0,
				WouldUnblock:  nil,
			},
			expected: "",
		},
		{
			name: "single unblock",
			bi: &BlockedIssue{
				TotalLeverage: 1,
				WouldUnblock:  []string{"test-1"},
			},
			expected: "unblocks: test-1",
		},
		{
			name: "multiple unblocks",
			bi: &BlockedIssue{
				TotalLeverage: 3,
				WouldUnblock:  []string{"test-1", "test-2", "test-3"},
			},
			expected: "unblocks: test-1, test-2, test-3",
		},
		{
			name: "many unblocks truncated",
			bi: &BlockedIssue{
				TotalLeverage: 5,
				WouldUnblock:  []string{"test-1", "test-2", "test-3", "test-4", "test-5"},
			},
			expected: "unblocks: test-1, test-2... (+3 more)",
		},
		{
			name: "transitive only",
			bi: &BlockedIssue{
				TotalLeverage: 3,
				WouldUnblock:  nil,
			},
			expected: "unblocks 3 (transitive)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatLeverage(tt.bi)
			if result != tt.expected {
				t.Errorf("FormatLeverage() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
