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
	Size     uint64
}

type VolumeReport struct {
	VolumeID storage.VolumeID
	Size     uint64
}

type HeartbeatRequest struct {
	Addr    string
	Volumes []VolumeReport
}
