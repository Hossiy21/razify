package sandbox

import (
	"fmt"
	"strings"
)

// ValidateGuardrails validates the configuration against security guardrails.
// Returns an error if any prohibited configuration is detected.
func ValidateGuardrails(cfg Config) error {
	if cfg.Privileged {
		return fmt.Errorf("guardrail violation: --privileged is prohibited as it bypasses container isolation")
	}

	if strings.ToLower(cfg.PidMode) == "host" {
		return fmt.Errorf("guardrail violation: --pid=host is prohibited as it exposes host processes")
	}

	if strings.ToLower(cfg.Network) == "host" {
		return fmt.Errorf("guardrail violation: --network host is prohibited as it bypasses network isolation")
	}

	if strings.Contains(strings.ToLower(cfg.Seccomp), "unconfined") {
		return fmt.Errorf("guardrail violation: disabling seccomp is prohibited")
	}

	for _, cap := range cfg.CapAdd {
		upper := strings.ToUpper(strings.TrimSpace(cap))
		if upper == "SYS_ADMIN" || upper == "CAP_SYS_ADMIN" || upper == "ALL" {
			return fmt.Errorf("guardrail violation: --cap-add %s is prohibited", cap)
		}
	}

	if len(cfg.Device) > 0 {
		return fmt.Errorf("guardrail violation: --device is prohibited as it exposes host devices")
	}

	if strings.ToLower(strings.TrimSpace(cfg.IpcMode)) == "host" {
		return fmt.Errorf("guardrail violation: --ipc=host is prohibited as it exposes host shared memory")
	}

	if strings.ToLower(strings.TrimSpace(cfg.UsernsMode)) == "host" {
		return fmt.Errorf("guardrail violation: --userns=host is prohibited as it disables user namespace isolation")
	}

	if strings.Contains(strings.ToLower(cfg.AppArmor), "unconfined") {
		return fmt.Errorf("guardrail violation: apparmor=unconfined is prohibited")
	}

	for _, opt := range cfg.SecurityOpt {
		lowerOpt := strings.ToLower(opt)
		if strings.Contains(lowerOpt, "seccomp=unconfined") {
			return fmt.Errorf("guardrail violation: seccomp=unconfined is prohibited")
		}
		if strings.Contains(lowerOpt, "apparmor=unconfined") {
			return fmt.Errorf("guardrail violation: apparmor=unconfined is prohibited")
		}
		if strings.Contains(lowerOpt, "label=disable") || strings.Contains(lowerOpt, "label:disable") {
			return fmt.Errorf("guardrail violation: label disable is prohibited")
		}
	}

	// Check custom tmpfs or volume mounts for Docker socket
	checkSocketPath := func(path string) error {
		lower := strings.ToLower(path)
		if strings.Contains(lower, "docker.sock") || strings.Contains(lower, "docker_engine") {
			return fmt.Errorf("guardrail violation: mounting the Docker daemon socket is prohibited")
		}
		return nil
	}

	if err := checkSocketPath(cfg.OutputDir); err != nil {
		return err
	}
	if err := checkSocketPath(cfg.CacheVolume); err != nil {
		return err
	}
	for _, m := range cfg.TmpfsMounts {
		if err := checkSocketPath(m); err != nil {
			return err
		}
	}

	return nil
}

// CheckWarnings inspects configuration and returns warnings for risky but allowable settings.
func CheckWarnings(cfg Config) []string {
	var warnings []string

	if cfg.Network != "" && cfg.Network != "none" {
		warnings = append(warnings, fmt.Sprintf("[WARNING] Network egress enabled (--network %s). Secret exfiltration mitigations are disabled.", cfg.Network))
	}

	if cfg.InjectSecrets {
		warnings = append(warnings, "[WARNING] Real secrets injected (--inject-secrets). Plaintext credentials exist in container memory.")
	}

	if !cfg.MaskEnvFiles {
		warnings = append(warnings, "[WARNING] .env file masking disabled (--no-mask-env). Raw secret files are readable within workspace.")
	}

	if !cfg.ReadOnlyWorkspace {
		warnings = append(warnings, "[WARNING] Workspace mounted read-write (--read-write). Containerized processes can modify files executed later on host.")
	}

	return warnings
}

