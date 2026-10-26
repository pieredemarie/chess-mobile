package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvPreservesInjectedValues(t *testing.T) {
	file := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(file, []byte("PORT=8080\nJWT_SECRET=from-file\n# comment\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JWT_SECRET", "injected")
	if err := loadDotEnv(file); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("JWT_SECRET"); got != "injected" {
		t.Fatalf("secret overwritten: %q", got)
	}
}
