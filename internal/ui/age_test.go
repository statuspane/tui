package ui

import (
	"testing"
	"time"
)

func TestAgeText(t *testing.T) {
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		ago  time.Duration
		want string
	}{
		{"seconds", 20 * time.Second, "just now"},
		{"minutes", 3*time.Minute + 30*time.Second, "3m ago"},
		{"hours", 5*time.Hour + 10*time.Minute, "5h ago"},
		{"days", 50 * time.Hour, "2d ago"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ageText(now, now.Add(-tc.ago)); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
