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
// VAR=value arguments. We deliberately do not forward PATH: sudo's default
// env_reset keeps root's PATH, so a root process cannot resolve containerlab
// or docker from a user-writable directory.
var sudoEnvVars = []string{"CLABLINGS_ROOT", "CLABLINGS_INSTANCE"}

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

	env := make(map[string]string, len(sudoEnvVars)+1)
	for _, key := range sudoEnvVars {
		if value, ok := os.LookupEnv(key); ok && value != "" {
			env[key] = value
		}
	}

	return syscall.Exec(sudo, sudoArgv(exe, args, env), os.Environ())
}

// sudoArgv builds the sudo command line: sudo, the forwarded env assignments,
// the absolute clablings path, and the original args without --sudo.
func sudoArgv(exe string, args []string, env map[string]string) []string {
	argv := []string{"sudo"}
	for _, key := range sudoEnvVars {
		if value := env[key]; value != "" {
			argv = append(argv, key+"="+value)
		}
	}
	argv = append(argv, exe)
	return append(argv, stripSudoFlag(args)...)
}

// shouldElevate reports whether the global --sudo appears before the
// subcommand. A --sudo after the subcommand belongs to that command; for
// `debug` it keeps its local clab-tui meaning.
func shouldElevate(args []string, euid int) bool {
	if euid == 0 {
		return false
	}
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == "--sudo" {
			return true
		}
		if strings.HasPrefix(arg, "--sudo=") {
			return !strings.EqualFold(strings.TrimPrefix(arg, "--sudo="), "false")
		}
		if !strings.HasPrefix(arg, "-") {
			return false
		}
	}
	return false
}

// stripSudoFlag removes --sudo and --sudo=... from args so the re-executed
// process does not try to elevate itself again. Everything after a `--`
// delimiter is positional and is preserved verbatim.
func stripSudoFlag(args []string) []string {
	out := make([]string, 0, len(args))
	for i, arg := range args {
		if arg == "--" {
			out = append(out, args[i:]...)
			return out
		}
		if arg == "--sudo" || strings.HasPrefix(arg, "--sudo=") {
			continue
		}
		out = append(out, arg)
	}
	return out
}
