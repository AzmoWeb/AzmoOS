package server

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/azmoweb/azmoos/internal/auth"
)

const maxWallpaper = 12 << 20

var wallpaperTypes = map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}

func (s *Server) wallpaperPath(user string) string {
	return filepath.Join(s.Config.Dir(), "wallpapers", user)
}

func (s *Server) routesWallpaper(handle routeFn) {
	handle("POST /api/me/wallpaper/preset", "", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ ID string }
		if !decode(w, r, &req) {
			return
		}
		if err := s.Auth.SetWallpaper(auth.From(r.Context()).Username, req.ID); err != nil {
			fail(w, 400, err)
			return
		}
		ok(w)
	})
	// Image personnelle : stockée dans /etc/azmoos/wallpapers/<utilisateur>.
	handle("GET /api/me/wallpaper", "", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(s.wallpaperPath(auth.From(r.Context()).Username))
		if err != nil {
			fail(w, 404, errString("aucune image personnelle"))
			return
		}
		defer f.Close()
		st, _ := f.Stat()
		w.Header().Set("Cache-Control", "private, max-age=3600")
		http.ServeContent(w, r, "wallpaper", st.ModTime(), f)
	})
	handle("POST /api/me/wallpaper", "", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxWallpaper+(1<<20))
		mr, err := r.MultipartReader()
		if err != nil {
			fail(w, 400, errString("image manquante"))
			return
		}
		part, err := mr.NextPart()
		if err != nil {
			fail(w, 400, errString("image manquante"))
			return
		}
		data, err := io.ReadAll(io.LimitReader(part, maxWallpaper+1))
		if err != nil || len(data) > maxWallpaper {
			fail(w, 400, errString("image trop lourde (12 Mo maximum)"))
			return
		}
		// Le type est déterminé à partir du contenu, pas du nom du fichier.
		if _, okType := wallpaperTypes[http.DetectContentType(data)]; !okType {
			fail(w, 400, errString("format non pris en charge (JPEG, PNG ou WebP)"))
			return
		}
		user := auth.From(r.Context()).Username
		p := s.wallpaperPath(user)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			fail(w, 500, err)
			return
		}
		if err := os.WriteFile(p+".tmp", data, 0o600); err != nil {
			fail(w, 500, err)
			return
		}
		if err := os.Rename(p+".tmp", p); err != nil {
			fail(w, 500, err)
			return
		}
		_ = s.Auth.SetWallpaper(user, "custom")
		ok(w)
	})
	handle("DELETE /api/me/wallpaper", "", func(w http.ResponseWriter, r *http.Request) {
		user := auth.From(r.Context()).Username
		_ = os.Remove(s.wallpaperPath(user))
		_ = s.Auth.SetWallpaper(user, "azmo")
		ok(w)
	})
}
