package doctor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Hossiy21/razify/internal/reporter"
)

type mockFS struct {
	files map[string][]byte
}

func (m *mockFS) FileExists(path string) bool {
	_, ok := m.files[path]
	return ok
}

func (m *mockFS) ReadFile(path string) ([]byte, error) {
	content, ok := m.files[path]
	if !ok {
		return nil, errors.New("file not found")
	}
	return content, nil
}

type mockRunner struct {
	outputs map[string]string
	errs    map[string]error
}

func (m *mockRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmdKey := name + " " + strings.Join(args, " ")
	if err, ok := m.errs[cmdKey]; ok {
		return "", err
	}
	if out, ok := m.outputs[cmdKey]; ok {
		return out, nil
	}
	return "", fmt.Errorf("command not mocked: %s", cmdKey)
}

type mockPortChecker struct {
	inUse map[int]bool
}

func (m *mockPortChecker) CheckPort(port int) (bool, error) {
	return m.inUse[port], nil
}

type mockProber struct {
	failures map[string]bool
	sleep    time.Duration
}

func (m *mockProber) Probe(network, address string, timeout time.Duration) error {
	if m.sleep > 0 {
		time.Sleep(m.sleep)
	}
	if m.failures[address] {
		return errors.New("connection refused")
	}
	return nil
}

type mockGitChecker struct {
	ignored bool
	tracked bool
	err     error
}

func (m *mockGitChecker) CheckIgnore(ctx context.Context, path string) (bool, error) {
	return m.ignored, m.err
}

func (m *mockGitChecker) IsTracked(ctx context.Context, path string) (bool, error) {
	return m.tracked, m.err
}

func defaultHealthyDeps() (Dependencies, *mockFS, *mockRunner, *mockPortChecker, *mockProber, *mockGitChecker) {
	fs := &mockFS{
		files: map[string][]byte{
			".gitignore":   []byte("# Git ignore\n.env\nnode_modules\n"),
			".env":         []byte("PORT=3000\nAPP_ENV=production\nDB_HOST=127.0.0.1\nDB_PORT=5432\n"),
			".env.example": []byte("# @type port\nPORT=3000\n# @type string\nAPP_ENV=production\nDB_HOST=127.0.0.1\n# @type port\nDB_PORT=5432\n"),
			"go.mod":       []byte("module test\ngo 1.25.0\n"),
		},
	}

	runner := &mockRunner{
		outputs: map[string]string{
			"go version": "go version go1.25.1 linux/amd64",
		},
		errs: map[string]error{},
	}

	portChecker := &mockPortChecker{inUse: map[int]bool{}}
	prober := &mockProber{failures: map[string]bool{}}
	git := &mockGitChecker{ignored: true, tracked: false}

	deps := Dependencies{
		FS:          fs,
		Runner:      runner,
		PortChecker: portChecker,
		Prober:      prober,
		Git:         git,
	}

	return deps, fs, runner, portChecker, prober, git
}

func TestDoctor_HealthyEnvironment(t *testing.T) {
	deps, _, _, _, _, _ := defaultHealthyDeps()

	report := RunDoctor(DoctorOptions{}, deps)

	if report.HealthScore != 100 {
		t.Fatalf("expected health score 100, got %d", report.HealthScore)
	}
	if report.HealthStatus != "HEALTHY" {
		t.Errorf("expected HEALTHY, got %s", report.HealthStatus)
	}
	if report.FailedCount != 0 {
		t.Errorf("expected 0 failures, got %d", report.FailedCount)
	}
	if report.Result.Status != "PASSED" {
		t.Errorf("expected PASSED result, got %s", report.Result.Status)
	}
}

func TestDoctor_ToolchainRedistribution(t *testing.T) {
	// 1. Only Go project: Go should receive all 15 points
	deps, fs, runner, _, _, _ := defaultHealthyDeps()
	report := RunDoctor(DoctorOptions{}, deps)

	for _, c := range report.Checks {
		if c.ID == "razify/doctor/toolchain-go" {
			if c.MaxPoints != 15 {
				t.Errorf("expected Go to receive all 15 toolchain points, got %d", c.MaxPoints)
			}
			if c.Status != StatusPassed {
				t.Errorf("expected Go to pass, got %s", c.Status)
			}
		}
		if c.ID == "razify/doctor/toolchain-node" || c.ID == "razify/doctor/toolchain-docker" {
			if c.Status != StatusSkipped {
				t.Errorf("expected %s to be SKIPPED, got %s", c.ID, c.Status)
			}
			if c.MaxPoints != 0 {
				t.Errorf("expected skipped toolchain to have 0 max points, got %d", c.MaxPoints)
			}
		}
	}

	// 2. Multi-toolchain project (Node + Docker): 15 points redistributed
	fs.files["package.json"] = []byte(`{"name":"test"}`)
	fs.files["Dockerfile"] = []byte("FROM alpine\n")
	delete(fs.files, "go.mod")

	runner.outputs["node --version"] = "v20.10.0"
	runner.outputs["docker --version"] = "Docker version 27.1.1"
	runner.outputs["docker info --format {{.ServerVersion}}"] = "27.1.1"

	reportMulti := RunDoctor(DoctorOptions{}, deps)
	totalToolchainPoints := 0
	for _, c := range reportMulti.Checks {
		if strings.HasPrefix(c.ID, "razify/doctor/toolchain-") && c.Status != StatusSkipped {
			totalToolchainPoints += c.MaxPoints
		}
	}
	if totalToolchainPoints != WeightToolchainsTotal {
		t.Errorf("expected toolchain total points to equal 15, got %d", totalToolchainPoints)
	}
}

func TestDoctor_GitCheck(t *testing.T) {
	// Case 1: .env tracked in git index despite gitignore -> FAIL
	deps, _, _, _, _, git := defaultHealthyDeps()
	git.ignored = true
	git.tracked = true

	reportTracked := RunDoctor(DoctorOptions{}, deps)
	if reportTracked.Result.Status != "FAILED" {
		t.Errorf("expected FAILED when .env is tracked in git index")
	}

	foundTracked := false
	for _, c := range reportTracked.Checks {
		if c.ID == "razify/doctor/gitignore-missing" {
			foundTracked = true
			if c.Status != StatusFailed {
				t.Errorf("expected StatusFailed for tracked .env, got %s", c.Status)
			}
			if !strings.Contains(c.Fix, "git rm --cached") {
				t.Errorf("expected fix with git rm --cached, got %s", c.Fix)
			}
		}
	}
	if !foundTracked {
		t.Fatal("gitignore check not evaluated")
	}

	// Case 2: .env not ignored by git -> FAIL
	git.ignored = false
	git.tracked = false
	reportUnignored := RunDoctor(DoctorOptions{}, deps)
	for _, c := range reportUnignored.Checks {
		if c.ID == "razify/doctor/gitignore-missing" {
			if c.Status != StatusFailed {
				t.Errorf("expected StatusFailed for unignored .env, got %s", c.Status)
			}
			if !strings.Contains(c.Fix, ">> .gitignore") {
				t.Errorf("expected fix appending to .gitignore, got %s", c.Fix)
			}
		}
	}
}

func TestDoctor_PortCheckWarning(t *testing.T) {
	deps, _, _, pc, _, _ := defaultHealthyDeps()
	pc.inUse[3000] = true // port 3000 is occupied (e.g. by user dev server)

	report := RunDoctor(DoctorOptions{}, deps)

	for _, c := range report.Checks {
		if c.ID == "razify/doctor/port-conflict" {
			if c.Status != StatusWarning {
				t.Fatalf("expected port in use to be reported as WARNING, got %s", c.Status)
			}
			if !strings.Contains(c.Message, "active local dev server") {
				t.Errorf("expected message mentioning dev server, got %s", c.Message)
			}
		}
	}

	// A port warning should NOT fail the run if all other checks pass
	if report.Result.Status != "PASSED" {
		t.Errorf("expected PASSED result for dev server port warning, got %s", report.Result.Status)
	}
}

func TestDoctor_MinScoreThreshold(t *testing.T) {
	deps, fs, _, _, _, _ := defaultHealthyDeps()
	// Remove .env file (10 points lost, score = 90)
	delete(fs.files, ".env")

	// 1. MinScore = 95 -> score 90 must FAIL
	reportFail := RunDoctor(DoctorOptions{MinScore: 95}, deps)
	if reportFail.Result.Status != "FAILED" {
		t.Errorf("expected FAILED when score < min-score")
	}

	// 2. MinScore = 80 -> score 90 passes threshold, but missing-env failure marks it FAILED
	// Let's test with a warning-only degradation (e.g. placeholder warning):
	fs.files[".env"] = []byte("PORT=3000\nAPP_ENV=placeholder\n")
	fs.files[".env.example"] = []byte("PORT=3000\nAPP_ENV=placeholder\n")

	reportCustom := RunDoctor(DoctorOptions{MinScore: 80}, deps)
	if reportCustom.HealthScore < 80 {
		t.Errorf("expected score >= 80, got %d", reportCustom.HealthScore)
	}
}

func TestDoctor_LiteralLocalhostDetection(t *testing.T) {
	deps, fs, _, _, prober, _ := defaultHealthyDeps()

	// 1. Literal remote hostname must be skipped without --probe-remote
	fs.files[".env"] = []byte("DATABASE_URL=postgres://user:pass@db.internal.corp:5432/app\n")
	reportRemote := RunDoctor(DoctorOptions{ProbeRemote: false}, deps)

	for _, c := range reportRemote.Checks {
		if c.ID == "razify/doctor/service-unreachable" {
			if c.Status != StatusSkipped {
				t.Errorf("expected remote host without --probe-remote to be SKIPPED, got %s", c.Status)
			}
		}
	}

	// 2. Literal 127.0.0.1 and localhost must be probed as local
	fs.files[".env"] = []byte("DATABASE_URL=postgres://localhost:5432/app\n")
	prober.failures["localhost:5432"] = true

	reportLocal := RunDoctor(DoctorOptions{ProbeRemote: false}, deps)
	for _, c := range reportLocal.Checks {
		if c.ID == "razify/doctor/service-unreachable" {
			if c.Status != StatusFailed {
				t.Errorf("expected local unreachable service to FAIL, got %s", c.Status)
			}
		}
	}
}

func TestDoctor_DockerDaemonDistinction(t *testing.T) {
	deps, fs, runner, _, _, _ := defaultHealthyDeps()
	fs.files["Dockerfile"] = []byte("FROM alpine\n")
	delete(fs.files, "go.mod")

	runner.outputs["docker --version"] = "Docker version 27.1.1"
	// Daemon is down
	runner.errs["docker info --format {{.ServerVersion}}"] = errors.New("cannot connect to docker daemon")

	// In normal mode: daemon down is flagged as WARNING
	reportFull := RunDoctor(DoctorOptions{Fast: false}, deps)
	for _, c := range reportFull.Checks {
		if c.ID == "razify/doctor/toolchain-docker" {
			if c.Status != StatusWarning {
				t.Errorf("expected WARNING when docker installed but daemon down, got %s", c.Status)
			}
			if !strings.Contains(c.Message, "daemon is not running") {
				t.Errorf("expected daemon not running message, got %s", c.Message)
			}
		}
	}

	// In --fast mode: daemon check is skipped
	reportFast := RunDoctor(DoctorOptions{Fast: true}, deps)
	for _, c := range reportFast.Checks {
		if c.ID == "razify/doctor/toolchain-docker" {
			if c.Status != StatusPassed {
				t.Errorf("expected PASSED in fast mode when CLI exists, got %s", c.Status)
			}
			if !strings.Contains(c.Message, "daemon check skipped") {
				t.Errorf("expected message mentioning skipped daemon check, got %s", c.Message)
			}
		}
	}
}

func TestDoctor_ConcurrentServiceProbing(t *testing.T) {
	deps, fs, _, _, prober, _ := defaultHealthyDeps()
	// 3 unreachable database services
	fs.files[".env"] = []byte(`
DATABASE_URL=postgres://localhost:5432/app
REDIS_URL=redis://127.0.0.1:6379/0
MONGO_URI=mongodb://127.0.0.1:27017/db
`)
	prober.failures["localhost:5432"] = true
	prober.failures["127.0.0.1:6379"] = true
	prober.failures["127.0.0.1:27017"] = true
	// Simulate 100ms probe latency on each
	prober.sleep = 100 * time.Millisecond

	start := time.Now()
	report := RunDoctor(DoctorOptions{}, deps)
	duration := time.Since(start)

	if report.Result.Status != "FAILED" {
		t.Fatalf("expected FAILED for unreachable services")
	}

	// If run sequentially, 3 * 100ms = 300ms.
	// Running concurrently, all 3 complete in ~100-150ms.
	if duration > 250*time.Millisecond {
		t.Errorf("expected concurrent probes to complete in ~100ms, took %v", duration)
	}
}

func TestDoctor_SecretsNeverPrinted(t *testing.T) {
	deps, fs, _, _, _, _ := defaultHealthyDeps()
	secretVal := "super_secret_db_password_xyz987"
	fs.files[".env"] = []byte(fmt.Sprintf("DATABASE_URL=postgres://appuser:%s@localhost:5432/app\n", secretVal))

	report := RunDoctor(DoctorOptions{}, deps)

	for _, c := range report.Checks {
		if strings.Contains(c.Message, secretVal) {
			t.Fatalf("LEAK in check message: %s", c.Message)
		}
		if strings.Contains(c.Fix, secretVal) {
			t.Fatalf("LEAK in check fix: %s", c.Fix)
		}
	}

	sarifText, err := reporter.RenderSarif(report.Result, "test-version")
	if err != nil {
		t.Fatalf("sarif render error: %v", err)
	}
	if strings.Contains(sarifText, secretVal) {
		t.Fatalf("LEAK in SARIF output: %s", sarifText)
	}
}
