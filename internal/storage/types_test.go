package storage

import (
	"fmt"
	"math"
	"testing"
)

func TestParseFileId(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    FileId
		wantErr bool
	}{
		{
			name:  "normal case",
			input: "3,0000000000000001deadbeef",
			want:  FileId{VolumeID: 3, Key: 1, Cookie: 0xdeadbeef},
		},
		{
			name:  "zero key and cookie",
			input: "1,000000000000000000000000",
			want:  FileId{VolumeID: 1, Key: 0, Cookie: 0},
		},
		{
			name:  "max values",
			input: fmt.Sprintf("%d,%016x%08x", uint32(math.MaxUint32), uint64(math.MaxUint64), uint32(math.MaxUint32)),
			want:  FileId{VolumeID: math.MaxUint32, Key: math.MaxUint64, Cookie: math.MaxUint32},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "no comma",
			input:   "12345",
			wantErr: true,
		},
		{
			name:    "rest too short",
			input:   "1,abc",
			wantErr: true,
		},
		{
			name:    "non-hex characters",
			input:   "1,gggggggggggggggggggggggg",
			wantErr: true,
		},
		{
			name:    "invalid volume id",
			input:   "abc,0000000000000001deadbeef",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFileID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFileId(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseFileId(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}
