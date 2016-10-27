package gardener

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
)

type MountNsExecer struct{}

func (e MountNsExecer) Exec(nsPath *os.File, cb func() error) error {
	return Exec(nsPath, cb)
}

func Exec(fd *os.File, cb func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	originalNsFd, err := GetMntNsFd()
	if err != nil {
		return err
	}
	defer originalNsFd.Close()

	newns := fd.Fd()
	if err := SetNs(newns); err != nil {
		return fmt.Errorf("set netns: %s", err)
	}

	err := cb()
	mustSetNs(originalNsFd) // if this happens we definitely can't recover
	return err
}

func mustSetNs(ns vishnetns.NsHandle) {
	if err := vishnetns.Set(ns); err != nil {
		panic(err)
	}
}

// Setns sets namespace using syscall. Note that this should be a method
// in syscall but it has not been added.
func Setns(ns uintptr) (err error) {
	_, _, e1 := syscall.Syscall(SYS_SETNS, ns, syscall.CLONE_NEWNS, 0)
	if e1 != 0 {
		err = e1
	}
	return
}

func GetMntNsFd() (int, error) {
	fd, err := syscall.Open("/proc/self/ns/mnt", syscall.O_RDONLY, 0)
	if err != nil {
		return -1, err
	}
	return NsHandle(fd), nil
}
