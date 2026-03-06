package ratio_setting

import "testing"

func TestGetHardcodedCompletionModelRatioGpt54(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected float64
	}{
		{
			name:     "gpt-5.4 uses dedicated ratio",
			model:    "gpt-5.4",
			expected: 6,
		},
		{
			name:     "other gpt-5 models keep default ratio",
			model:    "gpt-5.1",
			expected: 8,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := getHardcodedCompletionModelRatio(tc.model)
			if !ok {
				t.Fatalf("expected hardcoded ratio for %s", tc.model)
			}
			if got != tc.expected {
				t.Fatalf("unexpected ratio for %s: got %v want %v", tc.model, got, tc.expected)
			}
		})
	}
}
