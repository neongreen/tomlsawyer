package tomlsawyer

import (
	"testing"
)

// --- Section renames (in-place) ---

func TestMoveSectionBasic(t *testing.T) {
	doc, err := ParseString("[old]\na = 1\nb = 2\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("old", "new"); err != nil {
		t.Fatal(err)
	}
	wantGolden(t, doc.String(), "[new]\na = 1\nb = 2\n")
}

func TestMoveSectionNested(t *testing.T) {
	doc, err := ParseString("[foo.bar]\nx = 1\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("foo.bar", "foo.qux"); err != nil {
		t.Fatal(err)
	}
	wantGolden(t, doc.String(), "[foo.qux]\nx = 1\n")
}

func TestMoveSectionCascadesToChildren(t *testing.T) {
	doc, err := ParseString("[foo.bar]\nx = 1\n\n[foo.bar.baz]\ny = 2\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("foo.bar", "foo.qux"); err != nil {
		t.Fatal(err)
	}
	wantGolden(t, doc.String(), "[foo.qux]\nx = 1\n\n[foo.qux.baz]\ny = 2\n")
}

func TestMoveSectionCascadesDeeply(t *testing.T) {
	input := "[a.b]\nx = 1\n\n[a.b.c]\ny = 2\n\n[a.b.c.d]\nz = 3\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("a.b", "a.renamed"); err != nil {
		t.Fatal(err)
	}
	wantGolden(t, doc.String(), "[a.renamed]\nx = 1\n\n[a.renamed.c]\ny = 2\n\n[a.renamed.c.d]\nz = 3\n")
}

func TestMoveSectionPreservesBlockComment(t *testing.T) {
	input := "# This section is important\n[old]\nkey = \"val\"\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("old", "new"); err != nil {
		t.Fatal(err)
	}
	wantGolden(t, doc.String(), "# This section is important\n[new]\nkey = \"val\"\n")
}

func TestMoveSectionPreservesInlineComment(t *testing.T) {
	input := "[settings] # app config\ntimeout = 30\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("settings", "config"); err != nil {
		t.Fatal(err)
	}
	wantGolden(t, doc.String(), "[config]  # app config\ntimeout = 30\n")
}

func TestMoveSectionPreservesKeyComments(t *testing.T) {
	input := "[settings]\ntimeout = 30  # seconds\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("settings", "config"); err != nil {
		t.Fatal(err)
	}
	wantGolden(t, doc.String(), "[config]\ntimeout = 30  # seconds\n")
}

func TestMoveSectionDoesNotAffectSiblings(t *testing.T) {
	input := "[alpha]\na = 1\n\n[beta]\nb = 2\n\n[gamma]\nc = 3\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("beta", "bravo"); err != nil {
		t.Fatal(err)
	}
	wantGolden(t, doc.String(), "[alpha]\na = 1\n\n[bravo]\nb = 2\n\n[gamma]\nc = 3\n")
}

func TestMoveSectionArrayOfTables(t *testing.T) {
	// [[products]] uses IsArray on heading — must be preserved
	input := "[[products]]\nname = \"Hammer\"\n\n[[products]]\nname = \"Nail\"\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	// Move section heading — both [[products]] entries get renamed
	if err := doc.Move("products", "items"); err != nil {
		t.Fatal(err)
	}
	wantGolden(t, doc.String(), "[[items]]\nname = \"Hammer\"\n\n[[items]]\nname = \"Nail\"\n")
}

func TestMoveSectionTopLevelToNested(t *testing.T) {
	input := "[server]\nhost = \"localhost\"\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("server", "app.server"); err != nil {
		t.Fatal(err)
	}
	wantGolden(t, doc.String(), "[app.server]\nhost = \"localhost\"\n")
}

func TestMoveSectionNestedToTopLevel(t *testing.T) {
	input := "[app.server]\nhost = \"localhost\"\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("app.server", "server"); err != nil {
		t.Fatal(err)
	}
	wantGolden(t, doc.String(), "[server]\nhost = \"localhost\"\n")
}

func TestMoveSectionQuotedName(t *testing.T) {
	input := "[\"old.section\"]\nkey = 1\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move(`"old.section"`, `"new.section"`); err != nil {
		t.Fatal(err)
	}
	wantGolden(t, doc.String(), "[\"new.section\"]\nkey = 1\n")
}

// --- Key renames (same section) ---

func TestMoveKeySameSection(t *testing.T) {
	input := "[server]\nhost = \"localhost\"\nport = 8080\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("server.host", "server.addr"); err != nil {
		t.Fatal(err)
	}
	wantGolden(t, doc.String(), "[server]\naddr = \"localhost\"\nport = 8080\n")
}

func TestMoveKeyDottedStyle(t *testing.T) {
	input := "server.host = \"localhost\"\nserver.port = 8080\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("server.host", "server.addr"); err != nil {
		t.Fatal(err)
	}
	wantGolden(t, doc.String(), "server.addr = \"localhost\"\nserver.port = 8080\n")
}

func TestMoveKeyPreservesInlineComment(t *testing.T) {
	input := "[config]\ntimeout = 30  # seconds\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("config.timeout", "config.max_wait"); err != nil {
		t.Fatal(err)
	}
	wantGolden(t, doc.String(), "[config]\nmax_wait = 30  # seconds\n")
}

func TestMoveKeyTopLevel(t *testing.T) {
	input := "old_name = \"value\"\nother = 1\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("old_name", "new_name"); err != nil {
		t.Fatal(err)
	}
	wantGolden(t, doc.String(), "new_name = \"value\"\nother = 1\n")
}

// --- Cross-section key moves ---

func TestMoveKeyCrossSection(t *testing.T) {
	input := "[source]\nhost = \"localhost\"\nport = 8080\n\n[dest]\nurl = \"http://example.com\"\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("source.host", "dest.host"); err != nil {
		t.Fatal(err)
	}
	output := doc.String()
	// host should be gone from [source] and appear in [dest]
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v", err)
	}
	if ok, _ := doc2.Has("source.host"); ok {
		t.Error("source.host should be gone")
	}
	val, _, _ := doc2.Get("dest.host")
	if val != "localhost" {
		t.Errorf("dest.host = %v, want localhost", val)
	}
	// port should still be in source
	val, _, _ = doc2.Get("source.port")
	if val != int64(8080) {
		t.Errorf("source.port = %v, want 8080", val)
	}
}

func TestMoveKeyCrossSectionPreservesComment(t *testing.T) {
	input := "[old]\ntimeout = 30  # seconds\n\n[new]\nhost = \"localhost\"\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("old.timeout", "new.timeout"); err != nil {
		t.Fatal(err)
	}
	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v", err)
	}
	val, _, _ := doc2.Get("new.timeout")
	if val != int64(30) {
		t.Errorf("new.timeout = %v, want 30", val)
	}
	if ok, _ := doc2.Has("old.timeout"); ok {
		t.Error("old.timeout should be gone")
	}
}

func TestMoveKeyCrossSectionCreatesDestination(t *testing.T) {
	input := "[source]\nhost = \"localhost\"\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	// destination section doesn't exist yet
	if err := doc.Move("source.host", "newdest.host"); err != nil {
		t.Fatal(err)
	}
	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v", err)
	}
	val, _, _ := doc2.Get("newdest.host")
	if val != "localhost" {
		t.Errorf("newdest.host = %v, want localhost", val)
	}
	if ok, _ := doc2.Has("source.host"); ok {
		t.Error("source.host should be gone")
	}
}

func TestMoveKeyToGlobalSection(t *testing.T) {
	input := "[section]\nkey = \"value\"\nother = 1\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("section.key", "key"); err != nil {
		t.Fatal(err)
	}
	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v", err)
	}
	val, _, _ := doc2.Get("key")
	if val != "value" {
		t.Errorf("key = %v, want value", val)
	}
	if ok, _ := doc2.Has("section.key"); ok {
		t.Error("section.key should be gone")
	}
}

func TestMoveKeyFromGlobalToSection(t *testing.T) {
	input := "orphan = \"value\"\n\n[section]\nother = 1\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("orphan", "section.orphan"); err != nil {
		t.Fatal(err)
	}
	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v", err)
	}
	val, _, _ := doc2.Get("section.orphan")
	if val != "value" {
		t.Errorf("section.orphan = %v, want value", val)
	}
	if ok, _ := doc2.Has("orphan"); ok {
		t.Error("top-level orphan should be gone")
	}
}

func TestMoveKeyCrossSectionDeeplyNested(t *testing.T) {
	input := "[a.b.c]\nx = 1\n\n[d.e]\ny = 2\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("a.b.c.x", "d.e.x"); err != nil {
		t.Fatal(err)
	}
	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v", err)
	}
	val, _, _ := doc2.Get("d.e.x")
	if val != int64(1) {
		t.Errorf("d.e.x = %v, want 1", val)
	}
	if ok, _ := doc2.Has("a.b.c.x"); ok {
		t.Error("a.b.c.x should be gone")
	}
}

func TestMoveKeyCrossSectionWithRename(t *testing.T) {
	// Move AND rename simultaneously: source.host → dest.addr
	input := "[source]\nhost = \"localhost\"\n\n[dest]\nport = 8080\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("source.host", "dest.addr"); err != nil {
		t.Fatal(err)
	}
	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v", err)
	}
	val, _, _ := doc2.Get("dest.addr")
	if val != "localhost" {
		t.Errorf("dest.addr = %v, want localhost", val)
	}
	if ok, _ := doc2.Has("source.host"); ok {
		t.Error("source.host should be gone")
	}
	if ok, _ := doc2.Has("dest.host"); ok {
		t.Error("dest.host should not exist (renamed to addr)")
	}
}

// --- Error cases ---

func TestMoveNotFound(t *testing.T) {
	doc, _ := ParseString("[foo]\nx = 1\n")
	err := doc.Move("nonexistent", "whatever")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestMoveInvalidOldPath(t *testing.T) {
	doc, _ := ParseString("[foo]\nx = 1\n")
	err := doc.Move("foo.", "bar")
	if err == nil {
		t.Fatal("expected error for invalid old path")
	}
}

func TestMoveInvalidNewPath(t *testing.T) {
	doc, _ := ParseString("[foo]\nx = 1\n")
	err := doc.Move("foo", ".bar")
	if err == nil {
		t.Fatal("expected error for invalid new path")
	}
}

// --- Value type preservation ---

func TestMovePreservesStringValue(t *testing.T) {
	doc, _ := ParseString("[a]\ns = \"hello\"\n\n[b]\n")
	doc.Move("a.s", "b.s")
	val, _, _ := doc.Get("b.s")
	if val != "hello" {
		t.Errorf("b.s = %v, want hello", val)
	}
}

func TestMovePreservesIntValue(t *testing.T) {
	doc, _ := ParseString("[a]\nn = 42\n\n[b]\n")
	doc.Move("a.n", "b.n")
	val, _, _ := doc.Get("b.n")
	if val != int64(42) {
		t.Errorf("b.n = %v, want 42", val)
	}
}

func TestMovePreservesBoolValue(t *testing.T) {
	doc, _ := ParseString("[a]\nf = true\n\n[b]\n")
	doc.Move("a.f", "b.f")
	val, _, _ := doc.Get("b.f")
	if val != true {
		t.Errorf("b.f = %v, want true", val)
	}
}

func TestMovePreservesArrayValue(t *testing.T) {
	doc, _ := ParseString("[a]\narr = [1, 2, 3]\n\n[b]\n")
	doc.Move("a.arr", "b.arr")
	val, _, _ := doc.Get("b.arr")
	arr, ok := val.([]any)
	if !ok || len(arr) != 3 {
		t.Errorf("b.arr = %v, want [1,2,3]", val)
	}
}

func TestMovePreservesInlineTableValue(t *testing.T) {
	doc, _ := ParseString("[a]\ntbl = { x = 1, y = 2 }\n\n[b]\n")
	doc.Move("a.tbl", "b.tbl")
	val, _, _ := doc.Get("b.tbl")
	tbl, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("b.tbl = %T, want map", val)
	}
	if tbl["x"] != int64(1) || tbl["y"] != int64(2) {
		t.Errorf("b.tbl = %v, want {x:1, y:2}", tbl)
	}
}

// --- Dotted key style preservation on cross-section move ---

func TestMoveCrossSectionPreservesDottedStyle(t *testing.T) {
	// When destination already has dotted keys, the moved key should stay dotted
	input := "server.host = \"localhost\"\nserver.port = 8080\napp.name = \"myapp\"\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("server.host", "app.host"); err != nil {
		t.Fatal(err)
	}
	output := doc.String()
	// app.host should be dotted, not under a [app] section
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v", err)
	}
	val, _, _ := doc2.Get("app.host")
	if val != "localhost" {
		t.Errorf("app.host = %v, want localhost", val)
	}
	if ok, _ := doc2.Has("server.host"); ok {
		t.Error("server.host should be gone")
	}
	// Verify dotted style is used (no [app] section header)
	wantGolden(t, output, "server.port = 8080\napp.name = \"myapp\"\napp.host = \"localhost\"\n")
}

func TestMoveCrossSectionToExistingSection(t *testing.T) {
	// When destination is an existing section, insert into it
	input := "server.host = \"localhost\"\n\n[app]\nname = \"myapp\"\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("server.host", "app.host"); err != nil {
		t.Fatal(err)
	}
	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v", err)
	}
	val, _, _ := doc2.Get("app.host")
	if val != "localhost" {
		t.Errorf("app.host = %v, want localhost", val)
	}
}
