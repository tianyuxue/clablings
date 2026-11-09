package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// sudoEnvVars are the clablings environment variables that must survive the
// sudo re-exec. sudo resets the environment, so these are passed as explicit
// VAR=value arguments; PATH is deliberately not preserved, so the root process
// cannot resolve containerlab or docker from a user-writable directory.
var sudoEnvVars = []string{"CLABLINGS_ROOT", "CLABLINGS_INSTANCE", "CLABLINGS_LANG"}

// elevate re-executes the current command as root via sudo. It replaces the
// current process so the terminal, signals, and exit code stay intact.
func elevate(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating clablings executable: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	sudo, err := exec.LookPath("sudo")
	if err != nil {
		return fmt.Errorf("re-running as root: sudo not found in PATH: %w", err)
	}

	argv := []string{"sudo"}
	for _, key := range sudoEnvVars {
		if value, ok := os.LookupEnv(key); ok && value != "" {
			argv = append(argv, key+"="+value)
		}
	}
	argv = append(argv, exe)
	argv = append(argv, stripSudoFlag(args)...)
	return syscall.Exec(sudo, argv, os.Environ())
}

// stripSudoFlag removes --sudo and --sudo=... from args so the re-executed
// process does not try to elevate itself again.
func stripSudoFlag(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--sudo" || strings.HasPrefix(arg, "--sudo=") {
			continue
		}
		out = append(out, arg)
	}
	return out
}
