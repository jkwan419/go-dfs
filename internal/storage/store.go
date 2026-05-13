package storage

import "fmt"

// A VolumeServer contains one Store
type Store struct {
	Volumes map[VolumeID]*Volume
}

func NewStore() *Store {
	return &Store{
		Volumes: make(map[VolumeID]*Volume),
	}
}

func (s *Store) GetVolume(id VolumeID) (*Volume, error) {
	v, ok := s.Volumes[id]
	if !ok {
		return nil, fmt.Errorf("volume with id %s does not exist", id)
	}
	return v, nil
}

func (s *Store) AddVolume(id VolumeID, volume *Volume) error {
	_, ok := s.Volumes[id]
	if ok {
		return fmt.Errorf("volume id %s already exists", id)
	}
	s.Volumes[id] = volume
	return nil
}
