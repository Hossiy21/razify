package cmd

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func TestGoldenCheckJSON(t *testing.T) {
	testCases := []struct {
		name        string
		envFile     string
		exampleFile string
		goldenFile  string
		wantStatus  string
	}{
		{
			name:        "clean",
			envFile:     filepath.Join("..", "testdata", "fixtures", "clean.env"),
			exampleFile: filepath.Join("..", "testdata", "fixtures", "clean.env.example"),
			goldenFile:  filepath.Join("..", "testdata", "golden", "clean.json"),
			wantStatus:  "PASSED",
		},
		{
			name:        "missing",
			envFile:     filepath.Join("..", "testdata", "fixtures", "missing.env"),
			exampleFile: filepath.Join("..", "testdata", "fixtures", "missing.env.example"),
			goldenFile:  filepath.Join("..", "testdata", "golden", "missing.json"),
			wantStatus:  "FAILED",
		},
		{
			name:        "secret_leak",
			envFile:     filepath.Join("..", "testdata", "fixtures", "secret.env"),
			exampleFile: filepath.Join("..", "testdata", "fixtures", "secret.env.example"),
			goldenFile:  filepath.Join("..", "testdata", "golden", "secret_leak.json"),
			wantStatus:  "FAILED",
		},
		{
			name:        "invalid_type",
			envFile:     filepath.Join("..", "testdata", "fixtures", "invalid.env"),
			exampleFile: filepath.Join("..", "testdata", "fixtures", "invalid.env.example"),
			goldenFile:  filepath.Join("..", "testdata", "golden", "invalid_type.json"),
			wantStatus:  "FAILED",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := RunCheck(tc.envFile, tc.exampleFile)
			if err != nil {
				t.Fatalf("RunCheck failed: %v", err)
			}

			if output.Status != tc.wantStatus {
				t.Fatalf("expected status %s, got %s", tc.wantStatus, output.Status)
			}

			// Normalize paths to forward slashes for golden comparison across OSes
			normalizedOutput := output
			normalizedOutput.EnvFile = filepath.ToSlash(output.EnvFile)
			normalizedOutput.ExampleFile = filepath.ToSlash(output.ExampleFile)
			for i, rem := range normalizedOutput.RemediationSteps {
				normalizedOutput.RemediationSteps[i] = filepath.ToSlash(rem)
			}

			actualJSON, err := json.MarshalIndent(normalizedOutput, "", "  ")
			if err != nil {
				t.Fatalf("json.MarshalIndent failed: %v", err)
			}

			if *updateGolden {
				if err := os.WriteFile(tc.goldenFile, actualJSON, 0644); err != nil {
					t.Fatalf("failed to update golden file %s: %v", tc.goldenFile, err)
				}
				return
			}

			expectedJSON, err := os.ReadFile(tc.goldenFile)
			if err != nil {
				if os.IsNotExist(err) {
					if err := os.WriteFile(tc.goldenFile, actualJSON, 0644); err != nil {
						t.Fatalf("failed to create golden file %s: %v", tc.goldenFile, err)
					}
					return
				}
				t.Fatalf("failed to read golden file %s: %v", tc.goldenFile, err)
			}

			actualStr := strings.TrimSpace(string(actualJSON))
			expectedStr := strings.TrimSpace(strings.ReplaceAll(string(expectedJSON), "\r\n", "\n"))

			if actualStr != expectedStr {
				t.Errorf("JSON output does not match golden file %s.\nGot:\n%s\nWant:\n%s", tc.goldenFile, actualStr, expectedStr)
			}
		})
	}
}

func TestExitCodes(t *testing.T) {
	// Build razify binary for testing exit codes
	binName := "razify_test_bin"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(t.TempDir(), binName)
	buildCmd := exec.Command("go", "build", "-o", binPath, "..")
	var buildErr bytes.Buffer
	buildCmd.Stderr = &buildErr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build test binary: %v: %s", err, buildErr.String())
	}

	cleanEnv := filepath.Join("..", "testdata", "fixtures", "clean.env")
	cleanExample := filepath.Join("..", "testdata", "fixtures", "clean.env.example")
	missingEnv := filepath.Join("..", "testdata", "fixtures", "missing.env")
	missingExample := filepath.Join("..", "testdata", "fixtures", "missing.env.example")
	secretEnv := filepath.Join("..", "testdata", "fixtures", "secret.env")
	invalidEnv := filepath.Join("..", "testdata", "fixtures", "invalid.env")
	invalidExample := filepath.Join("..", "testdata", "fixtures", "invalid.env.example")
	nonExistent := filepath.Join("..", "testdata", "fixtures", "does_not_exist.env")

	tests := []struct {
		name         string
		args         []string
		expectedCode int
	}{
		// 1. check command
		{"check clean exit 0", []string{"check", cleanEnv, cleanExample}, 0},
		{"check clean --json exit 0", []string{"check", cleanEnv, cleanExample, "--json"}, 0},
		{"check clean --sarif exit 0", []string{"check", cleanEnv, cleanExample, "--sarif"}, 0},
		{"check missing keys exit 1", []string{"check", missingEnv, missingExample}, 1},
		{"check missing keys --json exit 1", []string{"check", missingEnv, missingExample, "--json"}, 1},
		{"check missing keys --sarif exit 1", []string{"check", missingEnv, missingExample, "--sarif"}, 1},
		{"check missing keys --exit-code 42", []string{"check", missingEnv, missingExample, "--exit-code", "42"}, 42},
		{"check reject invalid --exit-code 0", []string{"check", missingEnv, missingExample, "--exit-code", "0"}, 2},
		{"check reject invalid --exit-code 2", []string{"check", missingEnv, missingExample, "--exit-code", "2"}, 2},
		{"check fatal missing env file exit 2", []string{"check", nonExistent, cleanExample}, 2},
		{"check fatal missing example file exit 2", []string{"check", cleanEnv, nonExistent}, 2},

		// 2. scan command
		{"scan clean exit 0", []string{"scan", cleanEnv}, 0},
		{"scan secret leak exit 1", []string{"scan", secretEnv}, 1},
		{"scan secret leak --json exit 1", []string{"scan", secretEnv, "--json"}, 1},
		{"scan secret leak --sarif exit 1", []string{"scan", secretEnv, "--sarif"}, 1},
		{"scan secret leak --exit-code 99", []string{"scan", secretEnv, "--exit-code", "99"}, 99},
		{"scan fatal file missing exit 2", []string{"scan", nonExistent}, 2},

		// 3. validate command
		{"validate clean exit 0", []string{"validate", cleanEnv, cleanExample}, 0},
		{"validate missing exit 1", []string{"validate", missingEnv, missingExample}, 1},
		{"validate invalid exit 1", []string{"validate", invalidEnv, invalidExample}, 1},
		{"validate invalid --sarif exit 1", []string{"validate", invalidEnv, invalidExample, "--sarif"}, 1},
		{"validate invalid --exit-code 15", []string{"validate", invalidEnv, invalidExample, "--exit-code", "15"}, 15},
		{"validate fatal file missing exit 2", []string{"validate", nonExistent, cleanExample}, 2},

		// 4. diff command
		{"diff identical exit 0", []string{"diff", cleanEnv, cleanEnv}, 0},
		{"diff differing exit 1", []string{"diff", cleanEnv, missingEnv}, 1},
		{"diff differing --exit-code 7", []string{"diff", cleanEnv, missingEnv, "--exit-code", "7"}, 7},
		{"diff fatal file missing exit 2", []string{"diff", nonExistent, cleanEnv}, 2},

		// 5. audit command
		{"audit clean exit 0", []string{"audit", cleanEnv, cleanExample}, 0},
		{"audit issues exit 1", []string{"audit", secretEnv, cleanExample}, 1},
		{"audit fatal file missing exit 2", []string{"audit", nonExistent, cleanExample}, 2},

		// 6. version command (offline)
		{"version offline exit 0", []string{"version"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, tt.args...)
			err := cmd.Run()
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					t.Fatalf("unexpected error running command: %v", err)
				}
			}

			if exitCode != tt.expectedCode {
				t.Errorf("expected exit code %d, got %d", tt.expectedCode, exitCode)
			}
		})
	}
}

func BenchmarkCheck(b *testing.B) {
	cleanEnv := filepath.Join("..", "testdata", "fixtures", "clean.env")
	cleanExample := filepath.Join("..", "testdata", "fixtures", "clean.env.example")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := RunCheck(cleanEnv, cleanExample)
		if err != nil {
			b.Fatalf("RunCheck benchmark failed: %v", err)
		}
	}
}
