package doctor

// Documented health-score weights totaling 100 points.
//
// Weight Distribution:
// 1. Environment & Git Integrity (35 points):
//    - GitignoreLeakGuard: 15 pts (.env shielded in .gitignore and NOT tracked in git index)
//    - EnvFilePresent:     10 pts (.env file must exist)
//    - ExampleFilePresent: 10 pts (.env.example schema template must exist)
//
// 2. Schema & Secret Integrity (35 points):
//    - SecretLeaks:        15 pts (Zero high/critical secret leaks detected)
//    - SchemaRequiredVars: 12 pts (All required template variables defined)
//    - SchemaTypes:         8 pts (Variables satisfy @type, @enum, @range constraints)
//
// 3. Toolchain Readiness (15 points):
//    - Toolchains are evaluated ONLY if relevant to the project type:
//        * Go:         go.mod present
//        * Node.js:    package.json present
//        * Python:     requirements.txt, pyproject.toml, or Pipfile present
//        * Docker:     Dockerfile or docker-compose.yml present
//      Irrelevant toolchain points are redistributed evenly among relevant toolchains
//      so that the Toolchains category always totals 15 points.
//
// 4. Ports & Services (15 points):
//    - PortConflicts:       8 pts (Declared ports free; in-use ports flagged as warning)
//    - ServiceReachability: 7 pts (Declared DB/services accept TCP connect on localhost)
const (
	WeightGitignoreLeakGuard = 15
	WeightEnvFilePresent     = 10
	WeightExampleFilePresent = 10

	WeightSecretLeaks        = 15
	WeightSchemaRequiredVars = 12
	WeightSchemaTypes        = 8

	WeightToolchainsTotal     = 15
	WeightPortConflicts       = 8
	WeightServiceReachability = 7

	TotalMaxHealthScore = 100
)
