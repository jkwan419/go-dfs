package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type Volume struct {
	ID         VolumeID
	VolumeFile *os.File
	IndexFile  *os.File
	Index      map[uint64]int64
	Offset     int64
	NextID     uint64
	mu         sync.Mutex
}

func NewVolume(id VolumeID, vFile *os.File, iFile *os.File) *Volume {
	return &Volume{
		ID:         id,
		VolumeFile: vFile,
		IndexFile:  iFile,
		Index:      make(map[uint64]int64),
		Offset:     0,
		NextID:     0,
	}
}

func OpenVolume(id VolumeID, dataDir string) (*Volume, error) {
	datPath := filepath.Join(dataDir, fmt.Sprintf("%d.dat", id))
	datFile, err := os.OpenFile(datPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil { // TODO: Return + defer pattern
		return nil, fmt.Errorf("volume %d: open .dat: %w", id, err)
	}
	idxPath := filepath.Join(dataDir, fmt.Sprintf("%d.idx", id))
	idxFile, err := os.OpenFile(idxPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_APPEND, 0644)
	if err != nil {
		datFile.Close()
		return nil, fmt.Errorf("volume %d: open .idx: %w", id, err)
	}

	index := make(map[uint64]int64)
	var offset int64
	var nextID uint64

	for {
		needle, err := ReadNeedleAt(datFile, offset)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			datFile.Close()
			idxFile.Close()
			return nil, fmt.Errorf("volume %d: scan failed at offset %d: %w", id, offset, err)
		}

		index[needle.ID] = offset
		if err := WriteToFile(idxFile, needle.ID, offset); err != nil {
			datFile.Close()
			idxFile.Close()
			return nil, fmt.Errorf("volume %d: rewrite .idx at offset %d: %w", id, offset, err)
		}

		if needle.ID >= nextID {
			nextID = needle.ID + 1
		}
		offset += NeedleDiskSize(needle.Size)
	}

	return &Volume{
		ID:         id,
		VolumeFile: datFile,
		IndexFile:  idxFile,
		Index:      index,
		Offset:     offset,
		NextID:     nextID,
	}, nil
}

func (v *Volume) Read(needleID uint64) (*Needle, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	offset, ok := v.Index[needleID]
	if !ok {
		return nil, fmt.Errorf("needleID does not exist")
	}

	needle, err := ReadNeedleAt(v.VolumeFile, offset)
	if err != nil {
		return nil, err
	}

	return needle, nil
}

func (v *Volume) Write(data []byte, cookie uint32) (uint64, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	needle := &Needle{
		Cookie: cookie,
		ID:     v.NextID,
		Size:   uint32(len(data)),
		Data:   data,
	}

	_, err := v.VolumeFile.WriteAt(needle.Marshal(), v.Offset)
	if err != nil {
		return 0, err
	}

	err = WriteToFile(v.IndexFile, needle.ID, v.Offset)
	if err != nil {
		return 0, err
	}

	v.Index[needle.ID] = v.Offset
	v.Offset += NeedleDiskSize(needle.Size)
	v.NextID += 1

	return needle.ID, nil
}

func (v *Volume) HeartbeatStats() (VolumeID, uint64) {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.ID, uint64(v.Offset)
}
