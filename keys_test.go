package toml

import (
	"slices"
	"testing"
)

func TestKeysBasicSection(t *testing.T) {
	input := `
[foo]
a = 1
b = 2
c = 3
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("foo")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 3 {
		t.Fatalf("Keys() returned %d keys, want 3", len(keys))
	}

	for _, expected := range []string{"a", "b", "c"} {
		if !slices.Contains(keys, expected) {
			t.Errorf("Keys() missing %q, got %v", expected, keys)
		}
	}
}

func TestKeysIssue337(t *testing.T) {
	// Exact scenario from https://github.com/neongreen/mono/issues/337
	input := `
[foo]
a = { id = "123" }
b = { id = "456" }
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("foo")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("Keys() returned %d keys, want 2; got %v", len(keys), keys)
	}

	if !slices.Contains(keys, "a") || !slices.Contains(keys, "b") {
		t.Errorf("Keys() = %v, want [a, b]", keys)
	}

	// Verify we can use the discovered keys to access values
	for _, key := range keys {
		val, err := doc.Get("foo." + key + ".id")
		if err != nil {
			t.Errorf("Get(foo.%s.id) error = %v", key, err)
		}
		if val == nil {
			t.Errorf("Get(foo.%s.id) returned nil", key)
		}
	}
}

func TestKeysEmptySection(t *testing.T) {
	input := `
[empty]

[notempty]
x = 1
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("empty")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if keys != nil {
		t.Errorf("Keys() for empty section = %v, want nil", keys)
	}
}

func TestKeysNonExistentSection(t *testing.T) {
	input := `
[server]
host = "localhost"
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("nonexistent")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if keys != nil {
		t.Errorf("Keys() for nonexistent section = %v, want nil", keys)
	}
}

func TestKeysInvalidPath(t *testing.T) {
	doc, _ := ParseString("[foo]\na = 1")

	tests := []string{
		"",
		"foo.",
		".foo",
		"foo..bar",
	}

	for _, path := range tests {
		t.Run("path="+path, func(t *testing.T) {
			_, err := doc.Keys(path)
			if err == nil {
				t.Errorf("Keys(%q) should return error", path)
			}
		})
	}
}

func TestKeysInlineTable(t *testing.T) {
	input := `
person = { name = "Alice", age = 30, email = "alice@example.com" }
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("person")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 3 {
		t.Fatalf("Keys() returned %d keys, want 3; got %v", len(keys), keys)
	}

	// Inline table keys should be sorted
	expected := []string{"age", "email", "name"}
	for i, k := range expected {
		if keys[i] != k {
			t.Errorf("Keys()[%d] = %q, want %q", i, keys[i], k)
		}
	}
}

func TestKeysScalarValue(t *testing.T) {
	input := `name = "hello"`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("name")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if keys != nil {
		t.Errorf("Keys() for scalar = %v, want nil", keys)
	}
}

func TestKeysArrayValue(t *testing.T) {
	input := `tags = ["a", "b", "c"]`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("tags")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if keys != nil {
		t.Errorf("Keys() for array = %v, want nil", keys)
	}
}

func TestKeysManyKeys(t *testing.T) {
	input := `
[config]
a = 1
b = 2
c = 3
d = 4
e = 5
f = 6
g = 7
h = 8
i = 9
j = 10
k = 11
l = 12
m = 13
n = 14
o = 15
p = 16
q = 17
r = 18
s = 19
t_key = 20
u = 21
v = 22
w = 23
x = 24
y = 25
z = 26
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("config")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(keys) != 26 {
		t.Errorf("Keys() returned %d keys, want 26", len(keys))
	}
}
