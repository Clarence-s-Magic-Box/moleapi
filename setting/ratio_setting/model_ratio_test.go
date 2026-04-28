package ratio_setting

import "testing"

func TestGetHardcodedCompletionModelRatioGpt5(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected float64
	}{
		{
			name:     "gpt-5.5 uses official output multiplier",
			model:    "gpt-5.5",
			expected: 6,
		},
		{
			name:     "gpt-5.5 dated variant uses official output multiplier",
			model:    "gpt-5.5-2026-04-24",
			expected: 6,
		},
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

func TestGetCompletionRatioInfoGPT55UsesOfficialOutputMultiplier(t *testing.T) {
	info := GetCompletionRatioInfo("gpt-5.5")

	if info.Ratio != 6 {
		t.Fatalf("gpt-5.5 completion ratio = %v, want 6", info.Ratio)
	}
	if !info.Locked {
		t.Fatal("gpt-5.5 completion ratio should be locked to the official multiplier")
	}
}

func TestGetCompletionRatioGPT55DatedVariant(t *testing.T) {
	got := GetCompletionRatio("gpt-5.5-2026-04-24")

	if got != 6 {
		t.Fatalf("gpt-5.5 dated variant completion ratio = %v, want 6", got)
	}
}

func TestDefaultImageOutputRatioGPTImage2(t *testing.T) {
	got, ok := defaultImageOutputRatio["gpt-image-2"]

	if !ok {
		t.Fatal("gpt-image-2 should have a default image output ratio")
	}
	if got != 6 {
		t.Fatalf("gpt-image-2 image output ratio = %v, want 6", got)
	}
}
