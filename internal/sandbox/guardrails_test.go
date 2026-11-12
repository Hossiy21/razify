package sandbox_test

import (
	"strings"
	"testing"

	"github.com/Hossiy21/razify/internal/sandbox"
)

func TestValidateGuardrails_Rejections(t *testing.T) {
	testCases := []struct {
		name        string
		cfg         sandbox.Config
		wantErrText string
	}{
		{
			name: "reject privileged mode",
			cfg: sandbox.Config{
				Privileged: true,
			},
			wantErrText: "--privileged is prohibited",
		},
		{
			name: "reject host pid mode",
			cfg: sandbox.Config{
				PidMode: "host",
			},
			wantErrText: "--pid=host is prohibited",
		},
		{
			name: "reject host network",
			cfg: sandbox.Config{
				Network: "host",
			},
			wantErrText: "--network host is prohibited",
		},
		{
			name: "reject unconfined seccomp",
			cfg: sandbox.Config{
				Seccomp: "unconfined",
			},
			wantErrText: "disabling seccomp is prohibited",
		},
		{
			name: "reject mounting docker socket via output dir",
			cfg: sandbox.Config{
				OutputDir: "/var/run/docker.sock",
			},
			wantErrText: "mounting the Docker daemon socket is prohibited",
		},
		{
			name: "reject mounting windows docker engine pipe",
			cfg: sandbox.Config{
				OutputDir: `\\.\pipe\docker_engine`,
			},
			wantErrText: "mounting the Docker daemon socket is prohibited",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := sandbox.ValidateGuardrails(tc.cfg)
			if err == nil {
				t.Fatalf("expected guardrail rejection, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErrText) {
				t.Errorf("expected error containing %q, got %q", tc.wantErrText, err.Error())
			}
		})
	}
}

func TestCheckWarnings(t *testing.T) {
	cfg := sandbox.Config{
		Network:           "bridge",
		InjectSecrets:     true,
		MaskEnvFiles:      false,
		ReadOnlyWorkspace: false,
	}

	warnings := sandbox.CheckWarnings(cfg)
	if len(warnings) != 4 {
		t.Fatalf("expected 4 warnings, got %d: %+v", len(warnings), warnings)
	}

	hasWarning := func(substr string) bool {
		for _, w := range warnings {
			if strings.Contains(w, substr) {
				return true
			}
		}
		return false
	}

	if !hasWarning("Network egress enabled") {
		t.Errorf("missing network warning")
	}
	if !hasWarning("Real secrets injected") {
		t.Errorf("missing inject secrets warning")
	}
	if !hasWarning(".env file masking disabled") {
		t.Errorf("missing no-mask-env warning")
	}
	if !hasWarning("Workspace mounted read-write") {
		t.Errorf("missing read-write warning")
	}
}

