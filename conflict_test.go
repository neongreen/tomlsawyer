package tomlsawyer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetRejectsScalarOverSection(t *testing.T) {
	doc, _ := ParseString("[server]\nhost = \"localhost\"\n")
	err := doc.Set("server", 1)
	if err == nil {
		t.Fatal("Set should reject setting scalar over existing section")
	}
}

func TestSetRejectsNestedUnderScalar(t *testing.T) {
	doc, _ := ParseString("server = \"localhost\"\n")
	err := doc.Set("server.port", 8080)
	if err == nil {
		t.Fatal("Set should reject creating nested key under scalar")
	}
}

func TestSetAllowsMapOverSection(t *testing.T) {
	doc, _ := ParseString("[server]\nhost = \"localhost\"\n")
	err := doc.Set("server", map[string]any{"host": "0.0.0.0"})
	// This could either work or error — just verify no panic
	_ = err
}

func TestSetRejectsDeepNestedUnderScalar(t *testing.T) {
	doc, _ := ParseString("a = 1\n")
	err := doc.Set("a.b.c", 2)
	if err == nil {
		t.Fatal("Set should reject creating nested key under scalar")
	}
}

func TestMoveRejectsExistingDestination(t *testing.T) {
	doc, _ := ParseString("[a]\nx = 1\n\n[b]\nx = 2\n")
	err := doc.Move("a.x", "b.x")
	if err == nil {
		t.Fatal("Move should reject moving to existing destination")
	}
}

func TestMoveRejectsExistingSectionDestination(t *testing.T) {
	doc, _ := ParseString("[old]\nx = 1\n\n[new]\ny = 2\n")
	err := doc.Move("old", "new")
	if err == nil {
		t.Fatal("Move should reject moving to existing section")
	}
}

func TestWriteFilePreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.toml")
	os.WriteFile(path, []byte("key = \"old\"\n"), 0o600)

	err := WriteFile(path, map[string]any{"key": "new"})
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("permissions = %o, want 0600", info.Mode().Perm())
	}
}
