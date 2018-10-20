// +build darwin

package crash_dump

import (
	"os"
	"syscall"
)

func CrashLog(panicFile string) error {
	file, err := os.OpenFile(panicFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	if err = syscall.Dup2(int(file.Fd()), int(os.Stderr.Fd())); err != nil {
		return err
	}
	return nil
}
