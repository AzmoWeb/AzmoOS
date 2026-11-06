//go:build !linux

package terminal

import (
	"errors"
	"os"
	"syscall"
)

func openPTY() (*os.File, *os.File, error) {
	return nil, nil, errors.New("terminal disponible uniquement sous Linux")
}

func setSize(*os.File, uint16, uint16) error { return nil }

func sysProcAttr(uint32, uint32, []uint32, bool) *syscall.SysProcAttr { return &syscall.SysProcAttr{} }
