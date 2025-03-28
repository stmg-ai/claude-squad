//go:build !windows

package daemon

import (
	"syscall"
)

// getSysProcAttr returns platform-specific process attributes for detaching the child process
func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true, // Create a new session
	}
}
