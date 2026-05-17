package storage

import (
	"fmt"
	"sync"
)

// A VolumeServer contains one Store
type Store struct {
	Volumes map[VolumeID]*Volume
	mu      sync.Mutex
}

func NewStore() *Store {
	return &Store{
		Volumes: make(map[VolumeID]*Volume),
	}
}

func (s *Store) GetVolume(id VolumeID) (*Volume, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.Volumes[id]
	if !ok {
		return nil, fmt.Errorf("volume with id %s does not exist", id)
	}
	return v, nil
}

func (s *Store) AddVolume(id VolumeID, volume *Volume) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.Volumes[id]
	if ok {
		return fmt.Errorf("volume id %s already exists", id)
	}
	s.Volumes[id] = volume
	return nil
}

func (s *Store) Snapshot() []*Volume {
	s.mu.Lock()
	defer s.mu.Unlock()

	cpy := make([]*Volume, 0, len(s.Volumes))
	for _, v := range s.Volumes {
		cpy = append(cpy, v)
	}

	return cpy
}
