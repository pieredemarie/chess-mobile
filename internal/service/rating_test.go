package service

import "testing"

func TestEloRatings(t *testing.T) {
	cases := []struct {
		name                 string
		white, black         int
		result               string
		wantWhite, wantBlack int
	}{
		{"white wins equal", 1200, 1200, "1-0", 1216, 1184},
		{"black wins equal", 1200, 1200, "0-1", 1184, 1216},
		{"draw equal", 1200, 1200, "1/2-1/2", 1200, 1200},
		{"upset", 1600, 1200, "0-1", 1571, 1229},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, b := EloRatings(tc.white, tc.black, tc.result)
			if w != tc.wantWhite || b != tc.wantBlack {
				t.Fatalf("got %d,%d want %d,%d", w, b, tc.wantWhite, tc.wantBlack)
			}
		})
	}
}
