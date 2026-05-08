package storage

import (
	"fmt"
	"os"
)

type Volume struct {
	ID     uint64
	File   *os.File
	Index  map[uint64]int64
	Offset int64
}

func NewVolume(id uint64, file *os.File, index map[uint64]int64, offset int64) *Volume {
	return &Volume{
		ID:     id,
		File:   file,
		Index:  index,
		Offset: offset,
	}
}

func (v *Volume) Read(needleID uint64) (*Needle, error) {
	key, ok := v.Index[needleID]
	if !ok {
		return nil, fmt.Errorf("needleID does not exist")
	}

	needle, err := ReadNeedleAt(v.File, key)
	if err != nil {
		return nil, err
	}

	return needle, nil
}

func (v *Volume) Write(needle Needle) error {
	_, err := v.File.WriteAt(needle.Marshal(), v.Offset)
	if err != nil {
		return err
	}

	v.Index[needle.ID] = v.Offset
	v.Offset += NeedleDiskSize(needle.Size)

	return nil
}
