package migrations

import "testing"

func TestEmbeddedMigrations(t *testing.T) {
	names, err := pendingFiles()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"000001_create_users.up.sql", "000002_create_games.up.sql", "000003_rating.up.sql"}
	if len(names) != len(want) {
		t.Fatalf("found %v", names)
	}
	for i, name := range want {
		if names[i] != name {
			t.Fatalf("found %v", names)
		}
		data, err := sqlFiles.ReadFile(name)
		if err != nil || len(data) == 0 {
			t.Fatalf("read %s: %v", name, err)
		}
	}
}
