package storage

import (
	"encoding/binary"
	"io"
	"os"
)

func ReadFromFile(file *os.File) (map[uint64]int64, error) {
	out := make(map[uint64]int64)
	buf := make([]byte, 16)
	for {
		_, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		needleID := binary.BigEndian.Uint64(buf[0:8])
		offset := binary.BigEndian.Uint64(buf[8:16])
		out[needleID] = int64(offset)
	}
	return out, nil
}

func WriteToFile(file *os.File, needleID uint64, offset int64) error {
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[0:8], needleID)
	binary.BigEndian.PutUint64(buf[8:16], uint64(offset))
	_, err := file.Write(buf)
	if err != nil {
		return err
	}
	return nil
}
