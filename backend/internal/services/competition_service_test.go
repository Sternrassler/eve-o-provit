package services

import "testing"

func TestComputeChurn(t *testing.T) {
	tests := []struct {
		name  string
		prev  map[int64]float64
		curr  map[int64]float64
		hours float64
		want  float64
	}{
		{
			name:  "no changes over 2h",
			prev:  map[int64]float64{1: 5.0, 2: 6.0},
			curr:  map[int64]float64{1: 5.0, 2: 6.0},
			hours: 2,
			want:  0,
		},
		{
			name:  "one added one removed one repriced over 1h",
			prev:  map[int64]float64{1: 5.0, 2: 6.0}, // 2 removed
			curr:  map[int64]float64{1: 5.5, 3: 7.0}, // 1 repriced, 3 added
			hours: 1,
			want:  3, // added(3) + repriced(1) + removed(2) = 3 changes / 1h
		},
		{
			name:  "changes spread over 2h halves the rate",
			prev:  map[int64]float64{1: 5.0},
			curr:  map[int64]float64{2: 5.0, 3: 5.0}, // 2 added + 1 removed = 3
			hours: 2,
			want:  1.5,
		},
		{
			name:  "zero hours guarded",
			prev:  map[int64]float64{1: 5.0},
			curr:  map[int64]float64{2: 6.0},
			hours: 0,
			want:  0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeChurn(tt.prev, tt.curr, tt.hours)
			if got != tt.want {
				t.Errorf("ComputeChurn() = %v, want %v", got, tt.want)
			}
		})
	}
}
