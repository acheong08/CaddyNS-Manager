package database

import (
	"sync"
	"time"
)

type dnsCacheItem struct {
	Domain      string
	Dest        string
	RecordType  string
	LastUpdated time.Time
}

type dnsCacheList struct {
	Hits  int
	Items []*dnsCacheItem
}

func (l *dnsCacheList) Add(item dnsCacheItem) {
	if l.Items == nil {
		l.Items = make([]*dnsCacheItem, 0)
	}
	l.Items = append(l.Items, &item)
}

func (l *dnsCacheList) Remove(index int) {
	l.Items = append(l.Items[:index], l.Items[index+1:]...)
}

func (l *dnsCacheList) Clear() {
	l.Items = make([]*dnsCacheItem, 0)
}

type dnsCache struct {
	lock  sync.RWMutex
	Items map[string]*dnsCacheList
}

func newCache() *dnsCache {
	return &dnsCache{sync.RWMutex{}, make(map[string]*dnsCacheList, 0)}
}

func (c *dnsCache) Set(domain string, dest string, recordType string) {
	c.lock.Lock()
	if _, ok := c.Items[domain]; !ok {
		c.Items[domain] = &dnsCacheList{0, make([]*dnsCacheItem, 0)}
	}
	c.Items[domain].Add(dnsCacheItem{domain, dest, recordType, time.Now()})
	c.Items[domain].Hits++
	c.lock.Unlock()
}

func (c *dnsCache) Get(domain string) ([]*dnsCacheItem, bool) {
	c.lock.RLock()
	item, ok := c.Items[domain]
	if !ok {
		c.lock.RUnlock()
		return nil, false
	}
	item.Hits++
	c.lock.RUnlock()
	return item.Items, true
}

func (c *dnsCache) Delete(domain string) {
	c.lock.Lock()
	delete(c.Items, domain)
	c.lock.Unlock()
}

func (c *dnsCache) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.Items = make(map[string]*dnsCacheList)
}