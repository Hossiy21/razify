package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Hossiy21/razify/internal/sandbox"
	"github.com/Hossiy21/razify/internal/schema"
)

func TestGoldenSandboxDryRun(t *testing.T) {
	// Setup fixed deterministic config
	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = "/mock/workspace"
	cfg.User = "1000:1000"
	cfg.Command = []string{"go", "test", "./..."}

	exampleVars := []schema.EnvVar{
		{Key: "PORT", Value: "8080", Tags: map[string]string{"type": "int"}},
		{Key: "STRIPE_SECRET_KEY", Value: "", Tags: map[string]string{"type": "secret"}},
	}
	mockSecret := "sk_live_" + "000000000000000000000000"
	envValues := map[string]string{
		"PORT":              "8080",
		"STRIPE_SECRET_KEY": mockSecret,
	}

	injectedEnv, _ := sandbox.ProcessEnvForSandbox(exampleVars, envValues, nil, false)
	profile := sandbox.GetProfileConfig("go")
	maskedFiles := []string{".env", ".env.local"}
	emptyMask := "/tmp/mock_empty_mask.empty"

	args, _, err := sandbox.BuildDockerArgs(cfg, maskedFiles, emptyMask, injectedEnv, profile)
	if err != nil {
		t.Fatalf("BuildDockerArgs failed: %v", err)
	}

	rawOutput := "docker " + strings.Join(args, " ")
	// Normalize Windows drive letter on mock path for cross-platform golden comparison
	rawOutput = regexp.MustCompile(`[A-Za-z]:/mock/workspace`).ReplaceAllString(rawOutput, "/mock/workspace")

	goldenFile := filepath.Join("..", "testdata", "golden", "sandbox_dryrun.golden")

	if *updateGolden {
		if err := os.WriteFile(goldenFile, []byte(rawOutput+"\n"), 0644); err != nil {
			t.Fatalf("failed to update golden file: %v", err)
		}
		t.Logf("Updated golden file %s", goldenFile)
		return
	}

	expectedBytes, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v", goldenFile, err)
	}

	expected := strings.TrimSpace(string(expectedBytes))
	actual := strings.TrimSpace(rawOutput)

	if actual != expected {
		t.Fatalf("Golden file mismatch for sandbox dryrun.\nExpected:\n%s\nGot:\n%s", expected, actual)
	}
}
