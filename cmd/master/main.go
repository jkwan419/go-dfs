package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/jkwan419/go-dfs/internal/server"
)

func main() {
	addr := flag.String("addr", ":9333", "master listen address")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	s := server.NewMasterServer(*addr)
	log.Printf("master listening on %s", *addr)
	if err := s.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
