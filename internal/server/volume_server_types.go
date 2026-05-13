package server

import "github.com/jkwan419/go-dfs/internal/storage"

type VolumeCreateRequest struct {
	VolumeID storage.VolumeID
}
