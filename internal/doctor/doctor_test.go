package doctor

import (
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

func (m *mockRunner) Run(name string, args ...string) (string, error) {
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
	occupied map[int]bool
}

func (m *mockPortChecker) CheckPort(port int) error {
	if m.occupied[port] {
		return errors.New("address already in use")
	}
	return nil
}

type mockProber struct {
	failures map[string]bool
}

func (m *mockProber) Probe(network, address string, timeout time.Duration) error {
	if m.failures[address] {
		return errors.New("connection refused")
	}
	return nil
}

func defaultHealthyDeps() (Dependencies, *mockFS, *mockRunner, *mockPortChecker, *mockProber) {
	fs := &mockFS{
		files: map[string][]byte{
			".gitignore":   []byte("# Git ignore\n.env\nnode_modules\n"),
			".env":         []byte("PORT=3000\nAPP_ENV=production\nDB_HOST=localhost\nDB_PORT=5432\n"),
			".env.example": []byte("# @type port\nPORT=3000\n# @type string\nAPP_ENV=production\nDB_HOST=localhost\n# @type port\nDB_PORT=5432\n"),
		},
	}

	runner := &mockRunner{
		outputs: map[string]string{
			"go version":        "go version go1.25.1 linux/amd64",
			"node -v":           "v20.10.0",
			"docker --version":  "Docker version 27.1.1, build 6312585",
		},
		errs: map[string]error{},
	}

	portChecker := &mockPortChecker{occupied: map[int]bool{}}
	prober := &mockProber{failures: map[string]bool{}}

	deps := Dependencies{
		FS:          fs,
		Runner:      runner,
		PortChecker: portChecker,
		Prober:      prober,
	}

	return deps, fs, runner, portChecker, prober
}

func TestDoctor_HealthyEnvironment(t *testing.T) {
	deps, _, _, _, _ := defaultHealthyDeps()

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
	if len(report.Remediations) != 0 {
		t.Errorf("expected 0 remediations, got %d", len(report.Remediations))
	}
}

func TestDoctor_TableDrivenFailures(t *testing.T) {
	tests := []struct {
		name              string
		mutateDeps        func(fs *mockFS, runner *mockRunner, pc *mockPortChecker, p *mockProber)
		expectedStatus    string
		expectedScoreMax  int
		expectedFailedID  string
		expectedFixSubstr string
	}{
		{
			name: "missing .gitignore",
			mutateDeps: func(fs *mockFS, runner *mockRunner, pc *mockPortChecker, p *mockProber) {
				delete(fs.files, ".gitignore")
			},
			expectedStatus:    "DEGRADED",
			expectedScoreMax:  85,
			expectedFailedID:  "razify/doctor/gitignore-missing",
			expectedFixSubstr: ".gitignore",
		},
		{
			name: ".env not protected in .gitignore",
			mutateDeps: func(fs *mockFS, runner *mockRunner, pc *mockPortChecker, p *mockProber) {
				fs.files[".gitignore"] = []byte("node_modules/\ndist/\n")
			},
			expectedStatus:    "DEGRADED",
			expectedScoreMax:  85,
			expectedFailedID:  "razify/doctor/gitignore-missing",
			expectedFixSubstr: `echo ".env" >> .gitignore`,
		},
		{
			name: "missing .env file",
			mutateDeps: func(fs *mockFS, runner *mockRunner, pc *mockPortChecker, p *mockProber) {
				delete(fs.files, ".env")
			},
			expectedStatus:    "DEGRADED",
			expectedScoreMax:  90,
			expectedFailedID:  "razify/doctor/missing-env",
			expectedFixSubstr: "cp .env.example .env",
		},
		{
			name: "secret leak in .env",
			mutateDeps: func(fs *mockFS, runner *mockRunner, pc *mockPortChecker, p *mockProber) {
				// AWS fake credential fixture
				fs.files[".env"] = []byte("PORT=3000\nAWS_KEY=AKIA1234567890EXAMPLE\nAPP_ENV=production\n")
			},
			expectedStatus:    "DEGRADED",
			expectedScoreMax:  85,
			expectedFailedID:  "razify/doctor/secret-leak",
			expectedFixSubstr: "razify scan --fix",
		},
		{
			name: "outdated Go compiler",
			mutateDeps: func(fs *mockFS, runner *mockRunner, pc *mockPortChecker, p *mockProber) {
				runner.outputs["go version"] = "go version go1.22.4 linux/amd64"
			},
			expectedStatus:    "DEGRADED",
			expectedScoreMax:  95,
			expectedFailedID:  "razify/doctor/toolchain-go",
			expectedFixSubstr: "Upgrade Go to 1.25+",
		},
		{
			name: "Go not found in PATH",
			mutateDeps: func(fs *mockFS, runner *mockRunner, pc *mockPortChecker, p *mockProber) {
				runner.errs["go version"] = errors.New("executable file not found")
			},
			expectedStatus:    "DEGRADED",
			expectedScoreMax:  95,
			expectedFailedID:  "razify/doctor/toolchain-go",
			expectedFixSubstr: "https://go.dev/dl/",
		},
		{
			name: "port conflict detected",
			mutateDeps: func(fs *mockFS, runner *mockRunner, pc *mockPortChecker, p *mockProber) {
				pc.occupied[3000] = true
			},
			expectedStatus:    "DEGRADED",
			expectedScoreMax:  92,
			expectedFailedID:  "razify/doctor/port-conflict",
			expectedFixSubstr: "lsof -ti :3000",
		},
		{
			name: "database service unreachable",
			mutateDeps: func(fs *mockFS, runner *mockRunner, pc *mockPortChecker, p *mockProber) {
				p.failures["localhost:5432"] = true
			},
			expectedStatus:    "DEGRADED",
			expectedScoreMax:  93,
			expectedFailedID:  "razify/doctor/service-unreachable",
			expectedFixSubstr: "docker compose up -d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps, fs, runner, pc, prober := defaultHealthyDeps()
			tt.mutateDeps(fs, runner, pc, prober)

			report := RunDoctor(DoctorOptions{}, deps)

			if report.HealthScore > tt.expectedScoreMax {
				t.Errorf("expected score <= %d, got %d", tt.expectedScoreMax, report.HealthScore)
			}

			foundFailed := false
			for _, c := range report.Checks {
				if c.ID == tt.expectedFailedID {
					foundFailed = true
					if c.Status != StatusFailed {
						t.Errorf("expected check %s to have StatusFailed, got %s", c.ID, c.Status)
					}
					if !strings.Contains(c.Fix, tt.expectedFixSubstr) {
						t.Errorf("expected fix containing %q, got %q", tt.expectedFixSubstr, c.Fix)
					}
				}
			}

			if !foundFailed {
				t.Fatalf("expected check %s to fail, but it did not fail", tt.expectedFailedID)
			}
		})
	}
}

func TestDoctor_FastMode(t *testing.T) {
	deps, _, _, pc, p := defaultHealthyDeps()
	// Set ports occupied and services failing: in normal mode this would fail
	pc.occupied[3000] = true
	p.failures["localhost:5432"] = true

	// In fast mode, port checks and service probes must be skipped!
	report := RunDoctor(DoctorOptions{Fast: true}, deps)

	for _, c := range report.Checks {
		if c.ID == "razify/doctor/port-conflict" || c.ID == "razify/doctor/service-unreachable" {
			if c.Status != StatusSkipped {
				t.Errorf("expected check %s to be SKIPPED in fast mode, got %s", c.ID, c.Status)
			}
		}
	}

	if report.HealthScore != 100 {
		t.Errorf("expected score 100 in fast mode with no un-skipped failures, got %d", report.HealthScore)
	}
	if report.Result.Status != "PASSED" {
		t.Errorf("expected PASSED result in fast mode, got %s", report.Result.Status)
	}
}

func TestDoctor_RemoteServiceSkipping(t *testing.T) {
	deps, fs, _, _, p := defaultHealthyDeps()
	// Add remote database URL
	fs.files[".env"] = []byte("PORT=3000\nDATABASE_URL=postgres://user:secret@db.production.aws.com:5432/app\n")
	p.failures["db.production.aws.com:5432"] = true

	// 1. Without --probe-remote: remote host should be skipped
	reportDefault := RunDoctor(DoctorOptions{ProbeRemote: false}, deps)
	for _, c := range reportDefault.Checks {
		if c.ID == "razify/doctor/service-unreachable" {
			if c.Status != StatusSkipped {
				t.Errorf("expected remote service to be SKIPPED without --probe-remote, got %s", c.Status)
			}
		}
	}

	// 2. With --probe-remote: remote host is probed and fails
	reportRemote := RunDoctor(DoctorOptions{ProbeRemote: true}, deps)
	for _, c := range reportRemote.Checks {
		if c.ID == "razify/doctor/service-unreachable" {
			if c.Status != StatusFailed {
				t.Errorf("expected remote service to FAIL with --probe-remote when unreachable, got %s", c.Status)
			}
		}
	}
}

func TestDoctor_SecretsNeverPrinted(t *testing.T) {
	deps, fs, _, _, _ := defaultHealthyDeps()
	secretVal := "super_secret_db_password_xyz987"
	fs.files[".env"] = []byte(fmt.Sprintf("DATABASE_URL=postgres://appuser:%s@localhost:5432/app\n", secretVal))

	report := RunDoctor(DoctorOptions{}, deps)

	// Verify secret never appears in any message, fix, or diagnostic
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

