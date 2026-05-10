package storage

import (
	"fmt"
	"math"
	"testing"
)

func TestParseFileId(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    FileID
		expectedErr bool
	}{
		{
			name:     "normal case",
			input:    "3,0000000000000001deadbeef",
			expected: FileID{VolumeID: 3, Key: 1, Cookie: 0xdeadbeef},
		},
		{
			name:     "zero key and cookie",
			input:    "1,000000000000000000000000",
			expected: FileID{VolumeID: 1, Key: 0, Cookie: 0},
		},
		{
			name:     "max values",
			input:    fmt.Sprintf("%d,%016x%08x", uint32(math.MaxUint32), uint64(math.MaxUint64), uint32(math.MaxUint32)),
			expected: FileID{VolumeID: math.MaxUint32, Key: math.MaxUint64, Cookie: math.MaxUint32},
		},
		{
			name:        "empty string",
			input:       "",
			expectedErr: true,
		},
		{
			name:        "no comma",
			input:       "12345",
			expectedErr: true,
		},
		{
			name:        "rest too short",
			input:       "1,abc",
			expectedErr: true,
		},
		{
			name:        "non-hex characters",
			input:       "1,gggggggggggggggggggggggg",
			expectedErr: true,
		},
		{
			name:        "invalid volume id",
			input:       "abc,0000000000000001deadbeef",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFileID(tt.input)
			if (err != nil) != tt.expectedErr {
				t.Errorf("ParseFileId(%q) error = %v, expectedErr %v", tt.input, err, tt.expectedErr)
				return
			}
			if !tt.expectedErr && got != tt.expected {
				t.Errorf("ParseFileId(%q) = %+v, expected %+v", tt.input, got, tt.expected)
			}
		})
	}
}
