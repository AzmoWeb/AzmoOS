package terminal

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func ioctl(fd, req, arg uintptr) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, req, arg); e != 0 {
		return e
	}
	return nil
}

// openPTY ouvre une paire maître/esclave de pseudo-terminal.
func openPTY() (master, slave *os.File, err error) {
	master, err = os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}
	var unlock int32
	if err = ioctl(master.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock))); err != nil {
		master.Close()
		return nil, nil, err
	}
	var n uint32
	if err = ioctl(master.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n))); err != nil {
		master.Close()
		return nil, nil, err
	}
	slave, err = os.OpenFile(fmt.Sprintf("/dev/pts/%d", n), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		return nil, nil, err
	}
	return master, slave, nil
}

type winsize struct{ Rows, Cols, X, Y uint16 }

func setSize(f *os.File, rows, cols uint16) error {
	ws := winsize{Rows: rows, Cols: cols}
	return ioctl(f.Fd(), syscall.TIOCSWINSZ, uintptr(unsafe.Pointer(&ws)))
}

func sysProcAttr(uid, gid uint32, groups []uint32, asUser bool) *syscall.SysProcAttr {
	a := &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	if asUser {
		a.Credential = &syscall.Credential{Uid: uid, Gid: gid, Groups: groups}
	}
	return a
}
