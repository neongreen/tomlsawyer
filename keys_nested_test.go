package toml

import (
	"slices"
	"testing"
)

func TestKeysNestedSection(t *testing.T) {
	input := `
[database]
host = "localhost"
port = 5432

[database.connection]
max_retries = 5
timeout = 30
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	// Keys of top-level section should include both direct keys and sub-section names
	keys, err := doc.Keys("database")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if !slices.Contains(keys, "host") {
		t.Errorf("Keys(database) missing 'host', got %v", keys)
	}
	if !slices.Contains(keys, "port") {
		t.Errorf("Keys(database) missing 'port', got %v", keys)
	}
	if !slices.Contains(keys, "connection") {
		t.Errorf("Keys(database) missing 'connection' sub-section, got %v", keys)
	}

	// Keys of nested section
	connKeys, err := doc.Keys("database.connection")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(connKeys) != 2 {
		t.Fatalf("Keys(database.connection) returned %d keys, want 2; got %v", len(connKeys), connKeys)
	}
	if !slices.Contains(connKeys, "max_retries") || !slices.Contains(connKeys, "timeout") {
		t.Errorf("Keys(database.connection) = %v, want [max_retries, timeout]", connKeys)
	}
}

func TestKeysDeeplyNested(t *testing.T) {
	input := `
[a]
x = 1

[a.b]
y = 2

[a.b.c]
z = 3

[a.b.c.d]
w = 4
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("a")
	if err != nil {
		t.Fatalf("Keys(a) error = %v", err)
	}
	if !slices.Contains(keys, "x") || !slices.Contains(keys, "b") {
		t.Errorf("Keys(a) = %v, want to contain x and b", keys)
	}

	keys, err = doc.Keys("a.b")
	if err != nil {
		t.Fatalf("Keys(a.b) error = %v", err)
	}
	if !slices.Contains(keys, "y") || !slices.Contains(keys, "c") {
		t.Errorf("Keys(a.b) = %v, want to contain y and c", keys)
	}

	keys, err = doc.Keys("a.b.c")
	if err != nil {
		t.Fatalf("Keys(a.b.c) error = %v", err)
	}
	if !slices.Contains(keys, "z") || !slices.Contains(keys, "d") {
		t.Errorf("Keys(a.b.c) = %v, want to contain z and d", keys)
	}

	keys, err = doc.Keys("a.b.c.d")
	if err != nil {
		t.Fatalf("Keys(a.b.c.d) error = %v", err)
	}
	if len(keys) != 1 || keys[0] != "w" {
		t.Errorf("Keys(a.b.c.d) = %v, want [w]", keys)
	}
}

func TestKeysMultipleSubSections(t *testing.T) {
	input := `
[servers]

[servers.alpha]
ip = "10.0.0.1"
dc = "eqdc10"

[servers.beta]
ip = "10.0.0.2"
dc = "eqdc10"

[servers.gamma]
ip = "10.0.0.3"
dc = "eqdc20"
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("servers")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 3 {
		t.Fatalf("Keys(servers) returned %d keys, want 3; got %v", len(keys), keys)
	}

	for _, expected := range []string{"alpha", "beta", "gamma"} {
		if !slices.Contains(keys, expected) {
			t.Errorf("Keys(servers) missing %q, got %v", expected, keys)
		}
	}

	// Each sub-section should have its own keys
	for _, name := range keys {
		subKeys, err := doc.Keys("servers." + name)
		if err != nil {
			t.Errorf("Keys(servers.%s) error = %v", name, err)
			continue
		}
		if len(subKeys) != 2 {
			t.Errorf("Keys(servers.%s) = %v, want 2 keys", name, subKeys)
		}
	}
}

func TestKeysSubSectionsOnly(t *testing.T) {
	// Section with no direct keys, only sub-sections
	input := `
[parent]

[parent.child1]
a = 1

[parent.child2]
b = 2
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
		t.Fatalf("Keys() = %v, want [child1, child2]", keys)
	}
	if !slices.Contains(keys, "child1") || !slices.Contains(keys, "child2") {
		t.Errorf("Keys() = %v, want [child1, child2]", keys)
	}
}

func TestKeysNestedInlineTable(t *testing.T) {
	input := `
[config]
db = { host = "localhost", port = 5432, options = { ssl = true } }
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("config")
	if err != nil {
		t.Fatalf("Keys(config) error = %v", err)
	}
	if len(keys) != 1 || keys[0] != "db" {
		t.Errorf("Keys(config) = %v, want [db]", keys)
	}

	dbKeys, err := doc.Keys("config.db")
	if err != nil {
		t.Fatalf("Keys(config.db) error = %v", err)
	}
	if len(dbKeys) != 3 {
		t.Errorf("Keys(config.db) = %v, want 3 keys", dbKeys)
	}
}

func TestKeysMixedDirectAndSubSections(t *testing.T) {
	input := `
[app]
name = "myapp"
version = 1

[app.server]
host = "localhost"
port = 8080

[app.database]
url = "postgres://localhost"
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("app")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	expected := []string{"name", "version", "server", "database"}
	if len(keys) != len(expected) {
		t.Fatalf("Keys(app) returned %d keys, want %d; got %v", len(keys), len(expected), keys)
	}

	for _, e := range expected {
		if !slices.Contains(keys, e) {
			t.Errorf("Keys(app) missing %q, got %v", e, keys)
		}
	}
}
