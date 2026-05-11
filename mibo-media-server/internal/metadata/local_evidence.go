package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/settings"
)

type LocalScannerEvidence struct {
	ItemSource  database.MetadataItemSource
	ResourceID  uint
	Sidecars    []LocalScannerSidecarEvidence
	Images      []LocalScannerImageEvidence
	ExternalIDs []LocalScannerExternalIDEvidence
}

type LocalScannerSidecarEvidence struct {
	Path        string
	Hints       map[string]any
	ExternalIDs map[string]string
}

type LocalScannerImageEvidence struct {
	ImageType   string
	URL         string
	Path        string
	Source      string
	Priority    int
	Provisional bool
}

type LocalScannerExternalIDEvidence struct {
	Provider     string
	ProviderType string
	ExternalID   string
}

func (s *Service) loadMetadataItemLocalScannerEvidence(ctx context.Context, metadataItemID uint) (LocalScannerEvidence, error) {
	var source database.MetadataItemSource
	if err := s.db.WithContext(ctx).Where("metadata_item_id = ? AND source_type = ? AND source_name = ?", metadataItemID, catalog.SourceTypeLocalFile, "scanner").Order("fetched_at desc, id desc").First(&source).Error; err == nil {
		return localScannerEvidenceFromMetadataItemSource(source)
	}
	var resourceLink database.ResourceMetadataLink
	if err := s.db.WithContext(ctx).Where("metadata_item_id = ?", metadataItemID).Order("id asc").First(&resourceLink).Error; err != nil {
		return LocalScannerEvidence{}, err
	}
	var libraryLink database.ResourceLibraryLink
	if err := s.db.WithContext(ctx).Where("resource_id = ?", resourceLink.ResourceID).Order("last_seen_at desc, id asc").First(&libraryLink).Error; err != nil {
		return LocalScannerEvidence{}, err
	}
	source = database.MetadataItemSource{MetadataItemID: metadataItemID, SourceType: catalog.SourceTypeLocalFile, SourceName: "scanner", TriggeringLibraryID: &libraryLink.LibraryID, PayloadJSON: libraryLink.EvidenceJSON, EvidenceJSON: libraryLink.EvidenceJSON, FetchedAt: libraryLink.LastSeenAt}
	if err := s.db.WithContext(ctx).Create(&source).Error; err != nil {
		return LocalScannerEvidence{}, err
	}
	evidence, err := localScannerEvidenceFromMetadataItemSource(source)
	if err != nil {
		return LocalScannerEvidence{}, err
	}
	evidence.ResourceID = resourceLink.ResourceID
	return evidence, nil
}

func localScannerEvidenceFromMetadataItemSource(source database.MetadataItemSource) (LocalScannerEvidence, error) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(firstNonEmpty(source.PayloadJSON, source.EvidenceJSON))), &payload); err != nil {
		return LocalScannerEvidence{}, err
	}
	evidence := LocalScannerEvidence{ItemSource: source}
	evidence.Sidecars = localScannerSidecars(payload["metadata_sidecars"])
	evidence.Images = localScannerImages(payload["image_candidates"])
	evidence.ExternalIDs = localScannerExternalIDs(payload["external_ids"])
	return evidence, nil
}

func localScannerSidecars(raw any) []LocalScannerSidecarEvidence {
	items, _ := raw.([]any)
	result := make([]LocalScannerSidecarEvidence, 0, len(items))
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok || strings.TrimSpace(fmt.Sprint(item["parse_status"])) != "parsed" {
			continue
		}
		evidence := LocalScannerSidecarEvidence{Path: strings.TrimSpace(fmt.Sprint(item["path"])), Hints: map[string]any{}, ExternalIDs: map[string]string{}}
		if hints, ok := item["hints"].(map[string]any); ok {
			evidence.Hints = hints
		}
		if externalIDs, ok := item["external_ids"].(map[string]any); ok {
			for key, value := range externalIDs {
				if trimmed := strings.TrimSpace(fmt.Sprint(value)); trimmed != "" {
					evidence.ExternalIDs[strings.ToLower(strings.TrimSpace(key))] = trimmed
				}
			}
		}
		result = append(result, evidence)
	}
	return result
}

func localScannerImages(raw any) []LocalScannerImageEvidence {
	items, _ := raw.([]any)
	result := make([]LocalScannerImageEvidence, 0, len(items))
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, LocalScannerImageEvidence{ImageType: strings.TrimSpace(fmt.Sprint(item["image_type"])), URL: strings.TrimSpace(fmt.Sprint(item["url"])), Path: strings.TrimSpace(fmt.Sprint(item["path"])), Source: strings.TrimSpace(fmt.Sprint(item["source"])), Priority: intFromAny(item["priority"]), Provisional: boolFromAny(item["provisional"])})
	}
	return result
}

func localScannerExternalIDs(raw any) []LocalScannerExternalIDEvidence {
	items, _ := raw.([]any)
	result := make([]LocalScannerExternalIDEvidence, 0, len(items))
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, LocalScannerExternalIDEvidence{Provider: strings.TrimSpace(fmt.Sprint(item["provider"])), ProviderType: strings.TrimSpace(fmt.Sprint(item["provider_type"])), ExternalID: strings.TrimSpace(fmt.Sprint(item["external_id"]))})
	}
	return result
}

func intFromAny(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func boolFromAny(value any) bool {
	parsed, _ := value.(bool)
	return parsed
}

func localEvidenceCandidates(evidence LocalScannerEvidence, mediaType string) []NormalizedMetadataCandidate {
	result := make([]NormalizedMetadataCandidate, 0)
	for _, externalID := range evidence.ExternalIDs {
		candidate := localExternalIDCandidate(externalID.Provider, externalID.ProviderType, externalID.ExternalID, mediaType)
		if candidate.ExternalID != "" {
			result = append(result, candidate)
		}
	}
	for _, sidecar := range evidence.Sidecars {
		for provider, value := range sidecar.ExternalIDs {
			candidate := localExternalIDCandidate(provider, mediaType, value, mediaType)
			if candidate.ExternalID != "" {
				if title := strings.TrimSpace(fmt.Sprint(sidecar.Hints["title"])); title != "" {
					candidate.Title = title
				}
				if originalTitle := strings.TrimSpace(fmt.Sprint(sidecar.Hints["original_title"])); originalTitle != "" {
					candidate.OriginalTitle = originalTitle
				}
				if year := intFromAny(sidecar.Hints["year"]); year > 0 {
					candidate.Year = &year
				}
				result = append(result, candidate)
			}
		}
	}
	return result
}

func localExternalIDCandidate(provider string, providerType string, value string, mediaType string) NormalizedMetadataCandidate {
	provider = strings.ToLower(strings.TrimSpace(provider))
	providerType = strings.TrimSpace(providerType)
	value = strings.TrimSpace(value)
	if provider == "" || value == "" {
		return NormalizedMetadataCandidate{}
	}
	if providerType == "" {
		providerType = mediaType
	}
	switch provider {
	case "tmdb":
		if !strings.Contains(value, ":") {
			value = providerType + ":" + value
		}
		return NormalizedMetadataCandidate{Provider: provider, ProviderType: providerType, ExternalID: value, Confidence: 1, ReasonSummary: "local scanner external id"}
	case database.MetadataProviderTypeMetaTube:
		return NormalizedMetadataCandidate{Provider: provider, ProviderType: providerType, ExternalID: value, Confidence: 1, ReasonSummary: "local scanner external id"}
	default:
		return NormalizedMetadataCandidate{Provider: provider, ProviderType: providerType, ExternalID: value, Confidence: 1, ReasonSummary: "local scanner external id"}
	}
}

func localEvidenceDetail(evidence LocalScannerEvidence, itemType string) (NormalizedMetadataDetail, bool) {
	detail := NormalizedMetadataDetail{Provider: database.MetadataProviderTypeLocalScan, ProviderType: itemType, ExternalIDs: make([]NormalizedMetadataExternalID, 0)}
	if len(evidence.Sidecars) > 0 {
		sidecar := evidence.Sidecars[0]
		if itemType == catalog.ItemTypeSeries {
			detail.Title = strings.TrimSpace(fmt.Sprint(sidecar.Hints["series_title"]))
		}
		if detail.Title == "" {
			detail.Title = strings.TrimSpace(fmt.Sprint(sidecar.Hints["title"]))
		}
		detail.OriginalTitle = strings.TrimSpace(fmt.Sprint(sidecar.Hints["original_title"]))
		detail.Overview = strings.TrimSpace(fmt.Sprint(sidecar.Hints["overview"]))
		if year := intFromAny(sidecar.Hints["year"]); year > 0 {
			detail.Year = &year
		}
	}
	for _, candidate := range localEvidenceCandidates(evidence, catalogTMDBMediaType(itemType)) {
		detail.ExternalIDs = append(detail.ExternalIDs, NormalizedMetadataExternalID{Provider: candidate.Provider, ProviderType: candidate.ProviderType, ExternalID: candidate.ExternalID, IsPrimary: true, Confidence: &candidate.Confidence})
	}
	for index, image := range evidence.Images {
		imageType := strings.TrimSpace(image.ImageType)
		url := strings.TrimSpace(firstNonEmpty(image.URL, image.Path))
		if imageType == "" || url == "" {
			continue
		}
		sortOrder := image.Priority
		if sortOrder == 0 {
			sortOrder = index
		}
		detail.Images = append(detail.Images, NormalizedMetadataImage{ImageType: imageType, URL: url, SortOrder: sortOrder, Selected: !image.Provisional})
	}
	return detail, detail.Title != "" || detail.OriginalTitle != "" || detail.Overview != "" || detail.Year != nil || len(detail.ExternalIDs) > 0 || len(detail.Images) > 0
}

func localEvidenceProviderAttempt(provider settings.ResolvedMetadataProviderInstance, selected bool) MetadataProviderAttempt {
	outcome := ProviderAttemptOutcomeNoResult
	if selected {
		outcome = ProviderAttemptOutcomeSuccess
	}
	attempt := metadataProviderAttemptForProvider("local_evidence", provider, outcome)
	attempt.Selected = selected
	if selected {
		attempt.CandidateCount = 1
	}
	return attempt
}
