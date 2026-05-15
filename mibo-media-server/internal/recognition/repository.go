package recognition

import (
	"context"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

type ManifestScope struct {
	ManifestKey       string
	LibraryID         uint
	MediaSourceID     uint
	LibraryPathID     *uint
	StorageProvider   string
	RootPath          string
	ScopePath         string
	ClassifierVersion string
	Fingerprint       string
	EvidenceJSON      string
	ObservedAt        time.Time
}

type ManifestGraph struct {
	Manifest       database.RecognitionManifest
	GraphNodes     []database.MediaGraphNode
	GraphEdges     []database.MediaGraphEdge
	GroupDecisions []database.MediaGraphClassification
	Candidates     []database.RecognitionCandidate
	Evidence       []database.RecognitionEvidence
	Decisions      []database.RecognitionDecision
	Conflicts      []database.RecognitionConflict
}

func (r *Repository) UpsertManifest(ctx context.Context, scope ManifestScope) (database.RecognitionManifest, error) {
	manifestKey := strings.TrimSpace(scope.ManifestKey)
	if manifestKey == "" {
		manifestKey = ManifestKey(Scope{StorageProvider: scope.StorageProvider, RootPath: scope.RootPath, ScopePath: scope.ScopePath}, scope.ClassifierVersion)
	}
	manifest := database.RecognitionManifest{
		ManifestKey:       manifestKey,
		LibraryID:         scope.LibraryID,
		MediaSourceID:     scope.MediaSourceID,
		LibraryPathID:     scope.LibraryPathID,
		StorageProvider:   strings.TrimSpace(scope.StorageProvider),
		RootPath:          strings.TrimSpace(scope.RootPath),
		ScopePath:         strings.TrimSpace(scope.ScopePath),
		ClassifierVersion: strings.TrimSpace(scope.ClassifierVersion),
		Fingerprint:       strings.TrimSpace(scope.Fingerprint),
		Status:            "pending",
		EvidenceJSON:      strings.TrimSpace(scope.EvidenceJSON),
		ObservedAt:        scope.ObservedAt,
	}
	if manifest.ObservedAt.IsZero() {
		manifest.ObservedAt = time.Now().UTC()
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "manifest_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"library_id", "media_source_id", "library_path_id", "storage_provider", "root_path", "scope_path", "classifier_version", "fingerprint", "status", "evidence_json", "observed_at", "resolved_at", "superseded_at", "updated_at",
		}),
	}).Create(&manifest).Error; err != nil {
		return database.RecognitionManifest{}, err
	}
	if err := r.db.WithContext(ctx).Where("manifest_key = ?", manifest.ManifestKey).First(&manifest).Error; err != nil {
		return database.RecognitionManifest{}, err
	}
	return manifest, nil
}

func (r *Repository) ReplaceCandidatesAndEvidence(ctx context.Context, manifestID uint, candidates []database.RecognitionCandidate, evidence []database.RecognitionEvidence) error {
	if manifestID == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("manifest_id = ?", manifestID).Delete(&database.RecognitionCandidate{}).Error; err != nil {
			return err
		}
		if err := tx.Where("manifest_id = ?", manifestID).Delete(&database.RecognitionEvidence{}).Error; err != nil {
			return err
		}
		if len(candidates) > 0 {
			if err := NewRepository(tx).SaveCandidates(ctx, candidates); err != nil {
				return err
			}
		}
		if len(evidence) > 0 {
			if err := tx.CreateInBatches(&evidence, 100).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) LoadManifestByKey(ctx context.Context, manifestKey string) (database.RecognitionManifest, bool, error) {
	var manifest database.RecognitionManifest
	err := r.db.WithContext(ctx).Where("manifest_key = ?", strings.TrimSpace(manifestKey)).First(&manifest).Error
	if err == nil {
		return manifest, true, nil
	}
	if err == gorm.ErrRecordNotFound {
		return database.RecognitionManifest{}, false, nil
	}
	return database.RecognitionManifest{}, false, err
}

func (r *Repository) LoadManifestGraph(ctx context.Context, manifestID uint) (ManifestGraph, error) {
	var graph ManifestGraph
	if manifestID == 0 {
		return graph, nil
	}
	if err := r.db.WithContext(ctx).First(&graph.Manifest, manifestID).Error; err != nil {
		return graph, err
	}
	if err := r.db.WithContext(ctx).Where("manifest_id = ?", manifestID).Order("id asc").Find(&graph.Candidates).Error; err != nil {
		return graph, err
	}
	if err := r.db.WithContext(ctx).Where("manifest_id = ?", manifestID).Order("id asc").Find(&graph.GraphNodes).Error; err != nil {
		return graph, err
	}
	if err := r.db.WithContext(ctx).Where("manifest_id = ?", manifestID).Order("id asc").Find(&graph.GraphEdges).Error; err != nil {
		return graph, err
	}
	if err := r.db.WithContext(ctx).Where("manifest_id = ?", manifestID).Order("id asc").Find(&graph.GroupDecisions).Error; err != nil {
		return graph, err
	}
	if err := r.db.WithContext(ctx).Where("manifest_id = ?", manifestID).Order("id asc").Find(&graph.Evidence).Error; err != nil {
		return graph, err
	}
	if err := r.db.WithContext(ctx).Where("manifest_id = ?", manifestID).Order("id asc").Find(&graph.Decisions).Error; err != nil {
		return graph, err
	}
	if err := r.db.WithContext(ctx).Where("manifest_id = ?", manifestID).Order("id asc").Find(&graph.Conflicts).Error; err != nil {
		return graph, err
	}
	return graph, nil
}

func (r *Repository) SaveCandidates(ctx context.Context, candidates []database.RecognitionCandidate) error {
	if len(candidates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "manifest_id"}, {Name: "candidate_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"candidate_type", "candidate_role", "parent_candidate_key", "target_metadata_id", "target_resource_id", "primary_inventory_id", "canonical_key", "variant_key", "edition_key", "resource_shape", "review_state", "confidence", "evidence_json", "alternatives_json", "affected_files_json", "materialized_at", "superseded_at", "updated_at",
		}),
	}).CreateInBatches(&candidates, 100).Error
}

func (r *Repository) SaveMediaGraph(ctx context.Context, manifestID uint, nodes []database.MediaGraphNode, edges []database.MediaGraphEdge, classifications []database.MediaGraphClassification) error {
	if manifestID == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("manifest_id = ?", manifestID).Delete(&database.MediaGraphClassification{}).Error; err != nil {
			return err
		}
		if err := tx.Where("manifest_id = ?", manifestID).Delete(&database.MediaGraphEdge{}).Error; err != nil {
			return err
		}
		if err := tx.Where("manifest_id = ?", manifestID).Delete(&database.MediaGraphNode{}).Error; err != nil {
			return err
		}
		if len(nodes) > 0 {
			if err := tx.CreateInBatches(&nodes, 100).Error; err != nil {
				return err
			}
		}
		if len(edges) > 0 {
			if err := tx.CreateInBatches(&edges, 100).Error; err != nil {
				return err
			}
		}
		if len(classifications) > 0 {
			if err := tx.CreateInBatches(&classifications, 100).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) SaveEvidence(ctx context.Context, evidence []database.RecognitionEvidence) error {
	if len(evidence) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(&evidence, 100).Error
}

func (r *Repository) ReplaceEvidenceForInventoryFiles(ctx context.Context, manifestID uint, inventoryFileIDs []uint, evidence []database.RecognitionEvidence) error {
	if manifestID == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Where("manifest_id = ?", manifestID)
		if len(inventoryFileIDs) > 0 {
			query = query.Where("inventory_file_id IN ?", inventoryFileIDs)
		}
		if err := query.Delete(&database.RecognitionEvidence{}).Error; err != nil {
			return err
		}
		if len(evidence) == 0 {
			return nil
		}
		return tx.CreateInBatches(&evidence, 100).Error
	})
}

func (r *Repository) SaveDecisions(ctx context.Context, decisions []database.RecognitionDecision) error {
	if len(decisions) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(&decisions, 100).Error
}

func (r *Repository) SaveConflicts(ctx context.Context, conflicts []database.RecognitionConflict) error {
	if len(conflicts) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(&conflicts, 100).Error
}

func (r *Repository) ReplaceDecisionsAndConflicts(ctx context.Context, manifestID uint, decisions []database.RecognitionDecision, conflicts []database.RecognitionConflict) error {
	if manifestID == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("manifest_id = ?", manifestID).Delete(&database.RecognitionDecision{}).Error; err != nil {
			return err
		}
		if err := tx.Where("manifest_id = ?", manifestID).Delete(&database.RecognitionConflict{}).Error; err != nil {
			return err
		}
		if len(conflicts) > 0 {
			if err := tx.CreateInBatches(&conflicts, 100).Error; err != nil {
				return err
			}
		}
		if len(decisions) > 0 {
			if err := tx.CreateInBatches(&decisions, 100).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) SupersedeManifest(ctx context.Context, manifestID uint, at time.Time) error {
	if manifestID == 0 {
		return nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"status": "superseded", "superseded_at": at}
		if err := tx.Model(&database.RecognitionManifest{}).Where("id = ?", manifestID).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Model(&database.RecognitionCandidate{}).Where("manifest_id = ?", manifestID).Update("superseded_at", at).Error; err != nil {
			return err
		}
		return tx.Model(&database.RecognitionDecision{}).Where("manifest_id = ?", manifestID).Update("superseded_at", at).Error
	})
}

func (r *Repository) DeleteLibraryManifests(ctx context.Context, libraryID uint) error {
	if libraryID == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		subquery := tx.Model(&database.RecognitionManifest{}).Select("id").Where("library_id = ?", libraryID)
		if err := tx.Where("manifest_id IN (?)", subquery).Delete(&database.RecognitionConflict{}).Error; err != nil {
			return err
		}
		if err := tx.Where("manifest_id IN (?)", subquery).Delete(&database.RecognitionDecision{}).Error; err != nil {
			return err
		}
		if err := tx.Where("manifest_id IN (?)", subquery).Delete(&database.RecognitionEvidence{}).Error; err != nil {
			return err
		}
		if err := tx.Where("manifest_id IN (?)", subquery).Delete(&database.RecognitionCandidate{}).Error; err != nil {
			return err
		}
		return tx.Where("library_id = ?", libraryID).Delete(&database.RecognitionManifest{}).Error
	})
}
