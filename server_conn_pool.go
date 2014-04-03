package main

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

func (p *ServerConnPool) Get(host, port, auth string, timeout time.Duration) *ServerConn {
	key := poolKey(host, port, auth)

	// Ensure we don't create conflicting server connections
	p.lock(key)
	defer p.unlock(key)

	serverConn := p.pool[key]
	if serverConn == nil {
		serverConn = NewServerConn(host, port, auth, timeout)
		p.pool[key] = serverConn
	}

	return serverConn
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

func poolKey(host, port, auth string) string {
	return fmt.Sprintf("%s:%s:%s", host, port, auth)
}
