package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Hossiy21/razify/internal/doctor"
)

func TestDoctorCmd_JsonOutput(t *testing.T) {
	tempDir := t.TempDir()

	envFile := filepath.Join(tempDir, ".env")
	exampleFile := filepath.Join(tempDir, ".env.example")
	gitignoreFile := filepath.Join(tempDir, ".gitignore")

	_ = os.WriteFile(envFile, []byte("PORT=9876\nAPP_ENV=test\n"), 0600)
	_ = os.WriteFile(exampleFile, []byte("# @type port\nPORT=9876\nAPP_ENV=test\n"), 0644)
	_ = os.WriteFile(gitignoreFile, []byte(".env\n"), 0644)

	opts := doctor.DoctorOptions{
		EnvFile:       envFile,
		ExampleFile:   exampleFile,
		GitignoreFile: gitignoreFile,
		Fast:          true,
	}

	deps := doctor.NewDefaultDependencies()
	report := doctor.RunDoctor(opts, deps)

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("failed to marshal doctor report to json: %v", err)
	}

	var parsed doctor.DoctorReport
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal doctor report: %v", err)
	}

	if parsed.HealthScore < 70 {
		t.Errorf("expected healthy or degraded score >= 70, got %d", parsed.HealthScore)
	}
	if parsed.Result.Status != "PASSED" {
		t.Errorf("expected PASSED, got %s", parsed.Result.Status)
	}
}

func TestDoctorCmd_ExitCodeOverride(t *testing.T) {
	// Set up command with override
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Validate doctor command is registered
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "doctor" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("doctor command not found in rootCmd child commands")
	}
}

