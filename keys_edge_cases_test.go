package tomlsawyer

import (
	"fmt"
	"slices"
	"testing"
)

func TestKeysTopLevelKeys(t *testing.T) {
	// Keys at the document root (no section) — not supported by Keys() since
	// there's no path to pass for the global section. This test documents that
	// calling Keys on a non-section top-level key returns nil.
	input := `
name = "test"
version = 1
enabled = true
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	// "name" is a scalar, not a section
	keys, err := doc.Keys("name")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}
	if keys != nil {
		t.Errorf("Keys(name) = %v, want nil for scalar", keys)
	}
}

func TestKeysBooleanValues(t *testing.T) {
	input := `
[flags]
a = true
b = false
c = true
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("flags")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 3 {
		t.Fatalf("Keys() = %v, want 3 keys", keys)
	}
}

func TestKeysMixedValueTypes(t *testing.T) {
	input := `
[mixed]
str = "hello"
num = 42
float_val = 3.14
bool_val = true
arr = [1, 2, 3]
inline = { x = 1 }
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("mixed")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	expected := []string{"str", "num", "float_val", "bool_val", "arr", "inline"}
	if len(keys) != len(expected) {
		t.Fatalf("Keys() returned %d keys, want %d; got %v", len(keys), len(expected), keys)
	}

	for _, e := range expected {
		if !slices.Contains(keys, e) {
			t.Errorf("Keys() missing %q, got %v", e, keys)
		}
	}
}

func TestKeysSingleKey(t *testing.T) {
	input := `
[single]
only_key = "value"
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("single")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 1 || keys[0] != "only_key" {
		t.Errorf("Keys() = %v, want [only_key]", keys)
	}
}

func TestKeysWithComments(t *testing.T) {
	input := `
# Section with comments everywhere
[section]
# Comment before a
a = 1  # inline comment
# Comment before b
b = 2
# trailing comment
c = 3
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("section")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 3 {
		t.Fatalf("Keys() = %v, want 3 keys", keys)
	}

	for _, e := range []string{"a", "b", "c"} {
		if !slices.Contains(keys, e) {
			t.Errorf("Keys() missing %q, got %v", e, keys)
		}
	}
}

func TestKeysAfterModification(t *testing.T) {
	input := `
[section]
a = 1
b = 2
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	// Add a new key
	doc.Set("section.c", 3)

	keys, err := doc.Keys("section")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 3 {
		t.Fatalf("Keys() after adding = %v, want 3 keys", keys)
	}
	if !slices.Contains(keys, "c") {
		t.Errorf("Keys() missing newly added 'c', got %v", keys)
	}
}

func TestKeysAfterDeletion(t *testing.T) {
	input := `
[section]
a = 1
b = 2
c = 3
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	doc.Delete("section.b")

	keys, err := doc.Keys("section")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("Keys() after deletion = %v, want 2 keys", keys)
	}
	if slices.Contains(keys, "b") {
		t.Errorf("Keys() still contains deleted 'b', got %v", keys)
	}
}

func TestKeysEmptyInlineTable(t *testing.T) {
	input := `empty = {}`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("empty")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	// Empty inline table — parsed as map with 0 keys
	if keys != nil {
		t.Errorf("Keys() for empty inline table = %v, want nil", keys)
	}
}

func TestKeysPreservesOrder(t *testing.T) {
	// Keys should appear in document order for section items
	input := `
[ordered]
zebra = 1
apple = 2
mango = 3
banana = 4
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("ordered")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	expected := []string{"zebra", "apple", "mango", "banana"}
	if len(keys) != len(expected) {
		t.Fatalf("Keys() = %v, want %v", keys, expected)
	}

	for i, e := range expected {
		if keys[i] != e {
			t.Errorf("Keys()[%d] = %q, want %q (order not preserved)", i, keys[i], e)
		}
	}
}

func TestKeysSiblingSubSectionsNotLeaked(t *testing.T) {
	// Ensure Keys("a") doesn't include keys from [b]
	input := `
[a]
x = 1

[b]
y = 2
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("a")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 1 || keys[0] != "x" {
		t.Errorf("Keys(a) = %v, want [x]", keys)
	}

	if slices.Contains(keys, "y") {
		t.Errorf("Keys(a) leaked key 'y' from section [b]")
	}
}

func TestKeysNoDuplicates(t *testing.T) {
	// If a key appears as both a direct key and via a sub-section name, don't duplicate
	input := `
[parent]
child = "direct"

[parent.other]
x = 1
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("parent")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	seen := make(map[string]int)
	for _, k := range keys {
		seen[k]++
		if seen[k] > 1 {
			t.Errorf("Duplicate key %q in Keys() result", k)
		}
	}
}

func TestKeysDottedKeyStyle(t *testing.T) {
	// When the document uses dotted key style instead of sections
	input := `
server.host = "localhost"
server.port = 8080
database.url = "postgres://localhost"
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("server")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("Keys(server) = %v, want 2 keys", keys)
	}
	if !slices.Contains(keys, "host") {
		t.Errorf("Keys(server) missing 'host', got %v", keys)
	}
	if !slices.Contains(keys, "port") {
		t.Errorf("Keys(server) missing 'port', got %v", keys)
	}
}

func TestKeysLargeSection(t *testing.T) {
	// Build a section with 100 keys
	input := "[big]\n"
	for i := range 100 {
		input += "key_" + itoa(i) + " = " + itoa(i) + "\n"
	}

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("big")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 100 {
		t.Errorf("Keys() returned %d keys, want 100", len(keys))
	}
}

func TestHasSection(t *testing.T) {
	input := `
[server]
host = "localhost"
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if !doc.Has("server") {
		t.Error("Has(server) = false, want true for table section")
	}
	if !doc.Has("server.host") {
		t.Error("Has(server.host) = false, want true")
	}
}

func TestHasDottedPrefix(t *testing.T) {
	input := `server.host = "localhost"`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if !doc.Has("server") {
		t.Error("Has(server) = false, want true for dotted key prefix")
	}
	if !doc.Has("server.host") {
		t.Error("Has(server.host) = false, want true")
	}
}

func TestHasNonExistent(t *testing.T) {
	input := `name = "test"`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if doc.Has("nope") {
		t.Error("Has(nope) = true, want false")
	}
	if doc.Has("name.child") {
		t.Error("Has(name.child) = true, want false")
	}
}

func TestTopLevelKeys(t *testing.T) {
	input := `
name = "test"
server.host = "localhost"
server.port = 8080

[database]
url = "postgres://localhost"

[database.pool]
size = 10
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys := doc.TopLevelKeys()
	if len(keys) != 3 {
		t.Fatalf("TopLevelKeys() = %v, want 3 keys", keys)
	}
	if !slices.Contains(keys, "name") {
		t.Errorf("TopLevelKeys() missing 'name', got %v", keys)
	}
	if !slices.Contains(keys, "server") {
		t.Errorf("TopLevelKeys() missing 'server', got %v", keys)
	}
	if !slices.Contains(keys, "database") {
		t.Errorf("TopLevelKeys() missing 'database', got %v", keys)
	}
}

func TestTopLevelKeysEmpty(t *testing.T) {
	doc, err := ParseString("")
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys := doc.TopLevelKeys()
	if len(keys) != 0 {
		t.Errorf("TopLevelKeys() = %v, want empty", keys)
	}
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
