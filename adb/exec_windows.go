//go:build windows

package adb

import (
	"os/exec"
	"syscall"
)

var execCommand = exec.Command

func newCommand(name string, arg ...string) *exec.Cmd {
	cmd := execCommand(name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}
