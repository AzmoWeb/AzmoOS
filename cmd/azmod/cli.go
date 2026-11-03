package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/azmoweb/azmoos/internal/auth"
	"github.com/azmoweb/azmoos/internal/config"
)

const userUsage = `Gestion des comptes Azmo OS (à lancer avec sudo).

  azmod user list                          liste les comptes
  azmod user add <nom> [--display "Nom"] [--member]
                                           crée un compte (admin par défaut)
  azmod user passwd <nom>                  change le mot de passe
  azmod user delete <nom>                  supprime un compte

Le mot de passe est demandé au clavier, ou lu sur l'entrée standard :
  echo 'MotDePasse' | sudo azmod user add azmo

Options communes : --config /etc/azmoos
`

// runUser implémente « azmod user … ». Les changements sont pris en compte
// par le service en cours d'exécution sans redémarrage.
func runUser(args []string) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Print(userUsage)
		return 0
	}
	cmd := args[0]
	fs := flag.NewFlagSet("user "+cmd, flag.ContinueOnError)
	confDir := fs.String("config", "/etc/azmoos", "dossier de configuration")
	display := fs.String("display", "", "nom affiché")
	member := fs.Bool("member", false, "compte membre (sans droits d'administration)")
	// Autorise « add azmo --display X » comme « add --display X azmo ».
	var name string
	rest := args[1:]
	if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
		name, rest = rest[0], rest[1:]
	}
	if err := fs.Parse(rest); err != nil {
		return 2
	}
	if name == "" && fs.NArg() > 0 {
		name = fs.Arg(0)
	}
	name = strings.ToLower(strings.TrimSpace(name))

	cfg, err := config.Load(*confDir)
	if err != nil {
		return fail("configuration illisible : %v (as-tu lancé la commande avec sudo ?)", err)
	}
	a := auth.New(cfg)

	switch cmd {
	case "list", "ls":
		users := a.Users()
		if len(users) == 0 {
			fmt.Println("Aucun compte. Crée le premier avec : sudo azmod user add <nom>")
			return 0
		}
		for _, u := range users {
			role := "membre"
			if u.Role == config.RoleAdmin {
				role = "administrateur"
			}
			fmt.Printf("  %-20s %-24s %s\n", u.Username, u.Display, role)
		}
		return 0
	case "add", "create":
		if name == "" {
			return fail("indique un nom d'utilisateur : azmod user add <nom>")
		}
		pw, err := readPassword(true)
		if err != nil {
			return fail("%v", err)
		}
		role := config.RoleAdmin
		perms := []string(nil)
		if *member {
			role, perms = config.RoleMember, []string{auth.PermFilesRead}
		}
		if *display == "" {
			*display = strings.ToUpper(name[:1]) + name[1:]
		}
		if err := a.CreateUser(name, *display, pw, role, perms); err != nil {
			return fail("%v", err)
		}
		fmt.Printf("✔ Compte « %s » créé.\n", name)
		return 0
	case "passwd", "password":
		if name == "" {
			return fail("indique le compte : azmod user passwd <nom>")
		}
		pw, err := readPassword(true)
		if err != nil {
			return fail("%v", err)
		}
		if err := a.SetPassword(name, pw, ""); err != nil {
			return fail("%v", err)
		}
		fmt.Printf("✔ Mot de passe de « %s » changé. Ses sessions ont été fermées.\n", name)
		return 0
	case "delete", "rm":
		if name == "" {
			return fail("indique le compte : azmod user delete <nom>")
		}
		if err := a.DeleteUser(name); err != nil {
			return fail("%v", err)
		}
		fmt.Printf("✔ Compte « %s » supprimé.\n", name)
		return 0
	}
	fmt.Print(userUsage)
	return 2
}

func fail(f string, a ...any) int {
	fmt.Fprintf(os.Stderr, "✘ "+f+"\n", a...)
	return 1
}

// readPassword lit le mot de passe : au clavier (sans écho, avec confirmation)
// si on est dans un terminal, sinon une ligne sur l'entrée standard.
func readPassword(confirm bool) (string, error) {
	if !isTerminal(os.Stdin) {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if line == "" && err != nil {
			return "", fmt.Errorf("mot de passe attendu sur l'entrée standard")
		}
		return line, nil
	}
	pw, err := promptHidden("Mot de passe : ")
	if err != nil {
		return "", err
	}
	if confirm {
		again, err := promptHidden("Confirme le mot de passe : ")
		if err != nil {
			return "", err
		}
		if again != pw {
			return "", fmt.Errorf("les deux mots de passe ne sont pas identiques")
		}
	}
	return pw, nil
}
