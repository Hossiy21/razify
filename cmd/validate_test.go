package cmd

import (
	"os"
	"testing"
)

func TestValidate(t *testing.T) {
	exampleContent := `
REQUIRED_VAR=
PLACEHOLDER_VAR=your-value-here
OPTIONAL_VAR=
`
	envContent := `
REQUIRED_VAR=actual-value
PLACEHOLDER_VAR=your-value-here
# OPTIONAL_VAR is missing
`
	exampleFile, _ := os.CreateTemp("", "example.env")
	envFile, _ := os.CreateTemp("", "actual.env")

	defer os.Remove(exampleFile.Name())
	defer os.Remove(envFile.Name())

	os.WriteFile(exampleFile.Name(), []byte(exampleContent), 0644)
	os.WriteFile(envFile.Name(), []byte(envContent), 0644)

	missing, placeholder, empty, okCount, _, err := RunValidate(envFile.Name(), exampleFile.Name())
	if err != nil {
		t.Fatalf("RunValidate failed: %v", err)
	}

	if missing != 1 {
		t.Errorf("Expected 1 missing variable (OPTIONAL_VAR), got %d", missing)
	}
	if placeholder != 1 {
		t.Errorf("Expected 1 placeholder variable (PLACEHOLDER_VAR), got %d", placeholder)
	}
	if okCount != 1 {
		t.Errorf("Expected 1 OK variable (REQUIRED_VAR), got %d", okCount)
	}
	if empty != 0 {
		t.Errorf("Expected 0 empty variables, got %d", empty)
	}
}

func TestRichSchemaValidation(t *testing.T) {
	exampleContent := `
# @type(email)
ADMIN_EMAIL=admin@example.com

# @type(port) @range(1000-65535)
APP_PORT=8080

# @enum(dev,staging,prod)
NODE_ENV=dev

# @type(uuid)
INSTANCE_ID=550e8400-e29b-41d4-a716-446655440000

# @type(json)
APP_CONFIG={"feature":true}

# @type(ip)
SERVER_IP=192.168.1.1

# @requires(DB_HOST)
DB_PASS=secret
`

	validEnv := `
ADMIN_EMAIL=dev@company.org
APP_PORT=3000
NODE_ENV=staging
INSTANCE_ID=123e4567-e89b-12d3-a456-426614174000
APP_CONFIG={"debug":false}
SERVER_IP=10.0.0.1
DB_HOST=localhost
DB_PASS=secret
`

	invalidEnv := `
ADMIN_EMAIL=invalid-email
APP_PORT=70000
NODE_ENV=production_invalid
INSTANCE_ID=not-a-uuid
APP_CONFIG={bad_json}
SERVER_IP=999.999.999.999
# DB_HOST is missing
DB_PASS=secret
`

	exFile, _ := os.CreateTemp("", "example.env")
	vFile, _ := os.CreateTemp("", "valid.env")
	invFile, _ := os.CreateTemp("", "invalid.env")

	defer os.Remove(exFile.Name())
	defer os.Remove(vFile.Name())
	defer os.Remove(invFile.Name())

	os.WriteFile(exFile.Name(), []byte(exampleContent), 0644)
	os.WriteFile(vFile.Name(), []byte(validEnv), 0644)
	os.WriteFile(invFile.Name(), []byte(invalidEnv), 0644)

	// Valid environment should have 0 missing/invalid keys
	missingValid, _, _, okValid, _, err := RunValidate(vFile.Name(), exFile.Name())
	if err != nil {
		t.Fatalf("RunValidate valid failed: %v", err)
	}
	if missingValid != 0 {
		t.Errorf("Expected 0 missing/invalid for valid env, got %d", missingValid)
	}
	if okValid != 7 {
		t.Errorf("Expected 7 OK variables for valid env, got %d", okValid)
	}

	// Invalid environment should fail all 7 checks
	missingInvalid, _, _, _, results, err := RunValidate(invFile.Name(), exFile.Name())
	if err != nil {
		t.Fatalf("RunValidate invalid failed: %v", err)
	}
	if missingInvalid != 7 {
		t.Errorf("Expected 7 invalid variables, got %d", missingInvalid)
	}

	for _, r := range results {
		if r.Status != "INVALID" {
			t.Errorf("Expected status INVALID for %s, got %s", r.Key, r.Status)
		}
	}
}
