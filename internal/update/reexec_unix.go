//go:build !windows

package update

import "syscall"

// ReExec replaces the current process image with the given binary.
// On Unix, this preserves the PID and any supervisor's understanding of the process.
func ReExec(path string, args, env []string) error {
	return syscall.Exec(path, args, env)
}
