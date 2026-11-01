//go:build windows

package cli

import (
	"os/exec"
	"syscall"
	"unsafe"
)

func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000 | 0x00000200 | 0x00000008, // CREATE_NO_WINDOW | CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS
	}
}

// IsProcessAlive checks if a process with the specified PID is currently running.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	openProcess := kernel32.NewProc("OpenProcess")
	handle, _, _ := openProcess.Call(0x0400, 0, uintptr(pid)) // PROCESS_QUERY_INFORMATION
	if handle == 0 {
		return false
	}
	defer func() {
		closeHandle := kernel32.NewProc("CloseHandle")
		closeHandle.Call(handle)
	}()

	getExitCodeProcess := kernel32.NewProc("GetExitCodeProcess")
	var exitCode uint32
	ret, _, _ := getExitCodeProcess.Call(handle, uintptr(unsafe.Pointer(&exitCode)))
	if ret == 0 {
		return false
	}
	return exitCode == 259 // STILL_ACTIVE
}
