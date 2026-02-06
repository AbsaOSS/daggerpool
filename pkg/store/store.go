package store

import (
	"fmt"
	"strings"
	"sync"

	"github.com/absaoss/daggerpool/pkg/workerpool"
)

// Store thread safe key value store
type Store interface {
	Set(key string, value any)
	Get(key string) (any, bool)
	SetJobSnapshot(workerpool.Job, int)
	ClearJobSnapshot(workerpool.Job)
	GetSnapshotInProgress() map[string]*Snapshot
}

type DAGResultStore struct {
	mx    sync.RWMutex
	store map[string]any
}

type Snapshot struct {
	WorkerID int
	Job      workerpool.Job
}

func NewDAGResultStore() *DAGResultStore {
	return &DAGResultStore{store: make(map[string]any)}
}

func (s *DAGResultStore) Set(key string, value any) {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.store[key] = value
}

func (s *DAGResultStore) Get(key string) (any, bool) {
	s.mx.RLock()
	defer s.mx.RUnlock()
	value, ok := s.store[key]
	return value, ok
}

func (s *DAGResultStore) SetJobSnapshot(job workerpool.Job, id int) {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.store[s.snapshotKey(job)] = &Snapshot{Job: job, WorkerID: id}
}

func (s *DAGResultStore) ClearJobSnapshot(job workerpool.Job) {
	s.mx.Lock()
	defer s.mx.Unlock()
	delete(s.store, s.snapshotKey(job))
}

func (s *DAGResultStore) GetSnapshotInProgress() map[string]*Snapshot {
	s.mx.Lock()
	defer s.mx.Unlock()
	m := make(map[string]*Snapshot, 0)
	for key := range s.store {
		if strings.HasPrefix(key, "snapshot_") {
			m[key] = s.store[key].(*Snapshot)
		}
	}
	return m
}

func (s *DAGResultStore) snapshotKey(job workerpool.Job) string {
	return fmt.Sprintf("snapshot_%s", job)
}
