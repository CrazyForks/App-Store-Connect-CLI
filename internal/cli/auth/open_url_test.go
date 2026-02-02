package auth

import "testing"

func TestOpenURLRejectsEmpty(t *testing.T) {
	if err := openURL(" "); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestOpenURLRejectsInvalid(t *testing.T) {
	if err := openURL("://bad"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestOpenURLRejectsUnsupportedScheme(t *testing.T) {
	if err := openURL("file:///tmp/test"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
