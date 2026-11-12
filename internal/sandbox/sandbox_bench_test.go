package sandbox_test

import (
	"bytes"
	"testing"

	"github.com/Hossiy21/razify/internal/sandbox"
	"github.com/Hossiy21/razify/internal/schema"
)

func BenchmarkProcessEnvForSandbox(b *testing.B) {
	exampleVars := []schema.EnvVar{
		{Key: "PORT", Value: "8080", Tags: map[string]string{"type": "int"}},
		{Key: "DB_HOST", Value: "localhost"},
		{Key: "API_SECRET", Value: "", Tags: map[string]string{"type": "secret"}},
		{Key: "STRIPE_KEY", Value: "", Tags: map[string]string{"type": "key"}},
		{Key: "AUTH_TOKEN", Value: ""},
	}
	envValues := map[string]string{
		"PORT":       "8080",
		"DB_HOST":    "127.0.0.1",
		"API_SECRET": "sk_test_" + "000000000000000000000000",
		"STRIPE_KEY": "sk_live_" + "000000000000000000000000",
		"AUTH_TOKEN": "ghp_" + "000000000000000000000000000000000000",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = sandbox.ProcessEnvForSandbox(exampleVars, envValues, nil, false)
	}
}

func BenchmarkBuildDockerArgs(b *testing.B) {
	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = "/mock/workspace"
	cfg.User = "1000:1000"
	cfg.Command = []string{"go", "test", "./..."}

	profile := sandbox.GetProfileConfig("go")
	injectedEnv := map[string]string{
		"PORT":       "8080",
		"STRIPE_KEY": "razify_fake_stripe_key_12345678",
		"DB_PASS":    "razify_fake_db_pass_87654321",
	}
	maskedFiles := []string{".env", ".env.local"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, _ = sandbox.BuildDockerArgs(cfg, maskedFiles, "/tmp/mock.empty", injectedEnv, profile)
	}
}

func BenchmarkStreamMasker(b *testing.B) {
	mockStripe := "sk_live_" + "000000000000000000000000"
	mockGhp := "ghp_" + "000000000000000000000000000000000000"
	secrets := []string{
		mockStripe,
		mockGhp,
		"super_secret_database_password_999",
	}

	payload := []byte("2026-11-12T12:00:00Z INFO User logged in with " + mockStripe + " and " + mockGhp + "\n")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		masker := sandbox.NewStreamMasker(&buf, secrets)
		_, _ = masker.Write(payload)
		_ = masker.Close()
	}
}
