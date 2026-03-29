package toml

import (
	"testing"
)

// These tests document the current behavior of "renaming" a key using the
// Get → Set → Delete pattern. Renaming is not a first-class operation, so
// style and comment preservation may be lossy.

func TestRenameSectionStyleKey(t *testing.T) {
	// Renaming foo.bar.x to foo.qux.x — does the output use [foo.qux] section style?
	input := "[foo.bar]\nx = 1\n"

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	val, err := doc.Get("foo.bar.x")
	if err != nil {
		t.Fatal(err)
	}
	if val == nil {
		t.Fatal("foo.bar.x should exist")
	}

	if err := doc.Set("foo.qux.x", val); err != nil {
		t.Fatal(err)
	}
	if err := doc.Delete("foo.bar.x"); err != nil {
		t.Fatal(err)
	}

	output := doc.String()

	// Verify the value was moved
	doc2, err := Parse([]byte(output))
	if err != nil {
		t.Fatalf("output is not valid TOML: %v", err)
	}
	newVal, _ := doc2.Get("foo.qux.x")
	if newVal == nil {
		t.Fatal("foo.qux.x should exist in output")
	}
	oldVal, _ := doc2.Get("foo.bar.x")
	if oldVal != nil {
		t.Fatal("foo.bar.x should not exist in output")
	}

	// Stale [foo.bar] header remains (Delete removes the key, not the empty section).
	// New key goes under [foo.qux] via Set, which uses section style.
	wantGolden(t, output, `[foo.bar]

[foo]

[foo.qux]
x = 1
`)
}

func TestRenameDottedKey(t *testing.T) {
	// Renaming a dotted key: server.host → app.host
	input := "server.host = \"localhost\"\nserver.port = 8080\n"

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	val, err := doc.Get("server.host")
	if err != nil {
		t.Fatal(err)
	}

	if err := doc.Set("app.host", val); err != nil {
		t.Fatal(err)
	}
	if err := doc.Delete("server.host"); err != nil {
		t.Fatal(err)
	}

	output := doc.String()

	// Verify the value was moved
	doc2, err := Parse([]byte(output))
	if err != nil {
		t.Fatalf("output is not valid TOML: %v", err)
	}
	newVal, _ := doc2.Get("app.host")
	if newVal == nil {
		t.Fatal("app.host should exist in output")
	}

	// Set creates an [app] section rather than preserving dotted-key style.
	wantGolden(t, output, `server.port = 8080

[app]
host = "localhost"
`)
}

func TestRenamePreservesComment(t *testing.T) {
	// Key with an inline comment — does the comment survive a rename?
	input := "[settings]\ntimeout = 30 # seconds\n"

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	val, err := doc.Get("settings.timeout")
	if err != nil {
		t.Fatal(err)
	}

	if err := doc.Set("config.timeout", val); err != nil {
		t.Fatal(err)
	}
	if err := doc.Delete("settings.timeout"); err != nil {
		t.Fatal(err)
	}

	output := doc.String()

	// Verify the value was moved
	doc2, err := Parse([]byte(output))
	if err != nil {
		t.Fatalf("output is not valid TOML: %v", err)
	}
	newVal, _ := doc2.Get("config.timeout")
	if newVal == nil {
		t.Fatal("config.timeout should exist in output")
	}

	// Inline comment "# seconds" is lost: Get returns a plain value.
	wantGolden(t, output, `[settings]

[config]
timeout = 30
`)
}

func TestRenameSectionWithMultipleKeys(t *testing.T) {
	// Rename an entire section [old] → [new] by moving all keys.
	input := "[old]\na = 1\nb = 2\n"

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	keys, err := doc.Keys("old")
	if err != nil {
		t.Fatal(err)
	}

	for _, k := range keys {
		val, err := doc.Get("old." + k)
		if err != nil {
			t.Fatalf("Get old.%s: %v", k, err)
		}
		if err := doc.Set("new."+k, val); err != nil {
			t.Fatalf("Set new.%s: %v", k, err)
		}
		if err := doc.Delete("old." + k); err != nil {
			t.Fatalf("Delete old.%s: %v", k, err)
		}
	}

	output := doc.String()

	// Verify values were moved
	doc2, err := Parse([]byte(output))
	if err != nil {
		t.Fatalf("output is not valid TOML: %v", err)
	}
	for _, k := range keys {
		val, _ := doc2.Get("new." + k)
		if val == nil {
			t.Fatalf("new.%s should exist in output", k)
		}
		oldVal, _ := doc2.Get("old." + k)
		if oldVal != nil {
			t.Fatalf("old.%s should not exist in output", k)
		}
	}

	// Stale [old] header remains (Delete removes keys, not empty sections).
	wantGolden(t, output, `[old]

[new]
a = 1
b = 2
`)
}
