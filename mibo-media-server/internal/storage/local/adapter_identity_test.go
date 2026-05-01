package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/storage"
)

func TestAdapterExposesStableLocalIdentity(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "movie.mkv")
	if err := os.WriteFile(filePath, []byte("fixture"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	adapter := New(config.LocalStorageConfig{RootPath: root})
	object, err := adapter.Get(context.Background(), storage.GetRequest{Path: filePath})
	if err != nil {
		t.Fatalf("get object: %v", err)
	}
	if object.StableIdentity == "" {
		t.Fatalf("expected stable identity, got %#v", object)
	}
	if object.ProviderMeta["device"] == "" || object.ProviderMeta["inode"] == "" {
		t.Fatalf("expected device and inode metadata, got %#v", object.ProviderMeta)
	}

	objects, err := adapter.List(context.Background(), storage.ListRequest{Path: root})
	if err != nil {
		t.Fatalf("list objects: %v", err)
	}
	if len(objects) != 1 || objects[0].StableIdentity == "" {
		t.Fatalf("expected listed object stable identity, got %#v", objects)
	}
}
