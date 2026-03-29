package toml

import (
	"strings"
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
	t.Log("Section-style rename output:\n" + output)

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

	// Document whether section style is preserved.
	// Ideal: output contains [foo.qux] (section style matching the original [foo.bar]).
	if strings.Contains(output, "[foo.qux]") {
		t.Log("✓ Section style preserved — [foo.qux] header created")
	} else {
		t.Log("⚠ Section style NOT preserved — new key was not placed under a [foo.qux] header")
		// The empty [foo.bar] header may also linger.
	}
	if strings.Contains(output, "[foo.bar]") {
		t.Log("⚠ Stale [foo.bar] header remains in output (Delete only removed the key, not the empty section)")
	}
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
	t.Log("Dotted-key rename output:\n" + output)

	// Verify the value was moved
	doc2, err := Parse([]byte(output))
	if err != nil {
		t.Fatalf("output is not valid TOML: %v", err)
	}
	newVal, _ := doc2.Get("app.host")
	if newVal == nil {
		t.Fatal("app.host should exist in output")
	}

	// Document whether dotted style is preserved.
	// Ideal: output uses "app.host = ..." (dotted style) rather than a [app] section.
	if strings.Contains(output, "app.host") && !strings.Contains(output, "[app]") {
		t.Log("✓ Dotted key style preserved for new key")
	} else if strings.Contains(output, "[app]") {
		t.Log("⚠ Dotted key style NOT preserved — new key was placed under [app] section instead")
	}
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
	t.Log("Comment preservation after rename:\n" + output)

	// Verify the value was moved
	doc2, err := Parse([]byte(output))
	if err != nil {
		t.Fatalf("output is not valid TOML: %v", err)
	}
	newVal, _ := doc2.Get("config.timeout")
	if newVal == nil {
		t.Fatal("config.timeout should exist in output")
	}

	// Document whether the inline comment survives.
	// Ideal: "# seconds" appears on the same line as the new key.
	if strings.Contains(output, "# seconds") {
		t.Log("✓ Inline comment preserved after rename")
	} else {
		t.Log("⚠ Inline comment LOST during rename — Get returns a plain value, losing the comment")
	}
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
	t.Logf("Keys under [old]: %v", keys)

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
	t.Log("Multi-key section rename output:\n" + output)

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

	// Document behavior
	if strings.Contains(output, "[new]") {
		t.Log("✓ New [new] section header created")
	} else {
		t.Log("⚠ No [new] section header — keys may be dotted or placed elsewhere")
	}
	if strings.Contains(output, "[old]") {
		t.Log("⚠ Stale [old] header remains in output (Delete only removes keys, not empty sections)")
	} else {
		t.Log("✓ [old] header was cleaned up")
	}
}
