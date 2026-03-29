package tomlsawyer_test

import (
	"fmt"

	"github.com/neongreen/tomlsawyer"
)

func ExampleDocument_Get() {
	doc, _ := tomlsawyer.ParseString(`
[server]
host = "localhost"
port = 8080
`)
	host, _, _ := doc.Get("server.host")
	fmt.Println(host)
	// Output: localhost
}

func ExampleDocument_Set() {
	doc, _ := tomlsawyer.ParseString(`
[server]
host = "localhost"
port = 8080
`)
	doc.Set("server.port", 9090)
	fmt.Print(doc.String())
	// Output:
	// [server]
	// host = "localhost"
	// port = 9090
}

func ExampleDocument_Keys() {
	doc, _ := tomlsawyer.ParseString(`
[users]
alice = { role = "admin" }
bob = { role = "user" }
`)
	keys, _ := doc.Keys("users")
	fmt.Println(keys)
	// Output: [alice bob]
}

func ExampleDocument_Keys_quotedKeys() {
	doc, _ := tomlsawyer.ParseString(`
[aliases]
"." = "status"
".." = "show @-"
l = "log"
`)
	keys, _ := doc.Keys("aliases")
	for _, k := range keys {
		val, _, _ := doc.Get(fmt.Sprintf(`aliases."%s"`, k))
		fmt.Printf("%s = %s\n", k, val)
	}
	// Output:
	// . = status
	// .. = show @-
	// l = log
}

func ExampleDocument_Has() {
	doc, _ := tomlsawyer.ParseString(`
[server]
host = "localhost"
`)
	h1, _ := doc.Has("server")
	h2, _ := doc.Has("server.host")
	h3, _ := doc.Has("nonexistent")
	fmt.Println(h1)
	fmt.Println(h2)
	fmt.Println(h3)
	// Output:
	// true
	// true
	// false
}

func ExampleDocument_TopLevelKeys() {
	doc, _ := tomlsawyer.ParseString(`
name = "myapp"
version = 1

[server]
host = "localhost"

[database]
url = "postgres://localhost"
`)
	keys := doc.TopLevelKeys()
	fmt.Println(keys)
	// Output: [name version server database]
}

func ExampleDocument_Move() {
	doc, _ := tomlsawyer.ParseString(`
[server]
host = "localhost"
port = 8080
`)
	doc.Move("server", "app.server")
	fmt.Print(doc.String())
	// Output:
	// [app.server]
	// host = "localhost"
	// port = 8080
}

func ExampleDocument_Move_crossSection() {
	doc, _ := tomlsawyer.ParseString(`
[old]
timeout = 30

[new]
host = "localhost"
`)
	doc.Move("old.timeout", "new.timeout")
	doc.Prune() // remove empty [old] section
	fmt.Print(doc.String())
	// Output:
	// [new]
	// host = "localhost"
	// timeout = 30
}

func ExampleDocument_ApplyMap() {
	doc, _ := tomlsawyer.ParseString(`name = "myapp"
version = "1.0"
`)
	doc.ApplyMap(map[string]any{
		"version": "2.0",
		"author":  "Alice",
	})
	fmt.Print(doc.String())
	// Output:
	// name = "myapp"
	// version = "2.0"
	// author = "Alice"
}
