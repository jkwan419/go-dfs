package storage

import (
	"fmt"
	"strconv"
	"strings"
)

type VolumeID uint32

func (v VolumeID) String() string {
	return strconv.Itoa(int(v))
}

type FileID struct {
	VolumeID VolumeID
	Key      uint64
	Cookie   uint32
}

func (f FileID) String() string {
	return fmt.Sprintf("%d,%016x%08x", f.VolumeID, f.Key, f.Cookie)
}

func ParseFileID(s string) (FileID, error) {
	before, after, found := strings.Cut(s, ",")
	if !found {
		return FileID{}, fmt.Errorf("invalid file id %q: missing comma", s)
	}

	vid64, err := strconv.ParseUint(before, 10, 32)
	if err != nil {
		return FileID{}, fmt.Errorf("invalid volume id %q: %w", before, err)
	}

	if len(after) != 24 {
		return FileID{}, fmt.Errorf("invalid key+cookie length in %q", s)
	}

	key, err := strconv.ParseUint(after[:16], 16, 64)
	if err != nil {
		return FileID{}, fmt.Errorf("invalid key %q: %w", after[:16], err)
	}

	cookie, err := strconv.ParseUint(after[16:], 16, 32)
	if err != nil {
		return FileID{}, fmt.Errorf("invalid cookie %q: %w", after[16:], err)
	}

	return FileID{VolumeID: VolumeID(vid64), Key: key, Cookie: uint32(cookie)}, nil
}

func ParseVolumeID(s string) (VolumeID, error) {
	value, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}

	return VolumeID(value), nil
}
