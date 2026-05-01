package listener

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/fsnotify/fsnotify"
)

const localObserverReconcileInterval = 30 * time.Minute

func (s *Service) StartLocalObserver(ctx context.Context) {
	if s == nil || s.db == nil {
		return
	}
	observer := &localObserver{listener: s, watched: make(map[string]struct{})}
	if err := observer.start(ctx); err != nil {
		log.Printf("listener: local observer unavailable, using reconcile fallback only: %v", err)
		go observer.runReconcileFallback(ctx)
	}
}

type localObserver struct {
	listener *Service
	watcher  *fsnotify.Watcher
	watched  map[string]struct{}
	mu       sync.Mutex
}

func (o *localObserver) start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	o.watcher = watcher
	if err := o.refreshLibraries(ctx); err != nil {
		_ = watcher.Close()
		return err
	}
	go o.run(ctx)
	return nil
}

func (o *localObserver) run(ctx context.Context) {
	ticker := time.NewTicker(localObserverReconcileInterval)
	defer ticker.Stop()
	defer o.watcher.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-o.watcher.Events:
			if !ok {
				return
			}
			o.handleEvent(ctx, event)
		case err, ok := <-o.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("listener: local observer error: %v", err)
		case <-ticker.C:
			if err := o.refreshLibraries(ctx); err != nil {
				log.Printf("listener: refresh local observer libraries: %v", err)
			}
		}
	}
}

func (o *localObserver) runReconcileFallback(ctx context.Context) {
	ticker := time.NewTicker(localObserverReconcileInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			libraries, err := o.localLibraries(ctx)
			if err != nil {
				log.Printf("listener: local observer fallback list libraries: %v", err)
				continue
			}
			if err := o.listener.EnsureReconcileCoverage(ctx, libraries); err != nil {
				log.Printf("listener: local observer fallback reconcile: %v", err)
			}
		}
	}
}

func (o *localObserver) refreshLibraries(ctx context.Context) error {
	libraries, err := o.localLibraries(ctx)
	if err != nil {
		return err
	}
	for _, libraryRecord := range libraries {
		if err := o.addRecursive(libraryRecord.RootPath); err != nil {
			log.Printf("listener: watch local library %d path %s: %v", libraryRecord.ID, libraryRecord.RootPath, err)
		}
	}
	return o.listener.EnsureReconcileCoverage(ctx, libraries)
}

func (o *localObserver) localLibraries(ctx context.Context) ([]database.Library, error) {
	var libraries []database.Library
	err := o.listener.db.WithContext(ctx).
		Joins("JOIN media_sources ON media_sources.id = libraries.media_source_id").
		Where("libraries.status = ? AND libraries.scanner_enabled = ? AND media_sources.provider = ?", "active", true, "local").
		Order("libraries.id asc").
		Find(&libraries).Error
	return libraries, err
}

func (o *localObserver) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		return o.addWatch(path)
	})
}

func (o *localObserver) addWatch(path string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "" {
		return nil
	}
	if _, ok := o.watched[clean]; ok {
		return nil
	}
	if err := o.watcher.Add(clean); err != nil {
		return err
	}
	o.watched[clean] = struct{}{}
	return nil
}

func (o *localObserver) handleEvent(ctx context.Context, event fsnotify.Event) {
	kind := localEventKind(event)
	if kind == "" {
		return
	}
	if event.Has(fsnotify.Create) {
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			if err := o.addRecursive(event.Name); err != nil {
				log.Printf("listener: watch created directory %s: %v", event.Name, err)
			}
		}
	}
	libraryRecord, ok := o.libraryForPath(ctx, event.Name)
	if !ok {
		return
	}
	if _, err := o.listener.RecordStorageEvent(ctx, EventIngestInput{LibraryID: libraryRecord.ID, Kind: kind, Path: event.Name}); err != nil {
		log.Printf("listener: record local storage event library=%d path=%s kind=%s: %v", libraryRecord.ID, event.Name, kind, err)
	}
}

func (o *localObserver) libraryForPath(ctx context.Context, eventPath string) (database.Library, bool) {
	libraries, err := o.localLibraries(ctx)
	if err != nil {
		log.Printf("listener: resolve local event library: %v", err)
		return database.Library{}, false
	}
	cleanEvent := filepath.Clean(eventPath)
	var selected database.Library
	for _, libraryRecord := range libraries {
		root := filepath.Clean(libraryRecord.RootPath)
		rel, err := filepath.Rel(root, cleanEvent)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if selected.ID == 0 || len(root) > len(filepath.Clean(selected.RootPath)) {
			selected = libraryRecord
		}
	}
	return selected, selected.ID != 0
}

func localEventKind(event fsnotify.Event) string {
	if event.Has(fsnotify.Remove) {
		return "delete"
	}
	if event.Has(fsnotify.Rename) {
		return "delete"
	}
	if event.Has(fsnotify.Create) {
		return "create"
	}
	if event.Has(fsnotify.Write) || event.Has(fsnotify.Chmod) {
		return "update"
	}
	return ""
}
