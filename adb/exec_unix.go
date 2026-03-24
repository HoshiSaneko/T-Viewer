//go:build !windows

package adb

import (
	"os/exec"
)

var execCommand = exec.Command

func newCommand(name string, arg ...string) *exec.Cmd {
	return execCommand(name, arg...)
}
