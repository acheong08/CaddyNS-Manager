package database

import (
	"strings"

	"github.com/acheong08/nameserver/models"
)

type Storage struct {
	cache *dnsCache
	db    *database
	publicIP string
}

func NewStorage(publicIP string) (*Storage, error) {
	db, err := newDatabase()
	if err != nil {
		return nil, err
	}
	return &Storage{
		cache: newCache(),
		db:    db,
		publicIP: publicIP,
	}, nil
}

func (s *Storage) GetDNS(domain string) []*dnsCacheItem {
	// Check if the domain is in the cache
	items, ok := s.cache.Get(domain)
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
	// Get the owner of the domain
	owner, err := s.db.GetDomainOwner(rootDomain)
	if err != nil {
		return nil
	}
	// Get service from database
	service, err := s.db.GetService(owner.Username, domain)
	if err != nil {
		return nil
	}
	if service.Forwarding {
		// If it is, add it to the cache
			s.cache.Set(domain, s.publicIP, service.DNSRecordType)
	} else {
	// If forwarding is not enabled, directly return the destination
		s.cache.Set(domain, service.Destination, service.DNSRecordType)
	}
	items, ok = s.cache.Get(domain)
	if !ok {
		return nil
	}
	return items
}

func (s *Storage) ClearCache() {
	s.cache.Clear(100)
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) NewUser(user models.User) error {
	return s.db.NewUser(user)
}

func (s *Storage) GetUser(username string) (models.User, error) {
	return s.db.GetUser(username)
}

func (s *Storage) UserLogin(username, password string) error {
	return s.db.UserLogin(username, password)
}

func (s *Storage) NewService(service models.ServiceEntry) error {
	return s.db.NewService(service)
}

func (s *Storage) GetService(owner, subdomain string) (models.ServiceEntry, error) {
	return s.db.GetService(owner, subdomain)
}

func (s *Storage) DeleteService(owner, subdomain string) error {
	return s.db.DeleteService(owner, subdomain)
}

func (s *Storage) GetServices(owner string) ([]models.ServiceEntry, error) {
	return s.db.GetServices(owner)
}

func (s *Storage) UpdateService(service models.ServiceEntry) error {
	return s.db.UpdateService(service)
}
