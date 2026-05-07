package settings

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestListMetadataProviderInstancesIncludesBuiltInLocalScan(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db, config.MetadataConfig{})

	providers, err := svc.ListMetadataProviderInstances(ctx)
	if err != nil {
		t.Fatalf("list metadata provider instances: %v", err)
	}
	for _, provider := range providers {
		if provider.ProviderType != database.MetadataProviderTypeLocalScan {
			continue
		}
		if provider.Name != database.BuiltInLocalScanProviderInstanceName || !provider.SystemManaged || !provider.Locked || !provider.Configured {
			t.Fatalf("expected built-in local scan provider semantics, got %#v", provider)
		}
		return
	}
	t.Fatalf("expected built-in local scan provider instance to exist")
}

func TestUpsertTVDBMetadataProviderInstance(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db, config.MetadataConfig{TVDB: config.TVDBConfig{BaseURL: "https://tvdb.example/v4", Language: "en", Timeout: time.Second}})

	provider, err := svc.UpsertMetadataProviderInstance(ctx, 0, UpdateMetadataProviderInstanceInput{Name: "series-tvdb", ProviderType: database.MetadataProviderTypeTVDB, TVDB: &MetadataProviderInput{APIKey: "tvdb-key", BaseURL: "https://tvdb.override/v4", Language: "zh", Timeout: "2s"}})
	if err != nil {
		t.Fatalf("create tvdb provider instance: %v", err)
	}
	if provider.ProviderType != database.MetadataProviderTypeTVDB || !provider.Configured || provider.TVDB == nil {
		t.Fatalf("expected configured TVDB provider, got %#v", provider)
	}
	if provider.TVDB.BaseURL != "https://tvdb.override/v4" || provider.TVDB.Language != "zh" || provider.TVDB.Timeout != "2s" {
		t.Fatalf("unexpected TVDB config view: %#v", provider.TVDB)
	}

	_, err = svc.UpdateLibraryMetadataStrategy(ctx, 1, UpdateLibraryMetadataStrategyInput{SearchProviderIDs: []uint{provider.ID}})
	if err == nil {
		t.Fatalf("expected TVDB search stage validation error until execution support is implemented")
	}
}

func TestUpsertMetaTubeMetadataProviderInstance(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db, config.MetadataConfig{MetaTube: config.MetaTubeConfig{BaseURL: "https://metatube.example", Timeout: time.Second, FallbackEnabled: true}})
	fallbackEnabled := false

	provider, err := svc.UpsertMetadataProviderInstance(ctx, 0, UpdateMetadataProviderInstanceInput{Name: "private-metatube", ProviderType: database.MetadataProviderTypeMetaTube, MetaTube: &MetadataProviderInput{APIKey: "secret-token", BaseURL: "https://metatube.override", UpstreamProviderFilter: "fanza", FallbackEnabled: &fallbackEnabled, Timeout: "2s"}})
	if err != nil {
		t.Fatalf("create metatube provider instance: %v", err)
	}
	if provider.ProviderType != database.MetadataProviderTypeMetaTube || !provider.Configured || provider.MetaTube == nil || !provider.MetaTube.APIKeyMasked {
		t.Fatalf("expected configured MetaTube provider with masked token, got %#v", provider)
	}
	if provider.MetaTube.BaseURL != "https://metatube.override" || provider.MetaTube.UpstreamProviderFilter != "fanza" || provider.MetaTube.FallbackEnabled || provider.MetaTube.Timeout != "2s" {
		t.Fatalf("unexpected MetaTube config view: %#v", provider.MetaTube)
	}

	resolved, err := svc.ResolveMetadataProviderInstance(ctx, provider.ID)
	if err != nil {
		t.Fatalf("resolve metatube provider instance: %v", err)
	}
	if !resolved.Operational || resolved.MetaTube.Token != "secret-token" || resolved.MetaTube.UpstreamProviderFilter != "fanza" || resolved.MetaTube.FallbackEnabled {
		t.Fatalf("unexpected resolved MetaTube provider: %#v", resolved)
	}

	providers, err := svc.ListMetadataProviderInstances(ctx)
	if err != nil {
		t.Fatalf("list metadata provider instances: %v", err)
	}
	found := false
	for _, item := range providers {
		if item.ID == provider.ID {
			found = item.MetaTube != nil && item.MetaTube.APIKeyMasked
		}
	}
	if !found {
		t.Fatalf("expected listed MetaTube provider with masked config, got %#v", providers)
	}
}

func TestMetadataStrategyAcceptsMetaTubeSearchDetailAndRejectsHierarchy(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db, config.MetadataConfig{})
	provider, err := svc.UpsertMetadataProviderInstance(ctx, 0, UpdateMetadataProviderInstanceInput{Name: "metatube", ProviderType: database.MetadataProviderTypeMetaTube, MetaTube: &MetadataProviderInput{BaseURL: "https://metatube.example", Timeout: "1s"}})
	if err != nil {
		t.Fatalf("create metatube provider instance: %v", err)
	}
	if err := database.EnsureLibraryPolicyDefaults(db, 1); err != nil {
		t.Fatalf("ensure library policy defaults: %v", err)
	}

	if _, err := svc.UpdateLibraryMetadataStrategy(ctx, 1, UpdateLibraryMetadataStrategyInput{SearchProviderIDs: []uint{provider.ID}, DetailProviderIDs: []uint{provider.ID}}); err != nil {
		t.Fatalf("expected MetaTube search/detail strategy to be accepted: %v", err)
	}
	if _, err := svc.UpdateLibraryMetadataStrategy(ctx, 1, UpdateLibraryMetadataStrategyInput{HierarchyProviderIDs: []uint{provider.ID}}); err == nil {
		t.Fatalf("expected MetaTube hierarchy stage validation error")
	}
	if _, err := svc.UpsertMetadataProfile(ctx, 0, UpdateMetadataProfileInput{Name: "metatube-template", SearchProviderIDs: []uint{provider.ID}, DetailProviderIDs: []uint{provider.ID}}); err != nil {
		t.Fatalf("expected MetaTube template search/detail to be accepted: %v", err)
	}
	if _, err := svc.UpsertMetadataProfile(ctx, 0, UpdateMetadataProfileInput{Name: "bad-metatube-template", HierarchyProviderIDs: []uint{provider.ID}}); err == nil {
		t.Fatalf("expected MetaTube template hierarchy validation error")
	}
}

func TestUpdateLibraryMetadataStrategyRejectsLocalScanInSearchStage(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db, config.MetadataConfig{})

	providers, err := svc.ListMetadataProviderInstances(ctx)
	if err != nil {
		t.Fatalf("list metadata provider instances: %v", err)
	}
	localScanID := uint(0)
	for _, provider := range providers {
		if provider.ProviderType == database.MetadataProviderTypeLocalScan {
			localScanID = provider.ID
			break
		}
	}
	if localScanID == 0 {
		t.Fatalf("expected local scan provider id")
	}
	if err := database.EnsureLibraryPolicyDefaults(db, 1); err != nil {
		t.Fatalf("ensure library policy defaults: %v", err)
	}

	_, err = svc.UpdateLibraryMetadataStrategy(ctx, 1, UpdateLibraryMetadataStrategyInput{SearchProviderIDs: []uint{localScanID}})
	if err == nil {
		t.Fatalf("expected local_scan search stage validation error")
	}
}

func TestLibraryMetadataStrategyUsesLocalScanAsDetailFallback(t *testing.T) {
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	ctx := context.Background()
	svc := NewService(db, config.MetadataConfig{})
	if err := database.EnsureLibraryPolicyDefaults(db, 1); err != nil {
		t.Fatalf("ensure library policy defaults: %v", err)
	}

	if _, err := svc.UpdateLibraryMetadataStrategy(ctx, 1, UpdateLibraryMetadataStrategyInput{}); err != nil {
		t.Fatalf("update empty library metadata strategy: %v", err)
	}
	resolved, err := svc.ResolveLibraryMetadataProfile(ctx, 1)
	if err != nil {
		t.Fatalf("resolve library metadata profile: %v", err)
	}
	if len(resolved.DetailProviders) != 1 || resolved.DetailProviders[0].Record.ProviderType != database.MetadataProviderTypeLocalScan {
		t.Fatalf("expected local_scan detail fallback, got %#v", resolved.DetailProviders)
	}
}
