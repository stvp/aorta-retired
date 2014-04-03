package redis

import (
	"fmt"
	"sync"
	"time"
)

type ServerConnPool struct {
	pool         map[string]*ServerConn
	mutexes      map[string]*sync.Mutex
	mutexesMutex sync.Mutex
}

func NewServerConnPool() *ServerConnPool {
	return &ServerConnPool{
		pool:         map[string]*ServerConn{},
		mutexes:      map[string]*sync.Mutex{},
		mutexesMutex: sync.Mutex{},
	}
}

func (p *ServerConnPool) Get(address, auth string, timeout time.Duration) *ServerConn {
	key := poolKey(address, auth)

	// Ensure we don't create conflicting server connections
	p.lock(key)
	defer p.unlock(key)

	serverConn := p.pool[key]
	if serverConn == nil {
		serverConn = NewServerConn(address, auth, timeout)
		p.pool[key] = serverConn
	}

	return serverConn
}

func (p *ServerConnPool) Expire(limit time.Time) int {
	expired := 0
	for key, conn := range p.pool {
		if conn.LastUsed.Before(limit) {
			p.lock(key)
			delete(p.pool, key)
			delete(p.mutexes, key)
			conn.Close()
			expired++
		}
	}
	return expired
}

func (p *ServerConnPool) lock(key string) {
	p.mutexesMutex.Lock()
	mutex := p.mutexes[key]
	if mutex == nil {
		mutex = &sync.Mutex{}
		p.mutexes[key] = mutex
	}
	p.mutexesMutex.Unlock()
	mutex.Lock()
}

func (p *ServerConnPool) unlock(key string) {
	mutex := p.mutexes[key]
	mutex.Unlock()
}

func poolKey(address, auth string) string {
	return fmt.Sprintf("%s:%s", address, auth)
}
