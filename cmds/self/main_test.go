package self

import "testing"

func TestVersionDisplay(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"dev", "dev"},
		{"", ""},
		{"1.2.3", "v1.2.3"},
		{"0.1.0", "v0.1.0"},
		{"10.20.30", "v10.20.30"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := versionDisplay(tt.input)
			if got != tt.want {
				t.Errorf("versionDisplay(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
