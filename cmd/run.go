package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Hossiy21/razify/internal/reporter"
	"github.com/Hossiy21/razify/internal/sandbox"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [flags] -- <cmd> [args...]",
	Short: "Execute commands inside an agent-safe container sandbox",
	Long: `Run executes build, test, and development commands inside an unprivileged OCI container
with strict containment: synthetic secrets by default (--fake-secrets), isolated network (--network none),
read-only workspace (:ro) with .git protection, exec-capable tmpfs for compilers, and stream-masked output.`,
	DisableFlagParsing: false,
	Run: func(cmd *cobra.Command, args []string) {
		isSandbox, _ := cmd.Flags().GetBool("sandbox")
		if !isSandbox {
			fmt.Fprintln(os.Stderr, "Error: razify run requires the --sandbox flag for safe container execution")
			os.Exit(reporter.ExitError)
		}

		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Error: no command specified to run. Usage: razify run --sandbox -- <cmd> [args...]")
			os.Exit(reporter.ExitError)
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		fakeSecrets, _ := cmd.Flags().GetBool("fake-secrets")
		injectSecrets, _ := cmd.Flags().GetBool("inject-secrets")
		onlyStr, _ := cmd.Flags().GetString("only")
		envFile, _ := cmd.Flags().GetString("env-file")
		exampleFile, _ := cmd.Flags().GetString("example-file")
		image, _ := cmd.Flags().GetString("image")
		profile, _ := cmd.Flags().GetString("profile")
		network, _ := cmd.Flags().GetString("network")
		readWrite, _ := cmd.Flags().GetBool("read-write")
		writableRootfs, _ := cmd.Flags().GetBool("writable-rootfs")
		noMaskEnv, _ := cmd.Flags().GetBool("no-mask-env")
		memory, _ := cmd.Flags().GetString("memory")
		cpus, _ := cmd.Flags().GetString("cpus")
		pidsLimit, _ := cmd.Flags().GetInt("pids-limit")
		initFlag, _ := cmd.Flags().GetBool("init")
		rmFlag, _ := cmd.Flags().GetBool("rm")
		userFlag, _ := cmd.Flags().GetString("user")
		outputDir, _ := cmd.Flags().GetString("output-dir")
		tmpfsMounts, _ := cmd.Flags().GetStringSlice("tmpfs")
		cacheVolume, _ := cmd.Flags().GetString("cache-volume")
		noMaskOutput, _ := cmd.Flags().GetBool("no-mask-output")

		// Guardrail tracking flags
		privileged, _ := cmd.Flags().GetBool("privileged")
		pidMode, _ := cmd.Flags().GetString("pid")
		seccomp, _ := cmd.Flags().GetString("seccomp")
		capAdd, _ := cmd.Flags().GetStringSlice("cap-add")
		device, _ := cmd.Flags().GetStringSlice("device")
		ipcMode, _ := cmd.Flags().GetString("ipc")
		usernsMode, _ := cmd.Flags().GetString("userns")
		securityOpt, _ := cmd.Flags().GetStringSlice("security-opt")
		apparmor, _ := cmd.Flags().GetString("apparmor")

		var onlyKeys []string
		if onlyStr != "" {
			for _, k := range strings.Split(onlyStr, ",") {
				trimmed := strings.TrimSpace(k)
				if trimmed != "" {
					onlyKeys = append(onlyKeys, trimmed)
				}
			}
		}

		cfg := sandbox.Config{
			Command:           args,
			WorkspaceDir:      ".",
			ReadOnlyWorkspace: !readWrite,
			ReadOnlyRootfs:    !writableRootfs,
			MaskEnvFiles:      !noMaskEnv,
			OutputDir:         outputDir,
			TmpfsMounts:       tmpfsMounts,
			CacheVolume:       cacheVolume,
			Image:             image,
			ProjectProfile:    profile,
			Network:           network,
			DropCaps:          true,
			User:              userFlag,
			Memory:            memory,
			CPUs:              cpus,
			PidsLimit:         pidsLimit,
			Init:              initFlag,
			AutoRemove:        rmFlag,
			Privileged:        privileged,
			PidMode:           pidMode,
			Seccomp:           seccomp,
			CapAdd:            capAdd,
			Device:            device,
			IpcMode:           ipcMode,
			UsernsMode:        usernsMode,
			SecurityOpt:       securityOpt,
			AppArmor:          apparmor,
			FakeSecrets:       fakeSecrets && !injectSecrets,
			InjectSecrets:     injectSecrets,
			OnlyKeys:          onlyKeys,
			EnvFile:           envFile,
			ExampleFile:       exampleFile,
			NoMaskOutput:      noMaskOutput,
			DryRun:            dryRun,
		}

		ctx := context.Background()
		result, err := sandbox.Execute(ctx, cfg, os.Stdout, os.Stderr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(reporter.ExitError)
		}

		if dryRun {
			fmt.Printf("docker %s\n", strings.Join(result.Args, " "))
			os.Exit(0)
		}

		if result.ExitCode != 0 {
			os.Exit(result.ExitCode)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().Bool("sandbox", false, "Execute the command inside an agent-safe container sandbox")
	runCmd.Flags().Bool("dry-run", false, "Print the generated docker argv without executing")
	runCmd.Flags().Bool("fake-secrets", true, "Generate synthetic schema-conforming secrets (default true)")
	runCmd.Flags().Bool("inject-secrets", false, "Inject real secrets from .env file (warns loudly)")
	runCmd.Flags().String("only", "", "Comma-separated list of environment keys to inject")
	runCmd.Flags().String("env-file", "", "Path to .env file (auto-detected if omitted)")
	runCmd.Flags().String("example-file", "", "Path to .env.example file (auto-detected if omitted)")
	runCmd.Flags().String("image", "", "Container image to use (defaults by project profile, supports @sha256:...)")
	runCmd.Flags().String("profile", "", "Project profile: go, node, python, generic (auto-detected if omitted)")
	runCmd.Flags().String("network", "none", "Container network mode: none (default), bridge")
	runCmd.Flags().Bool("read-write", false, "Mount workspace read-write instead of read-only (.git stays read-only)")
	runCmd.Flags().Bool("writable-rootfs", false, "Allow container root filesystem to be writable")
	runCmd.Flags().Bool("no-mask-env", false, "Do not bind-mount empty files over workspace .env and .env.*")
	runCmd.Flags().String("memory", "2g", "Memory limit for the container")
	runCmd.Flags().String("cpus", "2.0", "CPU limit for the container")
	runCmd.Flags().Int("pids-limit", 256, "Maximum number of processes inside the container")
	runCmd.Flags().Bool("init", true, "Use tini/docker-init as PID 1 to reap zombies and forward signals")
	runCmd.Flags().Bool("rm", true, "Automatically remove container after execution")
	runCmd.Flags().String("user", "", "Container user mapping (defaults to current UID:GID on Unix)")
	runCmd.Flags().String("output-dir", "", "Optional host directory mounted writable at /output")
	runCmd.Flags().StringSlice("tmpfs", nil, "Custom tmpfs mount specifications")
	runCmd.Flags().String("cache-volume", "", "Named volume for persistent toolchain cache")
	runCmd.Flags().Bool("no-mask-output", false, "Disable stream-masking of secrets on stdout/stderr")

	// Guardrail flags to catch and prohibit unsafe configurations
	runCmd.Flags().Bool("privileged", false, "Prohibited: privileged mode bypasses container isolation")
	runCmd.Flags().String("pid", "", "PID namespace mode (host is prohibited)")
	runCmd.Flags().String("seccomp", "", "Seccomp profile (unconfined is prohibited)")
	runCmd.Flags().StringSlice("cap-add", nil, "Prohibited capabilities (e.g. SYS_ADMIN)")
	runCmd.Flags().StringSlice("device", nil, "Prohibited: host device passthrough")
	runCmd.Flags().String("ipc", "", "IPC mode (host is prohibited)")
	runCmd.Flags().String("userns", "", "User namespace mode (host is prohibited)")
	runCmd.Flags().StringSlice("security-opt", nil, "Security options (disabling seccomp/apparmor/label is prohibited)")
	runCmd.Flags().String("apparmor", "", "AppArmor profile (unconfined is prohibited)")
}

