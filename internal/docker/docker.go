// Package docker parle au démon Docker via son socket Unix (sans dépendance).
package docker

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	hc *http.Client
}

func New(socket string) *Client {
	tr := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "unix", socket)
	}}
	return &Client{hc: &http.Client{Transport: tr, Timeout: 60 * time.Second}}
}

func (c *Client) do(method, path string, q url.Values) (*http.Response, error) {
	u := "http://docker/v1.41" + path
	if q != nil {
		u += "?" + q.Encode()
	}
	req, _ := http.NewRequest(method, u, nil)
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docker indisponible: %w", err)
	}
	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		var e struct{ Message string }
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Message == "" {
			e.Message = resp.Status
		}
		return nil, fmt.Errorf("docker: %s", e.Message)
	}
	return resp, nil
}

type Port struct {
	PrivatePort int    `json:"privatePort"`
	PublicPort  int    `json:"publicPort"`
	Type        string `json:"type"`
}

type Container struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	State   string            `json:"state"`
	Status  string            `json:"status"`
	Created int64             `json:"created"`
	Ports   []Port            `json:"ports"`
	Labels  map[string]string `json:"labels"`
	Project string            `json:"project"`
}

// Available indique si Docker répond.
func (c *Client) Available() bool {
	r, err := c.do("GET", "/_ping", nil)
	if err != nil {
		return false
	}
	r.Body.Close()
	return true
}

// Info renvoie les infos Docker brutes.
func (c *Client) Info() (map[string]any, error) {
	r, err := c.do("GET", "/info", nil)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	var m map[string]any
	return m, json.NewDecoder(r.Body).Decode(&m)
}

// List liste tous les conteneurs.
func (c *Client) List() ([]Container, error) {
	r, err := c.do("GET", "/containers/json", url.Values{"all": {"1"}})
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	var raw []struct {
		ID      string `json:"Id"`
		Names   []string
		Image   string
		State   string
		Status  string
		Created int64
		Ports   []struct {
			PrivatePort int
			PublicPort  int
			Type        string
		}
		Labels map[string]string
	}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]Container, 0, len(raw))
	for _, x := range raw {
		ct := Container{ID: x.ID, Image: x.Image, State: x.State, Status: x.Status, Created: x.Created, Labels: x.Labels,
			Project: x.Labels["com.docker.compose.project"]}
		if len(x.Names) > 0 {
			ct.Name = strings.TrimPrefix(x.Names[0], "/")
		}
		seen := map[int]bool{}
		for _, p := range x.Ports {
			if p.PublicPort == 0 || seen[p.PublicPort] {
				continue
			}
			seen[p.PublicPort] = true
			ct.Ports = append(ct.Ports, Port{p.PrivatePort, p.PublicPort, p.Type})
		}
		out = append(out, ct)
	}
	return out, nil
}

// Action exécute start/stop/restart/remove.
func (c *Client) Action(id, action string) error {
	var r *http.Response
	var err error
	switch action {
	case "start", "stop", "restart":
		r, err = c.do("POST", "/containers/"+url.PathEscape(id)+"/"+action, nil)
	case "remove":
		r, err = c.do("DELETE", "/containers/"+url.PathEscape(id), url.Values{"force": {"1"}})
	default:
		return fmt.Errorf("action inconnue: %s", action)
	}
	if err != nil {
		return err
	}
	r.Body.Close()
	return nil
}

// Logs renvoie les dernières lignes de log.
func (c *Client) Logs(id string, tail int) (string, error) {
	r, err := c.do("GET", "/containers/"+url.PathEscape(id)+"/logs",
		url.Values{"stdout": {"1"}, "stderr": {"1"}, "timestamps": {"1"}, "tail": {fmt.Sprint(tail)}})
	if err != nil {
		return "", err
	}
	defer r.Body.Close()
	b, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		return "", err
	}
	return demux(b), nil
}

// demux retire l'en-tête de multiplexage stdout/stderr de Docker (si présent).
func demux(b []byte) string {
	var sb strings.Builder
	for len(b) >= 8 && (b[0] == 1 || b[0] == 2) && b[1] == 0 && b[2] == 0 && b[3] == 0 {
		n := int(binary.BigEndian.Uint32(b[4:8]))
		if 8+n > len(b) {
			n = len(b) - 8
		}
		sb.Write(b[8 : 8+n])
		b = b[8+n:]
	}
	sb.Write(b)
	return sb.String()
}
