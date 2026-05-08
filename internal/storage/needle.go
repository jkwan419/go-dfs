package storage

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
)

var magicBytes = []byte{0x08, 0x05, 0x18, 0x86}

const (
	NeedleHeaderSize  = 20
	NeedlePaddingSize = 8
	NeedleChecksum    = 4
)

type Needle struct {
	Cookie uint32
	ID     uint64
	Size   uint32
	Data   []byte
}

func NeedleDiskSize(dataSize uint32) int64 {
	total := NeedleHeaderSize + dataSize + NeedleChecksum
	if total%NeedlePaddingSize != 0 {
		total += NeedlePaddingSize - (total % NeedlePaddingSize)
	}

	return int64(total)
}

func (n *Needle) Marshal() []byte {
	buf := make([]byte, NeedleDiskSize(n.Size))

	// Magic bytes
	copy(buf[0:4], magicBytes)

	// Cookie
	binary.BigEndian.PutUint32(buf[4:8], n.Cookie)

	// ID
	binary.BigEndian.PutUint64(buf[8:16], n.ID)

	// Size
	binary.BigEndian.PutUint32(buf[16:20], n.Size)

	// Data
	copy(buf[NeedleHeaderSize:NeedleHeaderSize+int(n.Size)], n.Data)

	// Checksum
	checksum := crc32.ChecksumIEEE(n.Data)
	binary.BigEndian.PutUint32(buf[NeedleHeaderSize+int(n.Size):NeedleHeaderSize+int(n.Size)+NeedleChecksum], checksum)

	return buf
}

func ReadNeedleAt(r io.ReaderAt, offset int64) (*Needle, error) {
	header := make([]byte, NeedleHeaderSize)
	_, err := r.ReadAt(header, offset)
	if err != nil {
		return nil, err
	}

	// Check magic bytes
	if !bytes.Equal(header[0:4], magicBytes) {
		return nil, fmt.Errorf("incorrect offset")
	}

	size := binary.BigEndian.Uint32(header[16:20])
	data := make([]byte, int(size)+NeedleChecksum)
	_, err = r.ReadAt(data, offset+NeedleHeaderSize)
	if err != nil {
		return nil, err
	}

	// Check data checksum
	computedChecksum := crc32.ChecksumIEEE(data[:len(data)-4])
	storedChecksum := binary.BigEndian.Uint32(data[len(data)-4:])
	if computedChecksum != storedChecksum {
		return nil, fmt.Errorf("corrupted data")
	}

	n := &Needle{
		Cookie: binary.BigEndian.Uint32(header[4:8]),
		ID:     binary.BigEndian.Uint64(header[8:16]),
		Size:   binary.BigEndian.Uint32(header[16:20]),
		Data:   data[:len(data)-4],
	}

	return n, nil
}
