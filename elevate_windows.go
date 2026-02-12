//go:build windows

package main

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	shell32            = syscall.NewLazyDLL("shell32.dll")
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procShellExecuteEx = shell32.NewProc("ShellExecuteExW")
	procWaitForSingle  = kernel32.NewProc("WaitForSingleObject")
	procCloseHandle    = kernel32.NewProc("CloseHandle")
)

// SHELLEXECUTEINFO structure for ShellExecuteExW
type shellExecuteInfo struct {
	cbSize       uint32
	fMask        uint32
	hwnd         uintptr
	lpVerb       uintptr
	lpFile       uintptr
	lpParameters uintptr
	lpDirectory  uintptr
	nShow        int32
	hInstApp     uintptr
	lpIDList     uintptr
	lpClass      uintptr
	hkeyClass    uintptr
	dwHotKey     uint32
	hIcon        uintptr
	hProcess     uintptr
}

const (
	seeMaskNoCloseProcess = 0x00000040
	infinite              = 0xFFFFFFFF
)

// shellExecuteRunAsAndWait calls ShellExecuteExW with "runas" verb and waits
// for the elevated process to finish. This shows a foreground UAC dialog and
// blocks until the user responds and the elevated command completes.
func shellExecuteRunAsAndWait(program, args string) error {
	verbPtr, _ := syscall.UTF16PtrFromString("runas")
	exePtr, _ := syscall.UTF16PtrFromString(program)
	argsPtr, _ := syscall.UTF16PtrFromString(args)

	sei := shellExecuteInfo{
		fMask:        seeMaskNoCloseProcess,
		lpVerb:       uintptr(unsafe.Pointer(verbPtr)),
		lpFile:       uintptr(unsafe.Pointer(exePtr)),
		lpParameters: uintptr(unsafe.Pointer(argsPtr)),
		nShow:        0, // SW_HIDE
	}
	sei.cbSize = uint32(unsafe.Sizeof(sei))

	ret, _, err := procShellExecuteEx.Call(uintptr(unsafe.Pointer(&sei)))
	if ret == 0 {
		return fmt.Errorf("ShellExecuteExW: %v", err)
	}

	if sei.hProcess != 0 {
		// Wait for the elevated process to finish
		procWaitForSingle.Call(sei.hProcess, uintptr(infinite))
		procCloseHandle.Call(sei.hProcess)
	}

	return nil
}

// addDefenderExclusion adds the .archbox directory to Windows Defender's
// exclusion list by calling ShellExecuteExW with "runas" to trigger a
// foreground UAC dialog, then waits for the elevated process to complete
// before returning.
func addDefenderExclusion() error {
	dir := filepath.Dir(binDir()) // ~/.archbox
	fmt.Println("  Adding Windows Defender exclusion (click Yes on the UAC dialog)...")

	args := fmt.Sprintf(
		`-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -Command "Add-MpPreference -ExclusionPath '%s'"`,
		dir)

	if err := shellExecuteRunAsAndWait("powershell.exe", args); err != nil {
		return err
	}

	fmt.Printf("  Defender exclusion added for %s\n", dir)
	return nil
}
