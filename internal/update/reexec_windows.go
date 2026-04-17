//go:build windows

package update

import (
	"os"
	"os/exec"
	"syscall"
)

// ReExec spawns a fresh child process and exits the parent. Windows has no exec(2)
// equivalent that replaces the process image, so the PID will change. If you need
// PID stability, run under nssm / a Windows service wrapper.
func ReExec(path string, args, env []string) error {
	childArgs := []string{}
	if len(args) > 1 {
		childArgs = args[1:]
	}
	cmd := exec.Command(path, childArgs...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
