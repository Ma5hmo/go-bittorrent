package seedingstatus

import "sync"

type SeedingStatus struct {
	SeededBytes int64
	ActivePeers int
	mu          sync.RWMutex
}

func (s *SeedingStatus) GetSeededBytes() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SeededBytes
}

func (s *SeedingStatus) IncrementSeededBytes(bytes int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SeededBytes += bytes
}

func (s *SeedingStatus) GetActivePeers() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ActivePeers
}

func (s *SeedingStatus) IncrementActivePeers() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ActivePeers++
}

func (s *SeedingStatus) DecrementActivePeers() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ActivePeers--
}
