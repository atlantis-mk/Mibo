package library

import (
	"context"
	"encoding/json"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/storage"
)

const (
	SourceContentClassVideo = "video"
	SourceContentClassAudio = "audio"
	SourceContentClassText  = "text"
	SourceContentClassImage = "image"
	SourceContentClassOther = "other"

	SourceProbeStatusPending = "pending"
	SourceProbeStatusReady   = "ready"
	SourceProbeStatusPartial = "partial"
	SourceProbeStatusError   = "error"

	defaultSourceProbeMaxDuration = 3 * time.Second
	defaultSourceProbeMaxObjects  = 500
	defaultSourceProbeMaxDepth    = 3
	defaultSourceProbePageSize    = 200
)

type SourceProbeSummary struct {
	Status         string         `json:"status"`
	DominantClass  string         `json:"dominant_class"`
	Uncertain      bool           `json:"uncertain"`
	BudgetLimited  bool           `json:"budget_limited"`
	SampledObjects int            `json:"sampled_objects"`
	SampledFiles   int            `json:"sampled_files"`
	SampledDirs    int            `json:"sampled_dirs"`
	MaxObjects     int            `json:"max_objects"`
	MaxDepth       int            `json:"max_depth"`
	Classes        map[string]int `json:"classes"`
	Error          string         `json:"error,omitempty"`
}

type sourceProbeOptions struct {
	MaxDuration time.Duration
	MaxObjects  int
	MaxDepth    int
	PageSize    int
}

type sourceProbeNode struct {
	Path  string
	Depth int
}

var audioExtensions = map[string]struct{}{
	".mp3": {}, ".flac": {}, ".m4a": {}, ".aac": {}, ".ogg": {}, ".opus": {}, ".wav": {}, ".aiff": {}, ".alac": {}, ".wma": {},
}

var textExtensions = map[string]struct{}{
	".txt": {}, ".md": {}, ".pdf": {}, ".epub": {}, ".mobi": {}, ".azw3": {}, ".cbz": {}, ".cbr": {}, ".srt": {}, ".ass": {}, ".ssa": {}, ".vtt": {}, ".nfo": {}, ".json": {},
}

var imageExtensions = map[string]struct{}{
	".jpg": {}, ".jpeg": {}, ".png": {}, ".webp": {}, ".gif": {}, ".bmp": {}, ".avif": {}, ".heic": {}, ".tiff": {},
}

func (s *Service) ProbeSource(ctx context.Context, provider storage.Provider, rootPath string) SourceProbeSummary {
	return probeSource(ctx, provider, rootPath, sourceProbeOptions{
		MaxDuration: defaultSourceProbeMaxDuration,
		MaxObjects:  defaultSourceProbeMaxObjects,
		MaxDepth:    defaultSourceProbeMaxDepth,
		PageSize:    defaultSourceProbePageSize,
	})
}

func probeSource(ctx context.Context, provider storage.Provider, rootPath string, options sourceProbeOptions) SourceProbeSummary {
	options = normalizeSourceProbeOptions(options)
	deadline := time.Now().Add(options.MaxDuration)
	summary := SourceProbeSummary{Status: SourceProbeStatusReady, Classes: emptySourceProbeClassCounts(), MaxObjects: options.MaxObjects, MaxDepth: options.MaxDepth}
	queue := []sourceProbeNode{{Path: rootPath, Depth: 0}}

	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			summary.Status = SourceProbeStatusPartial
			summary.BudgetLimited = true
			break
		}
		if time.Now().After(deadline) || summary.SampledObjects >= options.MaxObjects {
			summary.Status = SourceProbeStatusPartial
			summary.BudgetLimited = true
			break
		}
		node := queue[0]
		queue = queue[1:]
		objects, err := provider.List(ctx, storage.ListRequest{Path: node.Path, Page: 1, PerPage: options.PageSize})
		if err != nil {
			if summary.SampledObjects == 0 {
				summary.Status = SourceProbeStatusError
				summary.Error = err.Error()
			} else {
				summary.Status = SourceProbeStatusPartial
				summary.BudgetLimited = true
			}
			break
		}
		sort.Slice(objects, func(i, j int) bool { return objects[i].Path < objects[j].Path })
		for _, object := range objects {
			if time.Now().After(deadline) || summary.SampledObjects >= options.MaxObjects {
				summary.Status = SourceProbeStatusPartial
				summary.BudgetLimited = true
				break
			}
			summary.SampledObjects++
			if object.IsDir {
				summary.SampledDirs++
				if node.Depth+1 <= options.MaxDepth {
					queue = append(queue, sourceProbeNode{Path: object.Path, Depth: node.Depth + 1})
				}
				continue
			}
			summary.SampledFiles++
			summary.Classes[classifySourceObject(object.Path)]++
		}
	}
	finalizeSourceProbeSummary(&summary)
	return summary
}

func normalizeSourceProbeOptions(options sourceProbeOptions) sourceProbeOptions {
	if options.MaxDuration <= 0 {
		options.MaxDuration = defaultSourceProbeMaxDuration
	}
	if options.MaxObjects <= 0 {
		options.MaxObjects = defaultSourceProbeMaxObjects
	}
	if options.MaxDepth < 0 {
		options.MaxDepth = defaultSourceProbeMaxDepth
	}
	if options.PageSize <= 0 || options.PageSize > defaultSourceProbePageSize {
		options.PageSize = defaultSourceProbePageSize
	}
	return options
}

func classifySourceObject(objectPath string) string {
	ext := strings.ToLower(path.Ext(objectPath))
	if _, ok := videoExtensions[ext]; ok {
		return SourceContentClassVideo
	}
	if _, ok := audioExtensions[ext]; ok {
		return SourceContentClassAudio
	}
	if _, ok := textExtensions[ext]; ok {
		return SourceContentClassText
	}
	if _, ok := imageExtensions[ext]; ok {
		return SourceContentClassImage
	}
	return SourceContentClassOther
}

func emptySourceProbeClassCounts() map[string]int {
	return map[string]int{SourceContentClassVideo: 0, SourceContentClassAudio: 0, SourceContentClassText: 0, SourceContentClassImage: 0, SourceContentClassOther: 0}
}

func finalizeSourceProbeSummary(summary *SourceProbeSummary) {
	if summary.Classes == nil {
		summary.Classes = emptySourceProbeClassCounts()
	}
	bestClass := ""
	bestCount := 0
	secondCount := 0
	for className, count := range summary.Classes {
		if count > bestCount {
			secondCount = bestCount
			bestClass = className
			bestCount = count
			continue
		}
		if count > secondCount {
			secondCount = count
		}
	}
	if bestCount > 0 {
		summary.DominantClass = bestClass
	}
	summary.Uncertain = bestCount == 0 || (secondCount > 0 && bestCount < secondCount*2)
	if summary.Status == "" {
		summary.Status = SourceProbeStatusReady
	}
}

func encodeSourceProbeSummary(summary SourceProbeSummary) string {
	encoded, err := json.Marshal(summary)
	if err != nil {
		return ""
	}
	return string(encoded)
}
