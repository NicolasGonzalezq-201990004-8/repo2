package main

import (
	"log"
	"sync"
)

type SessionMap struct {
	mu       sync.RWMutex
	sessions map[string]string
}

func NewSessionMap() *SessionMap {
	return &SessionMap{
		sessions: make(map[string]string),
	}
}

func (sm *SessionMap) RegisterAffinity(clientID string, datanodeAddr string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.sessions[clientID] = datanodeAddr
	log.Printf("[Coordinador] Afinidad registrada: Cliente %s -> Datanode %s", clientID, datanodeAddr)
}

func (sm *SessionMap) CheckAffinity(clientID string) (string, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	addr, exists := sm.sessions[clientID]
	return addr, exists
}