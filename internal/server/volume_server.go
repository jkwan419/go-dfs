package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jkwan419/go-dfs/internal/storage"
)

type VolumeServerConfig struct {
	Addr              string
	MasterAddr        string
	DataDir           string
	HeartbeatInterval time.Duration
}

type VolumeServer struct {
	Addr              string
	MasterAddr        string
	Store             *storage.Store
	DataDir           string
	HeartbeatInterval time.Duration
}

func NewVolumeServer(cfg VolumeServerConfig, store *storage.Store) *VolumeServer {
	interval := cfg.HeartbeatInterval
	if interval == 0 {
		interval = DefaultHeartbeatInterval
	}
	return &VolumeServer{
		Addr:              cfg.Addr,
		MasterAddr:        cfg.MasterAddr,
		Store:             store,
		DataDir:           cfg.DataDir,
		HeartbeatInterval: interval,
	}
}

func (s *VolumeServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/read", s.Read)
	mux.HandleFunc("/write", s.Write)
	mux.HandleFunc("/create", s.Create)

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

	go s.worker(ctx)

	err := srv.ListenAndServe()
	close(done)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *VolumeServer) Read(w http.ResponseWriter, r *http.Request) {
	vidStr := r.URL.Query().Get("volumeID")
	nidStr := r.URL.Query().Get("needleID")
	vid, err := storage.ParseVolumeID(vidStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	volume, err := s.Store.GetVolume(vid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	nid, err := strconv.ParseUint(nidStr, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	needle, err := volume.Read(nid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(needle.Data)
}

func (s *VolumeServer) Write(w http.ResponseWriter, r *http.Request) {
	vidStr := r.URL.Query().Get("volumeID")
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	vid, err := storage.ParseVolumeID(vidStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	volume, err := s.Store.GetVolume(vid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cookie := rand.Uint32()
	id, err := volume.Write(data, cookie)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fileID := &storage.FileID{
		VolumeID: vid,
		Key:      id,
		Cookie:   cookie,
	}

	w.Write([]byte(fileID.String()))
}

func (s *VolumeServer) Create(w http.ResponseWriter, r *http.Request) {
	var req VolumeCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	volumeFile, err := os.OpenFile(fmt.Sprintf("%s/%d.dat", s.DataDir, req.VolumeID), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	indexFile, err := os.OpenFile(fmt.Sprintf("%s/%d.idx", s.DataDir, req.VolumeID), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	volume := storage.NewVolume(req.VolumeID, volumeFile, indexFile)
	err = s.Store.AddVolume(req.VolumeID, volume)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *VolumeServer) worker(ctx context.Context) {
	client := &http.Client{Timeout: s.HeartbeatInterval / 2}

	sendOnce := func() {
		// Lock order: Store mutex first (via Snapshot), then per-Volume mutex
		// (via HeartbeatStats). Never the reverse - Snapshot already released the
		// Store lock before we touch any Volume, but the rule still applies to any
		// future code added here.
		vols := s.Store.Snapshot()
		reports := []VolumeReport{}
		for _, v := range vols {
			id, size := v.HeartbeatStats()
			reports = append(reports, VolumeReport{VolumeID: id, Size: size})
		}

		body := HeartbeatRequest{Addr: s.Addr, Volumes: reports}
		buf, err := json.Marshal(body)
		if err != nil {
			log.Printf("heartbeat failed: %v", err)
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", "http://"+s.MasterAddr+"/heartbeat", bytes.NewReader(buf))
		if err != nil {
			log.Printf("heartbeat failed: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("heartbeat failed: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			log.Printf("heartbeat got %d", resp.StatusCode)
		}
	}

	sendOnce()

	ticker := time.NewTicker(s.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sendOnce()
		}
	}
}
