package docker

import (
	"encoding/json"
	"net/url"
	"strings"
	"sync"
)

// Stats est la consommation instantanée d'un conteneur.
type Stats struct {
	ID       string  `json:"id"`
	CPU      float64 `json:"cpu"` // % d'un cœur × nb cœurs
	MemUsage uint64  `json:"memUsage"`
	MemLimit uint64  `json:"memLimit"`
	NetRx    uint64  `json:"netRx"`
	NetTx    uint64  `json:"netTx"`
}

// Stats renvoie la consommation d'un conteneur (non streamée).
func (c *Client) Stats(id string) (Stats, error) {
	r, err := c.do("GET", "/containers/"+url.PathEscape(id)+"/stats", url.Values{"stream": {"false"}})
	if err != nil {
		return Stats{}, err
	}
	defer r.Body.Close()
	var raw struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
			OnlineCPUs  int    `json:"online_cpus"`
		} `json:"cpu_stats"`
		Pre struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage uint64            `json:"usage"`
			Limit uint64            `json:"limit"`
			Stats map[string]uint64 `json:"stats"`
		} `json:"memory_stats"`
		Networks map[string]struct {
			RxBytes uint64 `json:"rx_bytes"`
			TxBytes uint64 `json:"tx_bytes"`
		} `json:"networks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return Stats{}, err
	}
	s := Stats{ID: id, MemLimit: raw.MemoryStats.Limit}
	s.MemUsage = raw.MemoryStats.Usage
	if v, ok := raw.MemoryStats.Stats["inactive_file"]; ok && v < s.MemUsage {
		s.MemUsage -= v
	}
	cpuDelta := float64(raw.CPUStats.CPUUsage.TotalUsage) - float64(raw.Pre.CPUUsage.TotalUsage)
	sysDelta := float64(raw.CPUStats.SystemUsage) - float64(raw.Pre.SystemUsage)
	if sysDelta > 0 && cpuDelta > 0 {
		n := raw.CPUStats.OnlineCPUs
		if n == 0 {
			n = 1
		}
		s.CPU = cpuDelta / sysDelta * float64(n) * 100
	}
	for _, n := range raw.Networks {
		s.NetRx += n.RxBytes
		s.NetTx += n.TxBytes
	}
	return s, nil
}

// AllStats récupère les stats de plusieurs conteneurs en parallèle.
func (c *Client) AllStats(ids []string) []Stats {
	out := make([]Stats, len(ids))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for i, id := range ids {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			s, err := c.Stats(id)
			if err != nil {
				s = Stats{ID: id}
			}
			out[i] = s
		}(i, id)
	}
	wg.Wait()
	return out
}

// Mount est un volume ou dossier monté dans un conteneur.
type Mount struct {
	Type        string `json:"type"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	RW          bool   `json:"rw"`
	Name        string `json:"name,omitempty"`
}

// Details est le résultat simplifié d'un inspect.
type Details struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Image    string            `json:"image"`
	Created  string            `json:"created"`
	Started  string            `json:"started"`
	State    string            `json:"state"`
	Restart  string            `json:"restart"`
	Network  string            `json:"network"`
	Mounts   []Mount           `json:"mounts"`
	Ports    []string          `json:"ports"`
	Env      []string          `json:"env"`
	Labels   map[string]string `json:"labels"`
	Restarts int               `json:"restarts"`
}

// Inspect renvoie les détails d'un conteneur.
func (c *Client) Inspect(id string) (Details, error) {
	r, err := c.do("GET", "/containers/"+url.PathEscape(id)+"/json", nil)
	if err != nil {
		return Details{}, err
	}
	defer r.Body.Close()
	var raw struct {
		ID, Name, Created string
		RestartCount      int
		State             struct {
			Status    string
			StartedAt string
		}
		Config struct {
			Image  string
			Env    []string
			Labels map[string]string
		}
		HostConfig struct {
			NetworkMode   string
			RestartPolicy struct{ Name string }
			PortBindings  map[string][]struct{ HostIP, HostPort string }
		}
		Mounts []struct {
			Type, Source, Destination, Name string
			RW                              bool
		}
	}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return Details{}, err
	}
	d := Details{ID: raw.ID, Name: strings.TrimPrefix(raw.Name, "/"), Image: raw.Config.Image, Created: raw.Created,
		Started: raw.State.StartedAt, State: raw.State.Status, Restart: raw.HostConfig.RestartPolicy.Name,
		Network: raw.HostConfig.NetworkMode, Labels: raw.Config.Labels, Restarts: raw.RestartCount, Mounts: []Mount{}, Ports: []string{}}
	for _, m := range raw.Mounts {
		d.Mounts = append(d.Mounts, Mount{Type: m.Type, Source: m.Source, Destination: m.Destination, RW: m.RW, Name: m.Name})
	}
	for p, bs := range raw.HostConfig.PortBindings {
		for _, b := range bs {
			d.Ports = append(d.Ports, b.HostPort+" → "+p)
		}
	}
	// On masque les valeurs qui ressemblent à des secrets.
	for _, e := range raw.Config.Env {
		k, _, _ := strings.Cut(e, "=")
		up := strings.ToUpper(k)
		if strings.Contains(up, "PASS") || strings.Contains(up, "SECRET") || strings.Contains(up, "TOKEN") || strings.Contains(up, "KEY") {
			e = k + "=••••••"
		}
		d.Env = append(d.Env, e)
	}
	return d, nil
}

type Image struct {
	ID       string   `json:"id"`
	Tags     []string `json:"tags"`
	Size     int64    `json:"size"`
	Created  int64    `json:"created"`
	InUse    bool     `json:"inUse"`
	Dangling bool     `json:"dangling"`
}

// Images liste les images.
func (c *Client) Images() ([]Image, error) {
	r, err := c.do("GET", "/images/json", nil)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	var raw []struct {
		ID         string `json:"Id"`
		RepoTags   []string
		Size       int64
		Created    int64
		Containers int64
	}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, err
	}
	used := map[string]bool{}
	if list, err := c.List(); err == nil {
		for _, ct := range list {
			used[ct.Image] = true
		}
	}
	out := []Image{}
	for _, x := range raw {
		img := Image{ID: x.ID, Tags: x.RepoTags, Size: x.Size, Created: x.Created}
		if img.Tags == nil {
			img.Tags = []string{}
		}
		img.Dangling = len(img.Tags) == 0 || img.Tags[0] == "<none>:<none>"
		for _, t := range img.Tags {
			if used[t] || used[strings.TrimSuffix(t, ":latest")] {
				img.InUse = true
			}
		}
		if used[x.ID] {
			img.InUse = true
		}
		out = append(out, img)
	}
	return out, nil
}

// RemoveImage supprime une image.
func (c *Client) RemoveImage(id string) error {
	r, err := c.do("DELETE", "/images/"+url.PathEscape(id), nil)
	if err != nil {
		return err
	}
	r.Body.Close()
	return nil
}

// Prune supprime les images inutilisées ; renvoie l'espace libéré.
func (c *Client) Prune() (uint64, error) {
	r, err := c.do("POST", "/images/prune", url.Values{"filters": {`{"dangling":["false"]}`}})
	if err != nil {
		return 0, err
	}
	defer r.Body.Close()
	var res struct{ SpaceReclaimed uint64 }
	_ = json.NewDecoder(r.Body).Decode(&res)
	return res.SpaceReclaimed, nil
}

type Volume struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
	Created    string `json:"created"`
	InUse      bool   `json:"inUse"`
	Project    string `json:"project"`
}

// Volumes liste les volumes Docker.
func (c *Client) Volumes() ([]Volume, error) {
	r, err := c.do("GET", "/volumes", nil)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	var raw struct {
		Volumes []struct {
			Name, Driver, Mountpoint, CreatedAt string
			Labels                              map[string]string
		}
	}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, err
	}
	used := map[string]bool{}
	if list, err := c.List(); err == nil {
		for _, ct := range list {
			if d, err := c.Inspect(ct.ID); err == nil {
				for _, m := range d.Mounts {
					if m.Name != "" {
						used[m.Name] = true
					}
				}
			}
		}
	}
	out := []Volume{}
	for _, v := range raw.Volumes {
		out = append(out, Volume{Name: v.Name, Driver: v.Driver, Mountpoint: v.Mountpoint, Created: v.CreatedAt,
			InUse: used[v.Name], Project: v.Labels["com.docker.compose.project"]})
	}
	return out, nil
}

// RemoveVolume supprime un volume.
func (c *Client) RemoveVolume(name string) error {
	r, err := c.do("DELETE", "/volumes/"+url.PathEscape(name), nil)
	if err != nil {
		return err
	}
	r.Body.Close()
	return nil
}
