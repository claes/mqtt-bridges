package lib

import (
	"testing"
)

func TestComputeChange(t *testing.T) {
	tests := []struct {
		name     string
		before   uint32
		change   int32
		max      uint32
		min      uint32
		expected uint32
	}{
		{
			name:     "Normal case within range",
			before:   50,
			change:   20,
			max:      100,
			min:      0,
			expected: 70,
		},
		{
			name:     "Exceeds max boundary",
			before:   80,
			change:   50,
			max:      100,
			min:      0,
			expected: 100,
		},
		{
			name:     "Falls below min boundary",
			before:   30,
			change:   -50,
			max:      100,
			min:      0,
			expected: 0,
		},
		{
			name:     "Exactly hits max boundary",
			before:   70,
			change:   30,
			max:      100,
			min:      0,
			expected: 100,
		},
		{
			name:     "Exactly hits min boundary",
			before:   20,
			change:   -20,
			max:      100,
			min:      0,
			expected: 0,
		},
		{
			name:     "No change",
			before:   50,
			change:   0,
			max:      100,
			min:      0,
			expected: 50,
		},
		{
			name:     "Max and min boundary equal",
			before:   50,
			change:   0,
			max:      50,
			min:      50,
			expected: 50,
		},
		{
			name:     "Negative change but within range",
			before:   60,
			change:   -10,
			max:      100,
			min:      0,
			expected: 50,
		},
		{
			name:     "Underflow with extreme change",
			before:   10,
			change:   -100,
			max:      100,
			min:      0,
			expected: 0,
		},
		{
			name:     "Overflow with extreme change",
			before:   10,
			change:   1000,
			max:      100,
			min:      0,
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeChange(tt.before, tt.change, tt.max, tt.min)
			if result != tt.expected {
				t.Errorf("computeChange(%d, %d, %d, %d) = %d; want %d", tt.before, tt.change, tt.max, tt.min, result, tt.expected)
			}
		})
	}
}
