package cmd

import (
	"os"
	"testing"
)

func TestShannonEntropy(t *testing.T) {
	tests := []struct {
		input    string
		min      float64
		max      float64
	}{
		{"aaaaa", 0.0, 0.5},                      // Low entropy
		{"abcde", 2.0, 3.0},                      // Medium
		{"sk_live_51MszZ4SDFvks92384729384", 3.5, 5.0}, // High (Secret)
		{"", 0.0, 0.0},                           // Empty
	}

	for _, tt := range tests {
		got := ShannonEntropy(tt.input)
		if got < tt.min || got > tt.max {
			t.Errorf("ShannonEntropy(%q) = %v; want between %v and %v", tt.input, got, tt.min, tt.max)
		}
	}
}

func TestValidateValue(t *testing.T) {
	tests := []struct {
		val  string
		tags map[string]string
		want bool // true if it should pass (empty error string)
	}{
		// Type: int
		{"123", map[string]string{"type": "int"}, true},
		{"abc", map[string]string{"type": "int"}, false},
		
		// Type: bool
		{"true", map[string]string{"type": "bool"}, true},
		{"false", map[string]string{"type": "bool"}, true},
		{"yes", map[string]string{"type": "bool"}, false},
		
		// Type: url
		{"https://google.com", map[string]string{"type": "url"}, true},
		{"not-a-url", map[string]string{"type": "url"}, false},
		
		// Range
		{"50", map[string]string{"range": "1-100"}, true},
		{"150", map[string]string{"range": "1-100"}, false},
		{"1024", map[string]string{"range": "1024-65535"}, true},
	}

	for _, tt := range tests {
		err := validateValue(tt.val, tt.tags)
		if (err == "") != tt.want {
			t.Errorf("validateValue(%q, %v) error = %q; want pass = %v", tt.val, tt.tags, err, tt.want)
		}
	}
}

func TestStripTags(t *testing.T) {
	input := "Database port. @type=int @range=1-100"
	want := "Database port."
	got := stripTags(input)
	if got != want {
		t.Errorf("stripTags(%q) = %q; want %q", input, got, want)
	}
}

func TestParseEnvWithMetadata(t *testing.T) {
	content := `
# @type=int @required=true
PORT=3000

# This is a secret but ignored # razify:ignore
SECRET=sk_live_123
`
	tmpfile, _ := os.CreateTemp("", "test.env")
	defer os.Remove(tmpfile.Name())
	tmpfile.WriteString(content)
	tmpfile.Close()

	vars, err := ParseEnvWithMetadata(tmpfile.Name())
	if err != nil {
		t.Fatalf("ParseEnvWithMetadata failed: %v", err)
	}

	if len(vars) != 2 {
		t.Fatalf("Expected 2 variables, got %d", len(vars))
	}

	// Check PORT
	if vars[0].Key != "PORT" || vars[0].Tags["type"] != "int" || !vars[0].Required {
		t.Errorf("PORT metadata incorrect: %+v", vars[0])
	}

	// Check SECRET
	if vars[1].Key != "SECRET" || !vars[1].Ignored {
		t.Errorf("SECRET ignore logic failed: %+v", vars[1])
	}
}
