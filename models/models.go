package models

import "github.com/miekg/dns"

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
	ID            int     `json:"id" db:"id"`
	// http://ip:port if forwarding
	// IP address if not forwarding
	Owner         string  `json:"owner,omitempty" db:"owner"`
	Destination   string  `json:"destination,omitempty" db:"destination"`
	Port          int     `json:"port" db:"port"`
	DNSRecordType string  `json:"dns_record_type,omitempty" db:"dns_record_type"`
	Subdomain     string  `json:"subdomain,omitempty" db:"subdomain"`
	Domain        string  `json:"domain,omitempty"`
	Forwarding    bool    `json:"forwarding" db:"forwarding"`
	RateLimit     int     `json:"rate_limit" db:"rate_limit"`
	LimitBy       limitBy `json:"limit_by" db:"limit_by"`
}

func (se *ServiceEntry) IsValidFOrPost() bool {
	// Check if all required fields are set
	if se.Destination == "" {
		return false
	}
	if !se.Forwarding && se.DNSRecordType == "" {
		return false
	}
	if se.Forwarding && se.Port == 0 {
		return false
	}
	if _, ok := dns.StringToType[se.DNSRecordType]; !ok {
		return false
	}
	return true
}
