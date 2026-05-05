package library

import (
	"context"
	"log"
	"strconv"

	"github.com/atlan/mibo-media-server/internal/database"
)

func (s *Service) markInventoryFileDirty(ctx context.Context, fileID uint, reason string) {
	if s.ingest == nil || fileID == 0 {
		return
	}
	if _, err := s.ingest.MarkInventoryFileDirty(ctx, fileID, reason); err != nil {
		log.Printf("library: mark inventory file %d ingest dirty: %v", fileID, err)
	}
}

func (s *Service) markLibraryScopeDirty(ctx context.Context, libraryID uint, rootPath string, reason string) {
	if s.ingest == nil || libraryID == 0 {
		return
	}
	if _, err := s.ingest.MarkLibraryScopeDirty(ctx, libraryID, rootPath, reason); err != nil {
		log.Printf("library: mark library %d ingest dirty: %v", libraryID, err)
	}
}

func (s *Service) markProjectionItemDirty(ctx context.Context, itemID uint, reason string) {
	if s.ingest == nil || itemID == 0 {
		return
	}
	if _, err := s.ingest.MarkProjectionItemDirty(ctx, itemID, reason); err != nil {
		log.Printf("library: mark item %d projection dirty: %v", itemID, err)
	}
}

func (s *Service) markProjectionLibraryDirty(ctx context.Context, libraryID uint, rootPath string, reason string) {
	if s.ingest == nil || libraryID == 0 {
		return
	}
	if _, err := s.ingest.MarkProjectionLibraryDirty(ctx, libraryID, rootPath, reason); err != nil {
		log.Printf("library: mark library %d projection dirty: %v", libraryID, err)
	}
}

func (s *Service) appendIngestEvent(ctx context.Context, event database.IngestEvent) {
	if s.ingest == nil {
		return
	}
	if _, err := s.ingest.AppendEvent(ctx, event); err != nil {
		log.Printf("library: append ingest event %q: %v", event.EventType, err)
	}
}

func inventoryFileUnitKey(fileID uint) string {
	return "inventory_file:" + strconv.FormatUint(uint64(fileID), 10)
}
