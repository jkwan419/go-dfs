package server

import (
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/jkwan419/go-dfs/internal/storage"
)

type VolumeServer struct {
	Addr  string
	Store *storage.Store
}

func NewVolumeServer(addr string, store *storage.Store) *VolumeServer {
	return &VolumeServer{
		Addr:  addr,
		Store: store,
	}
}

func (s *VolumeServer) Start() {
	mux := http.NewServeMux()

	mux.HandleFunc("/read", s.Read)
	mux.HandleFunc("/write", s.Write)

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
