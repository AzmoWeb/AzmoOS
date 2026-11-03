// azmod est le démon principal d'AzmoOS — made by Azmo.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/azmoweb/azmoos/internal/auth"
	"github.com/azmoweb/azmoos/internal/config"
	"github.com/azmoweb/azmoos/internal/docker"
	"github.com/azmoweb/azmoos/internal/files"
	"github.com/azmoweb/azmoos/internal/server"
	"github.com/azmoweb/azmoos/internal/store"
	"github.com/azmoweb/azmoos/internal/system"
	"github.com/azmoweb/azmoos/web"
)

var version = "dev"

func main() {
	addr := flag.String("addr", ":80", "adresse d'écoute")
	dataDir := flag.String("data", "/DATA", "dossier des données utilisateur")
	confDir := flag.String("config", "/etc/azmoos", "dossier de configuration")
	sock := flag.String("docker", "/var/run/docker.sock", "socket Docker")
	showVersion := flag.Bool("version", false, "affiche la version")
	flag.Parse()

	if *showVersion {
		fmt.Println("AzmoOS", version)
		return
	}
	for _, d := range []string{"", "AppData", "Media", "Media/Movies", "Media/TV", "Media/Music", "Downloads", "Documents", "Backups"} {
		_ = os.MkdirAll(*dataDir+"/"+d, 0o755)
	}
	cfg, err := config.Load(*confDir)
	if err != nil {
		log.Fatalf("configuration: %v", err)
	}
	st, err := store.New(*dataDir)
	if err != nil {
		log.Fatalf("app store: %v", err)
	}
	srv := &server.Server{
		Version: version,
		Auth:    auth.New(cfg),
		Docker:  docker.New(*sock),
		Store:   st,
		Files:   files.New(*dataDir),
		Monitor: system.NewMonitor(*dataDir),
		Web:     web.FS(),
	}
	hs := &http.Server{Addr: *addr, Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("AzmoOS %s — made by Azmo — écoute sur %s (données: %s)", version, *addr, *dataDir)
	log.Fatal(hs.ListenAndServe())
}
