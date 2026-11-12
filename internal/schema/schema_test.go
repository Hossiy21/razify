package schema_test

import (
	"strings"
	"testing"

	"github.com/Hossiy21/razify/internal/schema"
)

func TestParseEnvReader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:  "standard key-values",
			input: "FOO=bar\nBAZ=qux\n# comment\n\nEMPTY=\n",
			expected: map[string]string{
				"FOO":   "bar",
				"BAZ":   "qux",
				"EMPTY": "",
			},
		},
		{
			name:     "empty input",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:  "spaces around equals",
			input: "  KEY  =  value with spaces  \n",
			expected: map[string]string{
				"KEY": "value with spaces",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars, err := schema.ParseEnvReader(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(vars) != len(tt.expected) {
				t.Fatalf("expected %d entries, got %d", len(tt.expected), len(vars))
			}
			for k, v := range tt.expected {
				if vars[k] != v {
					t.Errorf("for key %q: expected %q, got %q", k, v, vars[k])
				}
			}
		})
	}
}

func TestValidateValueTable(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		tags     map[string]string
		env      map[string]string
		wantPass bool
	}{
		{"valid int", "42", map[string]string{"type": "int"}, nil, true},
		{"invalid int", "42a", map[string]string{"type": "int"}, nil, false},
		{"valid bool true", "true", map[string]string{"type": "bool"}, nil, true},
		{"valid bool 0", "0", map[string]string{"type": "bool"}, nil, true},
		{"invalid bool", "yes", map[string]string{"type": "bool"}, nil, false},
		{"valid url", "https://api.razify.dev/v1", map[string]string{"type": "url"}, nil, true},
		{"invalid url", "not_a_url", map[string]string{"type": "url"}, nil, false},
		{"valid email", "dev@example.com", map[string]string{"type": "email"}, nil, true},
		{"invalid email", "invalid@", map[string]string{"type": "email"}, nil, false},
		{"valid port", "8080", map[string]string{"type": "port"}, nil, true},
		{"port out of range", "70000", map[string]string{"type": "port"}, nil, false},
		{"valid uuid", "550e8400-e29b-41d4-a716-446655440000", map[string]string{"type": "uuid"}, nil, true},
		{"invalid uuid", "1234", map[string]string{"type": "uuid"}, nil, false},
		{"valid json", `{"key":"val"}`, map[string]string{"type": "json"}, nil, true},
		{"invalid json", `{bad_json}`, map[string]string{"type": "json"}, nil, false},
		{"valid ip", "192.168.1.1", map[string]string{"type": "ip"}, nil, true},
		{"invalid ip", "999.999.999.999", map[string]string{"type": "ip"}, nil, false},
		{"enum match", "staging", map[string]string{"enum": "dev,staging,prod"}, nil, true},
		{"enum mismatch", "test", map[string]string{"enum": "dev,staging,prod"}, nil, false},
		{"range match", "50", map[string]string{"range": "1-100"}, nil, true},
		{"range low", "0", map[string]string{"range": "1-100"}, nil, false},
		{"requires present", "secret", map[string]string{"requires": "DB_HOST"}, map[string]string{"DB_HOST": "localhost"}, true},
		{"requires missing", "secret", map[string]string{"requires": "DB_HOST"}, map[string]string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := schema.ValidateValue(tt.value, tt.tags, tt.env)
			if (errStr == "") != tt.wantPass {
				t.Errorf("validateValue(%q, %v): pass=%v, got err=%q", tt.value, tt.tags, tt.wantPass, errStr)
			}
		})
	}
}

func BenchmarkValidate(b *testing.B) {
	envVars := map[string]string{
		"PORT":        "3000",
		"NODE_ENV":    "production",
		"ADMIN_EMAIL": "admin@razify.internal",
		"API_KEY":     "some-configured-token-value",
	}

	exampleVars := []schema.EnvVar{
		{Key: "PORT", Value: "3000", Tags: map[string]string{"type": "port", "range": "1000-65535"}, Required: true},
		{Key: "NODE_ENV", Value: "development", Tags: map[string]string{"enum": "development,staging,production"}, Required: true},
		{Key: "ADMIN_EMAIL", Value: "", Tags: map[string]string{"type": "email"}, Required: true},
		{Key: "API_KEY", Value: "", Tags: map[string]string{"type": "string"}, Required: true},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = schema.Validate(envVars, exampleVars)
	}
}

