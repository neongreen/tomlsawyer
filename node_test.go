package tomlsawyer

import (
	"testing"
)

// --- Section access ---

func TestSectionsRoundTrip(t *testing.T) {
	doc, _ := ParseString("[alpha]\na = 1\n\n[beta]\nb = 2\n\n[gamma]\nc = 3\n")
	sections := doc.Sections()
	if len(sections) != 3 {
		t.Fatalf("Sections() = %d, want 3", len(sections))
	}
	if sections[0].Name() != "alpha" || sections[1].Name() != "beta" || sections[2].Name() != "gamma" {
		t.Errorf("names = [%s, %s, %s]", sections[0].Name(), sections[1].Name(), sections[2].Name())
	}
}

func TestSwapSections(t *testing.T) {
	doc, _ := ParseString("[alpha]\na = 1\n\n[beta]\nb = 2\n\n[gamma]\nc = 3\n")
	sections := doc.Sections()
	sections[0], sections[2] = sections[2], sections[0]
	doc.SetSections(sections)
	wantGolden(t, doc.String(), "[gamma]\nc = 3\n\n[beta]\nb = 2\n\n[alpha]\na = 1\n")
}

func TestSwapSectionsPreservesComments(t *testing.T) {
	input := "# Database\n[database]\nhost = \"localhost\"\n\n# Server\n[server]\nport = 8080\n"
	doc, _ := ParseString(input)
	sections := doc.Sections()
	sections[0], sections[1] = sections[1], sections[0]
	doc.SetSections(sections)
	output := doc.String()
	// Block comments travel with their sections
	wantGolden(t, output, "# Server\n[server]\nport = 8080\n\n# Database\n[database]\nhost = \"localhost\"\n")
}

func TestSectionComment(t *testing.T) {
	doc, _ := ParseString("[server]  # main\nhost = \"localhost\"\n")
	sec := doc.Sections()[0]
	if sec.Comment() != "# main" {
		t.Errorf("Comment() = %q, want %q", sec.Comment(), "# main")
	}
	sec.SetComment("# updated")
	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v", err)
	}
	_ = doc2
}

func TestSectionBlockComment(t *testing.T) {
	doc, _ := ParseString("# Important section\n# Do not modify\n[config]\nkey = 1\n")
	sec := doc.Sections()[0]
	bc := sec.BlockComment()
	if len(bc) != 2 {
		t.Fatalf("BlockComment() = %v, want 2 lines", bc)
	}
	sec.SetBlockComment([]string{"# New comment"})
	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v", err)
	}
	_ = doc2
}

// --- Entry access ---

func TestSectionEntries(t *testing.T) {
	doc, _ := ParseString("[server]\nhost = \"localhost\"\nport = 8080\ntls = true\n")
	sec := doc.Sections()[0]
	entries := sec.Entries()
	if len(entries) != 3 {
		t.Fatalf("Entries() = %d, want 3", len(entries))
	}
	if entries[0].Key() != "host" || entries[1].Key() != "port" || entries[2].Key() != "tls" {
		t.Errorf("keys = [%s, %s, %s]", entries[0].Key(), entries[1].Key(), entries[2].Key())
	}
}

func TestSwapEntries(t *testing.T) {
	doc, _ := ParseString("[server]\nhost = \"localhost\"\nport = 8080\n")
	sec := doc.Sections()[0]
	entries := sec.Entries()
	entries[0], entries[1] = entries[1], entries[0]
	sec.SetEntries(entries)
	wantGolden(t, doc.String(), "[server]\nport = 8080\nhost = \"localhost\"\n")
}

func TestSwapEntriesPreservesComments(t *testing.T) {
	input := "[config]\n# The hostname\nhost = \"localhost\"\n# The port number\nport = 8080\n"
	doc, _ := ParseString(input)
	sec := doc.Sections()[0]
	entries := sec.Entries()
	entries[0], entries[1] = entries[1], entries[0]
	sec.SetEntries(entries)
	wantGolden(t, doc.String(), "[config]\n\n# The port number\nport = 8080\n\n# The hostname\nhost = \"localhost\"\n")
}

func TestEntryComment(t *testing.T) {
	doc, _ := ParseString("[s]\nkey = 42  # the answer\n")
	entry := doc.Sections()[0].Entries()[0]
	if entry.Comment() != "# the answer" {
		t.Errorf("Comment() = %q", entry.Comment())
	}
	entry.SetComment("# updated")
	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v", err)
	}
	val, _, _ := doc2.Get("s.key")
	if val != int64(42) {
		t.Errorf("value lost after comment change: %v", val)
	}
}

func TestSetEntryBlockComment(t *testing.T) {
	doc, _ := ParseString("[s]\nkey = 1\n")
	entry := doc.Sections()[0].Entries()[0]
	entry.SetBlockComment([]string{"# added comment"})
	wantGolden(t, doc.String(), "[s]\n\n# added comment\nkey = 1\n")
}

func TestDeleteEntry(t *testing.T) {
	doc, _ := ParseString("[s]\na = 1\nb = 2\nc = 3\n")
	sec := doc.Sections()[0]
	entries := sec.Entries()
	// Remove middle entry
	entries = append(entries[:1], entries[2:]...)
	sec.SetEntries(entries)
	wantGolden(t, doc.String(), "[s]\na = 1\nc = 3\n")
}

func TestInsertEntry(t *testing.T) {
	doc, _ := ParseString("[s]\na = 1\nc = 3\n")
	sec := doc.Sections()[0]
	entries := sec.Entries()
	newEntry, err := NewEntry("b", 2)
	if err != nil {
		t.Fatal(err)
	}
	// Insert at position 1
	entries = append(entries[:1], append([]*Entry{newEntry}, entries[1:]...)...)
	sec.SetEntries(entries)
	wantGolden(t, doc.String(), "[s]\na = 1\nb = 2\nc = 3\n")
}

// --- Array elements ---

func TestArrayElements(t *testing.T) {
	doc, _ := ParseString("arr = [1, 2, 3]\n")
	v := doc.GetValue("arr")
	if v == nil {
		t.Fatal("GetValue returned nil")
	}
	elems := v.Elements()
	if len(elems) != 3 {
		t.Fatalf("Elements() = %d, want 3", len(elems))
	}
	if elems[0].AsAny() != int64(1) || elems[1].AsAny() != int64(2) || elems[2].AsAny() != int64(3) {
		t.Errorf("values = [%v, %v, %v]", elems[0].AsAny(), elems[1].AsAny(), elems[2].AsAny())
	}
}

func TestSwapArrayElements(t *testing.T) {
	doc, _ := ParseString("arr = [1, 2, 3]\n")
	v := doc.GetValue("arr")
	elems := v.Elements()
	elems[0], elems[2] = elems[2], elems[0]
	v.SetElements(elems)
	output := doc.String()
	doc2, _ := ParseString(output)
	val, _, _ := doc2.Get("arr")
	a := val.([]any)
	if a[0] != int64(3) || a[2] != int64(1) {
		t.Errorf("swap failed: %v", a)
	}
}

func TestArrayAppendElement(t *testing.T) {
	doc, _ := ParseString("arr = [1, 2]\n")
	v := doc.GetValue("arr")
	elems := v.Elements()
	newElem, _ := NewElement(3)
	elems = append(elems, newElem)
	v.SetElements(elems)
	output := doc.String()
	doc2, _ := ParseString(output)
	val, _, _ := doc2.Get("arr")
	a := val.([]any)
	if len(a) != 3 || a[2] != int64(3) {
		t.Errorf("append failed: %v", a)
	}
}

func TestArrayRemoveElement(t *testing.T) {
	doc, _ := ParseString("arr = [1, 2, 3]\n")
	v := doc.GetValue("arr")
	elems := v.Elements()
	elems = append(elems[:1], elems[2:]...)
	v.SetElements(elems)
	output := doc.String()
	doc2, _ := ParseString(output)
	val, _, _ := doc2.Get("arr")
	a := val.([]any)
	if len(a) != 2 || a[0] != int64(1) || a[1] != int64(3) {
		t.Errorf("remove failed: %v", a)
	}
}

func TestArrayCommentsTravel(t *testing.T) {
	input := "arr = [\n  # First\n  1,\n  # Second\n  2,\n  # Third\n  3,\n]\n"
	doc, _ := ParseString(input)
	v := doc.GetValue("arr")
	elems := v.Elements()

	if len(elems) != 3 {
		t.Fatalf("Elements() = %d, want 3", len(elems))
	}

	// Verify comments are associated with elements
	if len(elems[0].BlockComment()) == 0 || elems[0].BlockComment()[0] != "# First" {
		t.Errorf("elem 0 comment = %v", elems[0].BlockComment())
	}
	if len(elems[1].BlockComment()) == 0 || elems[1].BlockComment()[0] != "# Second" {
		t.Errorf("elem 1 comment = %v", elems[1].BlockComment())
	}

	// Swap — comments travel
	elems[0], elems[2] = elems[2], elems[0]
	v.SetElements(elems)

	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v\noutput:\n%s", err, output)
	}
	val, _, _ := doc2.Get("arr")
	a := val.([]any)
	if a[0] != int64(3) || a[2] != int64(1) {
		t.Errorf("swap values wrong: %v", a)
	}
}

func TestArrayDeleteWithComments(t *testing.T) {
	input := "arr = [\n  # Keep this\n  1,\n  # Remove this\n  2,\n  # Also keep\n  3,\n]\n"
	doc, _ := ParseString(input)
	v := doc.GetValue("arr")
	elems := v.Elements()
	// Remove element 1 (value=2, comment="# Remove this")
	elems = append(elems[:1], elems[2:]...)
	v.SetElements(elems)

	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v", err)
	}
	val, _, _ := doc2.Get("arr")
	a := val.([]any)
	if len(a) != 2 || a[0] != int64(1) || a[1] != int64(3) {
		t.Errorf("delete failed: %v", a)
	}
}

func TestArrayAppendWithComment(t *testing.T) {
	doc, _ := ParseString("arr = [1, 2]\n")
	v := doc.GetValue("arr")
	elems := v.Elements()

	newElem, _ := NewCommentedElement(3, []string{"# new entry"}, "# important")
	elems = append(elems, newElem)
	v.SetElements(elems)

	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v\noutput:\n%s", err, output)
	}
	val, _, _ := doc2.Get("arr")
	a := val.([]any)
	if len(a) != 3 || a[2] != int64(3) {
		t.Errorf("append with comment failed: %v", a)
	}
}

// --- Inline table fields ---

func TestInlineTableFields(t *testing.T) {
	doc, _ := ParseString("person = { name = \"Alice\", age = 30 }\n")
	v := doc.GetValue("person")
	fields := v.Fields()
	if len(fields) != 2 {
		t.Fatalf("Fields() = %d, want 2", len(fields))
	}
	if fields[0].Key() != "name" || fields[1].Key() != "age" {
		t.Errorf("keys = [%s, %s]", fields[0].Key(), fields[1].Key())
	}
}

func TestSwapInlineTableFields(t *testing.T) {
	doc, _ := ParseString("person = { name = \"Alice\", age = 30 }\n")
	v := doc.GetValue("person")
	fields := v.Fields()
	fields[0], fields[1] = fields[1], fields[0]
	v.SetFields(fields)
	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v", err)
	}
	val, _, _ := doc2.Get("person")
	tbl := val.(map[string]any)
	if tbl["name"] != "Alice" || tbl["age"] != int64(30) {
		t.Errorf("values wrong after swap: %v", tbl)
	}
}

// --- GetValue ---

func TestGetValueString(t *testing.T) {
	doc, _ := ParseString("s = \"hello\"\n")
	v := doc.GetValue("s")
	if v.Type() != StringVal {
		t.Errorf("Type() = %d, want StringVal", v.Type())
	}
	if v.AsString() != "hello" {
		t.Errorf("AsString() = %q", v.AsString())
	}
}

func TestGetValueInt(t *testing.T) {
	doc, _ := ParseString("n = 42\n")
	v := doc.GetValue("n")
	if v.Type() != IntVal {
		t.Errorf("Type() = %d, want IntVal", v.Type())
	}
	if v.AsInt() != 42 {
		t.Errorf("AsInt() = %d", v.AsInt())
	}
}

func TestGetValueBool(t *testing.T) {
	doc, _ := ParseString("b = true\n")
	v := doc.GetValue("b")
	if v.Type() != BoolVal {
		t.Errorf("Type() = %d, want BoolVal", v.Type())
	}
	if v.AsBool() != true {
		t.Errorf("AsBool() = %v", v.AsBool())
	}
}

func TestGetValueNonExistent(t *testing.T) {
	doc, _ := ParseString("a = 1\n")
	v := doc.GetValue("nope")
	if v != nil {
		t.Error("GetValue for nonexistent should return nil")
	}
}

func TestGetValueSection(t *testing.T) {
	doc, _ := ParseString("[sec]\na = 1\n")
	v := doc.GetValue("sec")
	if v != nil {
		t.Error("GetValue for section should return nil")
	}
}

// --- SetInlineComment via GetValue ---

func TestSetInlineCommentViaGetValue(t *testing.T) {
	doc, _ := ParseString("[s]\nport = 8080\n")
	v := doc.GetValue("s.port")
	v.SetInlineComment("# HTTP")
	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v", err)
	}
	val, _, _ := doc2.Get("s.port")
	if val != int64(8080) {
		t.Errorf("value changed: %v", val)
	}
}

// --- Global section ---

func TestGlobalSection(t *testing.T) {
	doc, _ := ParseString("name = \"app\"\nversion = 1\n\n[server]\nhost = \"localhost\"\n")
	global := doc.Global()
	entries := global.Entries()
	if len(entries) != 2 {
		t.Fatalf("Global entries = %d, want 2", len(entries))
	}
	if entries[0].Key() != "name" || entries[1].Key() != "version" {
		t.Errorf("keys = [%s, %s]", entries[0].Key(), entries[1].Key())
	}
}

func TestSwapGlobalEntries(t *testing.T) {
	doc, _ := ParseString("name = \"app\"\nversion = 1\n")
	global := doc.Global()
	entries := global.Entries()
	entries[0], entries[1] = entries[1], entries[0]
	global.SetEntries(entries)
	wantGolden(t, doc.String(), "version = 1\nname = \"app\"\n")
}
