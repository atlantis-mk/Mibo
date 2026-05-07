package database

import "strings"

const (
	MetadataItemTypeMovie      = "movie"
	MetadataItemTypeSeries     = "series"
	MetadataItemTypeSeason     = "season"
	MetadataItemTypeEpisode    = "episode"
	MetadataItemTypeCollection = "collection"
	MetadataItemTypePerson     = "person"

	MetadataContentFormStandard    = "standard"
	MetadataContentFormAnime       = "anime"
	MetadataContentFormDocumentary = "documentary"
	MetadataContentFormAdult       = "adult"

	ResourceTypePlayable = "playable"
	ResourceTypeRelated  = "related"

	ResourceShapeSingleFile   = "single_file"
	ResourceShapeMultiPart    = "multi_part"
	ResourceShapeMultiEpisode = "multi_episode"

	ResourceFileRoleSource   = "source"
	ResourceFileRoleSubtitle = "subtitle"
	ResourceFileRoleSidecar  = "sidecar"
	ResourceFileRoleImage    = "image"

	ResourceLinkRolePrimary = "primary"
	ResourceLinkRoleVersion = "version"
	ResourceLinkRoleTrailer = "trailer"
	ResourceLinkRoleExtra   = "extra"
	ResourceLinkRoleSample  = "sample"

	ProjectionAvailabilityAvailable   = "available"
	ProjectionAvailabilityUnavailable = "unavailable"
	ProjectionAvailabilityMissing     = "missing"
	ProjectionAvailabilityPartial     = "partial"

	ReviewStateAccepted    = "accepted"
	ReviewStatePending     = "pending"
	ReviewStateRejected    = "rejected"
	ReviewStateNeedsReview = "needs_review"
)

func NormalizeMetadataItemType(value string) string {
	switch normalizedToken(value) {
	case MetadataItemTypeMovie, MetadataItemTypeSeries, MetadataItemTypeSeason, MetadataItemTypeEpisode, MetadataItemTypeCollection, MetadataItemTypePerson:
		return normalizedToken(value)
	default:
		return MetadataItemTypeMovie
	}
}

func NormalizeMetadataContentForm(value string) string {
	switch normalizedToken(value) {
	case MetadataContentFormAnime, MetadataContentFormDocumentary, MetadataContentFormAdult:
		return normalizedToken(value)
	default:
		return MetadataContentFormStandard
	}
}

func NormalizeResourceType(value string) string {
	switch normalizedToken(value) {
	case ResourceTypeRelated:
		return ResourceTypeRelated
	default:
		return ResourceTypePlayable
	}
}

func NormalizeResourceShape(value string) string {
	switch normalizedToken(value) {
	case ResourceShapeMultiPart, ResourceShapeMultiEpisode:
		return normalizedToken(value)
	default:
		return ResourceShapeSingleFile
	}
}

func NormalizeResourceFileRole(value string) string {
	switch normalizedToken(value) {
	case ResourceFileRoleSubtitle, ResourceFileRoleSidecar, ResourceFileRoleImage:
		return normalizedToken(value)
	default:
		return ResourceFileRoleSource
	}
}

func NormalizeResourceLinkRole(value string) string {
	switch normalizedToken(value) {
	case ResourceLinkRoleVersion, ResourceLinkRoleTrailer, ResourceLinkRoleExtra, ResourceLinkRoleSample:
		return normalizedToken(value)
	default:
		return ResourceLinkRolePrimary
	}
}

func NormalizeProjectionAvailability(value string) string {
	switch normalizedToken(value) {
	case ProjectionAvailabilityAvailable, ProjectionAvailabilityMissing, ProjectionAvailabilityPartial:
		return normalizedToken(value)
	default:
		return ProjectionAvailabilityUnavailable
	}
}

func NormalizeReviewState(value string) string {
	switch normalizedToken(value) {
	case ReviewStatePending, ReviewStateRejected, ReviewStateNeedsReview:
		return normalizedToken(value)
	default:
		return ReviewStateAccepted
	}
}

func normalizedToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
