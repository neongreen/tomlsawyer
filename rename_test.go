package toml

import (
	"testing"
)

func TestRenameSectionStyleKey(t *testing.T) {
	input := "[foo.bar]\nx = 1\n"

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	if err := doc.Rename("foo.bar", "foo.qux"); err != nil {
		t.Fatal(err)
	}

	output := doc.String()
	wantGolden(t, output, "[foo.qux]\nx = 1\n")
}

func TestRenameDottedKey(t *testing.T) {
	input := "server.host = \"localhost\"\nserver.port = 8080\n"

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	if err := doc.Rename("server.host", "server.addr"); err != nil {
		t.Fatal(err)
	}

	output := doc.String()
	wantGolden(t, output, "server.addr = \"localhost\"\nserver.port = 8080\n")
}

func TestRenamePreservesComment(t *testing.T) {
	input := "[settings]\ntimeout = 30 # seconds\n"

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	if err := doc.Rename("settings", "config"); err != nil {
		t.Fatal(err)
	}

	output := doc.String()
	wantGolden(t, output, "[config]\ntimeout = 30  # seconds\n")
}

func TestRenameSectionWithMultipleKeys(t *testing.T) {
	input := "[old]\na = 1\nb = 2\n"

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	if err := doc.Rename("old", "new"); err != nil {
		t.Fatal(err)
	}

	output := doc.String()
	wantGolden(t, output, "[new]\na = 1\nb = 2\n")
}

func TestRenameWithChildSections(t *testing.T) {
	input := "[foo.bar]\nx = 1\n\n[foo.bar.baz]\ny = 2\n"

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	if err := doc.Rename("foo.bar", "foo.qux"); err != nil {
		t.Fatal(err)
	}

	output := doc.String()
	wantGolden(t, output, "[foo.qux]\nx = 1\n\n[foo.qux.baz]\ny = 2\n")
}

func TestRenameCrossSectionError(t *testing.T) {
	input := "[foo]\nx = 1\n"

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	err = doc.Rename("foo.x", "bar.x")
	if err == nil {
		t.Fatal("expected error for cross-section key rename")
	}
}
