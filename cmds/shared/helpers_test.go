package shared

import "testing"

func TestBoolCount(t *testing.T) {
	tests := []struct {
		name string
		vals []bool
		want int
	}{
		{"none", []bool{false, false, false}, 0},
		{"one", []bool{true, false, false}, 1},
		{"two", []bool{true, true, false}, 2},
		{"all", []bool{true, true, true}, 3},
		{"empty", []bool{}, 0},
		{"single true", []bool{true}, 1},
		{"single false", []bool{false}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BoolCount(tt.vals...)
			if got != tt.want {
				t.Errorf("BoolCount(%v) = %d, want %d", tt.vals, got, tt.want)
			}
		})
	}
}
