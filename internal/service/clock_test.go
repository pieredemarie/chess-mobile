package service

import (
	"errors"
	"testing"
	"time"
)

func TestClock(t *testing.T) {
	now := time.Unix(100, 0)
	cases := []struct {
		name         string
		base, inc    time.Duration
		elapsed      time.Duration
		wantW, wantB time.Duration
		expired      bool
	}{
		{"increment", 10 * time.Second, 2 * time.Second, 3 * time.Second, 9 * time.Second, 10 * time.Second, false},
		{"flag", 10 * time.Second, 2 * time.Second, 11 * time.Second, 0, 10 * time.Second, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := NewClock(tc.base, tc.inc, now)
			if err != nil {
				t.Fatal(err)
			}
			err = c.Move(now.Add(tc.elapsed))
			if (err != nil) != tc.expired {
				t.Fatalf("move error: %v", err)
			}
			if tc.expired && !errors.Is(err, ErrTimeExpired) {
				t.Fatalf("wrong error: %v", err)
			}
			w, b := c.Remaining(now.Add(tc.elapsed))
			if !tc.expired && (w != tc.wantW || b != tc.wantB) {
				t.Fatalf("remaining %v %v", w, b)
			}
		})
	}
}

func TestClockRejectsInvalidControl(t *testing.T) {
	if _, err := NewClock(0, 0, time.Now()); err == nil {
		t.Fatal("zero base accepted")
	}
	if _, err := NewClock(time.Second, -time.Second, time.Now()); err == nil {
		t.Fatal("negative increment accepted")
	}
}
