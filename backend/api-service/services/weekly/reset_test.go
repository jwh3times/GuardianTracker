package weekly

import (
	"testing"
	"time"
)

func TestNextWeeklyReset(t *testing.T) {
	cases := []struct {
		name     string
		now      string // RFC3339
		wantDay  time.Weekday
		wantHour int
	}{
		{"monday morning", "2026-06-08T09:00:00Z", time.Tuesday, 17},
		{"tuesday before reset", "2026-06-09T16:59:59Z", time.Tuesday, 17},
		{"exactly at reset", "2026-06-09T17:00:00Z", time.Tuesday, 17}, // next Tuesday
		{"tuesday after reset", "2026-06-09T17:01:00Z", time.Tuesday, 17},
		{"friday", "2026-06-12T12:00:00Z", time.Tuesday, 17},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			now, _ := time.Parse(time.RFC3339, tc.now)
			got := NextWeeklyReset(now)
			if got.Weekday() != tc.wantDay || got.Hour() != tc.wantHour {
				t.Errorf("NextWeeklyReset(%s) = %s, want %s %02d:00",
					tc.now, got, tc.wantDay, tc.wantHour)
			}
			if !got.After(now) {
				t.Errorf("reset %s is not after now %s", got, now)
			}
		})
	}
}

func TestXurPresent(t *testing.T) {
	cases := []struct {
		now  string
		want bool
	}{
		{"2026-06-12T16:59:00Z", false}, // Fri before 17:00
		{"2026-06-12T17:00:00Z", true},  // Fri exactly 17:00
		{"2026-06-13T12:00:00Z", true},  // Saturday
		{"2026-06-14T12:00:00Z", true},  // Sunday
		{"2026-06-15T12:00:00Z", true},  // Monday
		{"2026-06-16T16:59:00Z", true},  // Tue before reset
		{"2026-06-16T17:00:00Z", false}, // Tue at reset (Xur leaves)
		{"2026-06-17T12:00:00Z", false}, // Wednesday
	}
	for _, tc := range cases {
		t.Run(tc.now, func(t *testing.T) {
			now, _ := time.Parse(time.RFC3339, tc.now)
			got := XurPresent(now)
			if got != tc.want {
				t.Errorf("XurPresent(%s) = %v, want %v", tc.now, got, tc.want)
			}
		})
	}
}
