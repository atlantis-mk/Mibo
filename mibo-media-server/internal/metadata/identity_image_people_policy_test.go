package metadata

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestNormalizedIdentityImagePeopleAndProjectionPolicies(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	catalogSvc := catalog.NewService(db)
	item, err := catalogSvc.CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "Original", Path: "/movies/original.mkv", SortKey: "Original"})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	sourceID := uint(42)
	confidence := 0.9
	svc := NewService(db, config.MetadataConfig{}, nil)
	if err := applyNormalizedExternalIDs(ctx, catalogSvc, item.ID, []NormalizedMetadataExternalID{{Provider: "tmdb", ProviderType: "movie", ExternalID: "movie:42", IsPrimary: true, Confidence: &confidence}}, "metadata_match", nil); err != nil {
		t.Fatalf("apply normalized ids: %v", err)
	}
	if err := svc.applyNormalizedImages(ctx, item.ID, []NormalizedMetadataImage{{ImageType: "poster", URL: "https://img/poster.jpg", Selected: true}}, false, &sourceID); err != nil {
		t.Fatalf("apply normalized images: %v", err)
	}
	personID := 123
	if err := svc.applyNormalizedPeople(ctx, item.ID, []NormalizedMetadataPerson{{Name: "Actor One", Role: "actor", Character: "Lead", AvatarURL: "https://img/actor.jpg", TMDBPersonID: &personID}, {Name: "Director One", Role: "director"}}, &sourceID); err != nil {
		t.Fatalf("apply normalized people: %v", err)
	}
	if err := svc.refreshMetadataOperationProjectionScope(ctx, MetadataAffectedScope{ItemIDs: []uint{item.ID, item.ID}}); err != nil {
		t.Fatalf("refresh projection scope: %v", err)
	}

	var externalID database.CatalogExternalID
	if err := db.WithContext(ctx).Where("item_id = ? AND provider = ? AND provider_type = ?", item.ID, "tmdb", "movie").First(&externalID).Error; err != nil {
		t.Fatalf("load external id: %v", err)
	}
	var identity database.CatalogIdentity
	if err := db.WithContext(ctx).Where("item_id = ? AND provider = ? AND identity_type = ?", item.ID, "tmdb", "movie").First(&identity).Error; err != nil {
		t.Fatalf("load identity: %v", err)
	}
	var image database.ItemImage
	if err := db.WithContext(ctx).Where("item_id = ? AND image_type = ? AND is_selected = ?", item.ID, "poster", true).First(&image).Error; err != nil {
		t.Fatalf("load selected image: %v", err)
	}
	if image.SourceID == nil || *image.SourceID != sourceID {
		t.Fatalf("expected image source id %d, got %#v", sourceID, image)
	}
	var peopleCount int64
	if err := db.WithContext(ctx).Model(&database.ItemPerson{}).Where("item_id = ?", item.ID).Count(&peopleCount).Error; err != nil {
		t.Fatalf("count item people: %v", err)
	}
	if peopleCount != 2 {
		t.Fatalf("expected actor and director rows, got %d", peopleCount)
	}
	var doc database.CatalogSearchDocument
	if err := db.WithContext(ctx).Where("item_id = ?", item.ID).First(&doc).Error; err != nil {
		t.Fatalf("load search document: %v", err)
	}
}
