package config

import (
	"testing"
	"time"
)

// Une modification faite par un autre processus (CLI) doit être vue par le service.
func TestExternalChangeIsReloaded(t *testing.T) {
	dir := t.TempDir()
	service, _ := Load(dir)
	_ = service.Update(func(c *Config) error { c.Settings["a"] = "1"; return nil })

	time.Sleep(20 * time.Millisecond)
	cli, _ := Load(dir)
	_ = cli.Update(func(c *Config) error { c.Users = append(c.Users, &User{Username: "azmo", Role: RoleAdmin}); return nil })

	var found bool
	service.View(func(c *Config) { found = c.FindUser("azmo") != nil })
	if !found {
		t.Fatal("le compte créé en ligne de commande n'est pas visible par le service")
	}
	// Et une écriture du service ne doit pas l'effacer.
	_ = service.Update(func(c *Config) error { c.Settings["b"] = "2"; return nil })
	again, _ := Load(dir)
	again.View(func(c *Config) { found = c.FindUser("azmo") != nil && c.Settings["b"] == "2" })
	if !found {
		t.Fatal("l'écriture du service a écrasé la modification externe")
	}
}
