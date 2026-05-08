package storage

import (
	"fmt"
	"strconv"
	"strings"
)

type VolumeId uint32

func (v VolumeId) String() string {
	return strconv.Itoa(int(v))
}

type FileId struct {
	VolumeID VolumeId
	Key      uint64
	Cookie   uint32
}

func (f FileId) String() string {
	return fmt.Sprintf("%d,%016x%08x", f.VolumeID, f.Key, f.Cookie)
}

func ParseFileID(s string) (FileId, error) {
	before, after, found := strings.Cut(s, ",")
	if !found {
		return FileId{}, fmt.Errorf("invalid file id %q: missing comma", s)
	}

	vid64, err := strconv.ParseUint(before, 10, 32)
	if err != nil {
		return FileId{}, fmt.Errorf("invalid volume id %q: %w", before, err)
	}

	if len(after) != 24 {
		return FileId{}, fmt.Errorf("invalid key+cookie length in %q", s)
	}

	key, err := strconv.ParseUint(after[:16], 16, 64)
	if err != nil {
		return FileId{}, fmt.Errorf("invalid key %q: %w", after[:16], err)
	}

	cookie, err := strconv.ParseUint(after[16:], 16, 32)
	if err != nil {
		return FileId{}, fmt.Errorf("invalid cookie %q: %w", after[16:], err)
	}

	return FileId{VolumeID: VolumeId(vid64), Key: key, Cookie: uint32(cookie)}, nil
}
