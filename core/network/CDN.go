package network

import (
	"net"
	"sync"
)

type CDNInfo struct {
	mu      sync.RWMutex
	IsCDN   bool
	CDNIPs  []net.IP
	RealIPs []net.IP
}

func NewCDNInfo() *CDNInfo {
	return &CDNInfo{}
}

func (c *CDNInfo) MarkAsCDN() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.IsCDN = true
}

func (c *CDNInfo) AddCDNIP(ip net.IP) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CDNIPs = append(c.CDNIPs, ip)
}

func (c *CDNInfo) AddRealIP(ip net.IP) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.RealIPs = append(c.RealIPs, ip)
}

func (c *CDNInfo) GetSnapshot() (bool, []net.IP, []net.IP) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cdnIPsCopy := append([]net.IP(nil), c.CDNIPs...)
	realIPsCopy := append([]net.IP(nil), c.RealIPs...)
	return c.IsCDN, cdnIPsCopy, realIPsCopy
}
