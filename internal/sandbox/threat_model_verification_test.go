package sandbox_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hossiy21/razify/internal/sandbox"
	"github.com/Hossiy21/razify/internal/schema"
)

func TestSandboxArgv_FakeSecrets(t *testing.T) {
	fake := sandbox.GenerateFakeSecret("STRIPE_KEY", "real-key-value", "secret")
	if !strings.HasPrefix(fake, "razify_fake_") {
		t.Fatalf("expected razify_fake_ prefix, got %s", fake)
	}

	mockStripe := "sk_live_" + "000000000000000000000000"
	exampleVars := []schema.EnvVar{
		{Key: "STRIPE_KEY", Tags: map[string]string{"type": "secret"}},
	}
	envValues := map[string]string{
		"STRIPE_KEY": mockStripe,
	}

	injected, _ := sandbox.ProcessEnvForSandbox(exampleVars, envValues, nil, false)
	if injected["STRIPE_KEY"] == mockStripe {
		t.Fatalf("real secret was not replaced with fake secret")
	}
	if !strings.HasPrefix(injected["STRIPE_KEY"], "razify_fake_stripe_key_") {
		t.Fatalf("unexpected fake secret format: %s", injected["STRIPE_KEY"])
	}
}

func TestSandboxArgv_ScopedVariables(t *testing.T) {
	exampleVars := []schema.EnvVar{
		{Key: "PORT", Value: "8080"},
		{Key: "ALLOWED_KEY", Value: "allowed"},
	}
	envValues := map[string]string{
		"PORT":        "8080",
		"ALLOWED_KEY": "allowed",
		"SECRET_LEAK": "untracked_secret",
	}

	injected, _ := sandbox.ProcessEnvForSandbox(exampleVars, envValues, []string{"PORT"}, false)
	if len(injected) != 1 || injected["PORT"] != "8080" {
		t.Fatalf("expected only PORT in injected vars, got %+v", injected)
	}
	if _, exists := injected["SECRET_LEAK"]; exists {
		t.Fatalf("untracked secret was injected!")
	}
}

func TestSandboxArgv_NoInlineValues(t *testing.T) {
	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = t.TempDir()
	cfg.Command = []string{"echo", "hi"}

	injected := map[string]string{
		"SECRET_TOKEN": "dont_show_in_argv",
	}
	profile := sandbox.GetProfileConfig("generic")

	args, envMap, err := sandbox.BuildDockerArgs(cfg, nil, "", injected, profile)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if strings.Contains(argsStr, "dont_show_in_argv") {
		t.Fatalf("secret leaked in argv string: %s", argsStr)
	}
	if strings.Contains(argsStr, "-e SECRET_TOKEN=") {
		t.Fatalf("inline env assignment in argv: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-e SECRET_TOKEN") {
		t.Fatalf("missing -e SECRET_TOKEN in argv: %s", argsStr)
	}
	if envMap["SECRET_TOKEN"] != "dont_show_in_argv" {
		t.Fatalf("missing SECRET_TOKEN in process environment map")
	}
}

func TestSandboxArgv_NetworkNone(t *testing.T) {
	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = t.TempDir()
	cfg.Command = []string{"echo", "hi"}
	profile := sandbox.GetProfileConfig("generic")

	args, _, err := sandbox.BuildDockerArgs(cfg, nil, "", nil, profile)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "--network none") {
		t.Fatalf("expected --network none in argv: %s", argsStr)
	}
}

func TestSandboxArgv_WorkspaceReadOnly(t *testing.T) {
	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = t.TempDir()
	cfg.ReadOnlyWorkspace = true
	cfg.Command = []string{"echo", "hi"}
	profile := sandbox.GetProfileConfig("generic")

	args, _, err := sandbox.BuildDockerArgs(cfg, nil, "", nil, profile)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "/workspace:ro") {
		t.Fatalf("expected workspace to be mounted :ro in argv: %s", argsStr)
	}
}

func TestSandboxArgv_GitReadOnly(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755)

	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = tmpDir
	cfg.ReadOnlyWorkspace = false // --read-write requested
	cfg.Command = []string{"echo", "hi"}
	profile := sandbox.GetProfileConfig("generic")

	args, _, err := sandbox.BuildDockerArgs(cfg, nil, "", nil, profile)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "/workspace/.git:ro") {
		t.Fatalf(".git was not protected with :ro under --read-write: %s", argsStr)
	}
}

func TestSandboxArgv_ReadOnlyRootfs(t *testing.T) {
	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = t.TempDir()
	cfg.ReadOnlyRootfs = true
	cfg.Command = []string{"echo", "hi"}
	profile := sandbox.GetProfileConfig("generic")

	args, _, err := sandbox.BuildDockerArgs(cfg, nil, "", nil, profile)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "--read-only") {
		t.Fatalf("expected --read-only in argv: %s", argsStr)
	}
}

func TestSandboxArgv_TmpfsExec(t *testing.T) {
	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = t.TempDir()
	cfg.Command = []string{"echo", "hi"}
	profile := sandbox.GetProfileConfig("go")

	args, _, err := sandbox.BuildDockerArgs(cfg, nil, "", nil, profile)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "--tmpfs /tmp:rw,exec,nosuid,size=512m") {
		t.Fatalf("expected exec-capable /tmp tmpfs in argv: %s", argsStr)
	}
}

func TestSandboxArgv_ProjectProfiles(t *testing.T) {
	profiles := []struct {
		name         string
		expectedImg  string
		expectedEnv  string
	}{
		{"go", "golang:1.25-alpine", "-e GOCACHE"},
		{"node", "node:22-alpine", "-e npm_config_cache"},
		{"python", "python:3.12-alpine", "-e PYTHONPYCACHEPREFIX"},
		{"generic", "alpine:latest", "--tmpfs /tmp:rw,exec,nosuid,size=256m"},
	}

	for _, p := range profiles {
		t.Run(p.name, func(t *testing.T) {
			cfg := sandbox.DefaultConfig()
			cfg.WorkspaceDir = t.TempDir()
			cfg.Command = []string{"echo", "hi"}
			prof := sandbox.GetProfileConfig(p.name)

			args, _, err := sandbox.BuildDockerArgs(cfg, nil, "", nil, prof)
			if err != nil {
				t.Fatalf("failed: %v", err)
			}
			argsStr := strings.Join(args, " ")
			if !strings.Contains(argsStr, p.expectedImg) {
				t.Errorf("expected image %s in argv: %s", p.expectedImg, argsStr)
			}
			if !strings.Contains(argsStr, p.expectedEnv) {
				t.Errorf("expected %s in argv: %s", p.expectedEnv, argsStr)
			}
		})
	}
}

func TestSandboxArgv_MaskEnv(t *testing.T) {
	tmpDir := t.TempDir()
	emptyMask := filepath.Join(tmpDir, "empty.mask")
	_ = os.WriteFile(emptyMask, nil, 0644)

	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = tmpDir
	cfg.MaskEnvFiles = true
	cfg.Command = []string{"echo", "hi"}
	profile := sandbox.GetProfileConfig("generic")

	args, _, err := sandbox.BuildDockerArgs(cfg, []string{".env", ".env.test"}, emptyMask, nil, profile)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "/workspace/.env:ro") {
		t.Errorf("expected .env to be masked with empty file: %s", argsStr)
	}
	if !strings.Contains(argsStr, "/workspace/.env.test:ro") {
		t.Errorf("expected .env.test to be masked with empty file: %s", argsStr)
	}
}

func TestSandboxArgv_CapDropAll(t *testing.T) {
	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = t.TempDir()
	cfg.DropCaps = true
	cfg.Command = []string{"echo", "hi"}
	profile := sandbox.GetProfileConfig("generic")

	args, _, err := sandbox.BuildDockerArgs(cfg, nil, "", nil, profile)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "--cap-drop=ALL") {
		t.Fatalf("expected --cap-drop=ALL in argv: %s", argsStr)
	}
}

func TestSandboxArgv_NoNewPrivileges(t *testing.T) {
	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = t.TempDir()
	cfg.Command = []string{"echo", "hi"}
	profile := sandbox.GetProfileConfig("generic")

	args, _, err := sandbox.BuildDockerArgs(cfg, nil, "", nil, profile)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "--security-opt no-new-privileges") {
		t.Fatalf("expected --security-opt no-new-privileges in argv: %s", argsStr)
	}
}

func TestSandboxArgv_ResourceLimits(t *testing.T) {
	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = t.TempDir()
	cfg.Memory = "4g"
	cfg.CPUs = "4.0"
	cfg.PidsLimit = 512
	cfg.Command = []string{"echo", "hi"}
	profile := sandbox.GetProfileConfig("generic")

	args, _, err := sandbox.BuildDockerArgs(cfg, nil, "", nil, profile)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "--memory 4g") || !strings.Contains(argsStr, "--cpus 4.0") || !strings.Contains(argsStr, "--pids-limit 512") {
		t.Fatalf("expected resource limits in argv: %s", argsStr)
	}
}

func TestSandboxArgv_InitProcess(t *testing.T) {
	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = t.TempDir()
	cfg.Init = true
	cfg.Command = []string{"echo", "hi"}
	profile := sandbox.GetProfileConfig("generic")

	args, _, err := sandbox.BuildDockerArgs(cfg, nil, "", nil, profile)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "--init") {
		t.Fatalf("expected --init in argv: %s", argsStr)
	}
}

func TestSandboxArgv_AutoRemove(t *testing.T) {
	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = t.TempDir()
	cfg.AutoRemove = true
	cfg.Command = []string{"echo", "hi"}
	profile := sandbox.GetProfileConfig("generic")

	args, _, err := sandbox.BuildDockerArgs(cfg, nil, "", nil, profile)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "--rm") {
		t.Fatalf("expected --rm in argv: %s", argsStr)
	}
}

func TestSandboxArgv_UserMapping(t *testing.T) {
	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = t.TempDir()
	cfg.User = "1001:1001"
	cfg.Command = []string{"echo", "hi"}
	profile := sandbox.GetProfileConfig("generic")

	args, _, err := sandbox.BuildDockerArgs(cfg, nil, "", nil, profile)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "--user 1001:1001") {
		t.Fatalf("expected --user 1001:1001 in argv: %s", argsStr)
	}
}

func TestSandboxArgv_Rejections(t *testing.T) {
	rejections := []sandbox.Config{
		{Privileged: true},
		{PidMode: "host"},
		{Network: "host"},
		{Seccomp: "unconfined"},
		{OutputDir: "/var/run/docker.sock"},
	}

	for _, cfg := range rejections {
		if err := sandbox.ValidateGuardrails(cfg); err == nil {
			t.Fatalf("expected guardrail rejection for config %+v, got nil", cfg)
		}
	}
}

func TestSandboxWarnings(t *testing.T) {
	cfg := sandbox.Config{
		Network:           "bridge",
		InjectSecrets:     true,
		MaskEnvFiles:      false,
		ReadOnlyWorkspace: false,
	}

	warnings := sandbox.CheckWarnings(cfg)
	if len(warnings) != 4 {
		t.Fatalf("expected 4 warnings, got %d", len(warnings))
	}
}

func TestSandboxOutputMasking(t *testing.T) {
	secret := "sensitive_production_token_abc"
	input := "Connected with sensitive_production_token_abc to server"
	masked := sandbox.MaskString(input, []string{secret})

	if strings.Contains(masked, secret) {
		t.Fatalf("secret leaked: %s", masked)
	}
	if !strings.Contains(masked, "[REDACTED]") {
		t.Fatalf("expected [REDACTED], got: %s", masked)
	}
}

