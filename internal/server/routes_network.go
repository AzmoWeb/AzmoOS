package server

import (
	"net/http"
	"strconv"

	"github.com/azmoweb/azmoos/internal/auth"
	"github.com/azmoweb/azmoos/internal/events"
	"github.com/azmoweb/azmoos/internal/firewall"
	"github.com/azmoweb/azmoos/internal/network"
)

func (s *Server) routesNetwork(handle routeFn) {
	handle("GET /api/network", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"managed": network.Available(), "interfaces": network.Interfaces()})
	})
	handle("GET /api/network/wifi", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		l, err := network.ScanWiFi()
		if err != nil {
			fail(w, 503, err)
			return
		}
		writeJSON(w, 200, l)
	})
	handle("POST /api/network/wifi/connect", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var req struct{ SSID, Password string }
		if !decode(w, r, &req) {
			return
		}
		if err := network.ConnectWiFi(req.SSID, req.Password); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Info, "network", "Connecté au Wi-Fi « "+req.SSID+" »")
		ok(w)
	})
	handle("POST /api/network/wifi/forget", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var req struct{ SSID string }
		if !decode(w, r, &req) {
			return
		}
		if err := network.Forget(req.SSID); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Info, "network", "Wi-Fi oublié : "+req.SSID)
		ok(w)
	})
	handle("POST /api/network/ip", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var c network.IPConfig
		if !decode(w, r, &c) {
			return
		}
		if err := network.Configure(c); err != nil {
			fail(w, 400, err)
			return
		}
		msg := "DHCP (automatique)"
		if c.Method == "manual" {
			msg = "IP fixe " + c.Address
		}
		s.log(r, events.Warning, "network", c.Connection+" : "+msg)
		ok(w)
	})

	handle("GET /api/firewall", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, firewall.Get()) })
	handle("POST /api/firewall/enable", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		if err := firewall.Enable(s.WebPort); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Warning, "security", "Pare-feu activé")
		ok(w)
	})
	handle("POST /api/firewall/disable", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		if err := firewall.Disable(); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Warning, "security", "Pare-feu désactivé")
		ok(w)
	})
	handle("POST /api/firewall/rules", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Action, Port, Proto, From string }
		if !decode(w, r, &req) {
			return
		}
		if err := firewall.Add(req.Action, req.Port, req.Proto, req.From); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Info, "security", "Règle pare-feu : "+req.Action+" "+req.Port+"/"+req.Proto)
		ok(w)
	})
	handle("DELETE /api/firewall/rules/{n}", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		n, err := strconv.Atoi(r.PathValue("n"))
		if err != nil {
			fail(w, 400, errString("numéro invalide"))
			return
		}
		if err := firewall.Delete(n); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Info, "security", "Règle pare-feu n°"+r.PathValue("n")+" supprimée")
		ok(w)
	})
}
