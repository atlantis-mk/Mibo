package playback

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/probe"
	"github.com/atlan/mibo-media-server/internal/providers"
)

func TestResourcePlaybackReturnsOpenListDirectURLs(t *testing.T) {
	ctx := context.Background()
	rootPath := t.TempDir()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(rootPath, "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/fs/get" {
			http.NotFound(w, req)
			return
		}
		var payload struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"code":200,"message":"ok","data":{"name":"%s","is_dir":false,"size":5,"raw_url":"https://cdn.example.test%s"}}`, filepath.Base(payload.Path), payload.Path)
	}))
	defer server.Close()

	configJSON, err := providers.MarshalSourceConfig(providers.SourceConfig{OpenList: &providers.OpenListSourceConfig{BaseURL: server.URL}})
	if err != nil {
		t.Fatalf("marshal source config: %v", err)
	}
	source := database.MediaSource{Name: "OpenList", Provider: "openlist", StorageRef: "/library", RootPath: "/library", ConfigJSON: configJSON}
	if err := db.WithContext(ctx).Create(&source).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}
	libraryRecord := database.Library{Name: "Movies", Type: "movies", MediaSourceID: source.ID, RootPath: "/library", Status: "active", ScannerEnabled: true}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	service := NewService(db, providers.NewRegistry(config.Config{Local: config.LocalStorageConfig{RootPath: rootPath}}))

	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Remote Movie", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	resource := database.Resource{StableResourceKey: "resource:remote", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, DisplayName: "Remote Movie", Status: "available", ProbeStatus: probe.StatusReady}
	if err := db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	file := database.InventoryFile{LibraryID: libraryRecord.ID, StorageProvider: "openlist", StoragePath: "/movies/remote-movie.mp4", Container: "mp4", Status: "available"}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceFile{ResourceID: resource.ID, InventoryFileID: file.ID, Role: database.ResourceFileRoleSource}).Error; err != nil {
		t.Fatalf("create resource file: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceLibraryLink{ResourceID: resource.ID, LibraryID: libraryRecord.ID, Status: "available", FirstSeenAt: item.CreatedAt, LastSeenAt: item.CreatedAt}).Error; err != nil {
		t.Fatalf("create library link: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary}).Error; err != nil {
		t.Fatalf("create metadata link: %v", err)
	}
	width := 1920
	height := 1080
	if err := db.WithContext(ctx).Create(&database.MediaStream{FileID: file.ID, StreamIndex: 0, StreamType: "video", Codec: "h264", Width: &width, Height: &height}).Error; err != nil {
		t.Fatalf("create stream: %v", err)
	}

	subtitleFile := database.InventoryFile{LibraryID: libraryRecord.ID, StorageProvider: "openlist", StoragePath: "/movies/remote-movie.en.srt", Container: "srt", ContentClass: "subtitle", Status: "available"}
	if err := db.WithContext(ctx).Create(&subtitleFile).Error; err != nil {
		t.Fatalf("create subtitle file: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceFile{ResourceID: resource.ID, InventoryFileID: subtitleFile.ID, Role: database.ResourceFileRoleSubtitle}).Error; err != nil {
		t.Fatalf("create subtitle resource file: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.MediaStream{FileID: subtitleFile.ID, StreamIndex: 1, StreamType: "subtitle", Codec: "srt", Language: "en", DispositionJSON: `{"external":true}`}).Error; err != nil {
		t.Fatalf("create subtitle stream: %v", err)
	}

	link, err := service.GetPlaybackSource(ctx, PlaybackRequest{MetadataItemID: item.ID, ResourceID: resource.ID, LibraryID: libraryRecord.ID, ClientProfile: ClientProfileWeb})
	if err != nil {
		t.Fatalf("get playback source: %v", err)
	}
	if link.URL != "https://cdn.example.test/movies/remote-movie.mp4" {
		t.Fatalf("expected openlist direct url, got %q", link.URL)
	}
	if len(link.SubtitleTracks) != 1 || link.SubtitleTracks[0].URL != "https://cdn.example.test/movies/remote-movie.en.srt" {
		t.Fatalf("expected direct subtitle url, got %#v", link.SubtitleTracks)
	}
}
