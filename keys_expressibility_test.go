package toml

import (
	"testing"
)

func TestKeysWithLiteralDotInKeyName(t *testing.T) {
	// A key literally named "foo." should not be confused with path separator
	input := `[section]
"foo." = "value"
bar = "other"
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("section")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("Keys() = %v, want 2 keys", keys)
	}

	// "foo." should appear as a key name, not be split by dot
	found := false
	for _, k := range keys {
		if k == "foo." {
			found = true
		}
	}
	if !found {
		t.Errorf("Keys() = %v, missing literal key 'foo.'", keys)
	}
}

func TestHasLiteralDotInKeyName(t *testing.T) {
	input := `[section]
"foo." = "value"
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	// Has should find "section" as a section
	if !doc.Has("section") {
		t.Error("Has(section) should be true")
	}

	// Access the literal key via quoted path
	val, err := doc.Get(`section."foo."`)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if val != "value" {
		t.Errorf("Get(section.\"foo.\") = %v, want 'value'", val)
	}
}

func TestKeysDoNotConfusePathAndKeySegments(t *testing.T) {
	// key name "a.b" (single key with dot) vs path a.b (two segments)
	input := `[config]
"a.b" = 1
a = { b = 2 }
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("config")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	// Should have two keys: "a.b" (quoted, single key) and "a" (inline table)
	if len(keys) != 2 {
		t.Fatalf("Keys() = %v, want 2 keys", keys)
	}

	// Verify both values are accessible
	val1, _ := doc.Get(`config."a.b"`)
	if val1 == nil {
		t.Error("config.\"a.b\" should be accessible")
	}

	val2, _ := doc.Get("config.a.b")
	if val2 == nil {
		t.Error("config.a.b (nested) should be accessible")
	}
}

func TestKeysEmptyStringKey(t *testing.T) {
	// TOML allows empty quoted keys: "" = "value"
	input := `[section]
"" = "empty key"
normal = "value"
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("section")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("Keys() = %v, want 2 keys", keys)
	}

	// Empty string should appear as a key
	found := false
	for _, k := range keys {
		if k == "" {
			found = true
		}
	}
	if !found {
		t.Errorf("Keys() = %v, missing empty string key", keys)
	}
}

func TestHasDottedKeyPrefix(t *testing.T) {
	// Dotted keys like server.host create an implicit "server" namespace
	input := `server.host = "localhost"
server.port = 8080
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if !doc.Has("server") {
		t.Error("Has(server) should be true for dotted key prefix")
	}

	if !doc.Has("server.host") {
		t.Error("Has(server.host) should be true")
	}

	if doc.Has("server.nonexistent") {
		t.Error("Has(server.nonexistent) should be false")
	}
}

func TestKeysUnicodeKeyNames(t *testing.T) {
	input := `[i18n]
"日本語" = "Japanese"
"中文" = "Chinese"
english = "English"
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("i18n")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 3 {
		t.Fatalf("Keys() = %v, want 3 keys", keys)
	}
}
