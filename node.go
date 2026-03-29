package tomlsawyer

import (
	"fmt"

	"github.com/creachadair/tomledit"
	"github.com/creachadair/tomledit/parser"
)

// ValueType identifies the kind of a TOML value.
type ValueType int

const (
	StringVal      ValueType = iota
	IntVal
	FloatVal
	BoolVal
	DateTimeVal
	ArrayVal
	InlineTableVal
	UnknownVal
)

// Section wraps a table section in the document AST.
// Mutations go directly to the underlying AST.
type Section struct {
	sec *tomledit.Section
}

// Name returns the section's dotted name (e.g. "server" or "app.database").
// Returns "" for the global section.
func (s *Section) Name() string {
	if s.sec.Heading == nil {
		return ""
	}
	return s.sec.Heading.Name.String()
}

// SetName renames the section heading.
func (s *Section) SetName(name string) {
	if s.sec.Heading == nil {
		return
	}
	key, err := parser.ParseKey(name)
	if err != nil {
		return
	}
	s.sec.Heading.Name = key
}

// IsArray returns true for [[array_of_tables]] sections.
func (s *Section) IsArray() bool {
	return s.sec.Heading != nil && s.sec.Heading.IsArray
}

// Comment returns the inline comment on the section heading.
func (s *Section) Comment() string {
	if s.sec.Heading == nil {
		return ""
	}
	return s.sec.Heading.Trailer
}

// SetComment sets the inline comment on the section heading.
func (s *Section) SetComment(c string) {
	if s.sec.Heading == nil {
		return
	}
	s.sec.Heading.Trailer = c
}

// BlockComment returns the comment lines above the section heading.
func (s *Section) BlockComment() []string {
	if s.sec.Heading == nil {
		return nil
	}
	return []string(s.sec.Heading.Block)
}

// SetBlockComment sets the comment lines above the section heading.
func (s *Section) SetBlockComment(lines []string) {
	if s.sec.Heading == nil {
		return
	}
	s.sec.Heading.Block = parser.Comments(lines)
}

// Entries returns the key-value entries in document order.
// Each entry carries its associated block comment (comments above it).
func (s *Section) Entries() []*Entry {
	return extractEntries(s.sec.Items)
}

// SetEntries replaces the section's key-value entries, rebuilding the
// underlying item list with comments interleaved correctly.
func (s *Section) SetEntries(entries []*Entry) {
	s.sec.Items = buildItems(entries)
}

// Entry wraps a key-value pair with its comments.
type Entry struct {
	kv           *parser.KeyValue
	blockComment parser.Comments // comments above this entry (not on kv.Block)
}

// NewEntry creates a new entry from a key and Go value.
func NewEntry(key string, value any) (*Entry, error) {
	valueStr, err := FormatValueToString(value)
	if err != nil {
		return nil, fmt.Errorf("failed to format value: %w", err)
	}
	pv, err := parser.ParseValue(valueStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse value: %w", err)
	}
	return &Entry{
		kv: &parser.KeyValue{
			Name:  parser.Key{key},
			Value: pv,
		},
	}, nil
}

// Key returns the entry's key name.
func (e *Entry) Key() string {
	if len(e.kv.Name) == 0 {
		return ""
	}
	return e.kv.Name[len(e.kv.Name)-1]
}

// SetKey renames the entry's key.
func (e *Entry) SetKey(name string) {
	if len(e.kv.Name) == 0 {
		e.kv.Name = parser.Key{name}
	} else {
		e.kv.Name[len(e.kv.Name)-1] = name
	}
}

// Value returns a live reference to the entry's value in the AST.
func (e *Entry) Value() *Value {
	return &Value{pv: &e.kv.Value}
}

// Comment returns the inline comment on the value.
func (e *Entry) Comment() string {
	return e.kv.Value.Trailer
}

// SetComment sets the inline comment on the value.
func (e *Entry) SetComment(c string) {
	e.kv.Value.Trailer = c
}

// BlockComment returns the comment lines above this entry.
func (e *Entry) BlockComment() []string {
	// Combine the entry's own block comment with the KV's block comment
	var result []string
	if len(e.blockComment) > 0 {
		result = append(result, e.blockComment...)
	}
	if len(e.kv.Block) > 0 {
		result = append(result, e.kv.Block...)
	}
	return result
}

// SetBlockComment sets the comment lines above this entry.
func (e *Entry) SetBlockComment(lines []string) {
	e.blockComment = nil
	e.kv.Block = parser.Comments(lines)
}

// Value wraps a TOML value in the AST, providing typed access and mutation.
type Value struct {
	pv *parser.Value
}

// Type returns the TOML type of this value.
func (v *Value) Type() ValueType {
	switch v.pv.X.(type) {
	case parser.Token:
		tok := v.pv.X.(parser.Token)
		text := tok.String()
		// Detect type from token content
		if isQuotedString(text) {
			return StringVal
		}
		if text == "true" || text == "false" {
			return BoolVal
		}
		// Check for datetime patterns before numbers
		if isDateTimeLike(text) {
			return DateTimeVal
		}
		if isIntegerLike(text) {
			return IntVal
		}
		if isFloatLike(text) {
			return FloatVal
		}
		return UnknownVal
	case parser.Array:
		return ArrayVal
	case parser.Inline:
		return InlineTableVal
	default:
		return UnknownVal
	}
}

func isQuotedString(s string) bool {
	return len(s) >= 2 && (s[0] == '"' || s[0] == '\'')
}

func isDateTimeLike(s string) bool {
	// Simple heuristic: contains '-' and ':' or looks like a date
	for i, c := range s {
		if c == '-' && i == 4 {
			return true
		}
		if c == ':' && i > 2 {
			return true
		}
	}
	return false
}

func isIntegerLike(s string) bool {
	if len(s) == 0 {
		return false
	}
	start := 0
	if s[0] == '+' || s[0] == '-' {
		start = 1
	}
	if start >= len(s) {
		return false
	}
	if s[start] == '0' && start+1 < len(s) {
		next := s[start+1]
		if next == 'x' || next == 'o' || next == 'b' {
			return true
		}
	}
	for _, c := range s[start:] {
		if c != '_' && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func isFloatLike(s string) bool {
	for _, c := range s {
		if c == '.' || c == 'e' || c == 'E' {
			return true
		}
	}
	return s == "inf" || s == "+inf" || s == "-inf" ||
		s == "nan" || s == "+nan" || s == "-nan"
}

// AsAny returns the decoded Go value (string, int64, float64, bool, etc).
func (v *Value) AsAny() any {
	val, _ := parseValue(*v.pv)
	return val
}

// AsString returns the string value. Returns "" if not a string.
func (v *Value) AsString() string {
	val, _ := parseValue(*v.pv)
	s, _ := val.(string)
	return s
}

// AsInt returns the integer value. Returns 0 if not an integer.
func (v *Value) AsInt() int64 {
	val, _ := parseValue(*v.pv)
	i, _ := val.(int64)
	return i
}

// AsFloat returns the float value. Returns 0 if not a float.
func (v *Value) AsFloat() float64 {
	val, _ := parseValue(*v.pv)
	f, _ := val.(float64)
	return f
}

// AsBool returns the boolean value. Returns false if not a bool.
func (v *Value) AsBool() bool {
	val, _ := parseValue(*v.pv)
	b, _ := val.(bool)
	return b
}

// InlineComment returns the trailing comment on this value.
func (v *Value) InlineComment() string {
	return v.pv.Trailer
}

// SetInlineComment sets the trailing comment on this value.
func (v *Value) SetInlineComment(c string) {
	v.pv.Trailer = c
}

// Elements returns the array elements with their associated comments.
// Each element carries the block comment lines that precede it.
// Only valid for ArrayVal.
func (v *Value) Elements() []*Element {
	arr, ok := v.pv.X.(parser.Array)
	if !ok {
		return nil
	}
	return extractElements(arr)
}

// SetElements replaces the array contents, rebuilding the underlying
// ArrayItem list with comments interleaved correctly.
// Only valid for ArrayVal.
func (v *Value) SetElements(elems []*Element) {
	if _, ok := v.pv.X.(parser.Array); !ok {
		return
	}
	v.pv.X = buildArray(elems)
}

// Fields returns the inline table fields as entries in document order.
// Only valid for InlineTableVal.
func (v *Value) Fields() []*Entry {
	inline, ok := v.pv.X.(parser.Inline)
	if !ok {
		return nil
	}
	entries := make([]*Entry, len(inline))
	for i, kv := range inline {
		entries[i] = &Entry{kv: kv}
	}
	return entries
}

// SetFields replaces the inline table fields.
// Only valid for InlineTableVal.
func (v *Value) SetFields(entries []*Entry) {
	if _, ok := v.pv.X.(parser.Inline); !ok {
		return
	}
	inline := make(parser.Inline, len(entries))
	for i, e := range entries {
		inline[i] = e.kv
	}
	v.pv.X = inline
}

// Element is an array entry bundled with its associated comments.
type Element struct {
	blockComment parser.Comments
	value        parser.Value
}

// NewElement creates a new array element from a Go value.
func NewElement(val any) (*Element, error) {
	valueStr, err := FormatValueToString(val)
	if err != nil {
		return nil, fmt.Errorf("failed to format value: %w", err)
	}
	pv, err := parser.ParseValue(valueStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse value: %w", err)
	}
	return &Element{value: pv}, nil
}

// NewCommentedElement creates a new array element with a block comment and
// optional inline comment.
func NewCommentedElement(val any, blockComment []string, inlineComment string) (*Element, error) {
	elem, err := NewElement(val)
	if err != nil {
		return nil, err
	}
	elem.blockComment = parser.Comments(blockComment)
	elem.value.Trailer = inlineComment
	return elem, nil
}

// Value returns a live reference to this element's value.
func (e *Element) Value() *Value {
	return &Value{pv: &e.value}
}

// AsAny returns the decoded Go value.
func (e *Element) AsAny() any {
	val, _ := parseValue(e.value)
	return val
}

// InlineComment returns the trailing comment on this element.
func (e *Element) InlineComment() string {
	return e.value.Trailer
}

// SetInlineComment sets the trailing comment on this element.
func (e *Element) SetInlineComment(c string) {
	e.value.Trailer = c
}

// BlockComment returns the comment lines above this element.
func (e *Element) BlockComment() []string {
	return []string(e.blockComment)
}

// SetBlockComment sets the comment lines above this element.
func (e *Element) SetBlockComment(lines []string) {
	e.blockComment = parser.Comments(lines)
}

// ── Document-level section access ──

// Sections returns all named sections in document order.
func (d *Document) Sections() []*Section {
	sections := make([]*Section, len(d.doc.Sections))
	for i, sec := range d.doc.Sections {
		sections[i] = &Section{sec: sec}
	}
	return sections
}

// SetSections replaces the document's section list.
// Use with Sections() to reorder, insert, or remove sections.
func (d *Document) SetSections(sections []*Section) {
	result := make([]*tomledit.Section, len(sections))
	for i, s := range sections {
		result[i] = s.sec
	}
	d.doc.Sections = result
}

// Global returns the global (headerless) section.
func (d *Document) Global() *Section {
	if d.doc.Global == nil {
		d.doc.Global = &tomledit.Section{}
	}
	return &Section{sec: d.doc.Global}
}

// GetValue returns a live AST reference to the value at the given path.
// Returns nil if the path doesn't exist or refers to a section.
//
// Path syntax is the same as [Document.Get].
func (d *Document) GetValue(path string) *Value {
	keys, err := parseKeyPath(path)
	if err != nil {
		return nil
	}
	entry := d.doc.First(keys...)
	if entry == nil || entry.KeyValue == nil {
		return nil
	}
	return &Value{pv: &entry.KeyValue.Value}
}

// ── Helpers for comment-aware extraction/reconstruction ──

// extractEntries groups section items into entries, associating each block
// comment with the key-value that follows it.
func extractEntries(items []parser.Item) []*Entry {
	var entries []*Entry
	var pendingComments parser.Comments

	for _, item := range items {
		switch v := item.(type) {
		case parser.Comments:
			pendingComments = append(pendingComments, v...)
		case *parser.KeyValue:
			e := &Entry{kv: v}
			if len(pendingComments) > 0 {
				e.blockComment = pendingComments
				pendingComments = nil
			}
			entries = append(entries, e)
		}
	}
	return entries
}

// buildItems reconstructs a section's item list from entries, interleaving
// block comments correctly.
func buildItems(entries []*Entry) []parser.Item {
	var items []parser.Item
	for _, e := range entries {
		if len(e.blockComment) > 0 {
			items = append(items, parser.Comments(e.blockComment))
		}
		items = append(items, e.kv)
	}
	return items
}

// extractElements groups array items into elements, associating each block
// comment with the value that follows it.
func extractElements(arr parser.Array) []*Element {
	var elems []*Element
	var pendingComments parser.Comments

	for _, item := range arr {
		switch v := item.(type) {
		case parser.Comments:
			pendingComments = append(pendingComments, v...)
		case parser.Value:
			elem := &Element{value: v}
			if len(pendingComments) > 0 {
				elem.blockComment = pendingComments
				pendingComments = nil
			}
			elems = append(elems, elem)
		}
	}
	return elems
}

// buildArray reconstructs an Array from elements, interleaving block
// comments correctly.
func buildArray(elems []*Element) parser.Array {
	var arr parser.Array
	for _, e := range elems {
		if len(e.blockComment) > 0 {
			arr = append(arr, parser.Comments(e.blockComment))
		}
		arr = append(arr, e.value)
	}
	return arr
}
