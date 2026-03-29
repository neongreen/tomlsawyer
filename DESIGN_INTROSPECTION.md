# Design: Structural Introspection API

## Motivation

tomlsawyer already preserves structure during edits, but provides no way for callers to _query_ that structure. `Get("server.host")` returns `"localhost"` — but was it defined as a dotted key `server.host = "localhost"` in the global section, or as `host = "localhost"` under a `[server]` heading? Is a table value `{ x = 1 }` (inline) or `[foo]\nx = 1` (section)? What quote style does a string use? Are there comments attached?

This matters for tools that need to make format-aware decisions: linters, migration scripts, config diffing tools, and the `Rename` operation proposed below.

## What tomledit already provides

The underlying `tomledit` and `parser` packages expose rich AST information that tomlsawyer currently consumes internally but doesn't surface:

| Information | Where it lives in tomledit |
|---|---|
| Section vs global | `Entry.Section.Heading` — nil for global, non-nil for `[name]` sections |
| Section is array-of-tables | `Heading.IsArray` — true for `[[name]]` |
| Key is inside an inline table | `Entry.IsInline()` — checks parent type |
| Dotted key name | `KeyValue.Name` — multi-element `Key` in global section = dotted key |
| Value is inline table | `Value.X` is `parser.Inline` |
| Value is array | `Value.X` is `parser.Array` |
| String quote style | `Token.Type` — `scanner.String` (basic `"`), `scanner.LString` (literal `'`), `scanner.MString` (multiline `"""`), `scanner.MLString` (multiline `'''`) |
| Token type (int, float, bool, datetime, etc.) | `Token.Type` — `scanner.Integer`, `scanner.Float`, `scanner.Word` (for bools), `scanner.DateTime`, etc. |
| Block comment above a key | `KeyValue.Block` — `parser.Comments` (slice of strings) |
| Inline/trailing comment on a value | `Value.Trailer` — string |
| Block comment above a section heading | `Heading.Block` — `parser.Comments` |
| Trailing comment on a section heading | `Heading.Trailer` — string |
| Source line number | `KeyValue.Line`, `Value.Line`, `Heading.Line` |
| Raw text of a token value | `Token.String()` — returns the original text including quotes |

All of this is accessible through the `*tomledit.Entry` returned by `doc.First(keys...)`, which tomlsawyer already uses internally.

## Proposed API

### Core type: `KeyInfo`

Inspired by gjson's `Result` (which bundles `Type`, `Raw`, `Str`, `Num`, `Index` together), we define a single struct that bundles both the Go value and its structural metadata:

```go
// KeyInfo describes how a key-value pair is represented in the TOML document.
// It is the structural counterpart to the value returned by Get().
type KeyInfo struct {
    // Exists is true if the key was found in the document.
    Exists bool

    // Value is the parsed Go value (same as what Get() returns).
    // nil if the key doesn't exist or points to a section rather than a value.
    Value any

    // Raw is the original text of the value as it appears in the TOML source,
    // including quotes for strings. E.g. `"hello"`, `'literal'`, `42`.
    Raw string

    // ValueType describes the TOML type of the value.
    ValueType ValueType

    // StringStyle describes the quote style, if the value is a string.
    // Zero value if not a string.
    StringStyle StringStyle

    // TableStyle describes how a table-typed key is represented.
    // Only meaningful when ValueType is ValueInlineTable or the key
    // refers to a section heading.
    TableStyle TableStyle

    // Key describes how the key itself is written.
    Key KeyStyle

    // Comments holds the comments associated with this key.
    Comments CommentInfo

    // Line is the 1-based source line where this key-value was defined.
    // Zero if unknown or the key doesn't exist.
    Line int
}
```

### Enums

```go
type ValueType int

const (
    ValueNone         ValueType = iota // key doesn't exist or is a section heading
    ValueString                        // basic, literal, or multiline string
    ValueInteger                       // integer
    ValueFloat                         // float
    ValueBool                          // boolean
    ValueDateTime                      // offset date-time
    ValueLocalDate                     // local date
    ValueLocalTime                     // local time
    ValueLocalDateTime                 // local date-time
    ValueArray                         // array value [...]
    ValueInlineTable                   // inline table { ... }
)
```

```go
type StringStyle int

const (
    StringNone      StringStyle = iota // not a string
    StringBasic                        // "double quoted"
    StringLiteral                      // 'single quoted'
    StringMultiline                    // """multiline basic"""
    StringMultilineLiteral             // '''multiline literal'''
)
```

```go
type TableStyle int

const (
    TableNone    TableStyle = iota // not a table
    TableSection                   // [foo.bar] section heading
    TableArray                     // [[foo.bar]] array of tables
    TableInline                    // { key = value, ... }
)
```

```go
type KeyStyle struct {
    // Dotted is true if the key is written as a dotted key in its section.
    // E.g. `server.host = "localhost"` in the global section has Dotted=true,
    // while `host = "localhost"` under [server] has Dotted=false.
    Dotted bool

    // Section is the heading name of the section containing this key,
    // or nil if it's in the global section.
    // E.g. for `host = "localhost"` under [server], Section is ["server"].
    Section []string

    // LocalName is the key's name within its section (the left-hand side
    // of the `=`). For dotted keys this is the full dotted name.
    // E.g. for `server.host = "x"` in global, LocalName is ["server", "host"].
    // For `host = "x"` under [server], LocalName is ["host"].
    LocalName []string

    // IsInline is true if this key is inside an inline table value.
    IsInline bool
}
```

```go
type CommentInfo struct {
    // Block is the block comment immediately above the key-value pair.
    // Each string is one comment line (including the # prefix).
    // Empty if there is no block comment.
    Block []string

    // Inline is the trailing comment on the same line as the value.
    // Empty string if there is no inline comment.
    Inline string

    // HeadingBlock is the block comment above the section heading,
    // if this key refers to a section. Empty otherwise.
    HeadingBlock []string

    // HeadingInline is the trailing comment on the section heading line.
    HeadingInline string
}
```

### Methods

```go
// Inspect returns structural metadata about the key at the given path.
// Unlike Get, which returns only the value, Inspect returns how the key
// is represented in the document.
//
// If the path doesn't exist, returns a KeyInfo with Exists=false.
// If the path refers to a section heading (e.g. "server" when [server]
// exists), returns info about the section itself.
func (d *Document) Inspect(path string) (KeyInfo, error)
```

This is the primary new method. It parallels `Get` the same way gjson's `Result` parallels a plain `interface{}` return.

### Convenience accessors on `KeyInfo`

```go
// String returns the value as a string, or "" if not a string.
func (ki KeyInfo) String() string

// Int returns the value as int64, or 0.
func (ki KeyInfo) Int() int64

// Float returns the value as float64, or 0.
func (ki KeyInfo) Float() float64

// Bool returns the value as bool, or false.
func (ki KeyInfo) Bool() bool

// IsSection reports whether this key refers to a table section heading
// rather than a key-value pair.
func (ki KeyInfo) IsSection() bool

// IsDottedKey reports whether this key-value is expressed as a dotted key
// in its containing section (as opposed to being a simple key under a
// [section] heading).
func (ki KeyInfo) IsDottedKey() bool
```

## Usage Examples

### Discovering key representation style

```go
doc, _ := toml.ParseString(`
server.host = "localhost"

[server]
port = 8080
`)

info, _ := doc.Inspect("server.host")
fmt.Println(info.Key.Dotted)        // true — it's a dotted key in global
fmt.Println(info.Key.Section)       // nil — it's in the global section
fmt.Println(info.Key.LocalName)     // ["server", "host"]
fmt.Println(info.StringStyle)       // StringBasic

info2, _ := doc.Inspect("server.port")
fmt.Println(info2.Key.Dotted)       // false — simple key under [server]
fmt.Println(info2.Key.Section)      // ["server"]
fmt.Println(info2.Key.LocalName)    // ["port"]
```

### Checking quote style

```go
doc, _ := toml.ParseString(`
name = "Alice"
path = 'C:\Users\alice'
bio = """
Multi-line bio.
"""
`)

info, _ := doc.Inspect("name")
fmt.Println(info.StringStyle)  // StringBasic

info2, _ := doc.Inspect("path")
fmt.Println(info2.StringStyle) // StringLiteral

info3, _ := doc.Inspect("bio")
fmt.Println(info3.StringStyle) // StringMultiline
```

### Inspecting comments

```go
doc, _ := toml.ParseString(`
# Server configuration
# Handles all incoming requests
[server]
host = "localhost"  # bind address
# Port must be > 1024
port = 8080
`)

info, _ := doc.Inspect("server")
fmt.Println(info.Comments.HeadingBlock)
// ["# Server configuration", "# Handles all incoming requests"]

info2, _ := doc.Inspect("server.host")
fmt.Println(info2.Comments.Inline)  // "# bind address"

info3, _ := doc.Inspect("server.port")
fmt.Println(info3.Comments.Block)   // ["# Port must be > 1024"]
```

### Distinguishing inline tables from sections

```go
doc, _ := toml.ParseString(`
inline = { x = 1, y = 2 }

[section]
x = 1
y = 2
`)

info, _ := doc.Inspect("inline")
fmt.Println(info.ValueType)   // ValueInlineTable
fmt.Println(info.TableStyle)  // TableInline

info2, _ := doc.Inspect("section")
fmt.Println(info2.TableStyle) // TableSection
```

### Raw value access

```go
doc, _ := toml.ParseString(`count = 0x1F`)

info, _ := doc.Inspect("count")
fmt.Println(info.Value)      // 31 (int64)
fmt.Println(info.Raw)        // "0x1F"
fmt.Println(info.ValueType)  // ValueInteger
```

## Rename Operation

### Motivation

Renaming a key should preserve its structural representation. If `[foo.bar]` is renamed to `[foo.qux]`, the result should be `[foo.qux]`, not `[foo]\nqux = ...` or a flattened dotted key. Similarly, renaming a dotted key should keep it dotted, and renaming within an inline table should keep it inline.

### What tomledit already provides

`transform.Rename` does exactly this at the section/key level — it modifies the heading name or key name in-place without moving anything. However, it only renames exact key matches. If you rename `[foo.bar]` to `[foo.qux]`, any sub-keys like `[foo.bar.baz]` are not updated.

### Proposed API

```go
// Rename renames a key from oldPath to newPath, preserving the key's
// section style, comments, position, and all other structural properties.
//
// For section headings, all sub-sections that have oldPath as a prefix
// are also renamed. E.g. renaming "foo.bar" to "foo.qux" also renames
// [foo.bar.baz] to [foo.qux.baz].
//
// Rename does not move the key — it only changes its name. The key stays
// in the same position in the document.
//
// Returns an error if oldPath doesn't exist.
func (d *Document) Rename(oldPath, newPath string) error
```

### Design considerations for Rename

**Section cascade**: When renaming a section like `[foo.bar]`, all child sections (`[foo.bar.x]`, `[foo.bar.x.y]`, etc.) must have their headings updated too. tomledit's `transform.Rename` only handles a single key, so we'd need to iterate through `doc.Sections` and update any heading where the old key is a prefix.

**Dotted key handling**: If `server.host = "x"` exists as a dotted key in the global section and we rename `server` to `app`, the key should become `app.host = "x"`. This means replacing a prefix in `KeyValue.Name`.

**Cross-style rename**: What happens when renaming `server.host` (dotted key) to `database.url`? The section prefix changes. We should keep the same structural style: if it was dotted, it stays dotted. If it was under a section, we'd need to move it to a different section (or create one). This is actually a _move_ operation, not just a rename. For the initial implementation, `Rename` could be restricted to same-depth renames where only the final key segment changes, or renames within the same section.

### Rename examples

```go
// Renaming a section heading
doc, _ := toml.ParseString(`
[foo.bar]
x = 1

[foo.bar.baz]
y = 2
`)
doc.Rename("foo.bar", "foo.qux")
// Result:
// [foo.qux]
// x = 1
//
// [foo.qux.baz]
// y = 2

// Renaming a simple key under a section
doc2, _ := toml.ParseString(`
[server]
# Bind address
host = "localhost"  # change this in production
`)
doc2.Rename("server.host", "server.bind_address")
// Result:
// [server]
// # Bind address
// bind_address = "localhost"  # change this in production

// Renaming a dotted key
doc3, _ := toml.ParseString(`
server.host = "localhost"
server.port = 8080
`)
doc3.Rename("server.host", "server.bind")
// Result:
// server.bind = "localhost"
// server.port = 8080
```

## Interaction with existing API

`Inspect` is read-only and doesn't affect `Get`, `Set`, `Delete`, `Has`, or `Keys`. It's a parallel access path:

| Existing | New |
|---|---|
| `Get(path) → (any, error)` | `Inspect(path) → (KeyInfo, error)` |
| `Has(path) → bool` | `Inspect(path).Exists` |
| `Set(path, value)` | (unchanged — uses Inspect info internally to preserve style) |
| `Delete(path)` | (unchanged) |
| `Keys(path)` | (unchanged) |

The return type is a value struct (`KeyInfo`), not a pointer, following gjson's `Result` pattern. This makes it safe to store and compare without worrying about the caller mutating document internals.

## Implementation notes

`Inspect` would be implemented by:

1. Call `parseKeyPath(path)` (already exists)
2. Call `d.doc.First(keys...)` (already exists)
3. If entry is nil, return `KeyInfo{Exists: false}`
4. If entry is a section (`entry.KeyValue == nil`), populate from `entry.Section.Heading`
5. If entry is a key-value, populate from `entry.KeyValue` and `entry.Value`:
   - `Raw` = `entry.Value.X.String()`
   - `ValueType` = map from `Token.Type` (via type switch on `entry.Value.X`)
   - `StringStyle` = map from `scanner.String`/`LString`/`MString`/`MLString`
   - `Key.Section` = `entry.Section.TableName()`
   - `Key.Dotted` = `len(entry.KeyValue.Name) > 1` when in a section
   - `Key.LocalName` = `entry.KeyValue.Name`
   - `Key.IsInline` = `entry.IsInline()`
   - `Comments.Block` = `entry.KeyValue.Block`
   - `Comments.Inline` = `entry.Value.Trailer`
   - `Line` = `entry.KeyValue.Line`
6. Parse the value with the existing `parseValue()` for the `Value` field

The scanner token type is accessible through `Token.Type` on the `parser.Token` concrete type, which is the `Datum` inside `parser.Value`. tomlsawyer already does the type switch on `v.X.(parser.Token)` in `parseValue` — `Inspect` would do the same switch but extract additional metadata.

`Rename` would be implemented by:

1. Parse both paths
2. Find the entry via `doc.First()`
3. If it's a section, update `Heading.Name` and iterate `doc.Sections` to cascade prefix changes
4. If it's a key-value, update `KeyValue.Name` (adjusting for dotted key prefix changes)

## Open questions

1. **Should `Inspect` on a section return info about keys defined in that section?** Currently proposed to return just the section heading metadata. An alternative would be to include a `Children []string` or similar, but `Keys()` already covers that.

2. **Should `KeyInfo` include the raw text of the key name itself?** E.g., whether the key was written as `"host"` (quoted) vs `host` (bare). The current `parser.Key` normalizes to plain strings, losing this info. We could add a `RawKey string` field if there's demand, but this would require reaching into the scanner layer.

3. **Should `Rename` support cross-section moves?** E.g., renaming `server.host` to `database.url` where the section prefix changes. This is complex and arguably a different operation (move). Initial implementation could return an error for cross-section renames and offer a separate `Move` method later.

4. **Should there be an `InspectAll` or `Scan` that visits every key with its `KeyInfo`?** This would be useful for linting/analysis tools. The underlying `doc.Scan()` already supports this pattern.

5. **Should `CommentInfo.Block` include the raw `#` prefix or strip it?** tomledit's `Comments` type stores them with the `#` prefix. Preserving them as-is is simpler and lossless; callers can strip if needed.

6. **How should `Inspect` handle array-of-tables (`[[name]]`)?** There can be multiple sections with the same heading. Should it return info about the first one? All of them? The underlying `doc.Find(keys...)` returns all matches; we could offer both `Inspect` (first) and `InspectAll` (all matches).

7. **Naming: `Inspect` vs `Describe` vs `Info` vs `Meta`?** `Inspect` parallels the concept well and doesn't collide with existing method names. `Describe` is also reasonable. `GetInfo` follows the `Get` naming but is longer.
