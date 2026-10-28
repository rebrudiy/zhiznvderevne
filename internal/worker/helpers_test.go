package worker

import (
	"math/rand"
	"testing"
	"time"
)

func TestSanitizeSlug(t *testing.T) {
	cases := map[string]string{
		"":                    "post",
		"Hello World":         "hello-world",
		"  --a--b--  ":        "a-b",
		"Огород в мае":        "post",
		"Огород 2026 garden!": "2026-garden",
		"x__y..z":             "x-y-z",
	}
	for in, want := range cases {
		if got := sanitizeSlug(in); got != want {
			t.Errorf("sanitizeSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseClock(t *testing.T) {
	base := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	got, err := parseClock("09:30", base)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 5, 10, 9, 30, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for _, bad := range []string{"", "9", "09:30:00", "25:00", "10:60", "ab:cd", "-1:10"} {
		if _, err := parseClock(bad, base); err == nil {
			t.Errorf("parseClock(%q): expected error", bad)
		}
	}
}

func TestRandomTimeBetween(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	start := time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	for i := 0; i < 1000; i++ {
		got := randomTimeBetween(rng, start, end)
		if got.Before(start) || !got.Before(end) {
			t.Fatalf("%v not in [%v, %v)", got, start, end)
		}
	}
	if got := randomTimeBetween(rng, end, start); !got.Equal(end) {
		t.Fatalf("empty window: got %v, want start", got)
	}
}
