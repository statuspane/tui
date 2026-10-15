package ui

import "testing"

func TestStatusLabels(t *testing.T) {
	unknown := "something_new"
	cases := []struct {
		name   string
		status *string
		want   string
	}{
		{"operational", stringPtr("operational"), "Operational"},
		{"degraded", stringPtr("degraded_performance"), "Degraded"},
		{"major outage", stringPtr("major_outage"), "Major outage"},
		{"no check yet", nil, "Pending"},
		{"unknown value is shown as is", &unknown, "something_new"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := statusLabel(tc.status); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
