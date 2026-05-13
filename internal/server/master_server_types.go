package server

import "github.com/jkwan419/go-dfs/internal/storage"

type RegisterRequest struct {
	Addr string
}

type UploadResponse struct {
	Addr     string
	VolumeID storage.VolumeID
}

type UpdateVolumeRequest struct {
	VolumeID storage.VolumeID
	Size     uint32
}
