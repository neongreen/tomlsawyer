package tomlsawyer

import (
	"fmt"
	"slices"
	"testing"
)

func TestKeysWorkflowDiscoverAndModify(t *testing.T) {
	// The exact workflow from the issue: discover keys, match on values, then edit
	input := `
[foo]
a = { id = "123", name = "Alice" }
b = { id = "456", name = "Bob" }
c = { id = "789", name = "Charlie" }
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("foo")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	// Find the entry with id "456" and update it
	for _, key := range keys {
		val, err := doc.Get(fmt.Sprintf("foo.%s.id", key))
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if val == "456" {
			err = doc.Set(fmt.Sprintf("foo.%s.id", key), "789")
			if err != nil {
				t.Fatalf("Set() error = %v", err)
			}
			break
		}
	}

	// Verify the update
	val, _ := doc.Get("foo.b.id")
	if val != "789" {
		t.Errorf("After update, foo.b.id = %v, want 789", val)
	}

	// Verify other entries untouched
	val, _ = doc.Get("foo.a.id")
	if val != "123" {
		t.Errorf("foo.a.id = %v, want 123", val)
	}
}

func TestKeysWorkflowIterateAllSections(t *testing.T) {
	input := `
[servers.web]
host = "10.0.0.1"
port = 80

[servers.api]
host = "10.0.0.2"
port = 8080

[servers.db]
host = "10.0.0.3"
port = 5432
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	serverNames, err := doc.Keys("servers")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if len(serverNames) != 3 {
		t.Fatalf("Keys(servers) = %v, want 3 servers", serverNames)
	}

	// Collect all hosts
	hosts := make(map[string]string)
	for _, name := range serverNames {
		host, err := doc.Get(fmt.Sprintf("servers.%s.host", name))
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		hosts[name] = host.(string)
	}

	if hosts["web"] != "10.0.0.1" {
		t.Errorf("web host = %s, want 10.0.0.1", hosts["web"])
	}
	if hosts["api"] != "10.0.0.2" {
		t.Errorf("api host = %s, want 10.0.0.2", hosts["api"])
	}
	if hosts["db"] != "10.0.0.3" {
		t.Errorf("db host = %s, want 10.0.0.3", hosts["db"])
	}
}

func TestKeysWorkflowAddAndDiscover(t *testing.T) {
	doc, err := ParseString("")
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	// Build a document programmatically
	doc.Set("users.alice.email", "alice@example.com")
	doc.Set("users.alice.role", "admin")
	doc.Set("users.bob.email", "bob@example.com")
	doc.Set("users.bob.role", "user")

	// Discover the users
	keys, err := doc.Keys("users")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	if !slices.Contains(keys, "alice") || !slices.Contains(keys, "bob") {
		t.Errorf("Keys(users) = %v, want [alice, bob]", keys)
	}
}

func TestKeysWorkflowRoundTrip(t *testing.T) {
	input := `
# User configuration
[users]
admin = { name = "Admin", level = 10 }
guest = { name = "Guest", level = 1 }
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("users")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	// Serialize and re-parse
	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("Round-trip parse error = %v", err)
	}

	keys2, err := doc2.Keys("users")
	if err != nil {
		t.Fatalf("Keys() after round-trip error = %v", err)
	}

	if len(keys) != len(keys2) {
		t.Errorf("Keys changed after round-trip: %v -> %v", keys, keys2)
	}

	for _, k := range keys {
		if !slices.Contains(keys2, k) {
			t.Errorf("Key %q lost after round-trip", k)
		}
	}
}

func TestKeysWorkflowDeleteByDiscovery(t *testing.T) {
	input := `
[cache]
session = { ttl = 3600 }
tokens = { ttl = 86400 }
temp = { ttl = 60 }
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("cache")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	// Delete all cache entries with ttl < 3600
	for _, key := range keys {
		ttl, _ := doc.Get(fmt.Sprintf("cache.%s.ttl", key))
		if ttl.(int64) < 3600 {
			doc.Delete(fmt.Sprintf("cache.%s", key))
		}
	}

	// Only "session" and "tokens" should remain
	remaining, _ := doc.Keys("cache")
	if slices.Contains(remaining, "temp") {
		t.Errorf("temp should have been deleted, got %v", remaining)
	}
	if !slices.Contains(remaining, "session") {
		t.Errorf("session should remain, got %v", remaining)
	}
}

func TestKeysWorkflowBulkUpdate(t *testing.T) {
	input := `
[features]
dark_mode = false
notifications = false
beta_access = false
analytics = true
`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	keys, err := doc.Keys("features")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}

	// Enable all features
	for _, key := range keys {
		doc.Set("features."+key, true)
	}

	// Verify all are now true
	for _, key := range keys {
		val, _ := doc.Get("features." + key)
		if val != true {
			t.Errorf("features.%s = %v, want true", key, val)
		}
	}
}
