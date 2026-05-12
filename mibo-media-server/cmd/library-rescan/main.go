package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/metadata"
	"github.com/atlan/mibo-media-server/internal/probe"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
	"github.com/atlan/mibo-media-server/internal/workflow"
)

func main() {
	var libraryID uint
	var timeout time.Duration
	flag.UintVar(&libraryID, "library-id", 0, "library id to rescan")
	flag.DurationVar(&timeout, "timeout", 10*time.Minute, "maximum time to wait")
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
	registry := providers.NewRegistry(cfg)
	workflowSvc := workflow.NewService(db)
	ingestSvc := ingest.NewService(db, workflowSvc)
	settingsSvc := settings.NewService(db, cfg.Metadata)
	librarySvc := library.NewService(cfg, db, registry, nil, ingestSvc, workflowSvc)
	searchSvc := search.NewService(db, librarySvc)
	metadataSvc := metadata.NewService(db, cfg.Metadata, settingsSvc, searchSvc, ingestSvc)
	probeSvc := probe.NewService(db, registry, cfg.FFprobe, cfg.FFmpeg, ingestSvc)
	librarySvc.SetInventoryProbeExecutor(probeSvc.ProbeInventoryFile)
	librarySvc.SetMetadataMatchExecutor(func(ctx context.Context, metadataItemID uint, libraryID uint) error {
		_, err := metadataSvc.MatchMetadataItemOperation(ctx, metadataItemID, libraryID)
		return err
	})
	runner := workflow.NewRunner(workflowSvc, workflow.RunnerConfig{
		Enabled:       true,
		PollInterval:  200 * time.Millisecond,
		LeaseDuration: 30 * time.Second,
		TaskTimeout:   5 * time.Minute,
		MaxConcurrent: 1,
		Owner:         "library-rescan",
	})
	librarySvc.RegisterWorkflowHandlers(runner)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	go runner.Run(ctx)

	run, _, err := librarySvc.QueueLibraryWorkflow(ctx, library.QueueWorkflowInput{
		LibraryID: libraryID,
		Reason:    library.WorkflowReasonManualScan,
		Priority:  10,
	})
	if err != nil {
		log.Fatalf("queue library workflow: %v", err)
	}

	deadline := time.Now().Add(timeout)
	for {
		var refreshed database.WorkflowRun
		if err := db.WithContext(ctx).First(&refreshed, run.ID).Error; err != nil {
			log.Fatalf("load workflow run: %v", err)
		}
		switch refreshed.Status {
		case workflow.RunStatusCompleted:
			fmt.Printf("library %d rescan completed (run %d)\n", libraryID, refreshed.ID)
			return
		case workflow.RunStatusFailed:
			log.Fatalf("library %d rescan failed (run %d): %s", libraryID, refreshed.ID, refreshed.ErrorMessage)
		}
		if time.Now().After(deadline) {
			log.Fatalf("library %d rescan timed out waiting for run %d", libraryID, refreshed.ID)
		}
		time.Sleep(300 * time.Millisecond)
	}
}
