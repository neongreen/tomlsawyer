package tomlsawyer

import (
	"math"
	"testing"
)

// --- String conformance ---

func TestConformanceBasicString(t *testing.T) {
	tests := []struct{ input, key, want string }{
		{`s = "hello"`, "s", "hello"},
		{`s = ""`, "s", ""},
		{`s = "hello\tworld"`, "s", "hello\tworld"},
		{`s = "hello\nworld"`, "s", "hello\nworld"},
		{`s = "hello\\world"`, "s", "hello\\world"},
		{`s = "hello\"world"`, "s", "hello\"world"},
		{`s = "hello\rworld"`, "s", "hello\rworld"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			doc, err := ParseString(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			val, ok, _ := doc.Get(tt.key)
			if !ok {
				t.Fatal("key not found")
			}
			if val != tt.want {
				t.Errorf("Get = %q, want %q", val, tt.want)
			}
		})
	}
}

func TestConformanceLiteralString(t *testing.T) {
	tests := []struct{ input, key, want string }{
		{`s = 'hello'`, "s", "hello"},
		{`s = ''`, "s", ""},
		{`s = 'C:\temp'`, "s", `C:\temp`},
		{`s = 'hello\nworld'`, "s", `hello\nworld`},  // no escape processing
		{`s = '<\i\c*\s*>'`, "s", `<\i\c*\s*>`},
		{`s = 'Tom "Dubs" Preston-Werner'`, "s", `Tom "Dubs" Preston-Werner`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			doc, err := ParseString(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			val, ok, _ := doc.Get(tt.key)
			if !ok {
				t.Fatal("key not found")
			}
			if val != tt.want {
				t.Errorf("Get = %q, want %q", val, tt.want)
			}
		})
	}
}

func TestConformanceMultilineBasicString(t *testing.T) {
	// Leading newline after opening delimiter is trimmed
	doc, _ := ParseString("s = \"\"\"\nhello\nworld\"\"\"")
	val, ok, _ := doc.Get("s")
	if !ok { t.Fatal("not found") }
	if val != "hello\nworld" {
		t.Errorf("Get = %q, want %q", val, "hello\nworld")
	}
}

func TestConformanceMultilineLiteralString(t *testing.T) {
	doc, _ := ParseString("s = '''\nhello\\nworld\n'''")
	val, ok, _ := doc.Get("s")
	if !ok { t.Fatal("not found") }
	// Literal: no escape processing, leading newline trimmed
	if val != "hello\\nworld\n" {
		t.Errorf("Get = %q, want %q", val, "hello\\nworld\n")
	}
}

// --- Integer conformance ---

func TestConformanceIntegers(t *testing.T) {
	tests := []struct{ input, key string; want int64 }{
		{"n = 42", "n", 42},
		{"n = 0", "n", 0},
		{"n = -17", "n", -17},
		{"n = +99", "n", 99},
		{"n = 1_000", "n", 1000},
		{"n = 5_349_221", "n", 5349221},
		{"n = 1_2_3_4_5", "n", 12345},
		{"n = 0xDEADBEEF", "n", 0xDEADBEEF},
		{"n = 0xdead_beef", "n", 0xdeadbeef},
		{"n = 0o755", "n", 0o755},
		{"n = 0o7_5_5", "n", 0o755},
		{"n = 0b11010110", "n", 0b11010110},
		{"n = 0b1101_0110", "n", 0b11010110},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			doc, err := ParseString(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			val, ok, _ := doc.Get(tt.key)
			if !ok {
				t.Fatal("key not found")
			}
			if val != tt.want {
				t.Errorf("Get = %v (type %T), want %v", val, val, tt.want)
			}
		})
	}
}

// --- Float conformance ---

func TestConformanceFloats(t *testing.T) {
	tests := []struct{ input, key string; want float64 }{
		{"f = 3.14", "f", 3.14},
		{"f = -0.01", "f", -0.01},
		{"f = 5e+22", "f", 5e+22},
		{"f = 1e06", "f", 1e06},
		{"f = -2E-2", "f", -2e-2},
		{"f = 6.626e-34", "f", 6.626e-34},
		{"f = 224_617.445_991_228", "f", 224617.445991228},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			doc, err := ParseString(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			val, ok, _ := doc.Get(tt.key)
			if !ok {
				t.Fatal("key not found")
			}
			got, isFloat := val.(float64)
			if !isFloat {
				t.Fatalf("Get = %T, want float64", val)
			}
			if got != tt.want {
				t.Errorf("Get = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConformanceSpecialFloats(t *testing.T) {
	tests := []struct{ input, key string; check func(float64) bool; desc string }{
		{"f = inf", "f", func(v float64) bool { return math.IsInf(v, 1) }, "+inf"},
		{"f = +inf", "f", func(v float64) bool { return math.IsInf(v, 1) }, "+inf"},
		{"f = -inf", "f", func(v float64) bool { return math.IsInf(v, -1) }, "-inf"},
		{"f = nan", "f", func(v float64) bool { return math.IsNaN(v) }, "NaN"},
		{"f = +nan", "f", func(v float64) bool { return math.IsNaN(v) }, "NaN"},
		{"f = -nan", "f", func(v float64) bool { return math.IsNaN(v) }, "NaN"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			doc, err := ParseString(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			val, ok, _ := doc.Get(tt.key)
			if !ok {
				t.Fatal("key not found")
			}
			got, isFloat := val.(float64)
			if !isFloat {
				t.Fatalf("Get = %T, want float64", val)
			}
			if !tt.check(got) {
				t.Errorf("Get = %v, want %s", got, tt.desc)
			}
		})
	}
}

// --- Boolean conformance ---

func TestConformanceBooleans(t *testing.T) {
	doc, _ := ParseString("t = true\nf = false\n")
	v1, ok1, _ := doc.Get("t")
	v2, ok2, _ := doc.Get("f")
	if !ok1 || v1 != true { t.Errorf("true: got %v", v1) }
	if !ok2 || v2 != false { t.Errorf("false: got %v", v2) }
}

// --- Array conformance ---

func TestConformanceArrayMixed(t *testing.T) {
	doc, _ := ParseString(`arr = ["hello", 42, true, 3.14]`)
	val, ok, _ := doc.Get("arr")
	if !ok { t.Fatal("not found") }
	arr, isArr := val.([]any)
	if !isArr { t.Fatalf("type = %T", val) }
	if len(arr) != 4 { t.Fatalf("len = %d", len(arr)) }
	if arr[0] != "hello" { t.Errorf("[0] = %v", arr[0]) }
	if arr[1] != int64(42) { t.Errorf("[1] = %v", arr[1]) }
	if arr[2] != true { t.Errorf("[2] = %v", arr[2]) }
	if arr[3] != 3.14 { t.Errorf("[3] = %v", arr[3]) }
}

func TestConformanceNestedArray(t *testing.T) {
	doc, _ := ParseString(`arr = [[1, 2], [3, 4]]`)
	val, ok, _ := doc.Get("arr")
	if !ok { t.Fatal("not found") }
	arr, isArr := val.([]any)
	if !isArr { t.Fatalf("type = %T", val) }
	if len(arr) != 2 { t.Fatalf("len = %d", len(arr)) }
}

// --- Inline table conformance ---

func TestConformanceInlineTable(t *testing.T) {
	doc, _ := ParseString(`t = { name = "Alice", age = 30 }`)
	val, ok, _ := doc.Get("t")
	if !ok { t.Fatal("not found") }
	tbl, isMap := val.(map[string]any)
	if !isMap { t.Fatalf("type = %T", val) }
	if tbl["name"] != "Alice" { t.Errorf("name = %v", tbl["name"]) }
	if tbl["age"] != int64(30) { t.Errorf("age = %v", tbl["age"]) }
}

func TestConformanceEmptyInlineTable(t *testing.T) {
	doc, _ := ParseString(`t = {}`)
	val, ok, _ := doc.Get("t")
	if !ok { t.Fatal("not found") }
	tbl, isMap := val.(map[string]any)
	if !isMap { t.Fatalf("type = %T", val) }
	if len(tbl) != 0 { t.Errorf("len = %d", len(tbl)) }
}

func TestSetOrderedKeyValue(t *testing.T) {
	doc, _ := ParseString("")
	doc.Set("person", []KeyValue{
		{Key: "name", Value: "Alice"},
		{Key: "age", Value: 30},
		{Key: "email", Value: "alice@example.com"},
	})
	output := doc.String()
	// Order must be preserved: name, age, email (not sorted)
	wantGolden(t, output, "person = {name = \"Alice\", age = 30, email = \"alice@example.com\"}\n")
}

func TestSetOrderedKeyValueRoundTrip(t *testing.T) {
	doc, _ := ParseString("")
	doc.Set("t", []KeyValue{
		{Key: "z", Value: 1},
		{Key: "a", Value: 2},
		{Key: "m", Value: 3},
	})
	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v", err)
	}
	val, ok, _ := doc2.Get("t")
	if !ok {
		t.Fatal("not found")
	}
	tbl, isMap := val.(map[string]any)
	if !isMap {
		t.Fatalf("type = %T", val)
	}
	if tbl["z"] != int64(1) || tbl["a"] != int64(2) || tbl["m"] != int64(3) {
		t.Errorf("round-trip values wrong: %v", tbl)
	}
}

func TestSetOrderedKeyValueWithSpecialKeys(t *testing.T) {
	doc, _ := ParseString("")
	doc.Set("t", []KeyValue{
		{Key: "key with spaces", Value: "a"},
		{Key: "key.with.dots", Value: "b"},
	})
	output := doc.String()
	doc2, err := ParseString(output)
	if err != nil {
		t.Fatalf("invalid TOML: %v\noutput:\n%s", err, output)
	}
	v1, ok1, _ := doc2.Get(`t."key with spaces"`)
	v2, ok2, _ := doc2.Get(`t."key.with.dots"`)
	if !ok1 || v1 != "a" {
		t.Errorf("key with spaces = %v", v1)
	}
	if !ok2 || v2 != "b" {
		t.Errorf("key.with.dots = %v", v2)
	}
}
