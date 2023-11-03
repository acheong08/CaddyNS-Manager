package models

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Domain   string `json:"domain"`
	// Comma separated list of Caddy instances to call
	CaddyInstances string `json:"caddy_instances"`
}

type limitBy int

const (
	LimitBySecond limitBy = iota
	LimitByMinute
	LimitByHour
)

type ServiceEntry struct {
	// http://ip:port if forwarding
	// IP address if not forwarding
	Owner         string  `json:"owner"`
	Destination   string  `json:"destination"`
	DNSRecordType string  `json:"dns_record_type"`
	Subdomain     string  `json:"subdomain"`
	Forwarding    bool    `json:"forwarding"`
	RateLimit     int     `json:"rate_limit"`
	LimitBy       limitBy `json:"limit_by"`
}
