package render

import "testing"

func TestSparklineHandlesEmptyAndSinglePoint(t *testing.T) {
	if got := Sparkline(nil); got != "" {
		t.Fatalf("empty sparkline = %q", got)
	}
	if got := Sparkline([]int64{5}); got != "▁" {
		t.Fatalf("single sparkline = %q", got)
	}
}

func TestSparklineScalesValues(t *testing.T) {
	got := Sparkline([]int64{0, 5, 10})
	if got != "▁▄█" {
		t.Fatalf("sparkline = %q", got)
	}
}
