package tomlsawyer

import "testing"

func TestGetLiteralStringBackslash(t *testing.T) {
	doc, _ := ParseString(`path = 'C:\temp'`)
	val, _ := doc.Get("path")
	if val != `C:\temp` {
		t.Errorf("Get = %q, want %q", val, `C:\temp`)
	}
}

func TestGetLiteralStringNoEscape(t *testing.T) {
	doc, _ := ParseString(`regex = '<\i\c*\s*>'`)
	val, _ := doc.Get("regex")
	if val != `<\i\c*\s*>` {
		t.Errorf("Get = %q, want %q", val, `<\i\c*\s*>`)
	}
}

func TestGetBasicStringEscape(t *testing.T) {
	doc, _ := ParseString(`text = "hello\tworld\n"`)
	val, _ := doc.Get("text")
	if val != "hello\tworld\n" {
		t.Errorf("Get = %q, want %q", val, "hello\tworld\n")
	}
}

func TestGetMultilineLiteralStringNoEscape(t *testing.T) {
	doc, _ := ParseString("path = '''\nC:\\temp\\file\n'''")
	val, _ := doc.Get("path")
	if val != "C:\\temp\\file\n" {
		t.Errorf("Get = %q, want %q", val, "C:\\temp\\file\n")
	}
}

func TestGetHexInt(t *testing.T) {
	doc, _ := ParseString("hex = 0xDEADBEEF")
	val, _ := doc.Get("hex")
	if val != int64(0xDEADBEEF) {
		t.Errorf("Get = %v, want %v", val, int64(0xDEADBEEF))
	}
}

func TestGetOctalInt(t *testing.T) {
	doc, _ := ParseString("oct = 0o755")
	val, _ := doc.Get("oct")
	if val != int64(0o755) {
		t.Errorf("Get = %v, want %v", val, int64(0o755))
	}
}

func TestGetBinaryInt(t *testing.T) {
	doc, _ := ParseString("bin = 0b11010110")
	val, _ := doc.Get("bin")
	if val != int64(0b11010110) {
		t.Errorf("Get = %v, want %v", val, int64(0b11010110))
	}
}

func TestGetUnderscoredInt(t *testing.T) {
	doc, _ := ParseString("num = 1_000_000")
	val, _ := doc.Get("num")
	if val != int64(1_000_000) {
		t.Errorf("Get = %v, want %v", val, int64(1_000_000))
	}
}

func TestFormatInlineTableQuotesKeys(t *testing.T) {
	doc, _ := ParseString("")
	doc.Set("t", map[string]any{"key with spaces": 1, "normal": 2})
	output := doc.String()
	doc2, _ := ParseString(output)
	val, _ := doc2.Get(`t."key with spaces"`)
	if val != int64(1) {
		t.Errorf("round-trip failed for key with spaces: %v", val)
	}
}

func TestFormatInlineTableDeterministic(t *testing.T) {
	doc, _ := ParseString("")
	doc.Set("t", map[string]any{"z": 1, "a": 2, "m": 3})
	output1 := doc.String()
	for i := 0; i < 10; i++ {
		doc2, _ := ParseString("")
		doc2.Set("t", map[string]any{"z": 1, "a": 2, "m": 3})
		if doc2.String() != output1 {
			t.Fatal("inline table output is nondeterministic")
		}
	}
}
