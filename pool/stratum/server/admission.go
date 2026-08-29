package server

import (
	"net"
	"sync"
)

type admission struct {
	mu       sync.Mutex
	maxTotal int
	maxPerIP int
	total    int
	byIP     map[string]int
}

func newAdmission(total, perIP int) *admission {
	return &admission{
		maxTotal: total,
		maxPerIP: perIP,
		byIP:     make(map[string]int),
	}
}

func sourceKey(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	if tcpAddr, ok := addr.(*net.TCPAddr); ok {
		if ipv4 := tcpAddr.IP.To4(); ipv4 != nil {
			return ipv4.String()
		}
		return tcpAddr.IP.String()
	}
	return addr.String()
}

func (a *admission) Acquire(key string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.total >= a.maxTotal || a.byIP[key] >= a.maxPerIP {
		return false
	}
	a.total++
	a.byIP[key]++
	return true
}

func (a *admission) Release(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	count := a.byIP[key]
	if count <= 0 {
		return
	}
	if count == 1 {
		delete(a.byIP, key)
	} else {
		a.byIP[key] = count - 1
	}
	if a.total > 0 {
		a.total--
	}
}
