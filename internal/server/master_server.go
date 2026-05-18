package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/jkwan419/go-dfs/internal/storage"
)

type VolumeInfo struct {
	Addr string
	Size uint64
}

type MasterServer struct {
	Addr                 string
	Volumes              map[storage.VolumeID]*VolumeInfo
	VolumeServers        []string
	VolumeSizeLimitBytes uint64
	NextVolumeID         storage.VolumeID
	LastHeartbeat        map[string]time.Time
	StalenessThreshold   time.Duration
	mu                   sync.Mutex
}

func NewMasterServer(addr string) *MasterServer {
	return &MasterServer{
		Addr:                 addr,
		Volumes:              make(map[storage.VolumeID]*VolumeInfo),
		VolumeSizeLimitBytes: 3 * 1024 * 1024 * 1024,
		NextVolumeID:         0,
		LastHeartbeat:        make(map[string]time.Time),
		StalenessThreshold:   DefaultStalenessThreshold,
	}
}

func (s *MasterServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/read", s.Read)
	mux.HandleFunc("/upload", s.Upload)
	mux.HandleFunc("/register", s.Register)
	mux.HandleFunc("/update", s.UpdateVolume)

	srv := &http.Server{Addr: s.Addr, Handler: mux}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			srv.Shutdown(shutdownCtx)
		case <-done:
		}
	}()

	err := srv.ListenAndServe()
	close(done)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *MasterServer) Read(w http.ResponseWriter, r *http.Request) {
	vidStr := r.URL.Query().Get("volumeID")
	vid, err := storage.ParseVolumeID(vidStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	volume, ok := s.Volumes[vid]

	if !ok {
		http.Error(w, "invalid volume id", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(volume.Addr))
}

func (s *MasterServer) Upload(w http.ResponseWriter, r *http.Request) {
	var targetVid storage.VolumeID
	var targetAddr string
	var needsCreate bool

	s.mu.Lock()
	if vid, found := s.findWriteableVolume(); found {
		targetVid = vid
		targetAddr = s.Volumes[vid].Addr
		needsCreate = false
		s.mu.Unlock()
	} else if len(s.VolumeServers) == 0 {
		s.mu.Unlock()
		http.Error(w, "no volume servers registered", http.StatusBadRequest)
		return
	} else {
		targetAddr = s.VolumeServers[0]
		targetVid = s.NextVolumeID
		s.mu.Unlock()
		needsCreate = true
	}

	if needsCreate {
		body, err := json.Marshal(VolumeCreateRequest{VolumeID: targetVid})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp, err := http.Post("http://"+targetAddr+"/create", "application/json", bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer resp.Body.Close()

		s.mu.Lock()
		s.Volumes[targetVid] = &VolumeInfo{Addr: targetAddr, Size: 0}
		s.NextVolumeID++
		s.mu.Unlock()
	}

	resp := UploadResponse{
		Addr:     targetAddr,
		VolumeID: targetVid,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *MasterServer) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
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

	s.mu.Lock()
	defer s.mu.Unlock()
	volume, ok := s.Volumes[req.VolumeID]
	if !ok {
		http.Error(w, "invalid volume id", http.StatusBadRequest)
		return
	}
	volume.Size = req.Size
	w.WriteHeader(http.StatusOK)
}
