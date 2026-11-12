package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

// BuildDockerArgs constructs the docker run arguments and environment map according to the sandbox configuration.
func BuildDockerArgs(
	cfg Config,
	envFilesToMask []string,
	emptyMaskFilePath string,
	injectedEnv map[string]string,
	profile ProfileConfig,
) ([]string, map[string]string, error) {
	var args []string
	envMap := make(map[string]string)

	args = append(args, "run")

	if cfg.AutoRemove {
		args = append(args, "--rm")
	}

	if cfg.Init {
		args = append(args, "--init")
	}

	if cfg.DropCaps {
		args = append(args, "--cap-drop=ALL")
	}

	args = append(args, "--security-opt", "no-new-privileges")

	// Network configuration
	network := cfg.Network
	if network == "" {
		network = "none"
	}
	args = append(args, "--network", network)

	// Resource limits
	if cfg.Memory != "" {
		args = append(args, "--memory", cfg.Memory)
	}
	if cfg.CPUs != "" {
		args = append(args, "--cpus", cfg.CPUs)
	}
	if cfg.PidsLimit > 0 {
		args = append(args, "--pids-limit", strconv.Itoa(cfg.PidsLimit))
	}

	// Read-only container rootfs
	if cfg.ReadOnlyRootfs {
		args = append(args, "--read-only")
	}

	// Tmpfs mounts (Profile defaults + custom overrides)
	seenTmpfs := make(map[string]bool)
	tmpfsList := profile.DefaultTmpfs
	if len(cfg.TmpfsMounts) > 0 {
		tmpfsList = append(tmpfsList, cfg.TmpfsMounts...)
	}

	for _, t := range tmpfsList {
		if !seenTmpfs[t] {
			seenTmpfs[t] = true
			args = append(args, "--tmpfs", t)
		}
	}

	// Workspace mount
	workspaceMode := "ro"
	if !cfg.ReadOnlyWorkspace {
		workspaceMode = "rw"
	}
	absWorkspace, err := filepath.Abs(cfg.WorkspaceDir)
	if err != nil {
		absWorkspace = cfg.WorkspaceDir
	}
	args = append(args, "-v", FormatMountArg(absWorkspace, "/workspace", workspaceMode))

	// Protect .git (always read-only even under --read-write)
	gitDir := filepath.Join(absWorkspace, ".git")
	if stat, err := os.Stat(gitDir); err == nil && stat.IsDir() {
		args = append(args, "-v", FormatMountArg(gitDir, "/workspace/.git", "ro"))
	}

	// Mask .env and .env.* files with empty temp file
	if cfg.MaskEnvFiles && emptyMaskFilePath != "" {
		sort.Strings(envFilesToMask)
		for _, f := range envFilesToMask {
			containerTarget := "/workspace/" + filepath.ToSlash(f)
			args = append(args, "-v", FormatMountArg(emptyMaskFilePath, containerTarget, "ro"))
		}
	}

	// Output dir mount
	if cfg.OutputDir != "" {
		absOutput, err := filepath.Abs(cfg.OutputDir)
		if err != nil {
			absOutput = cfg.OutputDir
		}
		args = append(args, "-v", FormatMountArg(absOutput, "/output", "rw"))
	}

	// Named cache volume mount
	if cfg.CacheVolume != "" {
		args = append(args, "-v", fmt.Sprintf("%s:/cache:rw", cfg.CacheVolume))
	}

	// Working directory
	args = append(args, "-w", "/workspace")

	// User mapping (Unix UID:GID)
	if cfg.User != "" {
		args = append(args, "--user", cfg.User)
	}

	// Environment variables: profile defaults
	var profileEnvKeys []string
	for k := range profile.EnvVars {
		profileEnvKeys = append(profileEnvKeys, k)
	}
	sort.Strings(profileEnvKeys)
	for _, k := range profileEnvKeys {
		args = append(args, "-e", k)
		envMap[k] = profile.EnvVars[k]
	}

	// Environment variables: injected/fake env variables (Never passed inline in argv)
	var injectedKeys []string
	for k := range injectedEnv {
		injectedKeys = append(injectedKeys, k)
	}
	sort.Strings(injectedKeys)
	for _, k := range injectedKeys {
		args = append(args, "-e", k)
		envMap[k] = injectedEnv[k]
	}

	// Image specification
	image := cfg.Image
	if image == "" {
		image = profile.DefaultImage
	}
	args = append(args, image)

	// Command to execute
	args = append(args, cfg.Command...)

	return args, envMap, nil
}

