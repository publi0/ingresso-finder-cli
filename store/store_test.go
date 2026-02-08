package store

import "testing"

func setTestConfigDir(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", root)
}

func TestSetTheaterHidden_RoundTrip(t *testing.T) {
	setTestConfigDir(t)

	hidden, err := LoadHiddenTheaters("1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(hidden) != 0 {
		t.Fatalf("expected no hidden theaters, got %+v", hidden)
	}

	if err := SetTheaterHidden("1", "10", true); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := SetTheaterHidden("1", "11", true); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	hidden, err = LoadHiddenTheaters("1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !hidden["10"] || !hidden["11"] {
		t.Fatalf("expected theaters to be hidden, got %+v", hidden)
	}

	if err := SetTheaterHidden("1", "10", false); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	hidden, err = LoadHiddenTheaters("1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if hidden["10"] {
		t.Fatalf("expected theater 10 visible, got %+v", hidden)
	}
	if !hidden["11"] {
		t.Fatalf("expected theater 11 hidden, got %+v", hidden)
	}
}

func TestSetTheaterHidden_InvalidInput(t *testing.T) {
	setTestConfigDir(t)

	if err := SetTheaterHidden("", "10", true); err == nil {
		t.Fatal("expected error for empty city id")
	}
	if err := SetTheaterHidden("1", "", true); err == nil {
		t.Fatal("expected error for empty theater id")
	}
}
