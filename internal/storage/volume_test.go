package storage

import (
	"bytes"
	"os"
	"testing"
)

func TestVolumeRead(t *testing.T) {
	vFile, err := os.CreateTemp("", "volume-test-*.dat")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		vFile.Close()
		os.Remove(vFile.Name())
	})

	iFile, err := os.CreateTemp("", "index-test-*.idx")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		iFile.Close()
		os.Remove(iFile.Name())
	})

	volume := NewVolume(0x0000000000000000, vFile, iFile)

	needle := &Needle{
		Cookie: 0xDEADBEEF,
		Data:   []byte("test"),
		Size:   4,
	}

	id, err := volume.Write(needle.Data, needle.Cookie)
	if err != nil {
		t.Fatal(err)
	}
	needle.ID = id

	tests := []struct {
		name        string
		input       uint64
		expected    *Needle
		expectedErr bool
	}{
		{
			name:     "happy path",
			input:    id,
			expected: needle,
		},
		{
			name:        "invalid needle id",
			input:       id + 1,
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := volume.Read(tt.input)
			if (err != nil) != tt.expectedErr {
				t.Errorf("Read(%q) error = %v, expectedErr %v", tt.input, err, tt.expectedErr)
				return
			}
			if !tt.expectedErr {
				if got.Cookie != tt.expected.Cookie {
					t.Errorf("Cookie = %v, expected %v", got.Cookie, tt.expected.Cookie)
				}
				if got.ID != tt.expected.ID {
					t.Errorf("ID = %v, expected %v", got.ID, tt.expected.ID)
				}
				if !bytes.Equal(got.Data, tt.expected.Data) {
					t.Errorf("Data = %v, expected %v", got.Data, tt.expected.Data)
				}
				if got.Size != tt.expected.Size {
					t.Errorf("Size = %v, expected %v", got.Size, tt.expected.Size)
				}
			}
		})
	}
}

func TestVolumeWrite(t *testing.T) {
	vFile, err := os.CreateTemp("", "volume-test-*.dat")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		vFile.Close()
		os.Remove(vFile.Name())
	})

	iFile, err := os.CreateTemp("", "index-test-*.idx")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		iFile.Close()
		os.Remove(iFile.Name())
	})

	volume := NewVolume(0x0000000000000000, vFile, iFile)

	needle := &Needle{
		Cookie: 0xDEADBEEF,
		Data:   []byte("test"),
		Size:   4,
	}

	tests := []struct {
		name        string
		input       *Needle
		expectedErr bool
	}{
		{
			name:  "happy path",
			input: needle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := volume.Write(tt.input.Data, tt.input.Cookie)
			if (err != nil) != tt.expectedErr {
				t.Errorf("Write(%q) error = %v, expectedErr %v", tt.input, err, tt.expectedErr)
			}
			if !tt.expectedErr {
				if volume.Index[id] != 0 {
					t.Errorf("Offset = %v, expected = 0", volume.Index[id])
				}
				if volume.Offset != NeedleDiskSize(tt.input.Size) {
					t.Errorf("Offset = %v, expected = %v", volume.Offset, NeedleDiskSize(tt.input.Size))
				}
			}
		})
	}
}
