package sandbox_test

import (
	"testing"

	"github.com/Hossiy21/razify/internal/sandbox"
)

func TestFilterEnv(t *testing.T) {
	allEnv := map[string]string{
		"PORT":          "3000",
		"SECRET_TOKEN":  "dont-expose-me",
		"NODE_ENV":      "production",
		"UNTRACKED_KEY": "val",
	}

	approvedKeys := []string{"PORT", "NODE_ENV"}

	filtered := sandbox.FilterEnv(allEnv, approvedKeys)

	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered keys, got %d", len(filtered))
	}
	if filtered["PORT"] != "3000" || filtered["NODE_ENV"] != "production" {
		t.Errorf("expected approved keys in filtered env, got %+v", filtered)
	}
	if _, exists := filtered["SECRET_TOKEN"]; exists {
		t.Errorf("SECRET_TOKEN should not be present in filtered env")
	}
}

func TestMaskSecretsInEnv(t *testing.T) {
	mockSecret := "sk_live_" + "000000000000000000000000"
	env := map[string]string{
		"PORT":       "3000",
		"STRIPE_KEY": mockSecret,
	}

	masked := sandbox.MaskSecretsInEnv(env, []string{"STRIPE_KEY"})

	if masked["PORT"] != "3000" {
		t.Errorf("PORT should remain untouched: %s", masked["PORT"])
	}
	if masked["STRIPE_KEY"] == mockSecret {
		t.Errorf("STRIPE_KEY was not masked: %s", masked["STRIPE_KEY"])
	}
}
