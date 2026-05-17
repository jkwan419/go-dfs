package server

import (
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

type VolumeServer struct {
	Addr    string
	Store   *storage.Store
	DataDir string
}

func NewVolumeServer(addr string, store *storage.Store, dir string) *VolumeServer {
	return &VolumeServer{
		Addr:    addr,
		Store:   store,
		DataDir: dir,
	}
}

func (s *VolumeServer) Start() {
	mux := http.NewServeMux()

	mux.HandleFunc("/read", s.Read)
	mux.HandleFunc("/write", s.Write)
	mux.HandleFunc("/create", s.Create)

	err := http.ListenAndServe(s.Addr, mux)
	if err != nil {
		log.Fatal(err)
	}
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

	needle := &storage.Needle{
		Cookie: rand.Uint32(),
		ID:     volume.NextID,
		Size:   uint32(len(data)),
		Data:   data,
	}

	err = volume.Write(needle)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fileID := &storage.FileID{
		VolumeID: vid,
		Key:      needle.ID,
		Cookie:   needle.Cookie,
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

func (s *VolumeServer) worker(ctx context.Context, pulseInterval time.Duration) (<-chan struct{}, <-chan int) {
	heartbeat := make(chan struct{})
	results := make(chan int)

	go func() {
		defer close(heartbeat)
		defer close(results)

		pulse := time.NewTicker(pulseInterval)
		defer pulse.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-pulse.C:
				select {
				case heartbeat <- struct{}{}:
				default:
				}
			}
		}
	}()
	return heartbeat, results
}
