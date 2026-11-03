// Package cmdutil simplifie l'exécution des commandes système.
package cmdutil

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Run exécute une commande (timeout 60 s) et renvoie sa sortie combinée.
func Run(name string, args ...string) (string, error) {
	return RunTimeout(60*time.Second, name, args...)
}

// RunTimeout exécute une commande avec un délai maximal.
func RunTimeout(d time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil {
		if s != "" {
			return s, fmt.Errorf("%s", lastLine(s))
		}
		return s, fmt.Errorf("%s : %w", name, err)
	}
	return s, nil
}

// Has indique si un programme est installé.
func Has(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func lastLine(s string) string {
	l := strings.Split(strings.TrimSpace(s), "\n")
	return l[len(l)-1]
}
