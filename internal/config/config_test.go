package config

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		name, port, dsn, secret string
		valid                   bool
	}{
		{"valid", "8080", "postgres://db", "secret", true},
		{"missing port", "", "postgres://db", "secret", false},
		{"invalid port", "65536", "postgres://db", "secret", false},
		{"missing dsn", "8080", "", "secret", false},
		{"missing secret", "8080", "postgres://db", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := Parse(tc.port, tc.dsn, tc.secret)
			if (err == nil) != tc.valid {
				t.Fatalf("config=%+v err=%v", cfg, err)
			}
			if tc.valid && cfg.Address() != ":8080" {
				t.Fatal(cfg.Address())
			}
		})
	}
}
