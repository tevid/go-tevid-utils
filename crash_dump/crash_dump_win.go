// +build windows

package crash_dump

import (
	"os"
	"syscall"
)

const (
	kernel32dll = "kernel32.dll"
)

func CrashLog(panicFile string) error {

	file, err := os.OpenFile(panicFile, os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	kernel32 := syscall.NewLazyDLL(kernel32dll)
	setStdHandle := kernel32.NewProc("SetStdHandle")
	sh := syscall.STD_ERROR_HANDLE
	v, _, err := setStdHandle.Call(uintptr(sh), uintptr(file.Fd()))
	if v == 0 {
		return err
	}
	return nil
}
