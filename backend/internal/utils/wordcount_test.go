package utils

import "testing"

func TestCountStoryUnits(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int
	}{
		{name: "english words", text: "The quick brown fox", want: 4},
		{name: "chinese characters", text: "故事接龙", want: 4},
		{name: "mixed text", text: "hello 世界 2026", want: 4},
		{name: "punctuation", text: "hello, world! 不咕鸟", want: 5},
		{name: "empty", text: "   ", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CountStoryUnits(tt.text); got != tt.want {
				t.Fatalf("CountStoryUnits(%q) = %d, want %d", tt.text, got, tt.want)
			}
		})
	}
}
