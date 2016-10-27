package gardener

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
)

// SYS_SETNS syscall allows changing the namespace of the current process.
var SYS_SETNS = map[string]uintptr{
	"386":     346,
	"amd64":   308,
	"arm64":   268,
	"arm":     375,
	"ppc64":   350,
	"ppc64le": 350,
	"s390x":   339,
}[runtime.GOARCH]

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

	newns := fd.Fd()
	if err := Setns(newns); err != nil {
		return fmt.Errorf("set netns: %s", err)
	}

	err = cb()
	mustSetns(uintptr(originalNsFd)) // if this happens we definitely can't recover
	return err
}

func mustSetns(ns uintptr) {
	if err := Setns(ns); err != nil {
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
	return int(fd), nil
}
