package main

import (
	cspb "lab_3/proto/consensus/cspb"
	"sync"
)

// almacena las decisiones.
type PersistentLog struct {
	mu      sync.Mutex
	Entries []*cspb.Decision
}

// NewPersistentLog crea e inicializa el log.
func NewPersistentLog() *PersistentLog {
	return &PersistentLog{
		Entries: make([]*cspb.Decision, 0),
	}
}

// Append agrega una nueva decisión al log.
func (l *PersistentLog) Append(entry *cspb.Decision) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.Entries = append(l.Entries, entry)
}

// GetLastIndex devuelve el índice de la última entrada.
func (l *PersistentLog) GetLastIndex() int32 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return int32(len(l.Entries)) - 1
}

// GetLastTerm devuelve el término de la última entrada.
func (l *PersistentLog) GetLastTerm() int32 {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.Entries) == 0 {
		return 0 // Término inicial
	}
	return l.Entries[len(l.Entries)-1].PrevTerm
}