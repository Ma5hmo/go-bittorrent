package torrentstatus

import "sync"

type TorrentStatus struct {
	DonePieces  int
	PeersAmount int
	mu          sync.RWMutex
}

func (s *TorrentStatus) GetPeersAmount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.PeersAmount
}

func (s *TorrentStatus) GetDonePieces() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.DonePieces
}

func (s *TorrentStatus) IncrementDonePieces() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DonePieces++
}

func (s *TorrentStatus) DecrementPeersAmount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PeersAmount--
}
