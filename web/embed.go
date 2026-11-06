// Package web embarque l'interface compilée (frontend/ → web/dist) dans le binaire.
// Elle sert de secours : si /usr/share/azmoos/web existe, c'est cette version qui est utilisée.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var content embed.FS

// FS renvoie les fichiers de l'interface web.
func FS() fs.FS {
	sub, _ := fs.Sub(content, "dist")
	return sub
}
