package caddy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Define Go structs
type Config struct {
	Handle   []handle `json:"handle"`
	Match    []match  `json:"match"`
	Terminal bool     `json:"terminal"`
}

type handle struct {
	Handler   string      `json:"handler"`
	Routes    []route     `json:"routes,omitempty"`
	Upstreams []upstreamd `json:"upstreams,omitempty"`
	Transport *transport  `json:"transport,omitempty"`
}

type transport struct {
	Protocol string            `json:"protocol"`
	TLS      map[string]string `json:"tls"`
}

type route struct {
	Handle []handle `json:"handle"`
}

type match struct {
	Host []string `json:"host"`
}

type upstreamd struct {
	Dial string `json:"dial"`
}

func NewConfig(host, upstream string) Config {
	config := Config{
		Handle: []handle{
			{
				Handler: "subroute",
				Routes: []route{
					{
						Handle: []handle{
							{
								Handler: "reverse_proxy",
								Upstreams: []upstreamd{
									{
										Dial: upstream,
									},
								},
							},
						},
					},
				},
			},
		},
		Match: []match{
			{
				Host: []string{host},
			},
		},
		Terminal: true,
	}
	// Upstream must be longer than 8 characters to check for https://
	// Check if upstream ends with 443 or starts with https://
	if strings.HasSuffix(upstream, ":443") || strings.HasPrefix(upstream, "https://") {
		config.Handle[0].Routes[0].Handle[0].Transport = &transport{
			Protocol: "http",
			TLS:      make(map[string]string),
		}
	}
	if strings.HasPrefix(upstream, "https://") {
		// This should remove https:// from upstream
		config.Handle[0].Upstreams[0].Dial = strings.Replace(upstream, "https://", "", 1)
	}
	return config
}

func (c Config) JSON() []byte {
	b, _ := json.Marshal(c)
	return b
}

func ResetConfig(configs []Config) error {
	url := "http://127.0.0.01:2019/config/apps/http/servers/srv0/routes"
	body, _ := json.Marshal(configs)
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Caddy returned status code %d", resp.StatusCode)
	}
	return nil

}

func AddConfig(config Config) error {
	url := "http://127.0.0.1:2019/config/apps/http/servers/srv0/routes"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(config.JSON()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Caddy returned status code %d", resp.StatusCode)
	}
	return nil
}

func Update(config Config) error {
	updated := false
	url := "http://127.0.0.1:2019/config/apps/http/servers/srv0/routes"

	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var routes []Config
	err = json.NewDecoder(resp.Body).Decode(&routes)
	if err != nil {
		return err
	}

	for idx, route := range routes {
		for _, match := range route.Match {
			for _, host := range match.Host {
				if host == config.Match[0].Host[0] {
					url := fmt.Sprintf("http://127.0.0.1:2019/config/apps/http/servers/srv0/routes/%d", idx)
					req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(config.JSON()))
					if err != nil {
						return err
					}
					req.Header.Set("Content-Type", "application/json")

					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						return err
					}
					defer resp.Body.Close()
					if resp.StatusCode != 200 {
						return fmt.Errorf("Caddy returned status code %d", resp.StatusCode)
					}
					updated = true
				}
			}
		}
	}
	if !updated {
		return AddConfig(config)
	}

	return nil
}

func RemoveHost(domain string) error {
	url := "http://127.0.0.1:2019/config/apps/http/servers/srv0/routes"

	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var routes []Config
	err = json.NewDecoder(resp.Body).Decode(&routes)
	if err != nil {
		return err
	}

	for idx, route := range routes {
		for _, match := range route.Match {
			for _, host := range match.Host {
				if host == domain {
					url := fmt.Sprintf("http://127.0.0.1:2019/config/apps/http/servers/srv0/routes/%d", idx)
					req, err := http.NewRequest("DELETE", url, nil)
					if err != nil {
						return err
					}
					req.Header.Set("Content-Type", "application/json")

					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						return err
					}
					defer resp.Body.Close()
					if resp.StatusCode != 200 {
						return fmt.Errorf("Caddy returned status code %d", resp.StatusCode)
					}
				}
			}
		}
	}

	return nil
}
