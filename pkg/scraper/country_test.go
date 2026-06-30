package scraper

import "testing"

func strPtr(v string) *string {
	return &v
}

func TestResolveCountryName(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected *string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty input",
			input:    strPtr("  "),
			expected: nil,
		},
		{
			name:     "already iso code",
			input:    strPtr("US"),
			expected: strPtr("US"),
		},
		{
			name:     "demonym maps to country code",
			input:    strPtr("German"),
			expected: strPtr("DE"),
		},
		{
			name:     "us state maps to US",
			input:    strPtr("California"),
			expected: strPtr("US"),
		},
		{
			name:     "locality string maps using right-most token",
			input:    strPtr("Palm Springs, California, United States"),
			expected: strPtr("US"),
		},
		{
			name:     "unknown value returned as-is",
			input:    strPtr("Middle Earth"),
			expected: strPtr("Middle Earth"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveCountryName(tt.input)
			if tt.expected == nil {
				if got != nil {
					t.Fatalf("expected nil, got %v", *got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if *got != *tt.expected {
				t.Fatalf("expected %q, got %q", *tt.expected, *got)
			}
		})
	}
}
