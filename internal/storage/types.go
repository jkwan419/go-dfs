// package storage
//
// import (
// 	"encoding/hex"
// 	"fmt"
// 	"strconv"
// 	"strings"
// )
//
// type VolumeId uint32
//
// func NewVolumeId(vid string) (VolumeId, error) {
// 	volumeId, err := strconv.ParseUint(vid, 10, 64)
// 	return VolumeId(volumeId), err
// }
//
// func (vid VolumeId) String() string {
// 	return strconv.FormatUint(uint64(vid), 10)
// }
//
// type FileId struct {
// 	VolumeId VolumeId
// 	Key      uint64
// 	Cookie   uint32
// }
//
// func NewFileId(VolumeId VolumeId, key uint64, cookie uint32) *FileId {
// 	return &FileId{VolumeId: VolumeId, Key: key, Cookie: cookie}
// }
//
// func (fid *FileId) String() string {
// 	return fid.VolumeId.String()
// }
//
// func ParseFileId(fid string) (*FileId, error) {
// 	vid, cookie, err := splitFileId(fid)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	volumeId, err := NewVolumeId(vid)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	nid, cookie, err := ParseNeedleId(cookie)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	fileId := &FileId{VolumeId: volumeId, Key: nid, Cookie: cookie}
// 	return fileId, nil
// }
//
// func formatNeedleIdCookie(key NeedleId, cookie Cookie) string {
// 	bytes := make([]byte, NeedleIdSize+CookieSize)
// 	NeedleIdToBytes(bytes[0:NeedleIdSize], key)
// 	CookieToBytes(bytes[NeedleIdSize:NeedleIdSize+CookieSize], cookie)
// 	nonzero_idx := 0
// 	for ; bytes[nonzero_idx] == 0 && nonzero_idx < NeedleIdSize; nonzero_idx++ {
// 	}
// 	return hex.EncodeToString(bytes[nonzero_idx:])
// }
//
// func splitFileId(fileId string) (string, string, error) {
// 	commaIndex := strings.Index(fileId, ",")
// 	if commaIndex <= 0 {
// 		return "", "", fmt.Errorf("wrong fileid format")
// 	}
// 	return fileId[:commaIndex], fileId[commaIndex+1:], nil
// }

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
	VolumeId VolumeId
	Key      uint64
	Cookie   uint32
}

func (f FileId) String() string {
	return fmt.Sprintf("%d,%016x%08x", f.VolumeId, f.Key, f.Cookie)
}

func ParseFileId(s string) (FileId, error) {
	comma := strings.IndexByte(s, ',')
	if comma < 0 {
		return FileId{}, fmt.Errorf("invalid file id %q: missing comma", s)
	}
	vid64, err := strconv.ParseUint(s[:comma], 10, 32)
	if err != nil {
		return FileId{}, fmt.Errorf("invalid volume id %q: %w", s[:comma], err)
	}
	rest := s[comma+1:]
	if len(rest) != 24 {
		return FileId{}, fmt.Errorf("invalid key+cookie length in %q", s)
	}
	key, err := strconv.ParseUint(rest[:16], 16, 64)
	if err != nil {
		return FileId{}, fmt.Errorf("invalid key %q: %w", rest[:16], err)
	}
	cookie, err := strconv.ParseUint(rest[16:], 16, 32)
	if err != nil {
		return FileId{}, fmt.Errorf("invalid cookie %q: %w", rest[16:], err)
	}
	return FileId{VolumeId: VolumeId(vid64), Key: key, Cookie: uint32(cookie)}, nil
}
