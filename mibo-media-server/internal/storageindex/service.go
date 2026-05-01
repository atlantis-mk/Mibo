package storageindex

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ObservationStatusPresent = "present"
	ObservationStatusMissing = "missing"
	ObservationStatusUnknown = "unknown"
)

type Service struct {
	db  *gorm.DB
	now func() time.Time
}

type ObservationInput struct {
	LibraryID         uint
	StorageProvider   string
	StoragePath       string
	IsDir             bool
	SizeBytes         int64
	ModifiedAt        *time.Time
	StableIdentityKey string
	Hashes            map[string]string
	ProviderName      string
	ObjectType        string
	ProviderMeta      map[string]string
}

type FailureInput struct {
	LibraryID       uint
	StorageProvider string
	StoragePath     string
	Reason          string
	Error           error
	ErrorMessage    string
}

type ObserveTreeInput struct {
	LibraryID       uint
	StorageProvider string
	RootPath        string
	Provider        storage.Provider
	Refresh         bool
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) UpsertPresent(ctx context.Context, input ObservationInput) (database.StorageIndexEntry, error) {
	if err := validateObservationInput(input); err != nil {
		return database.StorageIndexEntry{}, err
	}
	observedAt := s.now()
	pathValue := normalizePath(input.StoragePath)
	provider := strings.TrimSpace(input.StorageProvider)
	hashesJSON, err := encodeStringMap(input.Hashes)
	if err != nil {
		return database.StorageIndexEntry{}, err
	}
	metaJSON, err := encodeStringMap(input.ProviderMeta)
	if err != nil {
		return database.StorageIndexEntry{}, err
	}

	entry := database.StorageIndexEntry{
		LibraryID:         input.LibraryID,
		StorageProvider:   provider,
		StoragePath:       pathValue,
		IsDir:             input.IsDir,
		SizeBytes:         input.SizeBytes,
		ModifiedAt:        input.ModifiedAt,
		StableIdentityKey: strings.TrimSpace(input.StableIdentityKey),
		HashesJSON:        hashesJSON,
		ProviderName:      strings.TrimSpace(input.ProviderName),
		ObjectType:        strings.TrimSpace(input.ObjectType),
		ProviderMetaJSON:  metaJSON,
		ObservationStatus: ObservationStatusPresent,
		FirstObservedAt:   observedAt,
		LastObservedAt:    observedAt,
	}
	updates := map[string]any{
		"is_dir":              entry.IsDir,
		"size_bytes":          entry.SizeBytes,
		"modified_at":         entry.ModifiedAt,
		"stable_identity_key": entry.StableIdentityKey,
		"hashes_json":         entry.HashesJSON,
		"provider_name":       entry.ProviderName,
		"object_type":         entry.ObjectType,
		"provider_meta_json":  entry.ProviderMetaJSON,
		"observation_status":  ObservationStatusPresent,
		"last_observed_at":    observedAt,
		"missing_since":       nil,
		"last_error":          "",
		"updated_at":          observedAt,
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "library_id"}, {Name: "storage_provider"}, {Name: "storage_path"}},
		DoUpdates: clause.Assignments(updates),
	}).Create(&entry).Error; err != nil {
		return database.StorageIndexEntry{}, err
	}
	return s.Find(ctx, input.LibraryID, provider, pathValue)
}

func (s *Service) ObservationFromObject(libraryID uint, providerName string, object storage.Object) ObservationInput {
	return ObservationInput{
		LibraryID:         libraryID,
		StorageProvider:   providerName,
		StoragePath:       object.Path,
		IsDir:             object.IsDir,
		SizeBytes:         object.Size,
		ModifiedAt:        object.Modified,
		StableIdentityKey: strings.TrimSpace(object.StableIdentity),
		Hashes:            storage.CloneStringMap(object.HashInfo),
		ProviderName:      strings.TrimSpace(object.Provider),
		ObjectType:        strings.TrimSpace(object.ObjectType),
		ProviderMeta:      object.SanitizedProviderMeta(),
	}
}

func (s *Service) ObserveTree(ctx context.Context, input ObserveTreeInput) ([]database.StorageIndexEntry, error) {
	if input.LibraryID == 0 {
		return nil, errors.New("library id is required")
	}
	if input.Provider == nil {
		return nil, errors.New("storage provider is required")
	}
	providerName := strings.TrimSpace(input.StorageProvider)
	if providerName == "" {
		providerName = input.Provider.Name()
	}
	root := normalizePath(input.RootPath)
	if root == "" {
		return nil, errors.New("root path is required")
	}
	previous, err := s.ListScoped(ctx, input.LibraryID, root)
	if err != nil {
		return nil, err
	}
	rootObject, err := input.Provider.Get(ctx, storage.GetRequest{Path: root})
	if err != nil {
		_, _ = s.RecordFailure(ctx, FailureInput{LibraryID: input.LibraryID, StorageProvider: providerName, StoragePath: root, Reason: "get_root_failed", Error: err})
		return nil, err
	}
	entries := make([]database.StorageIndexEntry, 0)
	observed := make(map[string]struct{})
	rootEntry, err := s.UpsertPresent(ctx, s.ObservationFromObject(input.LibraryID, providerName, rootObject))
	if err != nil {
		return nil, err
	}
	observed[rootEntry.StoragePath] = struct{}{}
	entries = append(entries, rootEntry)
	if !rootObject.IsDir {
		return entries, nil
	}
	if err := s.observeDirectory(ctx, input.Provider, input.LibraryID, providerName, root, input.Refresh, &entries, observed); err != nil {
		return nil, err
	}
	for _, entry := range previous {
		if _, ok := observed[entry.StoragePath]; ok {
			continue
		}
		missing, err := s.MarkMissing(ctx, entry.LibraryID, entry.StorageProvider, entry.StoragePath)
		if err != nil {
			return nil, err
		}
		entries = append(entries, missing)
	}
	return entries, nil
}

func (s *Service) observeDirectory(ctx context.Context, provider storage.Provider, libraryID uint, providerName string, dirPath string, refresh bool, entries *[]database.StorageIndexEntry, observed map[string]struct{}) error {
	objects, err := listAllDirectoryObjects(ctx, provider, dirPath, refresh)
	if err != nil {
		_, _ = s.RecordFailure(ctx, FailureInput{LibraryID: libraryID, StorageProvider: providerName, StoragePath: dirPath, Reason: "list_failed", Error: err})
		return err
	}
	for _, object := range objects {
		entry, err := s.UpsertPresent(ctx, s.ObservationFromObject(libraryID, providerName, object))
		if err != nil {
			return err
		}
		observed[entry.StoragePath] = struct{}{}
		*entries = append(*entries, entry)
		if object.IsDir {
			if err := s.observeDirectory(ctx, provider, libraryID, providerName, object.Path, refresh, entries, observed); err != nil {
				return err
			}
		}
	}
	return nil
}

func listAllDirectoryObjects(ctx context.Context, provider storage.Provider, dirPath string, refresh bool) ([]storage.Object, error) {
	const pageSize = 1000
	var all []storage.Object
	for page := 1; ; page++ {
		objects, err := provider.List(ctx, storage.ListRequest{Path: dirPath, Refresh: refresh, Page: page, PerPage: pageSize})
		if err != nil {
			return nil, err
		}
		all = append(all, objects...)
		if len(objects) < pageSize {
			break
		}
	}
	return all, nil
}

func (s *Service) MarkMissing(ctx context.Context, libraryID uint, storageProvider string, storagePath string) (database.StorageIndexEntry, error) {
	if libraryID == 0 {
		return database.StorageIndexEntry{}, errors.New("library id is required")
	}
	provider := strings.TrimSpace(storageProvider)
	pathValue := normalizePath(storagePath)
	if provider == "" || pathValue == "" {
		return database.StorageIndexEntry{}, errors.New("storage provider and path are required")
	}
	missingAt := s.now()
	updates := map[string]any{
		"observation_status": ObservationStatusMissing,
		"last_observed_at":   missingAt,
		"missing_since":      missingAt,
		"updated_at":         missingAt,
	}
	if err := s.db.WithContext(ctx).Model(&database.StorageIndexEntry{}).
		Where("library_id = ? AND storage_provider = ? AND storage_path = ?", libraryID, provider, pathValue).
		Updates(updates).Error; err != nil {
		return database.StorageIndexEntry{}, err
	}
	return s.Find(ctx, libraryID, provider, pathValue)
}

func (s *Service) Find(ctx context.Context, libraryID uint, storageProvider string, storagePath string) (database.StorageIndexEntry, error) {
	var entry database.StorageIndexEntry
	err := s.db.WithContext(ctx).
		Where("library_id = ? AND storage_provider = ? AND storage_path = ?", libraryID, strings.TrimSpace(storageProvider), normalizePath(storagePath)).
		First(&entry).Error
	return entry, err
}

func (s *Service) ListScoped(ctx context.Context, libraryID uint, rootPath string) ([]database.StorageIndexEntry, error) {
	if libraryID == 0 {
		return nil, errors.New("library id is required")
	}
	query := s.db.WithContext(ctx).Where("library_id = ?", libraryID)
	query = applyScopedPathFilter(query, "storage_path", rootPath)
	var entries []database.StorageIndexEntry
	err := query.Order("storage_path asc").Find(&entries).Error
	return entries, err
}

func (s *Service) RecordFailure(ctx context.Context, input FailureInput) (database.StorageObservationFailure, error) {
	if input.LibraryID == 0 {
		return database.StorageObservationFailure{}, errors.New("library id is required")
	}
	provider := strings.TrimSpace(input.StorageProvider)
	pathValue := normalizePath(input.StoragePath)
	if provider == "" || pathValue == "" {
		return database.StorageObservationFailure{}, errors.New("storage provider and path are required")
	}
	message := strings.TrimSpace(input.ErrorMessage)
	if message == "" && input.Error != nil {
		message = input.Error.Error()
	}
	failure := database.StorageObservationFailure{
		LibraryID:       input.LibraryID,
		StorageProvider: provider,
		StoragePath:     pathValue,
		Reason:          defaultString(input.Reason, "observation_failed"),
		ErrorMessage:    message,
		ObservedAt:      s.now(),
	}
	return failure, s.db.WithContext(ctx).Create(&failure).Error
}

func validateObservationInput(input ObservationInput) error {
	if input.LibraryID == 0 {
		return errors.New("library id is required")
	}
	if strings.TrimSpace(input.StorageProvider) == "" || normalizePath(input.StoragePath) == "" {
		return errors.New("storage provider and path are required")
	}
	return nil
}

func applyScopedPathFilter(query *gorm.DB, column string, rootPath string) *gorm.DB {
	root := normalizePath(rootPath)
	if root == "" || root == string(filepath.Separator) {
		return query
	}
	pattern := escapeSQLLikePattern(root) + string(filepath.Separator) + "%"
	return query.Where(column+" = ? OR "+column+" LIKE ? ESCAPE '\\'", root, pattern)
}

func normalizePath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	cleaned := filepath.Clean(trimmed)
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func encodeStringMap(input map[string]string) (string, error) {
	if len(input) == 0 {
		return "", nil
	}
	cleaned := make(map[string]string, len(input))
	keys := make([]string, 0, len(input))
	for key, value := range input {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		cleaned[key] = value
		keys = append(keys, key)
	}
	if len(cleaned) == 0 {
		return "", nil
	}
	sort.Strings(keys)
	ordered := make(map[string]string, len(cleaned))
	for _, key := range keys {
		ordered[key] = cleaned[key]
	}
	encoded, err := json.Marshal(ordered)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func escapeSQLLikePattern(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}

func defaultString(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
