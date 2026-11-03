// azmod est le démon principal d'Azmo OS — Made by Azmo.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/azmoweb/azmoos/internal/auth"
	"github.com/azmoweb/azmoos/internal/backup"
	"github.com/azmoweb/azmoos/internal/config"
	"github.com/azmoweb/azmoos/internal/docker"
	"github.com/azmoweb/azmoos/internal/events"
	"github.com/azmoweb/azmoos/internal/files"
	"github.com/azmoweb/azmoos/internal/jobs"
	"github.com/azmoweb/azmoos/internal/server"
	"github.com/azmoweb/azmoos/internal/shares"
	"github.com/azmoweb/azmoos/internal/storage"
	"github.com/azmoweb/azmoos/internal/store"
	"github.com/azmoweb/azmoos/internal/system"
	"github.com/azmoweb/azmoos/internal/updater"
	"github.com/azmoweb/azmoos/web"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "user" {
		os.Exit(runUser(os.Args[2:]))
	}
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Azmo OS %s — Made by Azmo\n\nUsage :\n  azmod [options]        lance le serveur\n  azmod user …           gère les comptes (azmod user help)\n\nOptions :\n", version)
		flag.PrintDefaults()
	}
	addr := flag.String("addr", ":80", "adresse d'écoute")
	dataDir := flag.String("data", "/DATA", "dossier des données utilisateur")
	confDir := flag.String("config", "/etc/azmoos", "dossier de configuration")
	webDir := flag.String("web", "/usr/share/azmoos/web", "interface web (si absente : version embarquée)")
	sock := flag.String("docker", "/var/run/docker.sock", "socket Docker")
	repo := flag.String("repo", "AzmoWeb/AzmoOS", "dépôt GitHub des mises à jour")
	showVersion := flag.Bool("version", false, "affiche la version")
	flag.Parse()

	if *showVersion {
		fmt.Println("Azmo OS", version)
		return
	}
	// Chemins absolus obligatoires : ils sont écrits dans les fichiers docker-compose.
	for _, p := range []*string{dataDir, confDir, webDir} {
		if abs, err := filepath.Abs(*p); err == nil {
			*p = abs
		}
	}
	// Dossiers utilisateur : créés pour l'utilisateur des apps (1000) afin qu'elles puissent y écrire.
	for _, d := range []string{"", "AppData", "Media", "Media/Movies", "Media/TV", "Media/Music", "Photos", "Downloads", "Documents", "Backups", "Disks"} {
		p := *dataDir + "/" + d
		if _, err := os.Stat(p); os.IsNotExist(err) {
			if os.MkdirAll(p, 0o755) == nil && d != "" && d != "AppData" && d != "Disks" {
				_ = os.Chown(p, store.AppUID, store.AppGID)
			}
		}
	}
	cfg, err := config.Load(*confDir)
	if err != nil {
		log.Fatalf("configuration : %v", err)
	}
	evlog := events.Open(*confDir)
	runner := jobs.NewRunner()
	runner.OnFinish = func(j jobs.Job) {
		lvl, verb := events.Info, "terminée"
		if j.Status == "error" {
			lvl, verb = events.Error, "en échec"
		}
		evlog.Add(lvl, category(j.Kind), "", "", fmt.Sprintf("Tâche %s « %s » %s", j.Action, j.Target, verb))
	}
	st, err := store.New(*dataDir, runner)
	if err != nil {
		log.Fatalf("app store : %v", err)
	}

	var webFS fs.FS = web.FS()
	if info, err := os.Stat(*webDir + "/index.html"); err == nil && !info.IsDir() {
		webFS = os.DirFS(*webDir)
		log.Printf("interface web : %s", *webDir)
	}
	_, portStr, _ := net.SplitHostPort(*addr)
	port, _ := strconv.Atoi(portStr)
	if port == 0 {
		port = 80
	}

	hw := system.DetectHardware()
	srv := &server.Server{
		Version:  version,
		WebPort:  port,
		Config:   cfg,
		Auth:     auth.New(cfg),
		Events:   evlog,
		Jobs:     runner,
		Docker:   docker.New(*sock),
		Store:    st,
		Files:    files.New(*dataDir),
		Storage:  storage.New(*dataDir, runner),
		Shares:   shares.New(cfg, *dataDir),
		Backup:   &backup.Manager{ConfDir: *confDir, AppsDir: st.AppsDir(), DataDir: *dataDir, Version: version},
		Updater:  updater.New(*repo, version, *webDir, runner),
		Monitor:  system.NewMonitor(*dataDir),
		Hardware: hw,
		Web:      webFS,
	}
	// Vérification des mises à jour en tâche de fond (au démarrage puis toutes les 12 h).
	go func() {
		time.Sleep(time.Minute)
		for {
			srv.Updater.Check()
			time.Sleep(12 * time.Hour)
		}
	}()

	evlog.Add(events.Info, "system", "", "", "Azmo OS "+version+" démarré sur "+hw.Model)
	hs := &http.Server{Addr: *addr, Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("Azmo OS %s — Made by Azmo — %s (%s, %d Mo, profil %s) — écoute sur %s", version, hw.Model, hw.Arch, hw.RAMMB, hw.Profile, *addr)
	log.Fatal(hs.ListenAndServe())
}

// category associe un type de tâche à une catégorie du journal.
func category(kind string) string {
	switch kind {
	case "app":
		return "apps"
	case "storage":
		return "storage"
	}
	return "system"
}
