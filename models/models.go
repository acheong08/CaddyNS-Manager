package models

type User struct {
	Username string `json:"username" db:"username"`
	Password string `json:"password" db:"password"`
	Domain   string `json:"domain" db:"domain"`
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
	Owner         string  `json:"owner" db:"owner"`
	Destination   string  `json:"destination" db:"destination"`
	Port          int     `json:"port" db:"port"`
	DNSRecordType string  `json:"dns_record_type" db:"dns_record_type"`
	Subdomain     string  `json:"subdomain" db:"subdomain"`
	Forwarding    bool    `json:"forwarding" db:"forwarding"`
	RateLimit     int     `json:"rate_limit" db:"rate_limit"`
	LimitBy       limitBy `json:"limit_by" db:"limit_by"`
}

func (se *ServiceEntry) IsValidFOrPost() bool {
	// Check if all required fields are set
	if se.Destination == "" || se.Subdomain == "" {
		return false
	}
	if !se.Forwarding && se.DNSRecordType == "" {
		return false
	}
	if se.Forwarding && se.Port == 0 {
		return false
	}
	return true
}
