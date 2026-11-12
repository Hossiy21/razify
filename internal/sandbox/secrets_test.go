package sandbox_test

import (
	"strings"
	"testing"

	"github.com/Hossiy21/razify/internal/sandbox"
	"github.com/Hossiy21/razify/internal/schema"
)

func TestIsSecretClassified(t *testing.T) {
	testCases := []struct {
		name       string
		key        string
		val        string
		tags       map[string]string
		wantSecret bool
	}{
		{
			name:       "schema tag secret",
			key:        "MY_CONFIDENTIAL_DATA",
			val:        "plain-text-data",
			tags:       map[string]string{"type": "secret"},
			wantSecret: true,
		},
		{
			name:       "schema tag password",
			key:        "USER_PASS",
			val:        "pass123",
			tags:       map[string]string{"type": "password"},
			wantSecret: true,
		},
		{
			name:       "sensitive name api_key",
			key:        "SERVICE_API_KEY",
			val:        "custom_value_1234",
			tags:       nil,
			wantSecret: true,
		},
		{
			name:       "sensitive name secret_token",
			key:        "APP_SECRET",
			val:        "some_secret",
			tags:       nil,
			wantSecret: true,
		},
		{
			name:       "provider pattern Stripe key",
			key:        "BILLING_KEY",
			val:        "sk_live_" + "000000000000000000000000",
			tags:       nil,
			wantSecret: true,
		},
		{
			name:       "high entropy string",
			key:        "CUSTOM_VAR",
			val:        "g9X#m2$Lp!q8@zWv1234567890",
			tags:       nil,
			wantSecret: true,
		},
		{
			name:       "non-secret config port",
			key:        "PORT",
			val:        "8080",
			tags:       map[string]string{"type": "int"},
			wantSecret: false,
		},
		{
			name:       "non-secret config node_env",
			key:        "NODE_ENV",
			val:        "production",
			tags:       map[string]string{"type": "string"},
			wantSecret: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := sandbox.IsSecretClassified(tc.key, tc.val, tc.tags)
			if got != tc.wantSecret {
				t.Errorf("IsSecretClassified(%q, %q) = %v, want %v", tc.key, tc.val, got, tc.wantSecret)
			}
		})
	}
}

func TestGenerateFakeSecret(t *testing.T) {
	fake := sandbox.GenerateFakeSecret("STRIPE_KEY", "real-stripe-val", "secret")
	if !strings.HasPrefix(fake, "razify_fake_stripe_key_") {
		t.Fatalf("unexpected fake secret format: %s", fake)
	}

	// Deterministic generation
	fake2 := sandbox.GenerateFakeSecret("STRIPE_KEY", "real-stripe-val", "secret")
	if fake != fake2 {
		t.Errorf("expected deterministic fake secret, got %s and %s", fake, fake2)
	}
}

func TestProcessEnvForSandbox_FakeSecretsDefault(t *testing.T) {
	exampleVars := []schema.EnvVar{
		{Key: "PORT", Value: "3000", Tags: map[string]string{"type": "int"}},
		{Key: "STRIPE_SECRET_KEY", Value: "", Tags: map[string]string{"type": "secret"}},
		{Key: "DB_PASSWORD", Value: "example_pass", Tags: map[string]string{"type": "password"}},
	}

	mockStripe := "sk_live_" + "000000000000000000000000"
	envValues := map[string]string{
		"PORT":              "8080",
		"STRIPE_SECRET_KEY": mockStripe,
		"DB_PASSWORD":       "supersecret123",
		"UNTRACKED_KEY":     "should_not_be_injected",
	}

	// Default: fakeSecrets = true, injectSecrets = false
	injected, rawSecrets := sandbox.ProcessEnvForSandbox(exampleVars, envValues, nil, false)

	if injected["PORT"] != "8080" {
		t.Errorf("PORT should be actual non-secret value, got %s", injected["PORT"])
	}

	if injected["STRIPE_SECRET_KEY"] == mockStripe {
		t.Errorf("STRIPE_SECRET_KEY was not faked!")
	}
	if !strings.HasPrefix(injected["STRIPE_SECRET_KEY"], "razify_fake_") {
		t.Errorf("STRIPE_SECRET_KEY should have razify_fake_ prefix, got %s", injected["STRIPE_SECRET_KEY"])
	}

	if injected["DB_PASSWORD"] == "supersecret123" {
		t.Errorf("DB_PASSWORD was not faked!")
	}

	if _, exists := injected["UNTRACKED_KEY"]; exists {
		t.Errorf("UNTRACKED_KEY should not be injected since it is not in example schema")
	}

	// Raw secrets collected for output masking
	if len(rawSecrets) != 2 {
		t.Fatalf("expected 2 raw secrets collected for masking, got %d", len(rawSecrets))
	}
}

func TestProcessEnvForSandbox_InjectSecrets(t *testing.T) {
	mockStripe := "sk_live_" + "000000000000000000000000"
	exampleVars := []schema.EnvVar{
		{Key: "STRIPE_SECRET_KEY", Value: "", Tags: map[string]string{"type": "secret"}},
	}
	envValues := map[string]string{
		"STRIPE_SECRET_KEY": mockStripe,
	}

	injected, rawSecrets := sandbox.ProcessEnvForSandbox(exampleVars, envValues, nil, true)

	if injected["STRIPE_SECRET_KEY"] != mockStripe {
		t.Errorf("expected real secret when injectSecrets is true, got %s", injected["STRIPE_SECRET_KEY"])
	}

	if len(rawSecrets) != 1 || rawSecrets[0] != mockStripe {
		t.Errorf("expected raw secret tracked for masking, got %+v", rawSecrets)
	}
}

func TestProcessEnvForSandbox_OnlyFilter(t *testing.T) {
	exampleVars := []schema.EnvVar{
		{Key: "PORT", Value: "3000"},
		{Key: "DB_URL", Value: "postgres://localhost/test"},
		{Key: "REDIS_HOST", Value: "localhost"},
	}
	envValues := map[string]string{
		"PORT":       "8080",
		"DB_URL":     "postgres://user:pass@host/db",
		"REDIS_HOST": "redis.local",
	}

	injected, _ := sandbox.ProcessEnvForSandbox(exampleVars, envValues, []string{"PORT", "REDIS_HOST"}, false)

	if len(injected) != 2 {
		t.Fatalf("expected 2 injected variables with --only, got %d", len(injected))
	}
	if injected["PORT"] != "8080" || injected["REDIS_HOST"] != "redis.local" {
		t.Errorf("unexpected injected vars: %+v", injected)
	}
	if _, exists := injected["DB_URL"]; exists {
		t.Errorf("DB_URL should have been filtered out by --only")
	}
}

