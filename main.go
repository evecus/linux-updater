package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var (
	dataDir string
	port    int
)

func main() {
	exePath, err := os.Executable()
	if err != nil {
		exePath = "."
	}
	defaultData := filepath.Join(filepath.Dir(exePath), "data")

	flag.StringVar(&dataDir, "dir", defaultData, "data directory")
	flag.IntVar(&port, "port", 9191, "web panel port")
	flag.Parse()

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("failed to create data dir: %v", err)
	}

	store := NewStore(dataDir)
	if err := store.Load(); err != nil {
		log.Fatalf("failed to load store: %v", err)
	}

	scheduler := NewScheduler(store)
	scheduler.Start()

	srv := NewServer(store, scheduler)
	addr := fmt.Sprintf(":%d", port)
	log.Printf("updater panel running at http://0.0.0.0%s  (data: %s)", addr, dataDir)
	if err := http.ListenAndServe(addr, srv.mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
