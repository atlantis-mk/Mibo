package catalog

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestUserMetadataFavoritesUseProjectionContext(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db)
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Favorite Metadata", SortTitle: "Favorite Metadata", GovernanceStatus: database.ReviewStateAccepted}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.LibraryMetadataProjection{LibraryID: 7, MetadataItemID: item.ID, ItemType: item.ItemType, Title: item.Title, AvailabilityStatus: database.ProjectionAvailabilityAvailable, ResourceCount: 1, AvailableCount: 1, LastProjectedAt: time.Now().UTC()}).Error; err != nil {
		t.Fatalf("create projection: %v", err)
	}

	favorite, err := svc.SetFavorite(ctx, 11, item.ID, true)
	if err != nil {
		t.Fatalf("set favorite: %v", err)
	}
	if !favorite.Favorite || favorite.Item.MetadataItemID != item.ID || favorite.Item.LibraryID != 7 {
		t.Fatalf("unexpected favorite entry: %#v", favorite)
	}
	favorites, err := svc.ListFavorites(ctx, 11, 10)
	if err != nil {
		t.Fatalf("list favorites: %v", err)
	}
	if len(favorites) != 1 || favorites[0].Item.MetadataItemID != item.ID {
		t.Fatalf("unexpected favorites: %#v", favorites)
	}
}
