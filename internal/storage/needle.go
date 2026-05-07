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
	SuperBlockSize    = 8
)

type Needle struct {
	Cookie uint32
	Id     uint64
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
	out := make([]byte, NeedleDiskSize(n.Size))

	// Magic bytes
	copy(out[0:4], magicBytes)

	// Cookie
	binary.BigEndian.PutUint32(out[4:8], n.Cookie)

	// ID
	binary.BigEndian.PutUint64(out[8:16], n.Id)

	// Size
	binary.BigEndian.PutUint32(out[16:20], n.Size)

	// Data
	copy(out[NeedleHeaderSize:NeedleHeaderSize+int(n.Size)], n.Data)

	// Checksum
	checksum := crc32.ChecksumIEEE(n.Data)
	binary.BigEndian.PutUint32(out[NeedleHeaderSize+int(n.Size):NeedleHeaderSize+int(n.Size)+NeedleChecksum], checksum)

	return out
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
	checksum := crc32.ChecksumIEEE(data[:len(data)-4])
	cs := binary.BigEndian.Uint32(data[len(data)-4:])
	if checksum != cs {
		return nil, fmt.Errorf("corrupted data")
	}

	in := &Needle{
		Cookie: binary.BigEndian.Uint32(header[4:8]),
		Id:     binary.BigEndian.Uint64(header[8:16]),
		Size:   binary.BigEndian.Uint32(header[16:20]),
		Data:   data[:len(data)-4],
	}

	return in, nil
}
