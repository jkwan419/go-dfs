package storage

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"io"
	"testing"
)

func TestNeedleDiskSize(t *testing.T) {
	tests := []struct {
		name     string
		input    uint32
		expected int64
	}{
		{
			name:     "no padding",
			input:    8,
			expected: 32,
		},
		{
			name:     "padding",
			input:    9,
			expected: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NeedleDiskSize(tt.input)
			if tt.expected != got {
				t.Errorf("got %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestNeedleMarshal(t *testing.T) {
	needle := &Needle{
		Cookie: 0xDEADBEEF,
		ID:     0x0000000000000001,
		Data:   []byte("test"),
		Size:   4,
	}

	buf := needle.Marshal()
	if !bytes.Equal(buf[0:4], magicBytes) {
		t.Errorf("incorrect magic bytes")
	}

	if binary.BigEndian.Uint32(buf[4:8]) != needle.Cookie {
		t.Errorf("incorrect cookie")
	}

	if binary.BigEndian.Uint64(buf[8:16]) != needle.ID {
		t.Errorf("incorrect id")
	}

	if binary.BigEndian.Uint32(buf[16:20]) != needle.Size {
		t.Errorf("incorrect size")
	}

	if !bytes.Equal(buf[NeedleHeaderSize:NeedleHeaderSize+int(needle.Size)], needle.Data) {
		t.Errorf("incorrect data")
	}

	testChecksum := crc32.ChecksumIEEE(needle.Data)
	if binary.BigEndian.Uint32(buf[NeedleHeaderSize+int(needle.Size):NeedleHeaderSize+int(needle.Size)+NeedleChecksum]) != testChecksum {
		t.Errorf("incorrect checksum")
	}
}

func TestNeedleReadNeedleAt(t *testing.T) {
	needle := &Needle{
		Cookie: 0xDEADBEEF,
		ID:     0x0000000000000001,
		Data:   []byte("test"),
		Size:   4,
	}

	validBuf := needle.Marshal()

	badMagicBuf := bytes.Clone(validBuf)
	badMagicBuf[0] = 0x0

	badChecksumBuf := bytes.Clone(validBuf)
	badChecksumBuf[20] ^= 0xFF

	tests := []struct {
		name        string
		reader      io.ReaderAt
		offset      int64
		expected    *Needle
		expectedErr bool
	}{
		{
			name:     "happy path",
			reader:   bytes.NewReader(validBuf),
			offset:   0,
			expected: needle,
		},
		{
			name:        "bad magic bytes",
			reader:      bytes.NewReader(badMagicBuf),
			offset:      0,
			expected:    nil,
			expectedErr: true,
		},
		{
			name:        "corrupted checksum",
			reader:      bytes.NewReader(badChecksumBuf),
			offset:      0,
			expected:    nil,
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := ReadNeedleAt(tt.reader, tt.offset)
			if (err != nil) != tt.expectedErr {
				t.Errorf("ReadNeedleAt(%q, %q) error = %v, expectedErr %v", tt.reader, tt.offset, err, tt.expectedErr)
				return
			}
			if !tt.expectedErr {
				if n.Cookie != tt.expected.Cookie {
					t.Errorf("Cookie = %v, expected %v", n.Cookie, tt.expected.Cookie)
				}
				if n.ID != tt.expected.ID {
					t.Errorf("ID = %v, expected %v", n.ID, tt.expected.ID)
				}
				if n.Size != tt.expected.Size {
					t.Errorf("Size = %v, expected %v", n.Size, tt.expected.Size)
				}
				if !bytes.Equal(n.Data, tt.expected.Data) {
					t.Errorf("Data = %v, expected %v", n.Data, tt.expected.Data)
				}
			}
		})
	}
}
