package library

import "github.com/atlan/mibo-media-server/internal/config"

const (
	ContentShapeClassifierVersion = "content-shape-v2"

	contentShapePlanReuseConfidenceThreshold    = 0.85
	contentShapeHighConfidenceThreshold         = 0.75
	contentShapeMediumReviewConfidenceThreshold = 0.65
	contentShapeLargeDirectoryVideoThreshold    = 50
	contentShapeLargeDirectoryEntryThreshold    = 100
)

type contentShapeSettings struct {
	ClassifierVersion               string
	PlanReuseConfidenceThreshold    float64
	HighConfidenceThreshold         float64
	MediumReviewConfidenceThreshold float64
	LargeDirectoryVideoThreshold    int
	LargeDirectoryEntryThreshold    int
}

func contentShapeSettingsFromConfig(cfg config.Config) contentShapeSettings {
	return contentShapeSettings{
		ClassifierVersion:               ContentShapeClassifierVersion,
		PlanReuseConfidenceThreshold:    contentShapePlanReuseConfidenceThreshold,
		HighConfidenceThreshold:         contentShapeHighConfidenceThreshold,
		MediumReviewConfidenceThreshold: contentShapeMediumReviewConfidenceThreshold,
		LargeDirectoryVideoThreshold:    contentShapeLargeDirectoryVideoThreshold,
		LargeDirectoryEntryThreshold:    contentShapeLargeDirectoryEntryThreshold,
	}
}
