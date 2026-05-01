package library

import (
	"context"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/storage"
)

func TestScanExclusionDecisionUsesPersistedExclusions(t *testing.T) {
	t.Parallel()

	db, svc, libraryRecord := newIdentityScanService(t)
	ctx := context.Background()
	exclusion := database.ScanExclusion{LibraryID: libraryRecord.ID, StorageProvider: "stable-test", StableIdentityKey: "stable-ad", StoragePath: "/library/renamed.mkv", Reason: ScanExclusionReasonAdvertisement, Enabled: true}
	if err := db.WithContext(ctx).Create(&exclusion).Error; err != nil {
		t.Fatalf("create exclusion: %v", err)
	}

	decision, err := svc.scanExclusionDecision(ctx, libraryRecord, "stable-test", storage.Object{Path: "/library/renamed-again.mkv", StableIdentity: "stable-ad"})
	if err != nil {
		t.Fatalf("scan exclusion decision: %v", err)
	}
	if !decision.Excluded || decision.Source != scanExclusionSkipUserExclusion || decision.Reason != ScanExclusionReasonAdvertisement {
		t.Fatalf("expected persisted exclusion decision, got %#v", decision)
	}
}

func TestScanExclusionDecisionIgnoresDisabledExclusions(t *testing.T) {
	t.Parallel()

	db, svc, libraryRecord := newIdentityScanService(t)
	ctx := context.Background()
	exclusion := database.ScanExclusion{LibraryID: libraryRecord.ID, StorageProvider: "stable-test", StoragePath: "/library/movie.mkv", Reason: ScanExclusionReasonAdvertisement, Enabled: false}
	if err := db.WithContext(ctx).Create(&exclusion).Error; err != nil {
		t.Fatalf("create exclusion: %v", err)
	}

	decision, err := svc.scanExclusionDecision(ctx, libraryRecord, "stable-test", storage.Object{Path: "/library/movie.mkv"})
	if err != nil {
		t.Fatalf("scan exclusion decision: %v", err)
	}
	if decision.Excluded {
		t.Fatalf("expected disabled exclusion to be ignored, got %#v", decision)
	}
}

func TestScanExclusionDecisionAppliesUserExclusionToSameFilename(t *testing.T) {
	t.Parallel()

	db, svc, libraryRecord := newIdentityScanService(t)
	ctx := context.Background()
	exclusion := database.ScanExclusion{LibraryID: libraryRecord.ID, StorageProvider: "stable-test", StoragePath: "/library/Movie A/promo.mp4", Reason: ScanExclusionReasonAdvertisement, Enabled: true}
	if err := db.WithContext(ctx).Create(&exclusion).Error; err != nil {
		t.Fatalf("create exclusion: %v", err)
	}

	decision, err := svc.scanExclusionDecision(ctx, libraryRecord, "stable-test", storage.Object{Path: "/library/Movie B/promo.mp4"})
	if err != nil {
		t.Fatalf("scan exclusion decision: %v", err)
	}
	if decision.Excluded {
		t.Fatalf("expected manual file exclusion not to match same filename, got %#v", decision)
	}

	if _, err := svc.SetScanExclusionEnabled(ctx, SetScanExclusionEnabledInput{ExclusionID: exclusion.ID, Enabled: false}); err != nil {
		t.Fatalf("disable exclusion: %v", err)
	}
	decision, err = svc.scanExclusionDecision(ctx, libraryRecord, "stable-test", storage.Object{Path: "/library/Movie B/promo.mp4"})
	if err != nil {
		t.Fatalf("scan exclusion decision after disable: %v", err)
	}
	if decision.Excluded {
		t.Fatalf("expected disabled user exclusion to stop same filename match, got %#v", decision)
	}
}

func TestScanExclusionDecisionAppliesFilenameRuleAcrossSources(t *testing.T) {
	t.Parallel()

	db, svc, libraryRecord := newIdentityScanService(t)
	ctx := context.Background()
	rule := database.FilenameExclusionRule{NormalizedFilename: "promo.mp4", Reason: ScanExclusionReasonAdvertisement, Enabled: true}
	if err := db.WithContext(ctx).Create(&rule).Error; err != nil {
		t.Fatalf("create filename rule: %v", err)
	}

	decision, err := svc.scanExclusionDecision(ctx, libraryRecord, "stable-test", storage.Object{Path: "/library/Movie B/PROMO.mp4"})
	if err != nil {
		t.Fatalf("scan exclusion decision: %v", err)
	}
	if !decision.Excluded || decision.Source != scanExclusionFilenameRuleSource(rule) || decision.Reason != ScanExclusionReasonAdvertisement {
		t.Fatalf("expected filename rule exclusion, got %#v", decision)
	}

	decision, err = svc.scanExclusionDecision(ctx, libraryRecord, "other-provider", storage.Object{Path: "/other/Movie B/PROMO.mp4"})
	if err != nil {
		t.Fatalf("cross-source scan exclusion decision: %v", err)
	}
	if !decision.Excluded || decision.Source != scanExclusionFilenameRuleSource(rule) {
		t.Fatalf("expected filename rule to apply across sources, got %#v", decision)
	}

	decision, err = svc.scanExclusionDecision(ctx, libraryRecord, "stable-test", storage.Object{Path: "/library/Movie B/promo.mkv"})
	if err != nil {
		t.Fatalf("extension scan exclusion decision: %v", err)
	}
	if decision.Excluded {
		t.Fatalf("expected different extension not to match, got %#v", decision)
	}
}

func TestFilenameRuleRestoreExceptionTakesPriority(t *testing.T) {
	t.Parallel()

	db, svc, libraryRecord := newIdentityScanService(t)
	ctx := context.Background()
	rule := database.FilenameExclusionRule{NormalizedFilename: "promo.mp4", Reason: ScanExclusionReasonAdvertisement, Enabled: true}
	if err := db.WithContext(ctx).Create(&rule).Error; err != nil {
		t.Fatalf("create filename rule: %v", err)
	}
	restore := database.FilenameExclusionRestore{RuleID: rule.ID, StableIdentityKey: "stable-restored", StoragePath: "/library/Movie A/promo.mp4"}
	if err := db.WithContext(ctx).Create(&restore).Error; err != nil {
		t.Fatalf("create restore: %v", err)
	}

	decision, err := svc.scanExclusionDecision(ctx, libraryRecord, "stable-test", storage.Object{Path: "/library/Movie A/promo.mp4", StableIdentity: "stable-restored"})
	if err != nil {
		t.Fatalf("restored scan exclusion decision: %v", err)
	}
	if decision.Excluded {
		t.Fatalf("expected restore exception to allow file, got %#v", decision)
	}

	decision, err = svc.scanExclusionDecision(ctx, libraryRecord, "stable-test", storage.Object{Path: "/library/Movie B/promo.mp4"})
	if err != nil {
		t.Fatalf("other scan exclusion decision: %v", err)
	}
	if !decision.Excluded || decision.Source != scanExclusionFilenameRuleSource(rule) {
		t.Fatalf("expected other same-name file to remain excluded, got %#v", decision)
	}
}

func TestScanExclusionDecisionAppliesConfigurableRule(t *testing.T) {
	t.Parallel()

	_, svc, libraryRecord := newIdentityScanService(t)
	ctx := context.Background()
	rule, err := svc.CreateScanExclusionRule(ctx, ScanExclusionRuleInput{Name: "Skip promo files", RuleType: ScanExclusionRuleTypeFilenameToken, Value: "promo", Reason: ScanExclusionReasonAdvertisement, Enabled: true})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	decision, err := svc.scanExclusionDecision(ctx, libraryRecord, "stable-test", storage.Object{Path: "/library/Movie A/promo.mkv"})
	if err != nil {
		t.Fatalf("scan exclusion decision: %v", err)
	}
	if !decision.Excluded || decision.Reason != ScanExclusionReasonAdvertisement || decision.Source != scanExclusionRuleSource(rule) {
		t.Fatalf("expected configurable rule decision, got %#v", decision)
	}

	if _, err := svc.SetScanExclusionRuleEnabled(ctx, SetScanExclusionRuleEnabledInput{RuleID: rule.ID, Enabled: false}); err != nil {
		t.Fatalf("disable rule: %v", err)
	}
	decision, err = svc.scanExclusionDecision(ctx, libraryRecord, "stable-test", storage.Object{Path: "/library/Movie A/promo.mkv"})
	if err != nil {
		t.Fatalf("scan exclusion decision after disable: %v", err)
	}
	if decision.Excluded {
		t.Fatalf("expected disabled configurable rule to stop matching, got %#v", decision)
	}
}

func TestScanExclusionDecisionAppliesGlobalAndScopedRules(t *testing.T) {
	t.Parallel()

	db, svc, libraryRecord := newIdentityScanService(t)
	ctx := context.Background()
	otherLibrary := database.Library{Name: "Other", Type: "movies", RootPath: "/other", Status: "active"}
	if err := db.WithContext(ctx).Create(&otherLibrary).Error; err != nil {
		t.Fatalf("create other library: %v", err)
	}
	if _, err := svc.CreateScanExclusionRule(ctx, ScanExclusionRuleInput{Name: "Skip global promo", RuleType: ScanExclusionRuleTypeFilenameToken, Value: "promo", Reason: ScanExclusionReasonAdvertisement, Enabled: true}); err != nil {
		t.Fatalf("create global rule: %v", err)
	}
	if _, err := svc.CreateScanExclusionRule(ctx, ScanExclusionRuleInput{LibraryID: &libraryRecord.ID, Name: "Skip scoped local", RuleType: ScanExclusionRuleTypeFilenameToken, Value: "localad", Reason: ScanExclusionReasonAdvertisement, Enabled: true}); err != nil {
		t.Fatalf("create scoped rule: %v", err)
	}
	if _, err := svc.CreateScanExclusionRule(ctx, ScanExclusionRuleInput{LibraryID: &otherLibrary.ID, Name: "Skip other only", RuleType: ScanExclusionRuleTypeFilenameToken, Value: "otherad", Reason: ScanExclusionReasonAdvertisement, Enabled: true}); err != nil {
		t.Fatalf("create other scoped rule: %v", err)
	}

	globalDecision, err := svc.scanExclusionDecision(ctx, libraryRecord, "stable-test", storage.Object{Path: "/library/Movie promo.mkv"})
	if err != nil {
		t.Fatalf("global scan exclusion decision: %v", err)
	}
	if !globalDecision.Excluded {
		t.Fatalf("expected global rule to apply, got %#v", globalDecision)
	}
	scopedDecision, err := svc.scanExclusionDecision(ctx, libraryRecord, "stable-test", storage.Object{Path: "/library/Movie localad.mkv"})
	if err != nil {
		t.Fatalf("scoped scan exclusion decision: %v", err)
	}
	if !scopedDecision.Excluded {
		t.Fatalf("expected matching scoped rule to apply, got %#v", scopedDecision)
	}
	otherDecision, err := svc.scanExclusionDecision(ctx, libraryRecord, "stable-test", storage.Object{Path: "/library/Movie otherad.mkv"})
	if err != nil {
		t.Fatalf("other scoped scan exclusion decision: %v", err)
	}
	if otherDecision.Excluded {
		t.Fatalf("expected other-library scoped rule not to apply, got %#v", otherDecision)
	}
}

func TestScanExclusionRuleScopeValidationAndUniqueness(t *testing.T) {
	t.Parallel()

	_, svc, libraryRecord := newIdentityScanService(t)
	ctx := context.Background()
	missingLibraryID := uint(9999)
	if _, err := svc.CreateScanExclusionRule(ctx, ScanExclusionRuleInput{LibraryID: &missingLibraryID, Name: "Missing library", RuleType: ScanExclusionRuleTypeFilenameToken, Value: "promo", Reason: ScanExclusionReasonAdvertisement, Enabled: true}); err == nil {
		t.Fatalf("expected missing library scope to be rejected")
	}
	if _, err := svc.CreateScanExclusionRule(ctx, ScanExclusionRuleInput{Name: "Global promo", RuleType: ScanExclusionRuleTypeFilenameToken, Value: "promo", Reason: ScanExclusionReasonAdvertisement, Enabled: true}); err != nil {
		t.Fatalf("create global rule: %v", err)
	}
	if _, err := svc.CreateScanExclusionRule(ctx, ScanExclusionRuleInput{Name: "Duplicate global promo", RuleType: ScanExclusionRuleTypeFilenameToken, Value: "PROMO", Reason: ScanExclusionReasonAdvertisement, Enabled: true}); err == nil {
		t.Fatalf("expected duplicate global rule to be rejected")
	}
	if _, err := svc.CreateScanExclusionRule(ctx, ScanExclusionRuleInput{LibraryID: &libraryRecord.ID, Name: "Scoped promo", RuleType: ScanExclusionRuleTypeFilenameToken, Value: "promo", Reason: ScanExclusionReasonAdvertisement, Enabled: true}); err != nil {
		t.Fatalf("expected equivalent scoped rule to be allowed: %v", err)
	}
	if _, err := svc.CreateScanExclusionRule(ctx, ScanExclusionRuleInput{LibraryID: &libraryRecord.ID, Name: "Duplicate scoped promo", RuleType: ScanExclusionRuleTypeFilenameToken, Value: "promo", Reason: ScanExclusionReasonAdvertisement, Enabled: true}); err == nil {
		t.Fatalf("expected duplicate scoped rule to be rejected")
	}
}

func TestDeleteLibraryRemovesOnlyScopedScanExclusionRules(t *testing.T) {
	t.Parallel()

	db, svc, libraryRecord := newIdentityScanService(t)
	ctx := context.Background()
	otherLibrary := database.Library{Name: "Other", Type: "movies", RootPath: "/other", Status: "active"}
	if err := db.WithContext(ctx).Create(&otherLibrary).Error; err != nil {
		t.Fatalf("create other library: %v", err)
	}
	globalRule, err := svc.CreateScanExclusionRule(ctx, ScanExclusionRuleInput{Name: "Global promo", RuleType: ScanExclusionRuleTypeFilenameToken, Value: "globalpromo", Reason: ScanExclusionReasonAdvertisement, Enabled: true})
	if err != nil {
		t.Fatalf("create global rule: %v", err)
	}
	libraryRule, err := svc.CreateScanExclusionRule(ctx, ScanExclusionRuleInput{LibraryID: &libraryRecord.ID, Name: "Scoped promo", RuleType: ScanExclusionRuleTypeFilenameToken, Value: "scopedpromo", Reason: ScanExclusionReasonAdvertisement, Enabled: true})
	if err != nil {
		t.Fatalf("create scoped rule: %v", err)
	}
	otherRule, err := svc.CreateScanExclusionRule(ctx, ScanExclusionRuleInput{LibraryID: &otherLibrary.ID, Name: "Other promo", RuleType: ScanExclusionRuleTypeFilenameToken, Value: "otherpromo", Reason: ScanExclusionReasonAdvertisement, Enabled: true})
	if err != nil {
		t.Fatalf("create other scoped rule: %v", err)
	}

	if err := svc.DeleteLibrary(ctx, libraryRecord.ID); err != nil {
		t.Fatalf("delete library: %v", err)
	}
	var remaining []database.ScanExclusionRule
	if err := db.WithContext(ctx).Where("id IN ?", []uint{globalRule.ID, libraryRule.ID, otherRule.ID}).Order("id asc").Find(&remaining).Error; err != nil {
		t.Fatalf("list remaining rules: %v", err)
	}
	remainingIDs := map[uint]bool{}
	for _, rule := range remaining {
		remainingIDs[rule.ID] = true
	}
	if !remainingIDs[globalRule.ID] || remainingIDs[libraryRule.ID] || !remainingIDs[otherRule.ID] {
		t.Fatalf("unexpected remaining scoped rules: %#v", remainingIDs)
	}
}

func TestScanExclusionDecisionUserExclusionTakesPriorityOverRule(t *testing.T) {
	t.Parallel()

	db, svc, libraryRecord := newIdentityScanService(t)
	ctx := context.Background()
	if _, err := svc.CreateScanExclusionRule(ctx, ScanExclusionRuleInput{Name: "Skip promo files", RuleType: ScanExclusionRuleTypeFilenameToken, Value: "promo", Reason: ScanExclusionReasonAdvertisement, Enabled: true}); err != nil {
		t.Fatalf("create rule: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ScanExclusion{LibraryID: libraryRecord.ID, StorageProvider: "stable-test", StoragePath: "/library/Movie A/promo.mkv", Reason: ScanExclusionReasonWrongImport, Enabled: true}).Error; err != nil {
		t.Fatalf("create user exclusion: %v", err)
	}

	decision, err := svc.scanExclusionDecision(ctx, libraryRecord, "stable-test", storage.Object{Path: "/library/Movie A/promo.mkv"})
	if err != nil {
		t.Fatalf("scan exclusion decision: %v", err)
	}
	if !decision.Excluded || decision.Source != scanExclusionSkipUserExclusion || decision.Reason != ScanExclusionReasonWrongImport {
		t.Fatalf("expected user exclusion priority, got %#v", decision)
	}
}

func TestScanExclusionRuleValidationRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	_, svc, _ := newIdentityScanService(t)
	ctx := context.Background()
	invalid := []ScanExclusionRuleInput{
		{Name: "Missing value", RuleType: ScanExclusionRuleTypeFilenameToken, Value: "", Reason: ScanExclusionReasonAdvertisement, Enabled: true},
		{Name: "Bad type", RuleType: "substring", Value: "ad", Reason: ScanExclusionReasonAdvertisement, Enabled: true},
		{Name: "Broad pattern", RuleType: ScanExclusionRuleTypePathPattern, Value: "*", Reason: ScanExclusionReasonAdvertisement, Enabled: true},
	}
	for _, input := range invalid {
		if _, err := svc.CreateScanExclusionRule(ctx, input); err == nil {
			t.Fatalf("expected invalid rule input to fail: %#v", input)
		}
	}
}

func TestDefaultScanExclusionRulesCanBeDisabled(t *testing.T) {
	t.Parallel()

	_, svc, libraryRecord := newIdentityScanService(t)
	ctx := context.Background()
	rules, err := svc.ListScanExclusionRules(ctx)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	var commercialRule database.ScanExclusionRule
	for _, rule := range rules {
		if rule.System && rule.RuleType == ScanExclusionRuleTypeFilenameToken && rule.Value == "commercial" {
			commercialRule = rule
			break
		}
	}
	if commercialRule.ID == 0 {
		t.Fatalf("expected seeded commercial rule in %#v", rules)
	}
	if _, err := svc.SetScanExclusionRuleEnabled(ctx, SetScanExclusionRuleEnabledInput{RuleID: commercialRule.ID, Enabled: false}); err != nil {
		t.Fatalf("disable commercial rule: %v", err)
	}
	decision, err := svc.scanExclusionDecision(ctx, libraryRecord, "stable-test", storage.Object{Path: "/library/Movie A/commercial.mkv"})
	if err != nil {
		t.Fatalf("scan exclusion decision: %v", err)
	}
	if decision.Excluded {
		t.Fatalf("expected disabled seeded rule to stop matching, got %#v", decision)
	}
}

func TestScanExclusionDecisionSameFilenameDoesNotCrossLibrary(t *testing.T) {
	t.Parallel()

	db, svc, libraryRecord := newIdentityScanService(t)
	ctx := context.Background()
	otherLibrary := database.Library{Name: "Other", Type: "movies", RootPath: "/other", Status: "active"}
	if err := db.WithContext(ctx).Create(&otherLibrary).Error; err != nil {
		t.Fatalf("create other library: %v", err)
	}
	exclusion := database.ScanExclusion{LibraryID: otherLibrary.ID, StorageProvider: "stable-test", StoragePath: "/other/Movie B/promo.mp4", Reason: ScanExclusionReasonAdvertisement, Enabled: true}
	if err := db.WithContext(ctx).Create(&exclusion).Error; err != nil {
		t.Fatalf("create exclusion: %v", err)
	}

	decision, err := svc.scanExclusionDecision(ctx, libraryRecord, "stable-test", storage.Object{Path: "/library/Movie A/promo.mp4"})
	if err != nil {
		t.Fatalf("scan exclusion decision: %v", err)
	}
	if decision.Excluded {
		t.Fatalf("expected same filename exclusion not to cross library, got %#v", decision)
	}
}

func TestAdvertisementPathMarkers(t *testing.T) {
	t.Parallel()

	positive := []string{
		"/movies/Movie A/ad.mp4",
		"/movies/Movie A/Movie A - ads.mkv",
		"/movies/Movie A/advertisement.mp4",
		"/movies/Movie A/commercial.mov",
		"/movies/Movie A/广告.mp4",
		"/movies/Movie A/ads/clip.mp4",
		"/movies/Movie A/commercials/clip.mp4",
	}
	for _, candidate := range positive {
		if !hasExplicitAdvertisementPathMarker(candidate) {
			t.Fatalf("expected %q to match advertisement marker", candidate)
		}
	}

	negative := []string{
		"/movies/Ad Astra/Ad Astra.mkv",
		"/movies/Adventure Movie/Adventure Movie.mp4",
		"/shows/Show/Season 01/Show.S01E01.mkv",
		"/movies/Movie A/trailer.mkv",
		"/movies/Movie A/sample.mp4",
		"/movies/Movie A/featurette.mkv",
	}
	for _, candidate := range negative {
		if hasExplicitAdvertisementPathMarker(candidate) {
			t.Fatalf("expected %q not to match advertisement marker", candidate)
		}
	}
}

func TestSupportedScanExclusionReasons(t *testing.T) {
	t.Parallel()

	for _, reason := range []string{ScanExclusionReasonAdvertisement, ScanExclusionReasonUnwanted, ScanExclusionReasonDuplicate, ScanExclusionReasonWrongImport, ScanExclusionReasonOther} {
		if !supportedScanExclusionReason(reason) {
			t.Fatalf("expected reason %q to be supported", reason)
		}
	}
	if supportedScanExclusionReason("") || supportedScanExclusionReason("promo") {
		t.Fatalf("expected unsupported reason to be rejected")
	}
}

func TestScanExclusionDecisionPathFallback(t *testing.T) {
	t.Parallel()

	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: t.TempDir() + "/mibo.db"})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, jobs.NewService(db))
	ctx := context.Background()
	libraryRecord := database.Library{Name: "Movies", Type: "movies", RootPath: "/library", Status: "active"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	if err := db.WithContext(ctx).Create(&database.ScanExclusion{LibraryID: libraryRecord.ID, StorageProvider: "stable-test", StoragePath: "/library/Movie-ad.mkv", Reason: ScanExclusionReasonAdvertisement, Enabled: true}).Error; err != nil {
		t.Fatalf("create exclusion: %v", err)
	}

	decision, err := svc.scanExclusionDecision(ctx, libraryRecord, "stable-test", storage.Object{Path: "/library/Movie-ad.mkv"})
	if err != nil {
		t.Fatalf("scan exclusion decision: %v", err)
	}
	if !decision.Excluded || decision.Source != scanExclusionSkipUserExclusion {
		t.Fatalf("expected path fallback exclusion, got %#v", decision)
	}
}

func TestScanExclusionDecisionUsesOrphanedLibraryExclusionAfterRecreate(t *testing.T) {
	t.Parallel()

	db, svc, libraryRecord := newIdentityScanService(t)
	ctx := context.Background()
	if err := db.WithContext(ctx).Create(&database.ScanExclusion{LibraryID: libraryRecord.ID, StorageProvider: "stable-test", StoragePath: "/library/Movie A/promo.mp4", Reason: ScanExclusionReasonAdvertisement, Enabled: true}).Error; err != nil {
		t.Fatalf("create exclusion: %v", err)
	}
	if err := db.WithContext(ctx).Delete(&database.Library{}, libraryRecord.ID).Error; err != nil {
		t.Fatalf("delete library: %v", err)
	}
	recreated := database.Library{Name: "Movies Recreated", Type: "movies", RootPath: "/library", Status: "active"}
	if err := db.WithContext(ctx).Create(&recreated).Error; err != nil {
		t.Fatalf("create recreated library: %v", err)
	}

	decision, err := svc.scanExclusionDecision(ctx, recreated, "stable-test", storage.Object{Path: "/library/Movie A/promo.mp4"})
	if err != nil {
		t.Fatalf("scan exclusion decision: %v", err)
	}
	if !decision.Excluded || decision.Source != scanExclusionSkipUserExclusion {
		t.Fatalf("expected orphaned exclusion to apply to recreated library, got %#v", decision)
	}
}

func TestFilenameRuleDecisionSurvivesLibraryRecreate(t *testing.T) {
	t.Parallel()

	db, svc, libraryRecord := newIdentityScanService(t)
	ctx := context.Background()
	rule := database.FilenameExclusionRule{NormalizedFilename: "promo.mp4", Reason: ScanExclusionReasonAdvertisement, Enabled: true}
	if err := db.WithContext(ctx).Create(&rule).Error; err != nil {
		t.Fatalf("create filename rule: %v", err)
	}
	if err := db.WithContext(ctx).Delete(&database.Library{}, libraryRecord.ID).Error; err != nil {
		t.Fatalf("delete library: %v", err)
	}
	recreated := database.Library{Name: "Movies Recreated", Type: "movies", RootPath: "/library", Status: "active"}
	if err := db.WithContext(ctx).Create(&recreated).Error; err != nil {
		t.Fatalf("create recreated library: %v", err)
	}

	decision, err := svc.scanExclusionDecision(ctx, recreated, "stable-test", storage.Object{Path: "/library/Other/promo.mp4"})
	if err != nil {
		t.Fatalf("scan exclusion decision: %v", err)
	}
	if !decision.Excluded || decision.Source != scanExclusionFilenameRuleSource(rule) {
		t.Fatalf("expected filename rule to apply to recreated library, got %#v", decision)
	}
}
