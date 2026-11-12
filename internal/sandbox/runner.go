package sandbox

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/Hossiy21/razify/internal/schema"
)

// CheckDockerAvailable verifies that docker CLI exists in PATH and the Docker daemon is accessible.
func CheckDockerAvailable(ctx context.Context) error {
	dockerPath, err := exec.LookPath("docker")
	if err != nil || dockerPath == "" {
		return fmt.Errorf("docker executable not found in PATH: please install Docker to use razify run --sandbox")
	}

	checkCtx, cancel := context.WithTimeout(ctx, 2500*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(checkCtx, "docker", "info", "--format", "{{.ServerVersion}}")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker daemon is not running or accessible: please start Docker to use razify run --sandbox")
	}

	return nil
}

// FindEnvFilesToMask discovers .env and .env.* files across the workspace recursively,
// excluding templates/samples (.env.example, .env.sample, .env.template, etc.)
// and ignoring common dependency directories (.git, node_modules, vendor, .venv).
// Returned paths are relative to workspaceDir using forward slashes.
func FindEnvFilesToMask(workspaceDir string) ([]string, error) {
	if workspaceDir == "" {
		workspaceDir = "."
	}

	var envFiles []string
	ignoredDirs := map[string]bool{
		".git":         true,
		"node_modules": true,
		"vendor":       true,
		".venv":        true,
	}

	err := filepath.WalkDir(workspaceDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if ignoredDirs[d.Name()] && path != workspaceDir {
				return filepath.SkipDir
			}
			return nil
		}

		name := d.Name()
		lower := strings.ToLower(name)

		// Exclude template, sample, and example files
		if strings.Contains(lower, ".example") ||
			strings.Contains(lower, ".sample") ||
			strings.Contains(lower, ".template") {
			return nil
		}

		// Match .env, .env.*, and *.env files
		if name == ".env" || strings.HasPrefix(name, ".env.") || strings.HasSuffix(name, ".env") {
			rel, err := filepath.Rel(workspaceDir, path)
			if err != nil {
				return nil
			}
			cleanRel := filepath.ToSlash(filepath.Clean(rel))
			cleanRel = strings.TrimPrefix(cleanRel, "./")
			envFiles = append(envFiles, cleanRel)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk workspace directory: %w", err)
	}

	sort.Strings(envFiles)
	return envFiles, nil
}

// ResolveUser determines the container UID:GID mapping.
// Defaults to "1000:1000" as a fixed non-root user on all platforms unless explicitly overridden.
func ResolveUser(configuredUser string) string {
	if configuredUser != "" {
		return configuredUser
	}
	return "1000:1000"
}

// RunResult holds the output and exit status of a sandbox execution.
type RunResult struct {
	ExitCode int
	Args     []string
	Warnings []string
}

// Execute runs the configured command in the docker sandbox.
func Execute(
	ctx context.Context,
	cfg Config,
	stdout io.Writer,
	stderr io.Writer,
) (RunResult, error) {
	result := RunResult{
		ExitCode: 0,
	}

	// 1. Guardrail validation
	if err := ValidateGuardrails(cfg); err != nil {
		result.ExitCode = 2
		return result, err
	}

	result.Warnings = CheckWarnings(cfg)
	for _, w := range result.Warnings {
		fmt.Fprintln(stderr, w)
	}

	// 2. Discover .env files to mask
	var envFilesToMask []string
	if cfg.MaskEnvFiles {
		discovered, err := FindEnvFilesToMask(cfg.WorkspaceDir)
		if err == nil {
			envFilesToMask = discovered
		}
	}

	// 3. Create empty temp file for bind-masking .env files
	var emptyTempPath string
	if cfg.MaskEnvFiles && len(envFilesToMask) > 0 {
		emptyFile, err := os.CreateTemp("", "razify_mask_*.empty")
		if err != nil {
			result.ExitCode = 2
			return result, fmt.Errorf("failed to create temporary mask file: %w", err)
		}
		emptyTempPath = emptyFile.Name()
		_ = emptyFile.Close()
		defer os.Remove(emptyTempPath)
	}

	// 4. Parse schema / example file and .env file for secrets
	var exampleVars []schema.EnvVar
	if cfg.ExampleFile != "" && fileExists(cfg.ExampleFile) {
		vars, err := schema.ParseEnvWithMetadata(cfg.ExampleFile)
		if err == nil {
			exampleVars = vars
		}
	} else {
		// Auto-discover example file in workspace
		autoExample := filepath.Join(cfg.WorkspaceDir, ".env.example")
		if fileExists(autoExample) {
			vars, err := schema.ParseEnvWithMetadata(autoExample)
			if err == nil {
				exampleVars = vars
			}
		}
	}

	envValues := make(map[string]string)
	if cfg.EnvFile != "" && fileExists(cfg.EnvFile) {
		vals, err := schema.ParseEnvFile(cfg.EnvFile)
		if err == nil {
			envValues = vals
		}
	} else {
		autoEnv := filepath.Join(cfg.WorkspaceDir, ".env")
		if fileExists(autoEnv) {
			vals, err := schema.ParseEnvFile(autoEnv)
			if err == nil {
				envValues = vals
			}
		}
	}

	// 5. Process and classify environment variables
	injectedEnv, rawSecretsToMask := ProcessEnvForSandbox(
		exampleVars,
		envValues,
		cfg.OnlyKeys,
		cfg.InjectSecrets,
	)

	// 6. Project profile and defaults
	profileName := cfg.ProjectProfile
	if profileName == "" {
		profileName = DetectProfile(cfg.WorkspaceDir)
	}
	profile := GetProfileConfig(profileName)

	if cfg.User == "" {
		cfg.User = ResolveUser("")
	}

	// 7. Build docker arguments
	args, processEnv, err := BuildDockerArgs(
		cfg,
		envFilesToMask,
		emptyTempPath,
		injectedEnv,
		profile,
	)
	if err != nil {
		result.ExitCode = 2
		return result, err
	}
	result.Args = args

	// 8. Handle Dry-Run
	if cfg.DryRun {
		return result, nil
	}

	// 9. Verify Docker availability
	if err := CheckDockerAvailable(ctx); err != nil {
		result.ExitCode = 2
		return result, err
	}

	// 10. Setup command execution
	cmd := exec.CommandContext(ctx, "docker", args...)

	// Pass environment variables to Docker CLI process environment (for -e KEY resolution)
	envSlice := os.Environ()
	for k, v := range processEnv {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = envSlice

	// Output streaming and masking
	var outWriter io.Writer = stdout
	var errWriter io.Writer = stderr
	var outMasker *StreamMasker
	var errMasker *StreamMasker

	if !cfg.NoMaskOutput && len(rawSecretsToMask) > 0 {
		outMasker = NewStreamMasker(stdout, rawSecretsToMask)
		errMasker = NewStreamMasker(stderr, rawSecretsToMask)
		outWriter = outMasker
		errWriter = errMasker
	}

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter
	cmd.Stdin = os.Stdin

	// Signal forwarding
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	if err := cmd.Start(); err != nil {
		result.ExitCode = 2
		return result, fmt.Errorf("failed to start container: %w", err)
	}

	go func() {
		select {
		case sig, ok := <-sigChan:
			if ok && cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		case <-ctx.Done():
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		}
	}()

	runErr := cmd.Wait()

	if outMasker != nil {
		_ = outMasker.Close()
	}
	if errMasker != nil {
		_ = errMasker.Close()
	}

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		result.ExitCode = 2
		return result, runErr
	}

	result.ExitCode = 0
	return result, nil
}
