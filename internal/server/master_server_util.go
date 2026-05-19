package server

import "github.com/jkwan419/go-dfs/internal/storage"

// findWriteableVolume must be called with s.mu held.
func (s *MasterServer) findWriteableVolume() (storage.VolumeID, bool) {
	var vid storage.VolumeID
	for k, v := range s.Volumes {
		if v.Size < s.VolumeSizeLimitBytes {
			vid = k
			return vid, true
		}
	}
	return 0, false
}
