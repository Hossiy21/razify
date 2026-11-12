package doctor

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"
)

// ToolchainRunner inspects available system developer toolchains.
type ToolchainRunner interface {
	Run(name string, args ...string) (string, error)
}

// PortChecker verifies whether a declared local TCP port is in conflict.
type PortChecker interface {
	CheckPort(port int) error
}

// ServiceProber tests read-only TCP connectivity to a declared service.
type ServiceProber interface {
	Probe(network, address string, timeout time.Duration) error
}

// FileSystem abstracts filesystem lookups for offline table-driven testing.
type FileSystem interface {
	FileExists(path string) bool
	ReadFile(path string) ([]byte, error)
}

// OSToolchainRunner executes real system binaries to check toolchain versions.
type OSToolchainRunner struct{}

func (r *OSToolchainRunner) Run(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return string(out), err
}

// NetPortChecker attempts a cross-platform net.Listen to detect port conflicts.
type NetPortChecker struct{}

func (p *NetPortChecker) CheckPort(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	return ln.Close()
}

// NetServiceProber conducts auth-less, strictly TCP-only connect probes.
type NetServiceProber struct{}

func (s *NetServiceProber) Probe(network, address string, timeout time.Duration) error {
	conn, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return err
	}
	return conn.Close()
}

// OSFileSystem reads files from the local filesystem.
type OSFileSystem struct{}

func (f *OSFileSystem) FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || info == nil || info.IsDir() {
		return false
	}
	return true
}

func (f *OSFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

