package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/azmoweb/azmoos/internal/auth"
	"github.com/azmoweb/azmoos/internal/events"
)

func (s *Server) routesFiles(handle routeFn) {
	handle("GET /api/files", auth.PermFilesRead, func(w http.ResponseWriter, r *http.Request) {
		entries, err := s.Files.List(r.URL.Query().Get("path"))
		if err != nil {
			fail(w, 400, err)
			return
		}
		writeJSON(w, 200, entries)
	})
	handle("GET /api/files/download", auth.PermFilesRead, s.fileDownload)
	handle("POST /api/files/upload", auth.PermFilesWrite, func(w http.ResponseWriter, r *http.Request) {
		mr, err := r.MultipartReader()
		if err != nil {
			fail(w, 400, err)
			return
		}
		dir := r.URL.Query().Get("path")
		n := 0
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			if part.FormName() != "files" || part.FileName() == "" {
				continue
			}
			if err := s.Files.SaveStream(dir, part.FileName(), part); err != nil {
				fail(w, 400, err)
				return
			}
			n++
		}
		s.log(r, events.Info, "files", strconv.Itoa(n)+" fichier(s) envoyé(s) dans "+dir)
		ok(w)
	})
	handle("POST /api/files/{op}", auth.PermFilesWrite, func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Path, Name, To string }
		if !decode(w, r, &req) {
			return
		}
		var err error
		op := r.PathValue("op")
		switch op {
		case "mkdir":
			err = s.Files.Mkdir(req.Path, req.Name)
		case "rename":
			err = s.Files.Rename(req.Path, req.Name)
		case "move":
			err = s.Files.Move(req.Path, req.To)
		case "delete":
			err = s.Files.Delete(req.Path)
			if err == nil {
				s.log(r, events.Info, "files", "Supprimé : "+req.Path)
			}
		default:
			err = errString("opération inconnue")
		}
		if err != nil {
			fail(w, 400, err)
			return
		}
		ok(w)
	})
}

func (s *Server) fileDownload(w http.ResponseWriter, r *http.Request) {
	p, err := s.Files.Resolve(r.URL.Query().Get("path"))
	if err != nil {
		fail(w, 400, err)
		return
	}
	f, err := os.Open(p)
	if err != nil {
		fail(w, 404, errString("fichier introuvable"))
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		fail(w, 400, errString("ce n'est pas un fichier"))
		return
	}
	disp := "attachment"
	if r.URL.Query().Get("inline") == "1" {
		disp = "inline"
	}
	w.Header().Set("Content-Disposition", disp+`; filename*=UTF-8''`+urlEscape(filepath.Base(p)))
	w.Header().Set("Content-Security-Policy", "sandbox")
	http.ServeContent(w, r, st.Name(), st.ModTime(), f)
}

func urlEscape(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || strings.IndexByte("-._~", c) >= 0 {
			b.WriteByte(c)
		} else {
			b.WriteString("%" + strings.ToUpper(strconv.FormatInt(int64(c)|0x100, 16)[1:]))
		}
	}
	return b.String()
}
