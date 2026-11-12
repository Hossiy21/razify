package doctor

// Documented health-score weights totaling 100 points.
//
// Weight Distribution:
// 1. Environment & Git Integrity (35 points):
//    - GitignoreLeakGuard: 15 pts (.env must be shielded in .gitignore)
//    - EnvFilePresent:     10 pts (.env file must exist)
//    - ExampleFilePresent: 10 pts (.env.example schema template must exist)
//
// 2. Schema & Secret Integrity (35 points):
//    - SecretLeaks:        15 pts (Zero high/critical secret leaks detected)
//    - SchemaRequiredVars: 12 pts (All required template variables defined)
//    - SchemaTypes:         8 pts (Variables satisfy @type, @enum, @range constraints)
//
// 3. Toolchain Readiness (15 points):
//    - ToolchainGo:         5 pts (Go 1.25+ detected in PATH)
//    - ToolchainNode:       5 pts (Node.js runtime detected in PATH)
//    - ToolchainDocker:     5 pts (Docker CLI installed and daemon responding)
//
// 4. Ports & Services (15 points):
//    - PortConflicts:       8 pts (All declared TCP ports are unallocated and free to bind)
//    - ServiceReachability: 7 pts (All declared DB/services accept TCP connect on localhost)
const (
	WeightGitignoreLeakGuard = 15
	WeightEnvFilePresent     = 10
	WeightExampleFilePresent = 10

	WeightSecretLeaks        = 15
	WeightSchemaRequiredVars = 12
	WeightSchemaTypes        = 8

	WeightToolchainGo     = 5
	WeightToolchainNode   = 5
	WeightToolchainDocker = 5

	WeightPortConflicts       = 8
	WeightServiceReachability = 7

	TotalMaxHealthScore = 100
)

