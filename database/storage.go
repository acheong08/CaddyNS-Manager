package database

import (
	"log"
	"strings"
)

type Storage struct {
	Cache    *dnsCache
	DB       *database
	publicIP string
}

func NewStorage(publicIP string) (*Storage, error) {
	db, err := newDatabase()
	if err != nil {
		return nil, err
	}
	return &Storage{
		Cache:    newCache(),
		DB:       db,
		publicIP: publicIP,
	}, nil
}

func (s *Storage) GetDNS(domain string) []*dnsCacheItem {
	if domain[len(domain)-1] == '.' {
		domain = domain[:len(domain)-1]
	}
	// Check if the domain is in the cache
	items, ok := s.Cache.Get(domain)
	if ok {
		return items
	}
	// Split the domain to find root domain
	domainList := strings.Split(domain, ".")
	// Prevent index out of range
	if len(domainList) < 2 {
		return nil
	}
	// Get the root domain
	rootDomain := domainList[len(domainList)-2] + "." + domainList[len(domainList)-1]
	// Get the owner of the domain
	owner, err := s.DB.GetDomainOwner(rootDomain)
	if err != nil {
		return nil
	}
	// Get the subdomain (remove root domain)
	var subdomain string
	if len(domain) == len(rootDomain) {
		subdomain = ""
	} else {
		subdomain = domain[:len(domain)-len(rootDomain)-1]
	}
	log.Println("subdomain", subdomain)
	// Get services from database
	services, err := s.DB.GetService(owner.Username, subdomain)
	if err != nil {
		return nil
	}
	for _, service := range services {
	if service.Forwarding {
		// If it is, add it to the cache
		s.Cache.Set(domain, s.publicIP, service.DNSRecordType)
	} else {
		// If forwarding is not enabled, directly return the destination
		s.Cache.Set(domain, service.Destination, service.DNSRecordType)
	}
	}
	items, ok = s.Cache.Get(domain)
	if !ok {
		return nil
	}
	return items
}
