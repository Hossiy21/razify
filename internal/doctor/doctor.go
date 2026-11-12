package doctor

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Hossiy21/razify/internal/reporter"
	"github.com/Hossiy21/razify/internal/scanner"
	"github.com/Hossiy21/razify/internal/schema"
)

// DoctorOptions specifies operational parameters for the health check.
type DoctorOptions struct {
	EnvFile       string
	ExampleFile   string
	GitignoreFile string
	Fast          bool          // Skip port and service network probes, skip docker daemon probe
	ProbeRemote   bool          // Probe remote databases/services (default: literal localhost only)
	Timeout       time.Duration // Probe connection timeout (default: 300ms)
	MinScore      int           // Minimum health score required to pass (default: 90)
}

// Dependencies contains the pluggable runners and probers for hermetic testing.
type Dependencies struct {
	FS          FileSystem
	Runner      ToolchainRunner
	PortChecker PortChecker
	Prober      ServiceProber
	Git         GitChecker
}

// NewDefaultDependencies constructs production OS, git, and network drivers.
func NewDefaultDependencies() Dependencies {
	return Dependencies{
		FS:          &OSFileSystem{},
		Runner:      &OSToolchainRunner{},
		PortChecker: &NetPortChecker{},
		Prober:      &NetServiceProber{},
		Git:         &OSGitChecker{},
	}
}

// RunDoctor performs the full suite of health, toolchain, integrity, and availability checks.
func RunDoctor(opts DoctorOptions, deps Dependencies) DoctorReport {
	if opts.EnvFile == "" {
		opts.EnvFile = ".env"
	}
	if opts.ExampleFile == "" {
		opts.ExampleFile = ".env.example"
	}
	if opts.GitignoreFile == "" {
		opts.GitignoreFile = ".gitignore"
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 300 * time.Millisecond
	}
	if opts.MinScore <= 0 {
		opts.MinScore = 90
	}

	var checks []CheckResult

	// 1. Environment & Git Integrity Checks
	gitCheck := checkGit(opts.GitignoreFile, opts.EnvFile, deps.FS, deps.Git)
	checks = append(checks, gitCheck)

	envFileCheck, envBytes := checkEnvFile(opts.EnvFile, deps.FS)
	checks = append(checks, envFileCheck)

	exampleCheck, exampleBytes := checkExampleFile(opts.ExampleFile, deps.FS)
	checks = append(checks, exampleCheck)

	// 2. Schema & Secret Validation Checks
	secretCheck := checkSecrets(opts.EnvFile, envBytes)
	checks = append(checks, secretCheck)

	reqCheck, typesCheck := checkSchema(opts.EnvFile, envBytes, exampleBytes)
	checks = append(checks, reqCheck, typesCheck)

	// 3. Toolchain Readiness Checks (scored only for detected project types)
	toolchainChecks := checkToolchains(deps.FS, deps.Runner, opts.Fast)
	checks = append(checks, toolchainChecks...)

	// 4. Port Conflict Checks (127.0.0.1 only, Dial first, warning on in-use)
	portCheck := checkPorts(opts.EnvFile, envBytes, exampleBytes, deps.PortChecker, opts.Fast)
	checks = append(checks, portCheck)

	// 5. Database & Service Probes (literal localhost only, concurrent timeouts)
	serviceCheck := checkServices(opts.EnvFile, envBytes, deps.Prober, opts.Fast, opts.ProbeRemote, opts.Timeout)
	checks = append(checks, serviceCheck)

	// Score Calculation & Aggregation
	totalEarned := 0
	totalMax := 0
	passedCount := 0
	failedCount := 0
	warningCount := 0
	skippedCount := 0
	var remediations []string
	var diagnostics []reporter.Diagnostic

	for _, c := range checks {
		switch c.Status {
		case StatusPassed:
			passedCount++
			totalEarned += c.PointsEarned
			totalMax += c.MaxPoints
		case StatusFailed:
			failedCount++
			totalEarned += c.PointsEarned
			totalMax += c.MaxPoints
			if c.Fix != "" {
				remediations = append(remediations, c.Fix)
			}
			diagnostics = append(diagnostics, reporter.Diagnostic{
				RuleID:   c.ID,
				Message:  c.Message,
				Severity: reporter.SeverityError,
				File:     c.Target,
				Line:     1,
			})
		case StatusWarning:
			warningCount++
			totalEarned += c.PointsEarned
			totalMax += c.MaxPoints
			if c.Fix != "" {
				remediations = append(remediations, c.Fix)
			}
			diagnostics = append(diagnostics, reporter.Diagnostic{
				RuleID:   c.ID,
				Message:  c.Message,
				Severity: reporter.SeverityWarning,
				File:     c.Target,
				Line:     1,
			})
		case StatusSkipped:
			skippedCount++
		}
	}

	healthScore := 100
	if totalMax > 0 {
		healthScore = int(math.Round(float64(totalEarned) / float64(totalMax) * 100.0))
		if healthScore < 0 {
			healthScore = 0
		} else if healthScore > 100 {
			healthScore = 100
		}
	}

	healthStatus := "HEALTHY"
	if healthScore < opts.MinScore-20 || failedCount >= 2 {
		healthStatus = "UNHEALTHY"
	} else if healthScore < opts.MinScore || failedCount > 0 {
		healthStatus = "DEGRADED"
	}

	unifiedStatus := "PASSED"
	if healthScore < opts.MinScore || failedCount > 0 {
		unifiedStatus = "FAILED"
	}

	unifiedResult := reporter.Result{
		TargetFile:       opts.EnvFile,
		TemplateFile:     opts.ExampleFile,
		Status:           unifiedStatus,
		Diagnostics:      diagnostics,
		RemediationSteps: remediations,
	}

	return DoctorReport{
		HealthScore:  healthScore,
		HealthStatus: healthStatus,
		Checks:       checks,
		Remediations: remediations,
		PassedCount:  passedCount,
		FailedCount:  failedCount,
		WarningCount: warningCount,
		SkippedCount: skippedCount,
		Result:       unifiedResult,
	}
}

func checkGit(gitignorePath, envFile string, fs FileSystem, git GitChecker) CheckResult {
	res := CheckResult{
		ID:        "razify/doctor/gitignore-missing",
		Category:  "git",
		Name:      "Gitignore Leak Guard",
		MaxPoints: WeightGitignoreLeakGuard,
		Target:    gitignorePath,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// 1. Try git commands if GitChecker is available
	if git != nil {
		tracked, errTrack := git.IsTracked(ctx, envFile)
		if errTrack == nil && tracked {
			res.Status = StatusFailed
			res.PointsEarned = 0
			res.Message = fmt.Sprintf("%s is tracked in the git index! Secrets will be committed on git push.", envFile)
			res.Fix = fmt.Sprintf("git rm --cached %s", envFile)
			return res
		}

		ignored, errIgnore := git.CheckIgnore(ctx, envFile)
		if errIgnore == nil {
			if ignored {
				res.Status = StatusPassed
				res.PointsEarned = WeightGitignoreLeakGuard
				res.Message = fmt.Sprintf("%s is properly shielded and untracked in git", envFile)
				return res
			}
			res.Status = StatusFailed
			res.PointsEarned = 0
			res.Message = fmt.Sprintf("%s is not ignored by git; risk of secret leaks", envFile)
			res.Fix = fmt.Sprintf("echo %q >> %s", envFile, gitignorePath)
			return res
		}
	}

	// 2. Fallback to reading .gitignore file directly
	if !fs.FileExists(gitignorePath) {
		res.Status = StatusFailed
		res.PointsEarned = 0
		res.Message = fmt.Sprintf("%s not found; environment secrets risk being committed to git", gitignorePath)
		res.Fix = fmt.Sprintf("echo %q >> %s", envFile, gitignorePath)
		return res
	}

	content, err := fs.ReadFile(gitignorePath)
	if err != nil {
		res.Status = StatusFailed
		res.PointsEarned = 0
		res.Message = fmt.Sprintf("Unable to read %s: %v", gitignorePath, err)
		res.Fix = fmt.Sprintf("chmod 644 %s", gitignorePath)
		return res
	}

	lines := strings.Split(string(content), "\n")
	envPatternFound := false
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}
		if trimmed == envFile || trimmed == ".env" || trimmed == "*.env" || trimmed == ".env*" || trimmed == "/.env" || trimmed == "**/.env" {
			envPatternFound = true
			break
		}
	}

	if envPatternFound {
		res.Status = StatusPassed
		res.PointsEarned = WeightGitignoreLeakGuard
		res.Message = fmt.Sprintf("%s shields %s from git commits", gitignorePath, envFile)
	} else {
		res.Status = StatusFailed
		res.PointsEarned = 0
		res.Message = fmt.Sprintf("%s is not ignored in %s, risking credential leaks", envFile, gitignorePath)
		res.Fix = fmt.Sprintf("echo %q >> %s", envFile, gitignorePath)
	}

	return res
}

func checkEnvFile(envPath string, fs FileSystem) (CheckResult, []byte) {
	res := CheckResult{
		ID:        "razify/doctor/missing-env",
		Category:  "env",
		Name:      "Environment File Present",
		MaxPoints: WeightEnvFilePresent,
		Target:    envPath,
	}

	if !fs.FileExists(envPath) {
		res.Status = StatusFailed
		res.PointsEarned = 0
		res.Message = fmt.Sprintf("Environment file %s does not exist", envPath)
		res.Fix = fmt.Sprintf("cp .env.example %s", envPath)
		return res, nil
	}

	bytes, err := fs.ReadFile(envPath)
	if err != nil {
		res.Status = StatusFailed
		res.PointsEarned = 0
		res.Message = fmt.Sprintf("Failed to read %s: %v", envPath, err)
		res.Fix = fmt.Sprintf("chmod 600 %s", envPath)
		return res, nil
	}

	res.Status = StatusPassed
	res.PointsEarned = WeightEnvFilePresent
	res.Message = fmt.Sprintf("Environment file %s found (%d bytes)", envPath, len(bytes))
	return res, bytes
}

func checkExampleFile(examplePath string, fs FileSystem) (CheckResult, []byte) {
	res := CheckResult{
		ID:        "razify/doctor/missing-example",
		Category:  "env",
		Name:      "Template Schema Present",
		MaxPoints: WeightExampleFilePresent,
		Target:    examplePath,
	}

	if !fs.FileExists(examplePath) {
		res.Status = StatusWarning
		res.PointsEarned = 0
		res.Message = fmt.Sprintf("Template schema %s missing; validation and drift detection disabled", examplePath)
		res.Fix = "razify init"
		return res, nil
	}

	bytes, err := fs.ReadFile(examplePath)
	if err != nil {
		res.Status = StatusWarning
		res.PointsEarned = 0
		res.Message = fmt.Sprintf("Failed to read %s: %v", examplePath, err)
		res.Fix = fmt.Sprintf("chmod 644 %s", examplePath)
		return res, nil
	}

	res.Status = StatusPassed
	res.PointsEarned = WeightExampleFilePresent
	res.Message = fmt.Sprintf("Template schema %s found", examplePath)
	return res, bytes
}

func checkSecrets(envPath string, envBytes []byte) CheckResult {
	res := CheckResult{
		ID:        "razify/doctor/secret-leak",
		Category:  "schema",
		Name:      "Secret Leak Scan",
		MaxPoints: WeightSecretLeaks,
		Target:    envPath,
	}

	if len(envBytes) == 0 {
		res.Status = StatusPassed
		res.PointsEarned = WeightSecretLeaks
		res.Message = "Zero secret leaks detected (file empty or absent)"
		return res
	}

	envVars, err := schema.ParseEnvWithMetadataReader(strings.NewReader(string(envBytes)))
	if err != nil {
		res.Status = StatusWarning
		res.PointsEarned = WeightSecretLeaks / 2
		res.Message = fmt.Sprintf("Failed to parse environment file for secret scanning: %v", err)
		return res
	}

	scanResults, scanSummary := scanner.ScanVars(envVars)

	if scanSummary.Critical > 0 || scanSummary.High > 0 {
		res.Status = StatusFailed
		res.PointsEarned = 0
		res.Message = fmt.Sprintf("Detected %d high-confidence secret leak(s) in %s", scanSummary.Critical+scanSummary.High, envPath)
		res.Fix = fmt.Sprintf("razify scan --fix %s", envPath)
		return res
	}

	if scanSummary.Medium > 0 {
		res.Status = StatusWarning
		res.PointsEarned = WeightSecretLeaks / 2
		res.Message = fmt.Sprintf("Detected %d suspicious credential pattern(s) in %s", len(scanResults), envPath)
		res.Fix = fmt.Sprintf("razify scan --entropy %s", envPath)
		return res
	}

	res.Status = StatusPassed
	res.PointsEarned = WeightSecretLeaks
	res.Message = fmt.Sprintf("Zero secret leaks detected in %s", envPath)
	return res
}

func checkSchema(envPath string, envBytes, exampleBytes []byte) (CheckResult, CheckResult) {
	reqRes := CheckResult{
		ID:        "razify/doctor/missing-required-vars",
		Category:  "schema",
		Name:      "Required Variables",
		MaxPoints: WeightSchemaRequiredVars,
		Target:    envPath,
	}

	typeRes := CheckResult{
		ID:        "razify/doctor/invalid-schema-types",
		Category:  "schema",
		Name:      "Schema Constraints",
		MaxPoints: WeightSchemaTypes,
		Target:    envPath,
	}

	if len(exampleBytes) == 0 {
		reqRes.Status = StatusSkipped
		reqRes.Message = "Schema checks skipped (no .env.example found)"
		typeRes.Status = StatusSkipped
		typeRes.Message = "Type checks skipped (no .env.example found)"
		return reqRes, typeRes
	}

	exampleVars, err := schema.ParseEnvWithMetadataReader(strings.NewReader(string(exampleBytes)))
	if err != nil {
		reqRes.Status = StatusWarning
		reqRes.PointsEarned = 0
		reqRes.Message = fmt.Sprintf("Failed to parse schema template: %v", err)
		reqRes.Fix = "razify check"

		typeRes.Status = StatusSkipped
		typeRes.Message = "Type checks skipped due to parse error"
		return reqRes, typeRes
	}

	envMap, err := schema.ParseEnvReader(strings.NewReader(string(envBytes)))
	if err != nil {
		reqRes.Status = StatusWarning
		reqRes.PointsEarned = 0
		reqRes.Message = fmt.Sprintf("Failed to parse environment file: %v", err)
		reqRes.Fix = "razify check"

		typeRes.Status = StatusSkipped
		typeRes.Message = "Type checks skipped due to parse error"
		return reqRes, typeRes
	}

	summary, results := schema.Validate(envMap, exampleVars)

	invalidCount := 0
	totalRequired := 0
	for _, ev := range exampleVars {
		if ev.Required {
			totalRequired++
		}
	}
	for _, vr := range results {
		if vr.Status == "INVALID" {
			invalidCount++
		}
	}

	if summary.Missing > 0 {
		reqRes.Status = StatusFailed
		reqRes.PointsEarned = 0
		reqRes.Message = fmt.Sprintf("%d required environment variable(s) missing from %s", summary.Missing, envPath)
		reqRes.Fix = fmt.Sprintf("razify fix %s", envPath)
	} else {
		reqRes.Status = StatusPassed
		reqRes.PointsEarned = WeightSchemaRequiredVars
		reqRes.Message = fmt.Sprintf("All required schema variables (%d defined) are present", totalRequired)
	}

	if invalidCount > 0 {
		typeRes.Status = StatusFailed
		typeRes.PointsEarned = 0
		typeRes.Message = fmt.Sprintf("%d variable(s) violate schema type constraints in %s", invalidCount, envPath)
		typeRes.Fix = fmt.Sprintf("razify check %s", envPath)
	} else {
		typeRes.Status = StatusPassed
		typeRes.PointsEarned = WeightSchemaTypes
		typeRes.Message = "All variables conform to schema type constraints"
	}

	return reqRes, typeRes
}

func checkToolchains(fs FileSystem, runner ToolchainRunner, fast bool) []CheckResult {
	// 1. Detect project types based on presence of manifests
	hasGo := fs.FileExists("go.mod")
	hasNode := fs.FileExists("package.json")
	hasPython := fs.FileExists("requirements.txt") || fs.FileExists("pyproject.toml") || fs.FileExists("Pipfile")
	hasDocker := fs.FileExists("Dockerfile") || fs.FileExists("docker-compose.yml") || fs.FileExists("docker-compose.yaml") || fs.FileExists("compose.yaml") || fs.FileExists("compose.yml")

	type toolPlan struct {
		name      string
		relevant  bool
		runProbe  func(ctx context.Context, points int) CheckResult
		skipMsg   string
	}

	plans := []toolPlan{
		{
			name:     "Go",
			relevant: hasGo,
			skipMsg:  "Skipped: not a Go project (no go.mod found)",
			runProbe: func(ctx context.Context, points int) CheckResult {
				return probeGo(ctx, runner, points)
			},
		},
		{
			name:     "Node.js",
			relevant: hasNode,
			skipMsg:  "Skipped: not a Node.js project (no package.json found)",
			runProbe: func(ctx context.Context, points int) CheckResult {
				return probeNode(ctx, runner, points)
			},
		},
		{
			name:     "Python",
			relevant: hasPython,
			skipMsg:  "Skipped: not a Python project (no requirements.txt/pyproject.toml found)",
			runProbe: func(ctx context.Context, points int) CheckResult {
				return probePython(ctx, runner, points)
			},
		},
		{
			name:     "Docker",
			relevant: hasDocker,
			skipMsg:  "Skipped: not a containerized project (no Dockerfile/compose.yaml found)",
			runProbe: func(ctx context.Context, points int) CheckResult {
				return probeDocker(ctx, runner, fast, points)
			},
		},
	}

	// Count relevant toolchains to redistribute WeightToolchainsTotal (15 pts)
	relevantCount := 0
	for _, p := range plans {
		if p.relevant {
			relevantCount++
		}
	}

	// If no project manifests detected at all, default to evaluating Go as baseline (or fallback equally)
	if relevantCount == 0 {
		plans[0].relevant = true
		relevantCount = 1
	}

	basePoints := WeightToolchainsTotal / relevantCount
	remainder := WeightToolchainsTotal % relevantCount

	var results []CheckResult
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := range plans {
		p := &plans[i]
		if !p.relevant {
			results = append(results, CheckResult{
				ID:        fmt.Sprintf("razify/doctor/toolchain-%s", strings.ToLower(p.name)),
				Category:  "toolchain",
				Name:      fmt.Sprintf("%s Toolchain", p.name),
				Status:    StatusSkipped,
				Message:   p.skipMsg,
				MaxPoints: 0,
			})
			continue
		}

		pts := basePoints
		if remainder > 0 {
			pts++
			remainder--
		}

		wg.Add(1)
		go func(plan toolPlan, points int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
			defer cancel()

			res := plan.runProbe(ctx, points)
			mu.Lock()
			results = append(results, res)
			mu.Unlock()
		}(*p, pts)
	}

	wg.Wait()
	return results
}

func probeGo(ctx context.Context, runner ToolchainRunner, points int) CheckResult {
	res := CheckResult{
		ID:        "razify/doctor/toolchain-go",
		Category:  "toolchain",
		Name:      "Go Toolchain",
		MaxPoints: points,
	}

	out, err := runner.Run(ctx, "go", "version")
	if err != nil {
		res.Status = StatusFailed
		res.PointsEarned = 0
		res.Message = "Go compiler (go) not found in PATH"
		res.Fix = "Install Go 1.25+ from https://go.dev/dl/"
		return res
	}

	re := regexp.MustCompile(`go([0-9]+)\.([0-9]+)`)
	matches := re.FindStringSubmatch(out)
	if len(matches) < 3 {
		res.Status = StatusWarning
		res.PointsEarned = points / 2
		res.Message = fmt.Sprintf("Go installed but unable to parse version: %s", strings.TrimSpace(out))
		res.Fix = "Upgrade Go to 1.25+ from https://go.dev/dl/"
		return res
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])

	if major > 1 || (major == 1 && minor >= 25) {
		res.Status = StatusPassed
		res.PointsEarned = points
		res.Message = fmt.Sprintf("Go %d.%d detected (satisfies >= 1.25)", major, minor)
	} else {
		res.Status = StatusFailed
		res.PointsEarned = 0
		res.Message = fmt.Sprintf("Go %d.%d detected (outdated: Razify requires Go 1.25+)", major, minor)
		res.Fix = "Upgrade Go to 1.25+ from https://go.dev/dl/"
	}

	return res
}

func probeNode(ctx context.Context, runner ToolchainRunner, points int) CheckResult {
	res := CheckResult{
		ID:        "razify/doctor/toolchain-node",
		Category:  "toolchain",
		Name:      "Node.js Runtime",
		MaxPoints: points,
	}

	out, err := runner.Run(ctx, "node", "--version")
	if err != nil {
		res.Status = StatusWarning
		res.PointsEarned = 0
		res.Message = "Node.js (node) not found in PATH"
		res.Fix = "Install Node.js 18+ from https://nodejs.org/"
		return res
	}

	version := strings.TrimSpace(out)
	res.Status = StatusPassed
	res.PointsEarned = points
	res.Message = fmt.Sprintf("Node.js runtime %s detected", version)
	return res
}

func probePython(ctx context.Context, runner ToolchainRunner, points int) CheckResult {
	res := CheckResult{
		ID:        "razify/doctor/toolchain-python",
		Category:  "toolchain",
		Name:      "Python Runtime",
		MaxPoints: points,
	}

	out, err := runner.Run(ctx, "python3", "--version")
	if err != nil {
		out, err = runner.Run(ctx, "python", "--version")
	}
	if err != nil {
		res.Status = StatusWarning
		res.PointsEarned = 0
		res.Message = "Python interpreter not found in PATH"
		res.Fix = "Install Python 3 from https://www.python.org/"
		return res
	}

	version := strings.TrimSpace(out)
	res.Status = StatusPassed
	res.PointsEarned = points
	res.Message = fmt.Sprintf("Python %s detected", version)
	return res
}

func probeDocker(ctx context.Context, runner ToolchainRunner, fast bool, points int) CheckResult {
	res := CheckResult{
		ID:        "razify/doctor/toolchain-docker",
		Category:  "toolchain",
		Name:      "Docker Engine",
		MaxPoints: points,
	}

	// 1. Check if Docker CLI binary is installed
	out, err := runner.Run(ctx, "docker", "--version")
	if err != nil {
		res.Status = StatusWarning
		res.PointsEarned = 0
		res.Message = "Docker CLI not found in PATH"
		res.Fix = "Install Docker from https://docs.docker.com/get-docker/"
		return res
	}
	cliVersion := strings.TrimSpace(out)

	// 2. In --fast mode, skip the daemon check
	if fast {
		res.Status = StatusPassed
		res.PointsEarned = points
		res.Message = fmt.Sprintf("%s detected (daemon check skipped in --fast mode)", cliVersion)
		return res
	}

	// 3. Probe if Docker daemon is actively responding
	daemonOut, daemonErr := runner.Run(ctx, "docker", "info", "--format", "{{.ServerVersion}}")
	if daemonErr != nil || strings.TrimSpace(daemonOut) == "" {
		res.Status = StatusWarning
		res.PointsEarned = points / 2
		res.Message = fmt.Sprintf("%s installed, but Docker daemon is not running", cliVersion)
		res.Fix = "Start Docker Desktop or system Docker service"
		return res
	}

	res.Status = StatusPassed
	res.PointsEarned = points
	res.Message = fmt.Sprintf("Docker CLI %s & daemon %s active", cliVersion, strings.TrimSpace(daemonOut))
	return res
}

func extractDeclaredPorts(envBytes, exampleBytes []byte) map[int]string {
	ports := make(map[int]string)

	parseContent := func(b []byte) {
		lines := strings.Split(string(b), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"' `)

			isPortKey := key == "PORT" ||
				strings.HasSuffix(key, "_PORT") ||
				strings.HasPrefix(key, "PORT_") ||
				strings.Contains(key, "PORT")

			if isPortKey {
				if p, err := strconv.Atoi(val); err == nil && p >= 1 && p <= 65535 {
					if _, exists := ports[p]; !exists {
						ports[p] = key
					}
				}
			}
		}
	}

	parseContent(envBytes)
	parseContent(exampleBytes)
	return ports
}

func checkPorts(envPath string, envBytes, exampleBytes []byte, checker PortChecker, fast bool) CheckResult {
	res := CheckResult{
		ID:        "razify/doctor/port-conflict",
		Category:  "network",
		Name:      "Port Conflict Detection",
		MaxPoints: WeightPortConflicts,
		Target:    envPath,
	}

	if fast {
		res.Status = StatusSkipped
		res.Message = "Port checks skipped (--fast mode)"
		return res
	}

	ports := extractDeclaredPorts(envBytes, exampleBytes)
	if len(ports) == 0 {
		res.Status = StatusPassed
		res.PointsEarned = WeightPortConflicts
		res.Message = "No application ports declared in environment"
		return res
	}

	var inUsePorts []string
	var conflictNumbers []int
	for port, key := range ports {
		inUse, _ := checker.CheckPort(port)
		if inUse {
			inUsePorts = append(inUsePorts, fmt.Sprintf("%d (%s)", port, key))
			conflictNumbers = append(conflictNumbers, port)
		}
	}

	// Notice: Per requirements, port in use is treated as a WARNING (may be the user's dev server), not failure.
	if len(inUsePorts) > 0 {
		res.Status = StatusWarning
		res.PointsEarned = WeightPortConflicts
		res.Message = fmt.Sprintf("Port(s) currently in use: %s (may be an active local dev server)", strings.Join(inUsePorts, ", "))
		res.Fix = fmt.Sprintf("lsof -ti :%d | xargs kill -9 (if occupied by an unexpected process)", conflictNumbers[0])
		return res
	}

	var declaredList []string
	for port, key := range ports {
		declaredList = append(declaredList, fmt.Sprintf("%d (%s)", port, key))
	}

	res.Status = StatusPassed
	res.PointsEarned = WeightPortConflicts
	res.Message = fmt.Sprintf("All declared port(s) free: %s", strings.Join(declaredList, ", "))
	return res
}

type serviceEndpoint struct {
	Key     string
	Host    string
	Port    string
	Scheme  string
	Masked  string
	IsLocal bool
}

func isLiteralLocalhost(host string) bool {
	h := strings.TrimSpace(host)
	h = strings.TrimPrefix(h, "[")
	h = strings.TrimSuffix(h, "]")
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

func extractDeclaredServices(envBytes []byte) []serviceEndpoint {
	var endpoints []serviceEndpoint
	lines := strings.Split(string(envBytes), "\n")

	envMap := make(map[string]string)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		v = strings.Trim(v, `"' `)
		envMap[k] = v
	}

	// 1. Look for URI connection strings
	for k, v := range envMap {
		if !strings.Contains(v, "://") {
			continue
		}
		u, err := url.Parse(v)
		if err != nil || u.Host == "" {
			continue
		}

		host := u.Hostname()
		port := u.Port()
		if port == "" {
			switch u.Scheme {
			case "postgres", "postgresql":
				port = "5432"
			case "redis", "rediss":
				port = "6379"
			case "mysql":
				port = "3306"
			case "mongodb", "mongodb+srv":
				port = "27017"
			case "http":
				port = "80"
			case "https":
				port = "443"
			default:
				continue
			}
		}

		isLocal := isLiteralLocalhost(host)
		masked := fmt.Sprintf("%s://%s:%s", u.Scheme, host, port)

		endpoints = append(endpoints, serviceEndpoint{
			Key:     k,
			Host:    host,
			Port:    port,
			Scheme:  u.Scheme,
			Masked:  masked,
			IsLocal: isLocal,
		})
	}

	// 2. Look for DB_HOST + DB_PORT or REDIS_HOST + REDIS_PORT pairs
	prefixes := []string{"DB", "DATABASE", "POSTGRES", "REDIS", "MYSQL", "MONGO"}
	for _, p := range prefixes {
		hostKey := p + "_HOST"
		portKey := p + "_PORT"
		host := envMap[hostKey]
		port := envMap[portKey]
		if host != "" && port != "" {
			isLocal := isLiteralLocalhost(host)
			endpoints = append(endpoints, serviceEndpoint{
				Key:     hostKey + "/" + portKey,
				Host:    host,
				Port:    port,
				Scheme:  strings.ToLower(p),
				Masked:  fmt.Sprintf("%s:%s", host, port),
				IsLocal: isLocal,
			})
		}
	}

	return endpoints
}

func checkServices(envPath string, envBytes []byte, prober ServiceProber, fast, probeRemote bool, timeout time.Duration) CheckResult {
	res := CheckResult{
		ID:        "razify/doctor/service-unreachable",
		Category:  "service",
		Name:      "Service Availability Probe",
		MaxPoints: WeightServiceReachability,
		Target:    envPath,
	}

	if fast {
		res.Status = StatusSkipped
		res.Message = "Service probes skipped (--fast mode)"
		return res
	}

	endpoints := extractDeclaredServices(envBytes)
	if len(endpoints) == 0 {
		res.Status = StatusPassed
		res.PointsEarned = WeightServiceReachability
		res.Message = "No database or backend services declared in environment"
		return res
	}

	var unreachable []string
	var skippedRemote []string
	var testedCount int
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, ep := range endpoints {
		if !ep.IsLocal && !probeRemote {
			skippedRemote = append(skippedRemote, ep.Masked)
			continue
		}

		testedCount++
		wg.Add(1)
		go func(target serviceEndpoint) {
			defer wg.Done()
			addr := net.JoinHostPort(target.Host, target.Port)
			err := prober.Probe("tcp", addr, timeout)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				unreachable = append(unreachable, fmt.Sprintf("%s (%s)", target.Masked, target.Key))
			}
		}(ep)
	}

	wg.Wait()

	if len(unreachable) > 0 {
		res.Status = StatusFailed
		res.PointsEarned = 0
		res.Message = fmt.Sprintf("Service connection failed: unreachable service(s): %s", strings.Join(unreachable, ", "))
		res.Fix = "docker compose up -d (or verify service is running)"
		return res
	}

	if testedCount == 0 && len(skippedRemote) > 0 {
		res.Status = StatusSkipped
		res.Message = fmt.Sprintf("Remote service(s) skipped (%s); pass --probe-remote to enable TCP probes", strings.Join(skippedRemote, ", "))
		return res
	}

	msg := fmt.Sprintf("All %d reachable service(s) accept TCP connect", testedCount)
	if len(skippedRemote) > 0 {
		msg += fmt.Sprintf(" (%d remote service(s) skipped, use --probe-remote)", len(skippedRemote))
	}

	res.Status = StatusPassed
	res.PointsEarned = WeightServiceReachability
	res.Message = msg
	return res
}
