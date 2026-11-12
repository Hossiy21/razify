package sandbox

// Config specifies runtime constraints and parameters for sandboxed command execution.
type Config struct {
	// Command and Arguments
	Command []string `json:"command"`

	// Workspace and Filesystem
	WorkspaceDir      string   `json:"workspace_dir"`
	ReadOnlyWorkspace bool     `json:"read_only_workspace"`
	ReadOnlyRootfs    bool     `json:"read_only_rootfs"`
	MaskEnvFiles      bool     `json:"mask_env_files"`
	OutputDir         string   `json:"output_dir"`
	TmpfsMounts       []string `json:"tmpfs_mounts"`
	CacheVolume       string   `json:"cache_volume"`

	// Image and Profile
	Image          string `json:"image"`
	ProjectProfile string `json:"project_profile"` // "go", "node", "python", "generic"

	// Security and Network Constraints
	Network   string `json:"network"` // "none" (default), "bridge"
	DropCaps  bool   `json:"drop_caps"`
	User      string `json:"user"`
	Memory    string `json:"memory"`     // default "2g"
	CPUs      string `json:"cpus"`       // default "2.0"
	PidsLimit int    `json:"pids_limit"` // default 256
	Init      bool   `json:"init"`       // default true
	AutoRemove bool  `json:"auto_remove"` // default true

	// Guardrail Overrides (flag tracking for validation)
	Privileged bool   `json:"privileged"`
	PidMode    string `json:"pid_mode"`
	Seccomp    string `json:"seccomp"`

	// Secret Handling
	FakeSecrets   bool     `json:"fake_secrets"`
	InjectSecrets bool     `json:"inject_secrets"`
	OnlyKeys      []string `json:"only_keys"`
	EnvFile       string   `json:"env_file"`
	ExampleFile   string   `json:"example_file"`
	NoMaskOutput  bool     `json:"no_mask_output"`

	// Execution Mode
	DryRun bool `json:"dry_run"`
}

// DefaultConfig returns the default configuration with all containment settings enabled.
func DefaultConfig() Config {
	return Config{
		ReadOnlyWorkspace: true,
		ReadOnlyRootfs:    true,
		MaskEnvFiles:      true,
		Network:           "none",
		DropCaps:          true,
		Memory:            "2g",
		CPUs:              "2.0",
		PidsLimit:         256,
		Init:              true,
		AutoRemove:        true,
		FakeSecrets:       true,
		InjectSecrets:     false,
		NoMaskOutput:      false,
		DryRun:            false,
	}
}

