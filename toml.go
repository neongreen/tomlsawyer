// Package tomlsawyer provides comment-preserving TOML parsing and serialization.
//
// This package wraps github.com/creachadair/tomledit to provide a stable,
// user-friendly API for reading, modifying, and writing TOML files while
// preserving comments, declaration order, and structural style choices
// (dotted keys vs sections, quote styles). Whitespace layout may be
// normalized.
//
// # Path Syntax
//
// All methods that accept a path parameter use TOML dotted-key syntax:
// simple keys like "name", nested keys like "server.host", and quoted keys
// for special characters like aliases."." or section."key with spaces".
// Paths are parsed using TOML's own key grammar, so any valid TOML key
// works as a path segment when quoted. Invalid paths like "foo." (trailing
// dot) or ".foo" (leading dot) are rejected.
//
// Path segments and raw key names are different things. A path like
// aliases."." has two segments (aliases and .). The [Document.Keys] method
// returns raw key names, not paths.
//
// Example usage:
//
//	doc, err := tomlsawyer.Parse([]byte(`
//	  # This is a comment
//	  [server]
//	  host = "localhost"
//	  port = 8080
//	`))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Modify a value
//	doc.Set("server.port", 9090)
//
//	// Use quoted keys for special characters
//	doc.Set(`aliases."."`, "status")
//
//	// Write back with comments preserved
//	output := doc.String()
package tomlsawyer

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/creachadair/tomledit"
	"github.com/creachadair/tomledit/parser"
	"github.com/creachadair/tomledit/transform"
)

// ErrNotValue is returned by Get when the path refers to a table section
// rather than a key-value entry.
var ErrNotValue = errors.New("path refers to a table section, not a value")

// Document represents a parsed TOML document with preserved structure,
// including comments, formatting, and declaration order.
type Document struct {
	doc *tomledit.Document
}

// Parse parses a TOML document from bytes and returns a Document that
// preserves all structural information including comments.
func Parse(input []byte) (*Document, error) {
	doc, err := tomledit.Parse(bytes.NewReader(input))
	if err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}
	return &Document{doc: doc}, nil
}

// ParseString is a convenience wrapper around Parse that accepts a string.
func ParseString(input string) (*Document, error) {
	return Parse([]byte(input))
}

// Get retrieves a value at the given path.
// Returns (value, true, nil) when the path exists and is a key-value entry,
// (nil, false, nil) when the path doesn't exist, and
// (nil, false, ErrNotValue) when the path refers to a table section.
//
// The path uses TOML dotted-key syntax: segments are separated by dots.
// Use quoted segments for keys containing special characters:
//
//	doc.Get("server.host")       // key "host" under section "server"
//	doc.Get(`aliases."."`)       // key "." under section "aliases"
func (d *Document) Get(path string) (any, bool, error) {
	keys, err := parseKeyPath(path)
	if err != nil {
		return nil, false, err
	}

	entry := d.doc.First(keys...)
	if entry == nil {
		return nil, false, nil
	}

	if entry.KeyValue == nil {
		return nil, false, ErrNotValue
	}

	val, err := parseValue(entry.Value)
	if err != nil {
		return nil, false, err
	}
	return val, true, nil
}

// Set sets a value at the given path, creating intermediate sections
// if necessary. The value can be a string, int, float64, bool, or []interface{}.
// This method preserves the original style (dotted keys vs sections, quote styles).
//
// Path syntax is the same as [Document.Get].
func (d *Document) Set(path string, value any) error {
	keys, err := parseKeyPath(path)
	if err != nil {
		return err
	}

	// Check if any prefix of the path is a scalar value (can't nest under scalars)
	if len(keys) > 1 {
		for i := 1; i < len(keys); i++ {
			prefix := keys[:i]
			entry := d.doc.First(prefix...)
			if entry != nil && entry.KeyValue != nil {
				val, err := parseValue(entry.Value)
				if err == nil {
					if _, isMap := val.(map[string]any); !isMap {
						return fmt.Errorf("cannot set %q: %q is a scalar value", path, prefix.String())
					}
				}
			}
		}
	}

	// Check if the exact path is a table section and we're setting a non-table value
	if section := transform.FindTable(d.doc, keys...); section != nil {
		if _, isMap := value.(map[string]any); !isMap {
			return fmt.Errorf("cannot set scalar at %q: path is a table section", path)
		}
	}

	// Check if the key already exists
	existingEntry := d.doc.First(keys...)

	if existingEntry != nil && existingEntry.KeyValue != nil {
		// Key exists - update it in place, preserving style and comments
		return d.updateExistingKey(existingEntry, value)
	}

	// Key doesn't exist - add it in an appropriate style
	return d.addNewKey(keys, value)
}

// updateExistingKey updates an existing key while preserving its formatting
func (d *Document) updateExistingKey(entry *tomledit.Entry, value any) error {
	oldValue := entry.Value

	// Try to format the new value in the same style as the old value
	newValue, err := d.formatValuePreservingStyle(value, oldValue)
	if err != nil {
		return fmt.Errorf("failed to format value: %w", err)
	}

	// Update the value while preserving comments
	entry.Value = newValue
	entry.Value.Trailer = oldValue.Trailer // Preserve trailing comment

	return nil
}

// formatValuePreservingStyle formats a value trying to match the style of an existing value
func (d *Document) formatValuePreservingStyle(value any, existingValue parser.Value) (parser.Value, error) {
	// For strings, try to preserve quote style
	if strValue, ok := value.(string); ok {
		if token, ok := existingValue.X.(parser.Token); ok {
			// Preserve the quote style from the original
			originalStr := token.String()

			var formatted string
			if strings.HasPrefix(originalStr, "'''") {
				// Multiline literal string
				formatted = "'''" + strValue + "'''"
			} else if strings.HasPrefix(originalStr, `"""`) {
				// Multiline basic string
				formatted = `"""` + strValue + `"""`
			} else if strings.HasPrefix(originalStr, "'") {
				// Single-quoted literal string
				formatted = "'" + strValue + "'"
			} else {
				// Double-quoted basic string (default)
				formatted = quoteString(strValue)
			}

			return parser.ParseValue(formatted)
		}
	}

	// For non-strings or when we can't preserve style, use default formatting
	valueStr, err := FormatValueToString(value)
	if err != nil {
		return parser.Value{}, err
	}

	return parser.ParseValue(valueStr)
}

// addNewKey adds a new key in an appropriate style (dotted key vs section)
func (d *Document) addNewKey(keys parser.Key, value any) error {
	// Format the value
	valueStr, err := FormatValueToString(value)
	if err != nil {
		return fmt.Errorf("failed to format value: %w", err)
	}

	parsedValue, err := parser.ParseValue(valueStr)
	if err != nil {
		return fmt.Errorf("failed to parse value: %w", err)
	}

	// Determine where and how to add the new key
	if len(keys) == 1 {
		// Top-level key - add to global section
		return d.addToGlobalSection(keys, parsedValue)
	}

	// Multi-part key - determine if we should use dotted key or section style
	// Check if there are existing dotted keys with the same prefix
	if d.hasDottedKeysWithPrefix(keys[:len(keys)-1]) {
		// Use dotted key style to match existing style
		return d.addDottedKey(keys, parsedValue)
	}

	// Check if there's an existing section for this prefix
	tableName := keys[:len(keys)-1]
	entry := transform.FindTable(d.doc, tableName...)

	if entry != nil {
		// Section exists - add to it
		return d.addToSection(entry.Section, parser.Key{keys[len(keys)-1]}, parsedValue)
	}

	// No existing pattern. Prefer creating table sections for nested keys.
	if len(keys) > 1 {
		section, err := d.ensureSection(tableName)
		if err != nil {
			return err
		}
		return d.addToSection(section, parser.Key{keys[len(keys)-1]}, parsedValue)
	}

	// Fallback to global section for single-part keys.
	return d.addToGlobalSection(keys, parsedValue)
}

// hasDottedKeysWithPrefix checks if there are any dotted keys with the given prefix
func (d *Document) hasDottedKeysWithPrefix(prefix parser.Key) bool {
	if d.doc.Global == nil {
		return false
	}

	prefixStr := strings.Join(prefix, ".") + "."

	for _, item := range d.doc.Global.Items {
		if kv, ok := item.(*parser.KeyValue); ok {
			keyStr := kv.Name.String()
			if strings.HasPrefix(keyStr, prefixStr) {
				return true
			}
		}
	}

	return false
}

// addToGlobalSection adds a key-value pair to the global section
func (d *Document) addToGlobalSection(keys parser.Key, value parser.Value) error {
	if d.doc.Global == nil {
		d.doc.Global = &tomledit.Section{}
	}

	kv := &parser.KeyValue{
		Name:  keys,
		Value: value,
	}

	transform.InsertMapping(d.doc.Global, kv, true)
	return nil
}

// addDottedKey adds a key as a dotted key in the global section
func (d *Document) addDottedKey(keys parser.Key, value parser.Value) error {
	if d.doc.Global == nil {
		d.doc.Global = &tomledit.Section{}
	}

	kv := &parser.KeyValue{
		Name:  keys, // Full dotted path
		Value: value,
	}

	transform.InsertMapping(d.doc.Global, kv, true)
	return nil
}

// addToSection adds a key-value pair to an existing section
func (d *Document) addToSection(section *tomledit.Section, key parser.Key, value parser.Value) error {
	kv := &parser.KeyValue{
		Name:  key,
		Value: value,
	}

	transform.InsertMapping(section, kv, true)
	return nil
}

// ensureSection finds or creates a table section for the given key.
func (d *Document) ensureSection(name parser.Key) (*tomledit.Section, error) {
	if len(name) == 0 {
		if d.doc.Global == nil {
			d.doc.Global = &tomledit.Section{}
		}
		return d.doc.Global, nil
	}

	if entry := transform.FindTable(d.doc, name...); entry != nil {
		return entry.Section, nil
	}

	// Ensure parent section exists to maintain hierarchy.
	if len(name) > 1 {
		if _, err := d.ensureSection(name[:len(name)-1]); err != nil {
			return nil, err
		}
	}

	section := &tomledit.Section{
		Heading: &parser.Heading{
			Name: copyKey(name),
		},
	}

	d.doc.Sections = append(d.doc.Sections, section)

	return section, nil
}

func copyKey(key parser.Key) parser.Key {
	if key == nil {
		return nil
	}
	out := make(parser.Key, len(key))
	copy(out, key)
	return out
}

// Move renames or moves a section or key, like Unix mv. Both oldPath and
// newPath use the same path syntax as [Document.Get].
//
// For sections, child sections are also renamed (e.g. moving "foo.bar" to
// "foo.qux" also renames "foo.bar.baz" to "foo.qux.baz"). Comments,
// formatting, and position are preserved.
//
// For key-value entries within the same section (e.g. "server.host" to
// "server.addr"), the key is renamed in place. For cross-section moves
// (e.g. "server.host" to "app.host"), the key is removed from the source
// section and inserted into the destination, creating it if needed. Block
// comments and inline comments on the key are preserved.
func (d *Document) Move(oldPath, newPath string) error {
	oldKeys, err := parseKeyPath(oldPath)
	if err != nil {
		return err
	}
	newKeys, err := parseKeyPath(newPath)
	if err != nil {
		return err
	}

	// Check if destination already exists
	if existing := d.doc.First(newKeys...); existing != nil {
		return fmt.Errorf("cannot move to %q: destination already exists", newPath)
	}

	entry := d.doc.First(oldKeys...)
	if entry == nil {
		return fmt.Errorf("failed to move: path %q not found", oldPath)
	}

	if entry.IsSection() {
		entry.Section.Heading.Name = copyKey(newKeys)

		for _, sec := range d.doc.Sections {
			if sec.Heading == nil || sec == entry.Section {
				continue
			}
			if oldKeys.IsPrefixOf(sec.Heading.Name) {
				tail := make(parser.Key, len(sec.Heading.Name)-len(oldKeys))
				copy(tail, sec.Heading.Name[len(oldKeys):])
				sec.Heading.Name = append(copyKey(newKeys), tail...)
			}
		}
		return nil
	}

	if entry.IsMapping() {
		oldParent := oldKeys[:len(oldKeys)-1]
		newParent := newKeys[:len(newKeys)-1]

		if parser.Key(oldParent).Equals(newParent) {
			// Same section — just rename the last key segment
			entry.KeyValue.Name[len(entry.KeyValue.Name)-1] = newKeys[len(newKeys)-1]
			return nil
		}

		// Cross-section move: remove from source, insert into destination
		kv := entry.KeyValue
		entry.Remove()

		destSectionKey := newKeys[:len(newKeys)-1]
		if len(destSectionKey) == 0 {
			// Moving to the global section with just the key name
			kv.Name = parser.Key{newKeys[len(newKeys)-1]}
			if d.doc.Global == nil {
				d.doc.Global = &tomledit.Section{}
			}
			transform.InsertMapping(d.doc.Global, kv, true)
		} else if d.hasDottedKeysWithPrefix(destSectionKey) {
			// Destination has dotted keys — use dotted style to match
			kv.Name = copyKey(newKeys)
			if d.doc.Global == nil {
				d.doc.Global = &tomledit.Section{}
			}
			transform.InsertMapping(d.doc.Global, kv, true)
		} else if t := transform.FindTable(d.doc, destSectionKey...); t != nil {
			// Destination section exists — add to it
			kv.Name = parser.Key{newKeys[len(newKeys)-1]}
			transform.InsertMapping(t.Section, kv, true)
		} else {
			// No existing context — create section
			kv.Name = parser.Key{newKeys[len(newKeys)-1]}
			section, err := d.ensureSection(destSectionKey)
			if err != nil {
				return fmt.Errorf("failed to create destination section: %w", err)
			}
			transform.InsertMapping(section, kv, true)
		}
		return nil
	}

	return fmt.Errorf("failed to move: path %q not found", oldPath)
}

// Delete removes a key at the given path.
// Returns nil if the path doesn't exist.
//
// Path syntax is the same as [Document.Get].
func (d *Document) Delete(path string) error {
	keys, err := parseKeyPath(path)
	if err != nil {
		return err
	}

	entry := d.doc.First(keys...)
	if entry == nil {
		return nil // Path doesn't exist, nothing to delete
	}

	entry.Remove()
	return nil
}

// Keys returns the child key names under the given path.
// For a table section like [foo], it returns the keys defined in that section.
// For an inline table value, it returns the keys of that inline table.
// Returns nil if the path doesn't exist or has no children.
//
// Keys are returned in document order. For inline tables, this is the order
// they appear in the source. For Go maps passed to Set or FormatValueToString,
// keys are sorted alphabetically since Go maps have no inherent order.
//
// The returned names are raw key names, not paths. For example,
// Keys("aliases") might return [".", "..", "l"] where "." is a literal
// key name containing a dot.
//
// Path syntax is the same as [Document.Get].
func (d *Document) Keys(path string) ([]string, error) {
	keys, err := parseKeyPath(path)
	if err != nil {
		return nil, err
	}

	var result []string

	// Check if it's a table section — look for sections whose heading starts with the path
	entry := d.doc.First(keys...)
	if entry != nil && entry.KeyValue != nil {
		// Check if the raw value is an inline table — preserve document order
		if inline, ok := entry.Value.X.(parser.Inline); ok {
			for _, kv := range inline {
				if len(kv.Name) > 0 {
					result = append(result, kv.Name[0])
				}
			}
			if len(result) == 0 {
				return nil, nil
			}
			return result, nil
		}
		// For other value types, fall back to parseValue
		val, err := parseValue(entry.Value)
		if err != nil {
			return nil, err
		}
		if _, ok := val.(map[string]any); ok {
			return nil, nil
		}
		return nil, nil
	}

	seen := make(map[string]bool)

	// Look for a table section matching the path exactly
	section := transform.FindTable(d.doc, keys...)
	if section != nil {
		for _, item := range section.Section.Items {
			if kv, ok := item.(*parser.KeyValue); ok {
				if len(kv.Name) > 0 {
					name := kv.Name[0]
					if !seen[name] {
						seen[name] = true
						result = append(result, name)
					}
				}
			}
		}
	}

	// Check for sub-sections (e.g. [foo.bar] when asking for keys of [foo])
	// This also handles the case where [foo] doesn't exist but [foo.bar] does
	for _, sec := range d.doc.Sections {
		if sec.Heading == nil {
			continue
		}
		hname := sec.Heading.Name
		if len(hname) > len(keys) {
			match := true
			for i, k := range keys {
				if hname[i] != k {
					match = false
					break
				}
			}
			if match {
				child := hname[len(keys)]
				if !seen[child] {
					seen[child] = true
					result = append(result, child)
				}
			}
		}
	}

	// Scan dotted keys in the global section whose prefix matches keys
	if d.doc.Global != nil {
		for _, item := range d.doc.Global.Items {
			if kv, ok := item.(*parser.KeyValue); ok {
				if len(kv.Name) > len(keys) && prefixMatches(kv.Name, keys) {
					child := kv.Name[len(keys)]
					if !seen[child] {
						seen[child] = true
						result = append(result, child)
					}
				}
			}
		}
	}

	// Scan dotted keys in all sections whose combined heading+key prefix matches
	for _, sec := range d.doc.Sections {
		if sec.Heading == nil {
			continue
		}
		hname := sec.Heading.Name
		for _, item := range sec.Items {
			kv, ok := item.(*parser.KeyValue)
			if !ok {
				continue
			}
			fullKey := append(parser.Key(nil), hname...)
			fullKey = append(fullKey, kv.Name...)
			if len(fullKey) > len(keys) && prefixMatches(fullKey, keys) {
				child := fullKey[len(keys)]
				if !seen[child] {
					seen[child] = true
					result = append(result, child)
				}
			}
		}
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

// prefixMatches returns true if key starts with prefix.
func prefixMatches(key, prefix parser.Key) bool {
	if len(key) < len(prefix) {
		return false
	}
	for i, p := range prefix {
		if key[i] != p {
			return false
		}
	}
	return true
}

// Has reports whether the given path exists in the document.
// This includes key-value entries, table sections, and dotted key prefixes.
// Parse errors (e.g. invalid paths like "foo.") are returned as the second value.
//
// Path syntax is the same as [Document.Get].
func (d *Document) Has(path string) (bool, error) {
	keys, err := parseKeyPath(path)
	if err != nil {
		return false, err
	}

	_, ok, err := d.Get(path)
	if err != nil && !errors.Is(err, ErrNotValue) {
		return false, err
	}
	if ok {
		return true, nil
	}

	// Check if a table section exists with this name
	if transform.FindTable(d.doc, keys...) != nil {
		return true, nil
	}

	// Check if any section heading starts with the given keys (implicit parents)
	for _, sec := range d.doc.Sections {
		if sec.Heading == nil {
			continue
		}
		if len(sec.Heading.Name) >= len(keys) && prefixMatches(sec.Heading.Name, keys) {
			return true, nil
		}
	}

	// Check if any dotted key in any section starts with the given keys
	if d.doc.Global != nil {
		for _, item := range d.doc.Global.Items {
			if kv, ok := item.(*parser.KeyValue); ok {
				if len(kv.Name) >= len(keys) && prefixMatches(kv.Name, keys) {
					return true, nil
				}
			}
		}
	}
	for _, sec := range d.doc.Sections {
		if sec.Heading == nil {
			continue
		}
		for _, item := range sec.Items {
			kv, ok := item.(*parser.KeyValue)
			if !ok {
				continue
			}
			fullKey := append(parser.Key(nil), sec.Heading.Name...)
			fullKey = append(fullKey, kv.Name...)
			if len(fullKey) >= len(keys) && prefixMatches(fullKey, keys) {
				return true, nil
			}
		}
	}

	return false, nil
}

// TopLevelKeys returns all top-level key names in the document, deduplicated
// and in document order. The returned strings are raw key names (not paths).
func (d *Document) TopLevelKeys() []string {
	var result []string
	seen := make(map[string]bool)

	// Keys from the global section
	if d.doc.Global != nil {
		for _, item := range d.doc.Global.Items {
			if kv, ok := item.(*parser.KeyValue); ok && len(kv.Name) > 0 {
				name := kv.Name[0]
				if !seen[name] {
					seen[name] = true
					result = append(result, name)
				}
			}
		}
	}

	// Top-level section heading names (first element only)
	for _, sec := range d.doc.Sections {
		if sec.Heading == nil || len(sec.Heading.Name) == 0 {
			continue
		}
		name := sec.Heading.Name[0]
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}

	return result
}

// Prune removes empty sections from the document. A section is considered
// empty if it contains no key-value items (comments-only sections are also
// removed). This is useful after Delete or Move operations that may leave
// behind stale section headers.
func (d *Document) Prune() {
	// Build a set of section names that have children.
	hasChild := make(map[string]bool)
	for _, sec := range d.doc.Sections {
		if sec.Heading == nil {
			continue
		}
		name := sec.Heading.Name
		for i := 1; i < len(name); i++ {
			hasChild[parser.Key(name[:i]).String()] = true
		}
	}

	filtered := d.doc.Sections[:0]
	for _, sec := range d.doc.Sections {
		if sec.Heading == nil {
			filtered = append(filtered, sec)
			continue
		}

		hasKV := false
		for _, item := range sec.Items {
			if _, ok := item.(*parser.KeyValue); ok {
				hasKV = true
				break
			}
		}

		if hasKV || hasChild[sec.Heading.Name.String()] {
			filtered = append(filtered, sec)
		}
	}
	d.doc.Sections = filtered
}

// String serializes the document back to TOML format, preserving all
// comments, formatting, and declaration order.
func (d *Document) String() string {
	return string(d.Bytes())
}

// Bytes serializes the document back to TOML format as a byte slice.
func (d *Document) Bytes() []byte {
	var buf bytes.Buffer
	var formatter tomledit.Formatter
	formatter.Format(&buf, d.doc)
	return buf.Bytes()
}

// parseKeyPath parses a dotted path into a parser.Key using the TOML-compliant
// parser from tomledit. This properly handles quoted keys like aliases."." and
// rejects invalid paths like "aliases." (trailing dot).
//
// Examples:
//   - "simple" -> ["simple"]
//   - "dotted.key" -> ["dotted", "key"]
//   - "aliases.\".\"" or `aliases."."` -> ["aliases", "."]
//   - "aliases." -> error (trailing dot not allowed)
//   - "" -> error (empty path not allowed)
func parseKeyPath(path string) (parser.Key, error) {
	if path == "" {
		return nil, fmt.Errorf("failed to parse path: empty path")
	}

	// Use tomledit's parser.ParseKey which handles quoted keys and validates
	// according to TOML specification
	key, err := parser.ParseKey(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path %q: %w", path, err)
	}

	return key, nil
}

// parseValue converts a parser.Value into a Go value.
func parseValue(v parser.Value) (any, error) {
	switch datum := v.X.(type) {
	case parser.Token:
		// Get the string representation and parse it
		text := datum.String()

		// Try to determine the type and parse accordingly
		// For strings, they'll be quoted, so unquote them
		if strings.HasPrefix(text, `"`) || strings.HasPrefix(text, `'`) {
			return unquoteString(text), nil
		}

		// Try boolean
		if text == "true" {
			return true, nil
		}
		if text == "false" {
			return false, nil
		}

		// Try integer (strip underscores, detect base prefix)
		cleanText := strings.ReplaceAll(text, "_", "")
		base := 10
		numStr := cleanText
		if strings.HasPrefix(cleanText, "0x") || strings.HasPrefix(cleanText, "0X") {
			base = 16
			numStr = cleanText[2:]
		} else if strings.HasPrefix(cleanText, "0o") || strings.HasPrefix(cleanText, "0O") {
			base = 8
			numStr = cleanText[2:]
		} else if strings.HasPrefix(cleanText, "0b") || strings.HasPrefix(cleanText, "0B") {
			base = 2
			numStr = cleanText[2:]
		}
		if i, err := strconv.ParseInt(numStr, base, 64); err == nil {
			return i, nil
		}

		// Try float (strip underscores, handle special values)
		switch cleanText {
		case "inf", "+inf":
			return math.Inf(1), nil
		case "-inf":
			return math.Inf(-1), nil
		case "nan", "+nan", "-nan":
			return math.NaN(), nil
		}
		if f, err := strconv.ParseFloat(cleanText, 64); err == nil {
			return f, nil
		}

		// Return as string if all else fails
		return text, nil

	case parser.Array:
		result := make([]any, 0, len(datum))
		for _, item := range datum {
			if arrayItem, ok := item.(parser.Value); ok {
				val, err := parseValue(arrayItem)
				if err != nil {
					return nil, err
				}
				result = append(result, val)
			}
		}
		return result, nil

	case parser.Inline:
		result := make(map[string]any)
		for _, kv := range datum {
			val, err := parseValue(kv.Value)
			if err != nil {
				return nil, err
			}
			result[kv.Name.String()] = val
		}
		return result, nil

	default:
		return nil, fmt.Errorf("failed to parse value: unsupported type %T", v.X)
	}
}

// FormatValueToString converts a Go value into a TOML value string.
// This is useful for displaying values in a TOML-compatible format.
func FormatValueToString(v any) (string, error) {
	switch val := v.(type) {
	case string:
		// Quote and escape the string
		return quoteString(val), nil

	case int:
		return strconv.FormatInt(int64(val), 10), nil

	case int64:
		return strconv.FormatInt(val, 10), nil

	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), nil

	case bool:
		if val {
			return "true", nil
		}
		return "false", nil

	case []any:
		parts := make([]string, len(val))
		for i, item := range val {
			s, err := FormatValueToString(item)
			if err != nil {
				return "", err
			}
			parts[i] = s
		}
		return "[" + strings.Join(parts, ", ") + "]", nil

	case []string:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = quoteString(item)
		}
		return "[" + strings.Join(parts, ", ") + "]", nil

	case []int:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = strconv.Itoa(item)
		}
		return "[" + strings.Join(parts, ", ") + "]", nil

	case map[string]any:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(val))
		for _, k := range keys {
			vs, err := FormatValueToString(val[k])
			if err != nil {
				return "", err
			}
			parts = append(parts, formatKeySegment(k)+" = "+vs)
		}
		return "{" + strings.Join(parts, ", ") + "}", nil

	default:
		return "", fmt.Errorf("failed to format value: unsupported type %T", v)
	}
}

// quoteString adds quotes around a string value and escapes special characters.
func quoteString(s string) string {
	// Use basic string (double quotes) and escape special characters
	escaped := strings.ReplaceAll(s, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	escaped = strings.ReplaceAll(escaped, "\n", "\\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\\r")
	escaped = strings.ReplaceAll(escaped, "\t", "\\t")
	return `"` + escaped + `"`
}

// unquoteString removes quotes and unescapes a string value.
func unquoteString(s string) string {
	// Multiline literal string: '''...'''
	if strings.HasPrefix(s, "'''") && strings.HasSuffix(s, "'''") && len(s) >= 6 {
		inner := s[3 : len(s)-3]
		if len(inner) > 0 && inner[0] == '\n' {
			inner = inner[1:]
		}
		return inner
	}

	// Literal string: '...' — no escape processing
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}

	// Multiline basic string: """..."""
	if strings.HasPrefix(s, `"""`) && strings.HasSuffix(s, `"""`) && len(s) >= 6 {
		inner := s[3 : len(s)-3]
		if len(inner) > 0 && inner[0] == '\n' {
			inner = inner[1:]
		}
		return unescapeBasicString(inner)
	}

	// Basic string: "..."
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return unescapeBasicString(s[1 : len(s)-1])
	}

	return s
}

// unescapeBasicString processes escape sequences in a TOML basic string.
func unescapeBasicString(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result = append(result, '\n')
				i++
			case 'r':
				result = append(result, '\r')
				i++
			case 't':
				result = append(result, '\t')
				i++
			case '"':
				result = append(result, '"')
				i++
			case '\'':
				result = append(result, '\'')
				i++
			case '\\':
				result = append(result, '\\')
				i++
			default:
				result = append(result, s[i])
			}
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
}
