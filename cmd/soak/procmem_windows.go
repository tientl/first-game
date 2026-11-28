//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	psapi                = syscall.NewLazyDLL("psapi.dll")
	getProcessMemoryInfo = psapi.NewProc("GetProcessMemoryInfo")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	getCurrentProcess    = kernel32.NewProc("GetCurrentProcess")
)

// processMemory is the working set and the private bytes (commit) of this
// process, in bytes; 0, 0 if Windows will not say.
func processMemory() (workingSet, private uint64) {
	var c struct {
		cb                         uint32
		PageFaultCount             uint32
		PeakWorkingSetSize         uintptr
		WorkingSetSize             uintptr
		QuotaPeakPagedPoolUsage    uintptr
		QuotaPagedPoolUsage        uintptr
		QuotaPeakNonPagedPoolUsage uintptr
		QuotaNonPagedPoolUsage     uintptr
		PagefileUsage              uintptr
		PeakPagefileUsage          uintptr
		PrivateUsage               uintptr
	}
	c.cb = uint32(unsafe.Sizeof(c))
	h, _, _ := getCurrentProcess.Call()
	if r, _, _ := getProcessMemoryInfo.Call(h, uintptr(unsafe.Pointer(&c)), uintptr(c.cb)); r == 0 {
		return 0, 0
	}
	return uint64(c.WorkingSetSize), uint64(c.PrivateUsage)
}
