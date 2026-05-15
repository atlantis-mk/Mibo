package recognition

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"gorm.io/gorm"
)

type Materializer struct {
	db *gorm.DB
}

func NewMaterializer(db *gorm.DB) *Materializer {
	return &Materializer{db: db}
}

type MaterializeResult struct {
	MetadataIDs           []uint
	ResourceIDs           []uint
	ProjectionMetadataIDs []uint
	ProjectionResourceIDs []uint
}

func (m *Materializer) MaterializeMetadata(ctx context.Context, graph ManifestGraph, decisions []database.RecognitionDecision) (MaterializeResult, error) {
	result := MaterializeResult{}
	candidatesByKey := make(map[string]database.RecognitionCandidate, len(graph.Candidates))
	for _, candidate := range graph.Candidates {
		candidatesByKey[strings.TrimSpace(candidate.CandidateKey)] = candidate
	}
	scopePath := strings.TrimSpace(graph.Manifest.ScopePath)
	for _, decision := range decisions {
		if decision.Outcome != DecisionOutcomeAccepted || decision.TargetKind != CandidateTypeWork && decision.TargetKind != CandidateTypeEpisode {
			continue
		}
		candidate, ok := candidatesByKey[strings.TrimSpace(decision.TargetKey)]
		if !ok {
			continue
		}
		item, err := m.upsertMetadataForCandidate(ctx, scopePath, candidate, candidatesByKey)
		if err != nil {
			return result, err
		}
		if item.ID != 0 {
			result.MetadataIDs = append(result.MetadataIDs, item.ID)
			result.ProjectionMetadataIDs = appendUniqueUint(result.ProjectionMetadataIDs, item.ID)
		}
	}
	return result, nil
}

func (m *Materializer) MaterializeResources(ctx context.Context, graph ManifestGraph, decisions []database.RecognitionDecision) (MaterializeResult, error) {
	result := MaterializeResult{}
	candidatesByKey := make(map[string]database.RecognitionCandidate, len(graph.Candidates))
	for _, candidate := range graph.Candidates {
		candidatesByKey[strings.TrimSpace(candidate.CandidateKey)] = candidate
	}
	metadataByKey, err := m.metadataByCandidateKey(ctx, graph.Manifest.ScopePath, graph.Candidates)
	if err != nil {
		return result, err
	}
	inventorySvc := inventory.NewService(m.db)
	for _, decision := range decisions {
		if decision.Outcome != DecisionOutcomeAccepted || decision.TargetKind != CandidateTypePlayableResource {
			continue
		}
		candidate, ok := candidatesByKey[strings.TrimSpace(decision.TargetKey)]
		if !ok || candidate.PrimaryInventoryID == nil {
			continue
		}
		resource, err := inventorySvc.UpsertResource(ctx, inventory.UpsertResourceInput{StableResourceKey: strings.TrimSpace(candidate.CandidateKey), ResourceType: database.ResourceTypePlayable, ResourceShape: database.NormalizeResourceShape(candidate.ResourceShape), DisplayName: titleFromCandidate(candidate), Edition: editionLabel(candidate), QualityLabel: variantLabel(candidate), Status: inventory.AssetStatusAvailable, ProbeStatus: "pending", TechnicalSummaryJSON: resolverResourceSummary(candidate)})
		if err != nil {
			return result, err
		}
		if _, err := inventorySvc.LinkResourceToFile(ctx, inventory.LinkResourceFileInput{ResourceID: resource.ID, InventoryFileID: *candidate.PrimaryInventoryID, Role: database.ResourceFileRoleSource}); err != nil {
			return result, err
		}
		if graph.Manifest.LibraryID != 0 {
			if _, err := inventorySvc.AttachResourceToLibrary(ctx, inventory.AttachResourceLibraryInput{ResourceID: resource.ID, LibraryID: graph.Manifest.LibraryID, Status: inventory.AssetStatusAvailable, EvidenceJSON: candidate.EvidenceJSON, ReviewState: database.ReviewStateAccepted}); err != nil {
				return result, err
			}
		}
		targetMetadataIDs := m.resourceTargetMetadataIDs(candidate, metadataByKey)
		for segmentIndex, parentMetadataID := range targetMetadataIDs {
			if parentMetadataID == 0 {
				continue
			}
			confidence := 0.95
			if candidate.Confidence != nil {
				confidence = *candidate.Confidence
			}
			if _, err := inventorySvc.LinkResourceToMetadata(ctx, inventory.LinkResourceMetadataInput{ResourceID: resource.ID, MetadataItemID: parentMetadataID, Role: resourceLinkRole(candidate), SegmentIndex: segmentIndex, Confidence: &confidence, EvidenceJSON: resolverLinkEvidence(candidate), Source: "recognition_resolver", ReviewState: database.ReviewStateAccepted}); err != nil {
				return result, err
			}
		}
		result.ResourceIDs = append(result.ResourceIDs, resource.ID)
		result.ProjectionResourceIDs = appendUniqueUint(result.ProjectionResourceIDs, resource.ID)
	}
	return result, nil
}

func (result MaterializeResult) Merge(other MaterializeResult) MaterializeResult {
	for _, id := range other.MetadataIDs {
		result.MetadataIDs = append(result.MetadataIDs, id)
	}
	for _, id := range other.ResourceIDs {
		result.ResourceIDs = append(result.ResourceIDs, id)
	}
	for _, id := range other.ProjectionMetadataIDs {
		result.ProjectionMetadataIDs = appendUniqueUint(result.ProjectionMetadataIDs, id)
	}
	for _, id := range other.ProjectionResourceIDs {
		result.ProjectionResourceIDs = appendUniqueUint(result.ProjectionResourceIDs, id)
	}
	return result
}

func resourceLinkRole(candidate database.RecognitionCandidate) string {
	switch strings.TrimSpace(candidate.CandidateRole) {
	case database.ResourceLinkRoleTrailer, database.ResourceLinkRoleExtra, database.ResourceLinkRoleSample:
		return strings.TrimSpace(candidate.CandidateRole)
	}
	if strings.TrimSpace(candidate.VariantKey) != "" || strings.TrimSpace(candidate.EditionKey) != "" {
		return database.ResourceLinkRoleVersion
	}
	return database.ResourceLinkRolePrimary
}

func editionLabel(candidate database.RecognitionCandidate) string {
	return strings.TrimPrefix(strings.TrimSpace(candidate.EditionKey), CandidateTypeEdition+":")
}

func variantLabel(candidate database.RecognitionCandidate) string {
	value := strings.TrimPrefix(strings.TrimSpace(candidate.VariantKey), CandidateTypeVariant+":")
	return strings.ReplaceAll(value, ":", " ")
}

func resolverResourceSummary(candidate database.RecognitionCandidate) string {
	return mustJSON(map[string]any{"candidate_key": candidate.CandidateKey, "variant_key": candidate.VariantKey, "edition_key": candidate.EditionKey, "resource_shape": candidate.ResourceShape, "role": candidate.CandidateRole})
}

func resolverLinkEvidence(candidate database.RecognitionCandidate) string {
	return mustJSON(map[string]any{"candidate_key": candidate.CandidateKey, "canonical_key": candidate.CanonicalKey, "parent_candidate_key": candidate.ParentCandidateKey, "variant_key": candidate.VariantKey, "edition_key": candidate.EditionKey, "resource_shape": candidate.ResourceShape, "role": candidate.CandidateRole, "evidence": candidate.EvidenceJSON})
}

func (m *Materializer) resourceTargetMetadataIDs(candidate database.RecognitionCandidate, metadataByKey map[string]uint) map[int]uint {
	targets := make(map[int]uint)
	if strings.TrimSpace(candidate.ResourceShape) != database.ResourceShapeMultiEpisode {
		parentKey := strings.TrimSpace(candidate.ParentCandidateKey)
		if parentKey != "" {
			if metadataID := metadataByKey[parentKey]; metadataID != 0 {
				targets[0] = metadataID
			}
		}
		return targets
	}
	var payload struct {
		EpisodeKeys []string `json:"episode_keys"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(candidate.EvidenceJSON)), &payload); err != nil {
		return targets
	}
	for idx, episodeKey := range payload.EpisodeKeys {
		if metadataID := metadataByKey[strings.TrimSpace(episodeKey)]; metadataID != 0 {
			targets[idx+1] = metadataID
		}
	}
	if len(targets) == 0 {
		parentKey := strings.TrimSpace(candidate.ParentCandidateKey)
		if parentKey != "" {
			if metadataID := metadataByKey[parentKey]; metadataID != 0 {
				targets[1] = metadataID
			}
		}
	}
	return targets
}

func (m *Materializer) metadataByCandidateKey(ctx context.Context, scopePath string, candidates []database.RecognitionCandidate) (map[string]uint, error) {
	result := make(map[string]uint)
	for _, candidate := range candidates {
		if candidate.CandidateType != CandidateTypeWork && candidate.CandidateType != CandidateTypeEpisode {
			continue
		}
		itemType := metadataItemTypeForCandidate(candidate)
		if itemType == "" || strings.TrimSpace(candidate.CanonicalKey) == "" {
			continue
		}
		var item database.MetadataItem
		query := m.db.WithContext(ctx).Where("item_type = ? AND sort_key = ? AND deleted_at IS NULL", itemType, candidateMetadataSortKey(scopePath, candidate, candidate.CanonicalKey))
		err := query.First(&item).Error
		if err == gorm.ErrRecordNotFound {
			continue
		}
		if err != nil {
			return nil, err
		}
		result[strings.TrimSpace(candidate.CandidateKey)] = item.ID
	}
	return result, nil
}

func (m *Materializer) upsertMetadataForCandidate(ctx context.Context, scopePath string, candidate database.RecognitionCandidate, candidatesByKey map[string]database.RecognitionCandidate) (database.MetadataItem, error) {
	itemType := metadataItemTypeForCandidate(candidate)
	if itemType == "" {
		return database.MetadataItem{}, nil
	}
	title := titleFromCandidate(candidate)
	if title == "" {
		title = strings.TrimSpace(candidate.CanonicalKey)
	}
	sortKey := candidateMetadataSortKey(scopePath, candidate, candidate.CanonicalKey)
	if sortKey == "" {
		sortKey = title
	}
	var item database.MetadataItem
	query := m.db.WithContext(ctx).Where("item_type = ? AND sort_key = ? AND deleted_at IS NULL", itemType, sortKey)
	err := query.First(&item).Error
	if err == nil {
		if err := m.updateMetadataForCandidate(ctx, &item, candidate, candidatesByKey); err != nil {
			return database.MetadataItem{}, err
		}
		return item, nil
	}
	if err != gorm.ErrRecordNotFound {
		return database.MetadataItem{}, err
	}
	item = database.MetadataItem{ItemType: itemType, ContentForm: database.MetadataContentFormStandard, Title: title, SortTitle: title, SortKey: sortKey, GovernanceStatus: database.ReviewStatePending}
	if err := m.populateMetadataHierarchy(ctx, &item, candidate, candidatesByKey); err != nil {
		return database.MetadataItem{}, err
	}
	if err := m.db.WithContext(ctx).Create(&item).Error; err != nil {
		return database.MetadataItem{}, err
	}
	return item, nil
}

func (m *Materializer) populateMetadataHierarchy(ctx context.Context, item *database.MetadataItem, candidate database.RecognitionCandidate, candidatesByKey map[string]database.RecognitionCandidate) error {
	if item == nil {
		return nil
	}
	switch metadataItemTypeForCandidate(candidate) {
	case database.MetadataItemTypeSeason:
		parentID, rootID, err := m.parentAndRootIDsForCandidate(ctx, item.SortKey, candidatesByKey[strings.TrimSpace(candidate.ParentCandidateKey)], candidatesByKey)
		if err != nil {
			return err
		}
		item.ParentID = parentID
		item.RootID = rootID
		item.IndexNumber = seasonNumberFromCandidate(candidate)
	case database.MetadataItemTypeEpisode:
		parentCandidate := candidatesByKey[strings.TrimSpace(candidate.ParentCandidateKey)]
		parentID, rootID, err := m.parentAndRootIDsForCandidate(ctx, item.SortKey, parentCandidate, candidatesByKey)
		if err != nil {
			return err
		}
		item.ParentID = parentID
		item.RootID = rootID
		item.ParentIndexNumber = seasonNumberFromCandidate(parentCandidate)
		item.IndexNumber = episodeNumberFromCandidate(candidate)
	case database.MetadataItemTypeSeries:
		item.RootID = nil
	}
	return nil
}

func candidateMetadataSortKey(scopePath string, candidate database.RecognitionCandidate, fallback string) string {
	sortKey := strings.TrimSpace(fallback)
	if sortKey == "" {
		return ""
	}
	if trimmedScope := strings.TrimSpace(scopePath); trimmedScope != "" {
		return trimmedScope + "\x00" + sortKey
	}
	return sortKey
}

func metadataScopeFromSortKey(sortKey string) string {
	parts := strings.SplitN(strings.TrimSpace(sortKey), "\x00", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}

func (m *Materializer) updateMetadataForCandidate(ctx context.Context, item *database.MetadataItem, candidate database.RecognitionCandidate, candidatesByKey map[string]database.RecognitionCandidate) error {
	if item == nil || item.ID == 0 {
		return nil
	}
	updated := *item
	if title := titleFromCandidate(candidate); title != "" {
		updated.Title = title
		updated.SortTitle = title
	}
	if err := m.populateMetadataHierarchy(ctx, &updated, candidate, candidatesByKey); err != nil {
		return err
	}
	updates := map[string]any{
		"title":               updated.Title,
		"sort_title":          updated.SortTitle,
		"parent_id":           updated.ParentID,
		"root_id":             updated.RootID,
		"index_number":        updated.IndexNumber,
		"parent_index_number": updated.ParentIndexNumber,
	}
	if err := m.db.WithContext(ctx).Model(&database.MetadataItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
		return err
	}
	*item = updated
	return nil
}

func (m *Materializer) parentAndRootIDsForCandidate(ctx context.Context, childSortKey string, candidate database.RecognitionCandidate, candidatesByKey map[string]database.RecognitionCandidate) (*uint, *uint, error) {
	if strings.TrimSpace(candidate.CandidateKey) == "" {
		return nil, nil, nil
	}
	parentItem, err := m.upsertMetadataForCandidate(ctx, metadataScopeFromSortKey(childSortKey), candidate, candidatesByKey)
	if err != nil {
		return nil, nil, err
	}
	if parentItem.ID == 0 {
		return nil, nil, nil
	}
	parentID := parentItem.ID
	rootID := parentID
	if parentItem.RootID != nil && *parentItem.RootID != 0 {
		rootID = *parentItem.RootID
	}
	return &parentID, &rootID, nil
}

func seasonNumberFromCandidate(candidate database.RecognitionCandidate) *int {
	key := strings.TrimSpace(candidate.CandidateKey)
	if key == "" {
		return nil
	}
	parts := strings.Split(key, ":")
	for _, part := range parts {
		if len(part) == 3 && strings.HasPrefix(part, "s") {
			if value := parseIndexToken(part[1:]); value != nil {
				return value
			}
		}
	}
	return nil
}

func episodeNumberFromCandidate(candidate database.RecognitionCandidate) *int {
	key := strings.TrimSpace(candidate.CandidateKey)
	if key == "" {
		return nil
	}
	parts := strings.Split(key, ":")
	for _, part := range parts {
		if len(part) == 3 && strings.HasPrefix(part, "e") {
			if value := parseIndexToken(part[1:]); value != nil {
				return value
			}
		}
	}
	return nil
}

func parseIndexToken(value string) *int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return nil
	}
	return &parsed
}

func metadataItemTypeForCandidate(candidate database.RecognitionCandidate) string {
	if candidate.CandidateType == CandidateTypeEpisode {
		return database.MetadataItemTypeEpisode
	}
	if candidate.CandidateType != CandidateTypeWork {
		return ""
	}
	switch strings.TrimSpace(candidate.CandidateRole) {
	case WorkKindSeries:
		return database.MetadataItemTypeSeries
	case WorkKindSeason:
		return database.MetadataItemTypeSeason
	case WorkKindEpisode:
		return database.MetadataItemTypeEpisode
	default:
		return database.MetadataItemTypeMovie
	}
}

func titleFromCandidate(candidate database.RecognitionCandidate) string {
	var evidence struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(candidate.EvidenceJSON)), &evidence); err == nil && strings.TrimSpace(evidence.Title) != "" {
		return strings.TrimSpace(evidence.Title)
	}
	key := strings.TrimSpace(candidate.CanonicalKey)
	if key == "" {
		key = strings.TrimSpace(candidate.CandidateKey)
	}
	parts := strings.Split(key, ":")
	for idx := len(parts) - 1; idx >= 0; idx-- {
		part := strings.TrimSpace(parts[idx])
		if part == "" || strings.HasPrefix(part, "s") || strings.HasPrefix(part, "e") || isNumeric(part) {
			continue
		}
		return strings.ReplaceAll(part, "-", " ")
	}
	return ""
}

func isNumeric(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func appendUniqueUint(values []uint, value uint) []uint {
	if value == 0 {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
