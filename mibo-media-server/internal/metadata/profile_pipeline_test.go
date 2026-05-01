package metadata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/settings"
)

func TestMatchCatalogItemSkipsUnavailableProviderInstanceInProfile(t *testing.T) {
	tmdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 101, "title": "Matched Movie", "release_date": "2024-02-02"}}})
		case "/movie/101":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 101, "title": "Matched Movie", "release_date": "2024-02-02", "runtime": 121, "genres": []map[string]any{}, "credits": map[string]any{"cast": []map[string]any{}, "crew": []map[string]any{}}, "images": map[string]any{"logos": []map[string]any{}}, "videos": map[string]any{"results": []map[string]any{}}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer tmdb.Close()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	settingsSvc := settings.NewService(db, config.MetadataConfig{TMDB: config.TMDBConfig{BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: time.Second}})
	enabledMigrated := true
	if _, err := settingsSvc.UpsertMetadataProviderInstance(ctx, 0, settings.UpdateMetadataProviderInstanceInput{Name: database.MigratedDefaultTMDBProviderInstanceName, ProviderType: database.MetadataProviderTypeTMDB, Enabled: &enabledMigrated, AvailabilityStatus: database.MetadataProviderAvailabilityAvailable, TMDB: &settings.MetadataProviderInput{APIKey: "fallback-key", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: "1s"}}); err != nil {
		t.Fatalf("create migrated tmdb provider instance: %v", err)
	}
	disabled := false
	primary, err := settingsSvc.UpsertMetadataProviderInstance(ctx, 0, settings.UpdateMetadataProviderInstanceInput{Name: "tmdb-cooldown", ProviderType: database.MetadataProviderTypeTMDB, Enabled: &disabled, AvailabilityStatus: database.MetadataProviderAvailabilityCooldown, TMDB: &settings.MetadataProviderInput{APIKey: "unused", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: "1s"}})
	if err != nil {
		t.Fatalf("create unavailable provider instance: %v", err)
	}
	enabled := true
	secondary, err := settingsSvc.UpsertMetadataProviderInstance(ctx, 0, settings.UpdateMetadataProviderInstanceInput{Name: "tmdb-active", ProviderType: database.MetadataProviderTypeTMDB, Enabled: &enabled, AvailabilityStatus: database.MetadataProviderAvailabilityAvailable, TMDB: &settings.MetadataProviderInput{APIKey: "active-key", BaseURL: tmdb.URL, ImageBaseURL: tmdb.URL + "/images", Language: "en-US", Timeout: "1s"}})
	if err != nil {
		t.Fatalf("create active provider instance: %v", err)
	}
	fallbackEnabled := true
	profile, err := settingsSvc.UpsertMetadataProfile(ctx, 0, settings.UpdateMetadataProfileInput{Name: "custom-fallback", SearchProviderIDs: []uint{primary.ID, secondary.ID}, DetailProviderIDs: []uint{primary.ID, secondary.ID}, ImageProviderIDs: []uint{secondary.ID}, PeopleProviderIDs: []uint{secondary.ID}, HierarchyProviderIDs: []uint{secondary.ID}, FallbackEnabled: &fallbackEnabled})
	if err != nil {
		t.Fatalf("create metadata profile: %v", err)
	}
	if err := database.EnsureLibraryPolicyDefaults(db, 1); err != nil {
		t.Fatalf("ensure policy defaults: %v", err)
	}
	if _, err := settingsSvc.UpdateLibraryMetadataStrategy(ctx, 1, settings.UpdateLibraryMetadataStrategyInput{TemplateProfileID: profile.ID, SearchProviderIDs: []uint{primary.ID, secondary.ID}, DetailProviderIDs: []uint{primary.ID, secondary.ID}, ImageProviderIDs: []uint{secondary.ID}, PeopleProviderIDs: []uint{secondary.ID}, HierarchyProviderIDs: []uint{secondary.ID}}); err != nil {
		t.Fatalf("save metadata strategy: %v", err)
	}
	item, err := catalog.NewService(db).CreateItem(ctx, catalog.CreateItemInput{LibraryID: 1, Type: catalog.ItemTypeMovie, Title: "MovieA", Path: "/movies/MovieA.2024.mkv", SortKey: "MovieA"})
	if err != nil {
		t.Fatalf("create catalog item: %v", err)
	}
	if _, err := NewService(db, config.MetadataConfig{}, settingsSvc).MatchCatalogItemOperation(ctx, item.ID); err != nil {
		t.Fatalf("match catalog item: %v", err)
	}
	var source database.MetadataSource
	if err := db.WithContext(ctx).Where("item_id = ?", item.ID).Order("id desc").First(&source).Error; err != nil {
		t.Fatalf("load metadata source: %v", err)
	}
	if source.ProviderInstanceName != "tmdb-active" || source.MetadataProfileName != "custom-fallback" {
		t.Fatalf("expected active provider provenance, got %#v", source)
	}
}

func TestMetadataOperationMatchabilityDocumentsStrategyShapes(t *testing.T) {
	tmdb := settings.ResolvedMetadataProviderInstance{Record: database.MetadataProviderInstance{ID: 1, Name: "tmdb", ProviderType: database.MetadataProviderTypeTMDB, Enabled: true, AvailabilityStatus: database.MetadataProviderAvailabilityAvailable}, Configured: true, Operational: true}
	metatube := settings.ResolvedMetadataProviderInstance{Record: database.MetadataProviderInstance{ID: 2, Name: "metatube", ProviderType: database.MetadataProviderTypeMetaTube, Enabled: true, AvailabilityStatus: database.MetadataProviderAvailabilityAvailable}, Configured: true, Operational: true}
	localScan := settings.ResolvedMetadataProviderInstance{Record: database.MetadataProviderInstance{ID: 3, Name: database.BuiltInLocalScanProviderInstanceName, ProviderType: database.MetadataProviderTypeLocalScan, Enabled: true, AvailabilityStatus: database.MetadataProviderAvailabilityAvailable}, Configured: true, Operational: true}
	disabled := settings.ResolvedMetadataProviderInstance{Record: database.MetadataProviderInstance{ID: 4, Name: "disabled", ProviderType: database.MetadataProviderTypeTMDB, Enabled: false, AvailabilityStatus: database.MetadataProviderAvailabilityAvailable}, Configured: true, Operational: false}
	cooldownUntil := time.Now().UTC().Add(time.Hour)
	cooldown := settings.ResolvedMetadataProviderInstance{Record: database.MetadataProviderInstance{ID: 5, Name: "cooldown", ProviderType: database.MetadataProviderTypeTMDB, Enabled: true, AvailabilityStatus: database.MetadataProviderAvailabilityCooldown, CooldownUntil: &cooldownUntil}, Configured: true, Operational: false}

	tests := []struct {
		name         string
		plan         MetadataExecutionPlan
		operation    string
		providerType string
		runnable     bool
	}{
		{name: "tmdb only match", plan: MetadataExecutionPlan{SearchProviders: []settings.ResolvedMetadataProviderInstance{tmdb}, DetailProviders: []settings.ResolvedMetadataProviderInstance{tmdb}}, operation: OperationTypeMatch, runnable: true},
		{name: "metatube only match", plan: MetadataExecutionPlan{SearchProviders: []settings.ResolvedMetadataProviderInstance{metatube}, DetailProviders: []settings.ResolvedMetadataProviderInstance{metatube}}, operation: OperationTypeMatch, runnable: true},
		{name: "local evidence only match", plan: MetadataExecutionPlan{DetailProviders: []settings.ResolvedMetadataProviderInstance{localScan}, LocalEvidenceEnabled: true}, operation: OperationTypeMatch, runnable: true},
		{name: "no provider match", plan: MetadataExecutionPlan{}, operation: OperationTypeMatch, runnable: false},
		{name: "disabled provider match", plan: MetadataExecutionPlan{SearchProviders: []settings.ResolvedMetadataProviderInstance{disabled}, DetailProviders: []settings.ResolvedMetadataProviderInstance{disabled}}, operation: OperationTypeMatch, runnable: false},
		{name: "cooldown provider match", plan: MetadataExecutionPlan{SearchProviders: []settings.ResolvedMetadataProviderInstance{cooldown}, DetailProviders: []settings.ResolvedMetadataProviderInstance{cooldown}}, operation: OperationTypeMatch, runnable: false},
		{name: "metatube refetch", plan: MetadataExecutionPlan{DetailProviders: []settings.ResolvedMetadataProviderInstance{metatube}}, operation: OperationTypeRefetch, providerType: database.MetadataProviderTypeMetaTube, runnable: true},
		{name: "manual apply disallowed provider", plan: MetadataExecutionPlan{DetailProviders: []settings.ResolvedMetadataProviderInstance{tmdb}}, operation: OperationTypeManualApply, providerType: database.MetadataProviderTypeMetaTube, runnable: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := metadataOperationMatchability(tc.plan, tc.operation, tc.providerType)
			if got.Runnable != tc.runnable {
				t.Fatalf("expected runnable=%v, got %#v", tc.runnable, got)
			}
		})
	}
}
