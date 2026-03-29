package tomlsawyer

import (
	"testing"
)

// TestKeyOrderPreservation tests that key order is preserved
func TestKeyOrderPreservation(t *testing.T) {
	input := `# Config in specific order
zebra = "last"
apple = "first"
middle = "middle"
banana = "second"

[section]
zulu = 1
alpha = 2
mike = 3
bravo = 4
`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	// Modify one value
	doc.Set("middle", "modified")

	output := doc.String()

	wantGolden(t, output, `# Config in specific order
zebra = "last"
apple = "first"
middle = "modified"
banana = "second"

[section]
zulu = 1
alpha = 2
mike = 3
bravo = 4
`)
}

// TestQuoteStylePreservation tests that different quote styles are preserved
func TestQuoteStylePreservation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		modifyKey   string
		modifyValue string
		want        string
	}{
		{
			name: "double quotes",
			input: `
double = "hello world"
single = 'hello world'
`,
			modifyKey:   "double",
			modifyValue: "hello universe",
			want: `double = "hello universe"
single = 'hello world'
`,
		},
		{
			name: "single quotes",
			input: `
single = 'hello'
double = "world"
`,
			modifyKey:   "double",
			modifyValue: "universe",
			want: `single = 'hello'
double = "universe"
`,
		},
		{
			name: "multiline basic string",
			input: `
text = """
Line 1
Line 2
Line 3"""
other = "simple"
`,
			modifyKey:   "other",
			modifyValue: "modified",
			want: `text = """
Line 1
Line 2
Line 3"""
other = "modified"
`,
		},
		{
			name: "multiline literal string",
			input: `
text = '''
Line 1
Line 2
Line 3'''
other = "simple"
`,
			modifyKey:   "other",
			modifyValue: "modified",
			want: `text = '''
Line 1
Line 2
Line 3'''
other = "modified"
`,
		},
		{
			name: "raw string literals",
			input: `
path = 'C:\Users\nodejs\templates'
regex = '<\i\c*\s*>'
other = "normal"
`,
			modifyKey:   "other",
			modifyValue: "changed",
			want: `path = 'C:\Users\nodejs\templates'
regex = '<\i\c*\s*>'
other = "changed"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ParseString(tt.input)
			if err != nil {
				t.Fatalf("ParseString() error = %v", err)
			}

			err = doc.Set(tt.modifyKey, tt.modifyValue)
			if err != nil {
				t.Errorf("Set() error = %v", err)
			}

			output := doc.String()
			wantGolden(t, output, tt.want)
		})
	}
}

// TestQuoteStylePreservationOnModification tests that quote style is preserved when modifying a value
func TestQuoteStylePreservationOnModification(t *testing.T) {
	input := `
single_quoted = 'hello'
double_quoted = "world"
multiline = """
text
here"""
`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	doc.Set("single_quoted", "goodbye")
	doc.Set("double_quoted", "universe")

	output := doc.String()

	wantGolden(t, output, `single_quoted = 'goodbye'
double_quoted = "universe"
multiline = """
text
here"""
`)
}

// TestNestednessStylePreservation tests that dotted keys vs sections are preserved
func TestNestednessStylePreservation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		modifyKey   string
		modifyValue any
		want        string
	}{
		{
			name: "dotted keys should stay dotted",
			input: `
# Dotted key style
server.host = "localhost"
server.port = 8080
database.url = "postgres://localhost"
`,
			modifyKey:   "server.host",
			modifyValue: "0.0.0.0",
			want: `# Dotted key style
server.host = "0.0.0.0"
server.port = 8080
database.url = "postgres://localhost"
`,
		},
		{
			name: "sections should stay sections",
			input: `
# Section style
[server]
host = "localhost"
port = 8080

[database]
url = "postgres://localhost"
`,
			modifyKey:   "server.host",
			modifyValue: "0.0.0.0",
			want: `# Section style
[server]
host = "0.0.0.0"
port = 8080

[database]
url = "postgres://localhost"
`,
		},
		{
			name: "mixed styles should be preserved",
			input: `
# Mixed styles
app.name = "myapp"
app.version = 1

[server]
host = "localhost"
port = 8080

database.url = "postgres://localhost"
database.port = 5432
`,
			modifyKey:   "app.version",
			modifyValue: 2,
			want: `# Mixed styles
app.name = "myapp"
app.version = 2

[server]
host = "localhost"
port = 8080
database.url = "postgres://localhost"
database.port = 5432
`,
		},
		{
			name: "nested sections",
			input: `
[server]
host = "localhost"

[server.tls]
enabled = true
cert = "/path/to/cert"
`,
			modifyKey:   "server.host",
			modifyValue: "0.0.0.0",
			want: `[server]
host = "0.0.0.0"

[server.tls]
enabled = true
cert = "/path/to/cert"
`,
		},
		{
			name: "deeply nested dotted keys",
			input: `
a.b.c.d = 1
a.b.c.e = 2
a.b.f = 3
`,
			modifyKey:   "a.b.c.d",
			modifyValue: 10,
			want: `a.b.c.d = 10
a.b.c.e = 2
a.b.f = 3
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ParseString(tt.input)
			if err != nil {
				t.Fatalf("ParseString() error = %v", err)
			}

			err = doc.Set(tt.modifyKey, tt.modifyValue)
			if err != nil {
				t.Errorf("Set() error = %v", err)
			}

			output := doc.String()
			wantGolden(t, output, tt.want)
		})
	}
}

// TestAddingNewKeysPreservesStyle tests that when adding new keys to existing structures,
// we follow the existing style
func TestAddingNewKeysPreservesStyle(t *testing.T) {
	t.Run("adding to section", func(t *testing.T) {
		input := `
[server]
host = "localhost"
port = 8080
`

		doc, _ := ParseString(input)
		doc.Set("server.timeout", 30)

		output := doc.String()

		wantGolden(t, output, `[server]
host = "localhost"
port = 8080
timeout = 30
`)
	})

	t.Run("adding to dotted keys", func(t *testing.T) {
		input := `
app.name = "myapp"
app.version = 1
`

		doc, _ := ParseString(input)
		doc.Set("app.debug", true)

		output := doc.String()

		wantGolden(t, output, `app.name = "myapp"
app.version = 1
app.debug = true
`)
	})
}

// TestComplexPreservation tests that all preservation features work together
func TestComplexPreservation(t *testing.T) {
	input := `# Application Configuration
# Version 1.0

# Basic settings
app_name = "myapp"
app_version = 1

# Server configuration with dotted keys
server.host = "localhost"
server.port = 8080

# Database section
[database]
driver = 'postgres'  # Using single quotes
host = "localhost"   # Using double quotes
port = 5432

# Multiline description
description = """
This is a
multiline string
with multiple lines"""

# Feature flags (specific order matters!)
[features]
feature_a = true
feature_z = false
feature_m = true
feature_b = false
`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	doc.Set("app_version", 2)
	doc.Set("server.port", 9090)
	doc.Set("database.host", "db.example.com")
	doc.Set("features.feature_m", false)

	output := doc.String()

	wantGolden(t, output, `# Application Configuration
# Version 1.0

# Basic settings
app_name = "myapp"
app_version = 2

# Server configuration with dotted keys
server.host = "localhost"
server.port = 9090

# Database section
[database]
driver = 'postgres'  # Using single quotes
host = "db.example.com"  # Using double quotes
port = 5432

# Multiline description
description = """
This is a
multiline string
with multiple lines"""

# Feature flags (specific order matters!)
[features]
feature_a = true
feature_z = false
feature_m = false
feature_b = false
`)

	// Verify the output is still valid TOML
	_, err = ParseString(output)
	if err != nil {
		t.Errorf("Output is not valid TOML: %v", err)
	}
}

// TestRoundTripPreservation tests that multiple round-trips preserve everything
func TestRoundTripPreservation(t *testing.T) {
	input := `# Config
zebra = 'last'
apple = "first"

[section]
zulu = 1
alpha = 2
`

	doc1, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	output1 := doc1.String()

	// Parse the output again
	doc2, err := ParseString(output1)
	if err != nil {
		t.Fatalf("Second ParseString() error = %v", err)
	}

	output2 := doc2.String()

	if output1 != output2 {
		t.Errorf("Round-trip not stable\nFirst output:\n%s\nSecond output:\n%s", output1, output2)
	}
}
