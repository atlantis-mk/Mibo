package catalog

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestRebuildMetadataSearchDocument(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie", OriginalTitle: "Original", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	person := database.Person{Name: "Actor One", SortName: "actor one"}
	tag := database.Tag{Kind: "genre", Name: "Drama"}
	if err := db.WithContext(ctx).Create(&person).Error; err != nil {
		t.Fatalf("create person: %v", err)
	}
	if err := db.WithContext(ctx).Create(&tag).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.MetadataItemPerson{MetadataItemID: item.ID, PersonID: person.ID, Role: "actor"}).Error; err != nil {
		t.Fatalf("create item person: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.MetadataItemTag{MetadataItemID: item.ID, TagID: tag.ID}).Error; err != nil {
		t.Fatalf("create item tag: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.MetadataExternalID{MetadataItemID: item.ID, Provider: "tmdb", ProviderType: "movie", ExternalID: "movie:42"}).Error; err != nil {
		t.Fatalf("create external id: %v", err)
	}

	doc, err := svc.RebuildMetadataSearchDocument(ctx, item.ID)
	if err != nil {
		t.Fatalf("rebuild search doc: %v", err)
	}
	if doc.Title != "Movie" || !strings.Contains(doc.PeopleText, "Actor One") || !strings.Contains(doc.TagsText, "Drama") || !strings.Contains(doc.ProviderIDsText, "movie:42") {
		t.Fatalf("unexpected metadata search document: %#v", doc)
	}
}

func TestRebuildLibrarySearchDocumentIncludesProjectionAndResourceText(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie", OriginalTitle: "Original", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	resource := database.Resource{StableResourceKey: "resource:search", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, DisplayName: "Movie 4K", QualityLabel: "2160p", Status: "available"}
	file := database.InventoryFile{LibraryID: 7, StorageProvider: "local", StoragePath: "/library/Movie.2160p.mkv", ContentClass: "video", Status: "available"}
	if err := db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	if err := db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceFile{ResourceID: resource.ID, InventoryFileID: file.ID, Role: database.ResourceFileRoleSource}).Error; err != nil {
		t.Fatalf("create resource file: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceLibraryLink{ResourceID: resource.ID, LibraryID: 7, Status: "available", FirstSeenAt: item.CreatedAt, LastSeenAt: item.CreatedAt}).Error; err != nil {
		t.Fatalf("create library link: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary}).Error; err != nil {
		t.Fatalf("create metadata link: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.LibraryMetadataProjection{LibraryID: 7, MetadataItemID: item.ID, ItemType: item.ItemType, Title: item.Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, LastProjectedAt: item.CreatedAt}).Error; err != nil {
		t.Fatalf("create projection: %v", err)
	}

	doc, err := svc.RebuildLibrarySearchDocument(ctx, 7, item.ID)
	if err != nil {
		t.Fatalf("rebuild library search doc: %v", err)
	}
	if doc.LibraryID != 7 || doc.MetadataItemID != item.ID || !strings.Contains(doc.ResourceText, "2160p") || !strings.Contains(doc.ResourceText, "Movie.2160p.mkv") {
		t.Fatalf("unexpected library search doc: %#v", doc)
	}
}
