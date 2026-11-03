package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

func tcget(fd uintptr) (*syscall.Termios, error) {
	var t syscall.Termios
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TCGETS, uintptr(unsafe.Pointer(&t))); e != 0 {
		return nil, e
	}
	return &t, nil
}

func tcset(fd uintptr, t *syscall.Termios) {
	syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TCSETS, uintptr(unsafe.Pointer(t)))
}

func isTerminal(f *os.File) bool {
	_, err := tcget(f.Fd())
	return err == nil
}

// promptHidden lit une ligne sans l'afficher (saisie de mot de passe).
func promptHidden(label string) (string, error) {
	fd := os.Stdin.Fd()
	fmt.Fprint(os.Stderr, label)
	old, err := tcget(fd)
	if err == nil {
		t := *old
		t.Lflag &^= syscall.ECHO
		tcset(fd, &t)
		defer tcset(fd, old)
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	fmt.Fprintln(os.Stderr)
	return strings.TrimRight(line, "\r\n"), err
}
