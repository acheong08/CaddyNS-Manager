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
	if len(domainList) == 0 {
		return nil
	}
	// Get the root domain
	rootDomain := domainList[len(domainList)-2] + "." + domainList[len(domainList)-1]
	log.Printf("Root domain: %s\n", rootDomain)
	// Get the owner of the domain
	owner, err := s.DB.GetDomainOwner(rootDomain)
	if err != nil {
		return nil
	}
	if len(domain) < len(rootDomain)+2 {
	 		return nil
	}
	// Get the subdomain (remove root domain)
	subdomain := domain[:len(domain)-len(rootDomain)-1]
	// Get service from database
	service, err := s.DB.GetService(owner.Username, subdomain)
	if err != nil {
		return nil
	}
	if service.Forwarding {
		// If it is, add it to the cache
		s.Cache.Set(domain, s.publicIP, service.DNSRecordType)
	} else {
		// If forwarding is not enabled, directly return the destination
		s.Cache.Set(domain, service.Destination, service.DNSRecordType)
	}
	items, ok = s.Cache.Get(domain)
	if !ok {
		return nil
	}
	return items
}
