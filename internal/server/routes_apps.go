package server

import (
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/azmoweb/azmoos/internal/auth"
	"github.com/azmoweb/azmoos/internal/docker"
	"github.com/azmoweb/azmoos/internal/events"
	"github.com/azmoweb/azmoos/internal/store"
	"github.com/azmoweb/azmoos/internal/system"
)

type appView struct {
	store.App
	State      string             `json:"state"` // running, stopped, partial, unknown
	Containers []docker.Container `json:"containers"`
}

type catalogView struct {
	store.App
	Compatible bool   `json:"compatible"`
	Warning    string `json:"warning,omitempty"`
	// WarnKind permet à l'interface de rédiger l'avertissement dans sa langue : « arch » ou « ram ».
	WarnKind string `json:"warnKind,omitempty"`
	CPUArch  string `json:"cpuArch,omitempty"`
}

func (s *Server) routesApps(handle routeFn) {
	handle("GET /api/store/catalog", "", s.catalog)
	handle("GET /api/apps", "", s.apps)
	handle("GET /api/apps/{id}/compose", auth.PermAppsManage, func(w http.ResponseWriter, r *http.Request) {
		c, err := s.Store.Compose(r.PathValue("id"))
		if err != nil {
			fail(w, 404, err)
			return
		}
		writeJSON(w, 200, map[string]string{"compose": c})
	})
	handle("POST /api/apps/{id}/{action}", auth.PermAppsManage, s.appAction)

	handle("GET /api/docker/containers", auth.PermAppsManage, func(w http.ResponseWriter, r *http.Request) {
		list, err := s.Docker.List()
		if err != nil {
			fail(w, 503, err)
			return
		}
		writeJSON(w, 200, list)
	})
	handle("GET /api/docker/stats", "", func(w http.ResponseWriter, r *http.Request) {
		list, err := s.Docker.List()
		if err != nil {
			fail(w, 503, err)
			return
		}
		var ids []string
		for _, c := range list {
			if c.State == "running" {
				ids = append(ids, c.ID)
			}
		}
		writeJSON(w, 200, s.Docker.AllStats(ids))
	})
	handle("GET /api/docker/containers/{id}", auth.PermAppsManage, func(w http.ResponseWriter, r *http.Request) {
		d, err := s.Docker.Inspect(r.PathValue("id"))
		if err != nil {
			fail(w, 404, err)
			return
		}
		writeJSON(w, 200, d)
	})
	handle("POST /api/docker/containers/{id}/{action}", auth.PermAppsManage, func(w http.ResponseWriter, r *http.Request) {
		if err := s.Docker.Action(r.PathValue("id"), r.PathValue("action")); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Info, "apps", "Conteneur "+short(r.PathValue("id"))+" : "+r.PathValue("action"))
		ok(w)
	})
	handle("GET /api/docker/containers/{id}/logs", auth.PermAppsManage, func(w http.ResponseWriter, r *http.Request) {
		tail, _ := strconv.Atoi(r.URL.Query().Get("tail"))
		if tail <= 0 || tail > 5000 {
			tail = 400
		}
		logs, err := s.Docker.Logs(r.PathValue("id"), tail)
		if err != nil {
			fail(w, 400, err)
			return
		}
		writeJSON(w, 200, map[string]string{"logs": logs})
	})
	handle("GET /api/docker/images", auth.PermAppsManage, func(w http.ResponseWriter, r *http.Request) {
		l, err := s.Docker.Images()
		if err != nil {
			fail(w, 503, err)
			return
		}
		writeJSON(w, 200, l)
	})
	handle("DELETE /api/docker/images/{id}", auth.PermAppsManage, func(w http.ResponseWriter, r *http.Request) {
		if err := s.Docker.RemoveImage(r.PathValue("id")); err != nil {
			fail(w, 400, err)
			return
		}
		ok(w)
	})
	handle("POST /api/docker/prune", auth.PermAppsManage, func(w http.ResponseWriter, r *http.Request) {
		n, err := s.Docker.Prune()
		if err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Info, "apps", "Nettoyage des images inutilisées")
		writeJSON(w, 200, map[string]uint64{"reclaimed": n})
	})
	handle("GET /api/docker/volumes", auth.PermAppsManage, func(w http.ResponseWriter, r *http.Request) {
		l, err := s.Docker.Volumes()
		if err != nil {
			fail(w, 503, err)
			return
		}
		writeJSON(w, 200, l)
	})
	handle("DELETE /api/docker/volumes/{name}", auth.PermAppsManage, func(w http.ResponseWriter, r *http.Request) {
		if err := s.Docker.RemoveVolume(r.PathValue("name")); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Warning, "apps", "Volume supprimé : "+r.PathValue("name"))
		ok(w)
	})
}

func short(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func (s *Server) catalog(w http.ResponseWriter, r *http.Request) {
	arch := system.CatalogArch()
	out := []catalogView{}
	for _, a := range s.Store.Catalog() {
		v := catalogView{App: a, Compatible: true}
		if len(a.Arch) > 0 && !slices.Contains(a.Arch, arch) {
			v.Compatible, v.WarnKind, v.CPUArch = false, "arch", arch
			v.Warning = "Cette application n'existe pas pour ton processeur (" + arch + "). Un système 64 bits est nécessaire."
		} else if a.MinRAM > 0 && s.Hardware.RAMMB > 0 && a.MinRAM > s.Hardware.RAMMB {
			v.Compatible, v.WarnKind = s.Hardware.Profile != system.ProfileLight, "ram"
			v.Warning = "Cette application demande environ " + strconv.Itoa(a.MinRAM/1024) + " Go de RAM, ton Pi en a " + ramLabel(s.Hardware.RAMMB) + ". Elle risque d'être lente."
			if a.MinRAM < 1024 {
				v.Warning = "Cette application demande environ " + strconv.Itoa(a.MinRAM) + " Mo de RAM, ton Pi en a " + ramLabel(s.Hardware.RAMMB) + "."
			}
		}
		out = append(out, v)
	}
	writeJSON(w, 200, out)
}

func ramLabel(mb int) string {
	if mb >= 1024 {
		return strconv.FormatFloat(float64(mb)/1024, 'f', 1, 64) + " Go"
	}
	return strconv.Itoa(mb) + " Mo"
}

func (s *Server) apps(w http.ResponseWriter, r *http.Request) {
	list, derr := s.Docker.List()
	byProject := map[string][]docker.Container{}
	for _, c := range list {
		byProject[c.Project] = append(byProject[c.Project], c)
	}
	out := []appView{}
	for _, a := range s.Store.Installed() {
		v := appView{App: a, Containers: byProject[store.ProjectPrefix+a.ID], State: "stopped"}
		running := 0
		for _, c := range v.Containers {
			if c.State == "running" {
				running++
			}
		}
		switch {
		case derr != nil:
			v.State = "unknown"
		case running > 0 && running == len(v.Containers):
			v.State = "running"
		case running > 0:
			v.State = "partial"
		}
		if v.Containers == nil {
			v.Containers = []docker.Container{}
		}
		out = append(out, v)
	}
	others := []docker.Container{}
	for _, c := range list {
		if !strings.HasPrefix(c.Project, store.ProjectPrefix) {
			others = append(others, c)
		}
	}
	res := map[string]any{"apps": out, "others": others, "docker": derr == nil}
	writeJSON(w, 200, res)
}

var actionLabels = map[string]string{"install": "installation", "update": "mise à jour", "uninstall": "désinstallation", "start": "démarrage", "stop": "arrêt", "restart": "redémarrage"}

func (s *Server) appAction(w http.ResponseWriter, r *http.Request) {
	id, action := r.PathValue("id"), r.PathValue("action")
	var err error
	switch action {
	case "install":
		var req struct {
			Compose, Name, Icon, Path string
			Port                      int
		}
		if r.ContentLength != 0 && !decode(w, r, &req) {
			return
		}
		err = s.Store.Install(id, req.Compose, &store.App{Name: req.Name, Icon: req.Icon, Port: req.Port, Path: req.Path, Category: "Perso"})
	case "update":
		err = s.Store.Update(id)
	case "uninstall":
		err = s.Store.Uninstall(id)
	case "start", "stop", "restart":
		err = s.Store.Control(id, action)
	default:
		err = errString("action inconnue")
	}
	if err != nil {
		fail(w, 400, err)
		return
	}
	s.log(r, events.Info, "apps", "Application « "+id+" » : "+actionLabels[action]+" lancée")
	ok(w)
}
