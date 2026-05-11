package metadata

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

func TestApplyNormalizedMetadataItemDetailWritesMetadataOwnedTables(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db, config.MetadataConfig{}, nil)
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Old", SortTitle: "Old", GovernanceStatus: database.ReviewStatePending}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	confidence := 0.95
	detail := NormalizedMetadataDetail{Provider: database.MetadataProviderTypeTMDB, ProviderType: "movie", ExternalID: "movie:42", Title: "New Movie", Year: intPtrForMetadataItemPolicy(2026), Images: []NormalizedMetadataImage{{ImageType: "poster", URL: "https://image/poster.jpg", Selected: true}}, Tags: []NormalizedMetadataTag{{Kind: "genre", Name: "Drama"}}, People: []NormalizedMetadataPerson{{Name: "Actor One", Role: "actor", Character: "Lead", SortOrder: 1}}, ExternalIDs: []NormalizedMetadataExternalID{{Provider: database.MetadataProviderTypeTMDB, ProviderType: "movie", ExternalID: "movie:42", IsPrimary: true}}}
	plan := MetadataExecutionPlan{LibraryID: 7, MetadataProfileName: "Movie Profile", PreferredMetadataLanguage: "zh-CN", PreferredImageLanguage: "zh"}
	result := metadataItemOperationResult(OperationTypeMatch, MetadataOperationRequest{OriginMetadataItemID: item.ID}, item, plan)

	if _, _, err := svc.applyNormalizedMetadataItemDetail(ctx, item.CreatedAt, item, plan, result, detail, NormalizedMetadataCandidate{Provider: database.MetadataProviderTypeTMDB, ExternalID: "movie:42"}, confidence, OperationTypeMatch); err != nil {
		t.Fatalf("apply metadata item detail: %v", err)
	}

	var stored database.MetadataItem
	if err := db.WithContext(ctx).First(&stored, item.ID).Error; err != nil {
		t.Fatalf("load metadata item: %v", err)
	}
	if stored.Title != "New Movie" || stored.Year == nil || *stored.Year != 2026 {
		t.Fatalf("expected canonical metadata fields, got %#v", stored)
	}
	assertMetadataItemPolicyCount(t, ctx, db, &database.MetadataItemSource{}, 1)
	assertMetadataItemPolicyCount(t, ctx, db, &database.MetadataItemFieldState{}, 4)
	assertMetadataItemPolicyCount(t, ctx, db, &database.MetadataExternalID{}, 1)
	assertMetadataItemPolicyCount(t, ctx, db, &database.MetadataItemImage{}, 1)
	assertMetadataItemPolicyCount(t, ctx, db, &database.MetadataItemPerson{}, 1)
	assertMetadataItemPolicyCount(t, ctx, db, &database.MetadataItemTag{}, 1)
	var source database.MetadataItemSource
	if err := db.WithContext(ctx).First(&source).Error; err != nil {
		t.Fatalf("load metadata item source: %v", err)
	}
	if source.Language != "zh-CN" || source.MetadataProfileName != "Movie Profile" || source.TriggeringLibraryID == nil || *source.TriggeringLibraryID != 7 || source.FallbackSummaryJSON == "" {
		t.Fatalf("expected source trigger context, got %#v", source)
	}
}

func TestLoadMetadataItemProviderIdentityUsesMetadataExternalIDs(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db, config.MetadataConfig{}, nil)
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Movie", SortTitle: "Movie", GovernanceStatus: database.ReviewStatePending}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	confidence := 0.97
	if err := db.WithContext(ctx).Create(&database.MetadataExternalID{MetadataItemID: item.ID, Provider: database.MetadataProviderTypeTMDB, ProviderType: "movie", ExternalID: "movie:42", IsPrimary: true, Confidence: &confidence}).Error; err != nil {
		t.Fatalf("create external id: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.MetadataItemSource{MetadataItemID: item.ID, SourceType: "provider", SourceName: database.MetadataProviderTypeTMDB, ExternalID: "movie:42", ProviderInstanceName: "tmdb-main", FetchedAt: item.CreatedAt}).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}

	providerInstance, externalID, gotConfidence, err := svc.loadMetadataItemProviderIdentity(ctx, item.ID, database.MetadataProviderTypeTMDB, "movie")
	if err != nil {
		t.Fatalf("load metadata identity: %v", err)
	}
	if providerInstance != "tmdb-main" || externalID != "movie:42" || gotConfidence != confidence {
		t.Fatalf("unexpected identity shortcut: instance=%q external=%q confidence=%f", providerInstance, externalID, gotConfidence)
	}
}

func TestLoadMetadataItemLocalScannerEvidenceFromResourceEvidence(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db, config.MetadataConfig{}, nil)
	library := database.Library{Name: "Movies", Type: "movies", RootPath: "/library", Status: "active"}
	if err := db.WithContext(ctx).Create(&library).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	item := database.MetadataItem{ItemType: database.MetadataItemTypeMovie, ContentForm: database.MetadataContentFormStandard, Title: "Old", SortTitle: "Old", GovernanceStatus: database.ReviewStatePending}
	resource := database.Resource{StableResourceKey: "resource:1", ResourceType: database.ResourceTypePlayable, ResourceShape: database.ResourceShapeSingleFile, Status: "available"}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		t.Fatalf("create metadata item: %v", err)
	}
	if err := db.WithContext(ctx).Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}
	payload := `{"metadata_sidecars":[{"path":"/library/movie.nfo","parse_status":"parsed","hints":{"title":"Local Title","year":2026},"external_ids":{"tmdb":"42"}}]}`
	if err := db.WithContext(ctx).Create(&database.ResourceMetadataLink{ResourceID: resource.ID, MetadataItemID: item.ID, Role: database.ResourceLinkRolePrimary}).Error; err != nil {
		t.Fatalf("create resource metadata link: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ResourceLibraryLink{ResourceID: resource.ID, LibraryID: library.ID, Status: "available", FirstSeenAt: item.CreatedAt, LastSeenAt: item.CreatedAt, EvidenceJSON: payload}).Error; err != nil {
		t.Fatalf("create resource library link: %v", err)
	}

	evidence, err := svc.loadMetadataItemLocalScannerEvidence(ctx, item.ID)
	if err != nil {
		t.Fatalf("load local scanner evidence: %v", err)
	}
	detail, ok := localEvidenceDetail(evidence, database.MetadataItemTypeMovie)
	if !ok || detail.Title != "Local Title" || detail.Year == nil || *detail.Year != 2026 || len(detail.ExternalIDs) != 1 {
		t.Fatalf("unexpected local evidence detail: ok=%v detail=%#v", ok, detail)
	}
	assertMetadataItemPolicyCount(t, ctx, db, &database.MetadataItemSource{}, 1)
}

func TestLocalEvidenceDetailSupportsImagesOnly(t *testing.T) {
	evidence := LocalScannerEvidence{Images: []LocalScannerImageEvidence{{ImageType: "poster", URL: "https://image/poster.jpg", Priority: 1}}}
	detail, ok := localEvidenceDetail(evidence, database.MetadataItemTypeMovie)
	if !ok {
		t.Fatal("expected images-only local evidence to be applicable")
	}
	if len(detail.Images) != 1 || detail.Images[0].ImageType != "poster" || detail.Images[0].URL != "https://image/poster.jpg" || !detail.Images[0].Selected {
		t.Fatalf("unexpected images-only detail: %#v", detail)
	}
}

func TestMetadataItemOperationDedupKeyIsStableAcrossLibraries(t *testing.T) {
	profileID := uint(3)
	resultA := MetadataOperationResult{Operation: OperationTypeMatch, TargetMetadataItemID: 10, Plan: MetadataExecutionPlanSummary{LibraryID: 1, MetadataProfileID: &profileID, MetadataProfileName: "Default", PreferredMetadataLanguage: "en", DetailProviders: []MetadataPlanProviderSummary{{ID: 7, Name: "tmdb", ProviderType: database.MetadataProviderTypeTMDB}}}}
	resultB := resultA
	resultB.Plan.LibraryID = 2
	if metadataOperationDeduplicationKey(resultA) != metadataOperationDeduplicationKey(resultB) {
		t.Fatalf("expected shared metadata/profile/provider requests to deduplicate across libraries")
	}
	resultB.Plan.PreferredMetadataLanguage = "zh-CN"
	if metadataOperationDeduplicationKey(resultA) == metadataOperationDeduplicationKey(resultB) {
		t.Fatalf("expected language context to affect metadata deduplication key")
	}
}

func assertMetadataItemPolicyCount(t *testing.T, ctx context.Context, db *gorm.DB, model any, want int64) {
	t.Helper()
	var got int64
	if err := db.WithContext(ctx).Model(model).Count(&got).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	if got != want {
		t.Fatalf("expected %d %T rows, got %d", want, model, got)
	}
}

func intPtrForMetadataItemPolicy(value int) *int {
	return &value
}
