package cmd

import (
	"strings"
	"testing"
)

func TestBuildMarkdown(t *testing.T) {
	vars := []EnvVar{
		{
			Key:      "DB_HOST",
			Value:    "localhost",
			Comment:  "Main database",
			Required: false,
			Category: "Database",
		},
		{
			Key:      "API_KEY",
			Value:    "",
			Comment:  "External API",
			Required: true,
			Category: "API",
		},
	}

	md := buildMarkdown(vars, ".env.example")

	if !strings.Contains(md, "# Environment Variables") {
		t.Error("Markdown does not contain title")
	}
	if !strings.Contains(md, "| `DB_HOST` | No | `localhost` | Main database |") {
		t.Error("Markdown table does not contain DB_HOST entry")
	}
	if !strings.Contains(md, "| `API_KEY` | **Yes** | `—` | External API |") {
		t.Error("Markdown table does not contain API_KEY entry")
	}
	if !strings.Contains(md, "## Database") {
		t.Error("Markdown does not contain Category header")
	}
}
