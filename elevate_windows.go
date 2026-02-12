//go:build windows

package main

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	shell32          = syscall.NewLazyDLL("shell32.dll")
	procShellExecute = shell32.NewProc("ShellExecuteW")
)

// shellExecuteRunAs calls the Win32 ShellExecuteW API with "runas" verb.
// This always triggers a foreground UAC consent dialog.
// It blocks until the elevated process exits (via SEE_MASK_NOCLOSEPROCESS would
// be needed for waiting, but ShellExecuteW is fire-and-forget).
func shellExecuteRunAs(program, args string) error {
	verbPtr, _ := syscall.UTF16PtrFromString("runas")
	exePtr, _ := syscall.UTF16PtrFromString(program)
	argsPtr, _ := syscall.UTF16PtrFromString(args)
	dirPtr, _ := syscall.UTF16PtrFromString("")

	ret, _, _ := procShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(exePtr)),
		uintptr(unsafe.Pointer(argsPtr)),
		uintptr(unsafe.Pointer(dirPtr)),
		0, // SW_HIDE
	)
	// ShellExecuteW returns >32 on success
	if ret <= 32 {
		return fmt.Errorf("ShellExecuteW failed with code %d", ret)
	}
	return nil
}

// addDefenderExclusion adds the .archbox directory to Windows Defender's
// exclusion list by calling ShellExecuteW with "runas" to trigger a
// foreground UAC dialog. This is how real Windows programs request elevation.
func addDefenderExclusion() error {
	dir := filepath.Dir(binDir()) // ~/.archbox
	fmt.Println("  Adding Windows Defender exclusion (click Yes on the UAC dialog)...")

	args := fmt.Sprintf(
		`-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -Command "Add-MpPreference -ExclusionPath '%s'"`,
		dir)

	if err := shellExecuteRunAs("powershell.exe", args); err != nil {
		return err
	}

	fmt.Printf("  Defender exclusion added for %s\n", dir)
	return nil
}
