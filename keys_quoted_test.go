package tomlsawyer

import (
	"slices"
	"testing"
)

func TestKeysQuotedKeys(t *testing.T) {
	input := `
[aliases]
"." = "status"
".." = "show @-"
"..." = "show @--"
l = "log"
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("aliases")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 4 {
		t.Fatalf("Keys() returned %d keys, want 4; got %v", len(keys), keys)
	}

	for _, expected := range []string{".", "..", "...", "l"} {
		if !slices.Contains(keys, expected) {
			t.Errorf("Keys() missing %q, got %v", expected, keys)
		}
	}
}

func TestKeysQuotedSectionName(t *testing.T) {
	input := `
["section:with:colons"]
key1 = "value1"
key2 = "value2"
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys(`"section:with:colons"`)
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("Keys() = %v, want 2 keys", keys)
	}
	if !slices.Contains(keys, "key1") || !slices.Contains(keys, "key2") {
		t.Errorf("Keys() = %v, want [key1, key2]", keys)
	}
}

func TestKeysWithSpacesInKeyNames(t *testing.T) {
	input := `
[config]
"key with spaces" = "value1"
"another spaced key" = "value2"
normal_key = "value3"
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("config")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 3 {
		t.Fatalf("Keys() returned %d, want 3; got %v", len(keys), keys)
	}

	for _, expected := range []string{"key with spaces", "another spaced key", "normal_key"} {
		if !slices.Contains(keys, expected) {
			t.Errorf("Keys() missing %q, got %v", expected, keys)
		}
	}
}

func TestKeysWithDashesAndColons(t *testing.T) {
	input := `
[tools]
"key.with.dots" = "v1"
key-with-dashes = "v2"
"key:with:colons" = "v3"
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("tools")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 3 {
		t.Fatalf("Keys() returned %d, want 3; got %v", len(keys), keys)
	}

	for _, expected := range []string{"key.with.dots", "key-with-dashes", "key:with:colons"} {
		if !slices.Contains(keys, expected) {
			t.Errorf("Keys() missing %q, got %v", expected, keys)
		}
	}
}

func TestKeysQuotedSubSections(t *testing.T) {
	input := `
[parent]

[parent."child.with.dots"]
x = 1

[parent."child with spaces"]
y = 2
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("parent")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("Keys(parent) = %v, want 2 keys", keys)
	}

	if !slices.Contains(keys, "child.with.dots") || !slices.Contains(keys, "child with spaces") {
		t.Errorf("Keys(parent) = %v, want [child.with.dots, child with spaces]", keys)
	}
}
