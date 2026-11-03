//go:build !linux

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Hors Linux (développement sur Mac/Windows) : saisie simple, sans masquage.
func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func promptHidden(label string) (string, error) {
	fmt.Fprint(os.Stderr, label)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimRight(line, "\r\n"), err
}
