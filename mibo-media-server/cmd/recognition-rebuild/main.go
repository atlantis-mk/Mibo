package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/library"
)

func main() {
	var libraryID uint
	var resetOnly bool
	flag.UintVar(&libraryID, "library-id", 0, "library id to rebuild")
	flag.BoolVar(&resetOnly, "reset-only", false, "only reset recognition state without rebuilding projections")
	flag.Parse()

	if libraryID == 0 {
		log.Fatal("library-id is required")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	svc := library.NewService(cfg, db, nil, nil)
	ctx := context.Background()

	if resetOnly {
		if err := svc.ResetRecognitionLibraryState(ctx, libraryID); err != nil {
			log.Fatalf("reset recognition library state: %v", err)
		}
		fmt.Printf("reset recognition state for library %d\n", libraryID)
		return
	}
	if err := svc.RebuildRecognitionLibraryState(ctx, libraryID); err != nil {
		log.Fatalf("rebuild recognition library state: %v", err)
	}
	fmt.Printf("rebuilt recognition state for library %d\n", libraryID)
}
