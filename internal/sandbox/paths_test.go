package sandbox_test

import (
	"strings"
	"testing"

	"github.com/Hossiy21/razify/internal/sandbox"
)

func TestToDockerMountPath(t *testing.T) {
	testCases := []struct {
		input string
		want  string
	}{
		{`C:\Users\test\project`, "C:/Users/test/project"},
		{`c:\workspace\repo`, "c:/workspace/repo"},
		{`/home/user/project`, "/home/user/project"},
		{`./sub/dir`, "sub/dir"},
	}

	for _, tc := range testCases {
		got := sandbox.ToDockerMountPath(tc.input)
		// On non-Windows platforms, C:\Users is treated as a relative filename,
		// but ToDockerMountPath must never contain backslashes.
		if strings.Contains(got, `\`) {
			t.Errorf("ToDockerMountPath(%q) contains backslashes: %s", tc.input, got)
		}
	}
}

func TestFormatMountArg(t *testing.T) {
	got := sandbox.FormatMountArg(`/home/user/repo`, `/workspace`, `ro`)
	want := `/home/user/repo:/workspace:ro`
	if got != want {
		t.Errorf("FormatMountArg = %s, want %s", got, want)
	}

	gotNoMode := sandbox.FormatMountArg(`/home/user/cache`, `/cache`, ``)
	wantNoMode := `/home/user/cache:/cache`
	if gotNoMode != wantNoMode {
		t.Errorf("FormatMountArg (no mode) = %s, want %s", gotNoMode, wantNoMode)
	}
}

