package storage

import (
	"fmt"
	"os"
)

type Volume struct {
	ID         uint64
	VolumeFile *os.File
	IndexFile  *os.File
	Index      map[uint64]int64
	Offset     int64
	NextID     uint64
}

func NewVolume(id uint64, vFile *os.File, iFile *os.File) *Volume {
	return &Volume{
		ID:         id,
		VolumeFile: vFile,
		IndexFile:  iFile,
		Index:      make(map[uint64]int64),
		Offset:     0,
		NextID:     0,
	}
}

func (v *Volume) Read(needleID uint64) (*Needle, error) {
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

func (v *Volume) Write(needle *Needle) error {
	_, err := v.VolumeFile.WriteAt(needle.Marshal(), v.Offset)
	if err != nil {
		return err
	}

	err = WriteToFile(v.IndexFile, needle.ID, v.Offset)
	if err != nil {
		return err
	}

	v.Index[needle.ID] = v.Offset
	v.Offset += NeedleDiskSize(needle.Size)
	v.NextID += 1

	return nil
}
