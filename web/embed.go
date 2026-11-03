// Package web contient l'interface web embarquée dans le binaire.
package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var content embed.FS

// FS renvoie le système de fichiers des ressources statiques.
func FS() fs.FS {
	sub, _ := fs.Sub(content, "static")
	return sub
}
