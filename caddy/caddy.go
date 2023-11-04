package caddy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Define Go structs
type Config struct {
	Handle   []Handle `json:"handle"`
	Match    []Match  `json:"match"`
	Terminal bool     `json:"terminal"`
}

type Handle struct {
	Handler   string     `json:"handler"`
	Routes    []Route    `json:"routes,omitempty"`
	Upstreams []Upstream `json:"upstreams,omitempty"`
}

type Route struct {
	Handle []Handle `json:"handle"`
}

type Match struct {
	Host []string `json:"host"`
}

type Upstream struct {
	Dial string `json:"dial"`
}

func NewConfig(host, upstream string) Config {
	return Config{
		Handle: []Handle{
			{
				Handler: "subroute",
				Routes: []Route{
					{
						Handle: []Handle{
							{
								Handler: "reverse_proxy",
								Upstreams: []Upstream{
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
		Match: []Match{
			{
				Host: []string{host},
			},
		},
		Terminal: true,
	}
}

func (c Config) JSON() []byte {
	b, _ := json.Marshal(c)
	return b
}

func ResetConfig()

func AddConfig(config Config) error {
	url := "http://127.0.0.1:2019/config/apps/http/servers/srv0/routes"

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(config.JSON()))
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
