package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jkwan419/go-dfs/internal/server"
	"github.com/jkwan419/go-dfs/internal/storage"
)

func main() {
	addr := flag.String("addr", ":9001", "volume server listen address")
	master := flag.String("master", "localhost:9333", "master address (host:port)")
	dir := flag.String("data", "./data", "data directory")
	flag.Parse()

	if err := os.MkdirAll(*dir, 0755); err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store := storage.NewStore()
	s := server.NewVolumeServer(server.VolumeServerConfig{
		Addr:       *addr,
		MasterAddr: *master,
		DataDir:    *dir,
	}, store)

	log.Printf("volume server listening on %s, master at %s, data dir %s", *addr, *master, *dir)
	if err := s.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
