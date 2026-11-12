package reporter_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/Hossiy21/razify/internal/reporter"
	"github.com/Hossiy21/razify/internal/scanner"
	"github.com/Hossiy21/razify/internal/schema"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

func TestSarifValidationAgainstOfficialSchema(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "testdata", "sarif-schema-2.1.0.json")
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft7

	sch, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("failed to compile official SARIF v2.1.0 JSON schema: %v", err)
	}

	res := reporter.Result{
		TargetFile:   ".env",
		TemplateFile: ".env.example",
		Status:       "FAILED",
		Diagnostics: []reporter.Diagnostic{
			{
				RuleID:   "razify/secret-leak",
				Message:  "AWS Access Key ID detected",
				Severity: reporter.SeverityError,
				File:     ".env",
				Line:     5,
				Key:      "AWS_ACCESS_KEY",
			},
			{
				RuleID:   "razify/missing-key",
				Message:  "Required key not found",
				Severity: reporter.SeverityError,
				File:     ".env",
				Line:     1,
				Key:      "DATABASE_URL",
			},
		},
		ValidationSummary: schema.ValidateSummary{Missing: 1},
		SecretSummary:     scanner.ScanSummary{Critical: 1, Total: 1},
	}

	sarifJSON, err := reporter.RenderSarif(res, "1.0.5")
	if err != nil {
		t.Fatalf("RenderSarif failed: %v", err)
	}

	var rawDoc interface{}
	if err := json.Unmarshal([]byte(sarifJSON), &rawDoc); err != nil {
		t.Fatalf("failed to unmarshal generated SARIF: %v", err)
	}

	if err := sch.Validate(rawDoc); err != nil {
		t.Fatalf("SARIF output failed official v2.1.0 JSON schema validation:\n%v\nGenerated SARIF:\n%s", err, sarifJSON)
	}
}

func TestSarifCleanValidationAgainstOfficialSchema(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "testdata", "sarif-schema-2.1.0.json")
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft7

	sch, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("failed to compile official SARIF v2.1.0 JSON schema: %v", err)
	}

	res := reporter.Result{
		TargetFile:   ".env",
		TemplateFile: ".env.example",
		Status:       "PASSED",
		Diagnostics:  []reporter.Diagnostic{},
	}

	sarifJSON, err := reporter.RenderSarif(res, "1.0.5")
	if err != nil {
		t.Fatalf("RenderSarif failed: %v", err)
	}

	var rawDoc interface{}
	if err := json.Unmarshal([]byte(sarifJSON), &rawDoc); err != nil {
		t.Fatalf("failed to unmarshal generated SARIF: %v", err)
	}

	if err := sch.Validate(rawDoc); err != nil {
		t.Fatalf("Clean SARIF output failed official v2.1.0 JSON schema validation:\n%v\nGenerated SARIF:\n%s", err, sarifJSON)
	}
}
