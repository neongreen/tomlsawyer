package toml_test

import (
	"fmt"

	toml "github.com/neongreen/tomlsawyer"
)

func ExampleDocument_Get() {
	doc, _ := toml.ParseString(`
[server]
host = "localhost"
port = 8080
`)
	host, _ := doc.Get("server.host")
	fmt.Println(host)
	// Output: localhost
}

func ExampleDocument_Set() {
	doc, _ := toml.ParseString(`
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
	doc, _ := toml.ParseString(`
[users]
alice = { role = "admin" }
bob = { role = "user" }
`)
	keys, _ := doc.Keys("users")
	fmt.Println(keys)
	// Output: [alice bob]
}

func ExampleDocument_Keys_quotedKeys() {
	doc, _ := toml.ParseString(`
[aliases]
"." = "status"
".." = "show @-"
l = "log"
`)
	keys, _ := doc.Keys("aliases")
	for _, k := range keys {
		val, _ := doc.Get(fmt.Sprintf(`aliases."%s"`, k))
		fmt.Printf("%s = %s\n", k, val)
	}
	// Output:
	// . = status
	// .. = show @-
	// l = log
}

func ExampleDocument_Has() {
	doc, _ := toml.ParseString(`
[server]
host = "localhost"
`)
	fmt.Println(doc.Has("server"))
	fmt.Println(doc.Has("server.host"))
	fmt.Println(doc.Has("nonexistent"))
	// Output:
	// true
	// true
	// false
}

func ExampleDocument_TopLevelKeys() {
	doc, _ := toml.ParseString(`
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
