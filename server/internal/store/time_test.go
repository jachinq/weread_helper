package store

import "testing"

func TestAnnualBaseTime2026(t *testing.T) {
	got := AnnualBaseTime(2026)
	const want int64 = 1767196800
	if got != want {
		t.Fatalf("AnnualBaseTime(2026)=%d want %d", got, want)
	}
}
