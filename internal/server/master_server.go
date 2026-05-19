package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"slices"
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
	mux.HandleFunc("/heartbeat", s.Heartbeat)

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

	go s.sweeper(ctx)

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

func (s *MasterServer) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !slices.Contains(s.VolumeServers, req.Addr) {
		s.VolumeServers = append(s.VolumeServers, req.Addr)
	}
	s.LastHeartbeat[req.Addr] = time.Now()

	for vid, info := range s.Volumes {
		if info.Addr == req.Addr {
			delete(s.Volumes, vid)
		}
	}

	for _, report := range req.Volumes {
		s.Volumes[report.VolumeID] = &VolumeInfo{Addr: req.Addr, Size: report.Size}
		if report.VolumeID >= s.NextVolumeID {
			s.NextVolumeID = report.VolumeID + 1
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (s *MasterServer) sweep() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for addr, ts := range s.LastHeartbeat {
		if now.Sub(ts) <= s.StalenessThreshold {
			continue
		}
		// Evict:
		//	- remove addr from VolumeServers
		s.VolumeServers = slices.DeleteFunc(s.VolumeServers, func(v string) bool {
			return v == addr
		})
		//	- delete all volumes whose Addr == addr
		for vid, info := range s.Volumes {
			if info.Addr == addr {
				delete(s.Volumes, vid)
			}
		}
		//	- delete LastHeartbeat[addr]
		delete(s.LastHeartbeat, addr)
	}
}

func (s *MasterServer) sweeper(ctx context.Context) {
	ticker := time.NewTicker(s.StalenessThreshold / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweep()
		}
	}
}
