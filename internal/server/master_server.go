package server

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/jkwan419/go-dfs/internal/storage"
)

type VolumeInfo struct {
	Addr string
	Size uint32
}

type MasterServer struct {
	Addr              string
	Volumes           map[storage.VolumeID]*VolumeInfo
	VolumeServers     []string
	VolumeSizeLimitMB uint32
	NextVolumeID      storage.VolumeID
}

func NewMasterServer(addr string) *MasterServer {
	return &MasterServer{
		Addr:              addr,
		Volumes:           make(map[storage.VolumeID]*VolumeInfo),
		VolumeSizeLimitMB: 3000,
		NextVolumeID:      0,
	}
}

func (s *MasterServer) Start() {
	mux := http.NewServeMux()

	mux.HandleFunc("/read", s.Read)
	mux.HandleFunc("/upload", s.Upload)
	mux.HandleFunc("/register", s.Register)
	mux.HandleFunc("/update", s.UpdateVolume)

	err := http.ListenAndServe(s.Addr, mux)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *MasterServer) Read(w http.ResponseWriter, r *http.Request) {
	vidStr := r.URL.Query().Get("volumeID")
	vid, err := storage.ParseVolumeID(vidStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	volume, ok := s.Volumes[vid]
	if !ok {
		http.Error(w, "invalid volume id", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(volume.Addr))
}

func (s *MasterServer) Upload(w http.ResponseWriter, r *http.Request) {
	vid, found := s.findWriteableVolume()
	if !found {
		if len(s.VolumeServers) == 0 {
			http.Error(w, "no volume servers registered", http.StatusBadRequest)
			return
		}

		body, err := json.Marshal(VolumeCreateRequest{VolumeID: s.NextVolumeID})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp, err := http.Post("http://"+s.VolumeServers[0]+"/create", "application/json", bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer resp.Body.Close()

		s.Volumes[s.NextVolumeID] = &VolumeInfo{
			Addr: s.VolumeServers[0],
			Size: 0,
		}

		vid = s.NextVolumeID
		s.NextVolumeID += 1
	}

	res := &UploadResponse{
		Addr:     s.Volumes[vid].Addr,
		VolumeID: vid,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (s *MasterServer) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	s.VolumeServers = append(s.VolumeServers, req.Addr)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
}

func (s *MasterServer) UpdateVolume(w http.ResponseWriter, r *http.Request) {
	var req UpdateVolumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	volume, ok := s.Volumes[req.VolumeID]
	if !ok {
		http.Error(w, "invalid volume id", http.StatusBadRequest)
		return
	}

	volume.Size = req.Size

	w.WriteHeader(http.StatusOK)
}
