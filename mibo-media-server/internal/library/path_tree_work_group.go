package library

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/inventory"
	"github.com/atlan/mibo-media-server/internal/scanrecognition"
	"github.com/atlan/mibo-media-server/internal/storage"
	"github.com/atlan/mibo-media-server/internal/titleclean"
)

const pathTreeWorkGroupRuleType = "path_tree_work_group"

const (
	pathTreeWorkGroupShapeMovie             = "movie"
	pathTreeWorkGroupShapeMovieVersionGroup = "movie_version_group"
	pathTreeWorkGroupShapeMovieCollection   = "movie_collection"
	pathTreeWorkGroupShapeSeries            = "series"
	pathTreeWorkGroupShapeSeason            = "season"
	pathTreeWorkGroupShapeEpisodePack       = "episode_pack"
	pathTreeWorkGroupShapeAttachment        = "attachment"
	pathTreeWorkGroupShapeReview            = "review_required"

	pathTreeAssignmentMovie      = contentShapeAssignmentMovie
	pathTreeAssignmentVersion    = contentShapeAssignmentVersion
	pathTreeAssignmentEpisode    = contentShapeAssignmentEpisode
	pathTreeAssignmentAttachment = contentShapeAssignmentAttachment
	pathTreeAssignmentReview     = contentShapeAssignmentReview
)

type pathTreeWorkKey struct {
	Kind       string
	Title      string
	Year       *int
	Normalized string
}

type pathTreeWorkGroupAlternative struct {
	Shape      string
	Confidence float64
	Reason     string
}

type pathTreeWorkGroupAssignment struct {
	StoragePath    string
	AssignmentType string
	TargetKey      string
	SeriesTitle    string
	SeasonNumber   *int
	EpisodeNumber  *int
	AbsoluteNumber *int
	AssetRole      string
	Confidence     float64
	ReviewState    string
	Evidence       map[string]any
}

type pathTreeWorkGroup struct {
	Shape        string
	WorkKey      pathTreeWorkKey
	Confidence   float64
	ReviewState  string
	Evidence     map[string]any
	Alternatives []pathTreeWorkGroupAlternative
	Assignments  []pathTreeWorkGroupAssignment
}

type pathTreeParentSummary struct {
	LibraryID         uint
	StorageProvider   string
	RootPath          string
	ParentPath        string
	ClassifierVersion string
	ScanPolicy        database.LibraryScanPolicy
	ExclusionRules    []database.ScanExclusionRule
	Snapshot          scanDirectorySnapshot
	Children          []pathTreeChildSummary
	Files             []pathTreeFileSummary
	Fingerprint       string
}

type pathTreeChildSummary struct {
	Path        string
	Snapshot    scanDirectorySnapshot
	Plan        contentShapeDirectoryPlan
	Files       []pathTreeFileSummary
	Fingerprint string
}

type pathTreeFileSummary struct {
	StoragePath  string
	ParentPath   string
	Basename     string
	Signal       filenameSignalModel
	MovieWorkKey pathTreeWorkKey
	IsVideo      bool
	IsExtra      bool
}

type pathTreeSiblingMovieVersionChild struct {
	File    database.InventoryFile
	Summary pathTreeFileSummary
}

type pathTreeMovieCollectionChild struct {
	File    database.InventoryFile
	Summary pathTreeFileSummary
}

func buildPathTreeParentSummary(scope contentShapeScope, parentSnapshot scanDirectorySnapshot, childSnapshots map[string]scanDirectorySnapshot, childPlans map[string]contentShapeDirectoryPlan, indexedSignals map[string]filenameSignalModel, scanPolicy database.LibraryScanPolicy, exclusionRules []database.ScanExclusionRule, tokenCache *filenameTokenProfileCache) pathTreeParentSummary {
	summary := pathTreeParentSummary{LibraryID: scope.LibraryID, StorageProvider: strings.TrimSpace(scope.StorageProvider), RootPath: strings.TrimSpace(scope.RootPath), ParentPath: strings.TrimSpace(parentSnapshot.Path), ClassifierVersion: strings.TrimSpace(scope.ClassifierVersion), ScanPolicy: scanPolicy, ExclusionRules: exclusionRules, Snapshot: parentSnapshot}
	for _, object := range parentSnapshot.Objects {
		if object.IsDir {
			childPath := strings.TrimSpace(object.Path)
			childSnapshot := childSnapshots[childPath]
			child := pathTreeChildSummary{Path: childPath, Snapshot: childSnapshot, Plan: childPlans[childPath]}
			child.Files = pathTreeFileSummaries(childSnapshot.Objects, indexedSignals, tokenCache)
			child.Fingerprint = pathTreeChildFingerprint(child)
			summary.Children = append(summary.Children, child)
			continue
		}
		if file, ok := pathTreeFileSummaryFromObject(object, indexedSignals, tokenCache); ok {
			summary.Files = append(summary.Files, file)
		}
	}
	sort.Slice(summary.Children, func(i, j int) bool { return summary.Children[i].Path < summary.Children[j].Path })
	sort.Slice(summary.Files, func(i, j int) bool { return summary.Files[i].StoragePath < summary.Files[j].StoragePath })
	summary.Fingerprint = pathTreeParentFingerprint(summary)
	return summary
}

func pathTreeFileSummaries(objects []storage.Object, indexedSignals map[string]filenameSignalModel, tokenCache *filenameTokenProfileCache) []pathTreeFileSummary {
	files := make([]pathTreeFileSummary, 0)
	for _, object := range objects {
		file, ok := pathTreeFileSummaryFromObject(object, indexedSignals, tokenCache)
		if ok {
			files = append(files, file)
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].StoragePath < files[j].StoragePath })
	return files
}

func pathTreeFileSummaryFromObject(object storage.Object, indexedSignals map[string]filenameSignalModel, tokenCache *filenameTokenProfileCache) (pathTreeFileSummary, bool) {
	storagePath := strings.TrimSpace(object.Path)
	if object.IsDir || storagePath == "" || !isVideoFile(storagePath) {
		return pathTreeFileSummary{}, false
	}
	signal, ok := indexedSignals[storagePath]
	if !ok {
		signal = filenameTokenProfileForPath(tokenCache, storagePath)
	}
	return pathTreeFileSummary{StoragePath: storagePath, ParentPath: path.Dir(storagePath), Basename: path.Base(storagePath), Signal: signal, MovieWorkKey: normalizedMovieWorkKeyFromSignal(signal.VideoSignal, signal.RawPathData, signal.Identity.TitleCandidate, signal.Identity.Year), IsVideo: true, IsExtra: signal.VideoSignal.IsExtra}, true
}

func normalizedMovieWorkKeyFromSignal(videoSignal scanrecognition.VideoSignal, rawPathData filenameRawPathData, fallbackTitle string, fallbackYear *int) pathTreeWorkKey {
	title := pathTreeMovieTitleFromSignal(videoSignal, rawPathData, fallbackTitle, fallbackYear)
	if title == "" {
		return pathTreeWorkKey{Kind: pathTreeWorkGroupShapeMovie}
	}
	normalizedTitle := titleclean.NormalizeMovieWorkTitle(title)
	displayTitle := scanrecognition.CleanTitle(title)
	if strings.EqualFold(titleclean.NormalizeMovieWorkTitle(displayTitle), normalizedTitle) {
		displayTitle = titleTitleCase(normalizedTitle)
	}
	key := pathTreeWorkKey{Kind: pathTreeWorkGroupShapeMovie, Title: displayTitle, Year: preferredMovieYear(videoSignal, rawPathData, fallbackYear)}
	key.Normalized = normalizedMovieWorkKeyString(key.Title, key.Year)
	return key
}

func titleTitleCase(input string) string {
	parts := strings.Fields(strings.TrimSpace(input))
	for idx, part := range parts {
		if part == "" {
			continue
		}
		parts[idx] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func pathTreeMovieTitleFromSignal(videoSignal scanrecognition.VideoSignal, rawPathData filenameRawPathData, fallbackTitle string, fallbackYear *int) string {
	if title := movieFolderTitleCandidateFromParts(videoSignal, rawPathData, fallbackTitle, fallbackYear); title != "" {
		return title
	}
	if title := firstScanTitle(videoSignal.TitleCandidates); strings.TrimSpace(title) != "" {
		return title
	}
	if title := strings.TrimSpace(fallbackTitle); title != "" {
		return title
	}
	rawTitle := strings.TrimSuffix(rawPathData.Basename, rawPathData.Extension)
	return scanrecognition.CleanTitle(titleclean.MovieWorkTitle(rawTitle))
}

func preferredMovieYear(videoSignal scanrecognition.VideoSignal, rawPathData filenameRawPathData, fallbackYear *int) *int {
	folderSignal := scanrecognition.ParseFolderName(path.Base(strings.TrimSpace(rawPathData.Directory)))
	if folderSignal.Season == nil && folderSignal.Year != nil {
		return firstNonNilInt(folderSignal.Year, videoSignal.Year, fallbackYear)
	}
	return firstNonNilInt(videoSignal.Year, fallbackYear)
}

func normalizedMovieWorkKeyString(title string, year *int) string {
	normalizedTitle := titleclean.NormalizeMovieWorkTitle(title)
	if normalizedTitle == "" {
		return ""
	}
	if year != nil {
		return fmt.Sprintf("%s:%d", normalizedTitle, *year)
	}
	return normalizedTitle
}
func pathTreeChildFingerprint(child pathTreeChildSummary) string {
	parts := []string{"child=" + strings.TrimSpace(child.Path), "shape=" + strings.TrimSpace(child.Plan.Shape), fmt.Sprintf("confidence=%.3f", child.Plan.Confidence), "review=" + strings.TrimSpace(child.Plan.ReviewState)}
	for _, file := range child.Files {
		parts = append(parts, strings.Join([]string{"file=" + file.StoragePath, "work=" + file.MovieWorkKey.Normalized, fmt.Sprintf("extra=%t", file.IsExtra)}, "|"))
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(digest[:])
}

func pathTreeParentFingerprint(summary pathTreeParentSummary) string {
	parts := []string{fmt.Sprintf("library=%d", summary.LibraryID), "provider=" + summary.StorageProvider, "root=" + summary.RootPath, "parent=" + summary.ParentPath, "classifier=" + summary.ClassifierVersion, contentShapeScanPolicyFingerprint(summary.ScanPolicy), contentShapeExclusionFingerprint(summary.ExclusionRules)}
	for _, child := range summary.Children {
		parts = append(parts, "child="+child.Fingerprint)
	}
	for _, file := range summary.Files {
		parts = append(parts, strings.Join([]string{"file=" + file.StoragePath, "work=" + file.MovieWorkKey.Normalized, fmt.Sprintf("extra=%t", file.IsExtra)}, "|"))
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(digest[:])
}

func pathTreeReleaseHintCount(videoSignal scanrecognition.VideoSignal) int {
	count := 0
	for _, value := range []string{videoSignal.Quality, videoSignal.Codec, videoSignal.Audio, videoSignal.Subtitle, videoSignal.HDR, videoSignal.Edition, videoSignal.ReleaseGroup} {
		if strings.TrimSpace(value) != "" {
			count++
		}
	}
	count += len(videoSignal.SourceTags)
	return count
}

func compileSiblingMovieVersionAssignmentsFromFiles(files []database.InventoryFile, indexedSignals map[string]filenameSignalModel, tokenCache *filenameTokenProfileCache) map[string]pathTreeWorkGroupAssignment {
	type parentGroup struct {
		children map[string][]pathTreeSiblingMovieVersionChild
	}
	parents := make(map[string]*parentGroup)
	for _, file := range files {
		storagePath := strings.TrimSpace(file.StoragePath)
		if file.ID == 0 || storagePath == "" || file.Status != inventory.FileStatusAvailable || file.ContentClass != SourceContentClassVideo || !isVideoFile(storagePath) {
			continue
		}
		signal, ok := indexedSignals[storagePath]
		if !ok {
			signal = filenameTokenProfileForPath(tokenCache, storagePath)
		}
		if signal.VideoSignal.IsExtra {
			continue
		}
		workKey := normalizedMovieWorkKeyFromSignal(signal.VideoSignal, signal.RawPathData, signal.Identity.TitleCandidate, signal.Identity.Year)
		if strings.TrimSpace(workKey.Normalized) == "" || workKey.Year == nil || signal.VideoSignal.Episode != nil || len(signal.VideoSignal.EpisodeNumbers) > 0 {
			continue
		}
		childDir := path.Dir(storagePath)
		parentDir := path.Dir(childDir)
		if childDir == "." || parentDir == "." || childDir == parentDir {
			continue
		}
		group := parents[parentDir]
		if group == nil {
			group = &parentGroup{children: make(map[string][]pathTreeSiblingMovieVersionChild)}
			parents[parentDir] = group
		}
		group.children[childDir] = append(group.children[childDir], pathTreeSiblingMovieVersionChild{File: file, Summary: pathTreeFileSummary{StoragePath: storagePath, ParentPath: childDir, Basename: path.Base(storagePath), Signal: signal, MovieWorkKey: workKey, IsVideo: true}})
	}
	assignments := make(map[string]pathTreeWorkGroupAssignment)
	for parentDir, group := range parents {
		childrenByWorkKey := make(map[string][]pathTreeSiblingMovieVersionChild)
		for _, childFiles := range group.children {
			if len(childFiles) != 1 {
				continue
			}
			child := childFiles[0]
			childrenByWorkKey[child.Summary.MovieWorkKey.Normalized] = append(childrenByWorkKey[child.Summary.MovieWorkKey.Normalized], child)
		}
		for workKey, childFiles := range childrenByWorkKey {
			if len(childFiles) < 2 || !pathTreeSiblingMovieVersionsHaveReleaseEvidence(childFiles) {
				continue
			}
			sort.Slice(childFiles, func(i, j int) bool { return childFiles[i].Summary.StoragePath < childFiles[j].Summary.StoragePath })
			targetKey := pathTreeMovieVersionTargetPath(parentDir, childFiles[0].Summary.MovieWorkKey)
			for _, child := range childFiles {
				evidence := map[string]any{"source": "path_tree_work_group", "shape": pathTreeWorkGroupShapeMovieVersionGroup, "work_key": workKey, "parent_path": parentDir, "release_hint_count": pathTreeReleaseHintCount(child.Summary.Signal.VideoSignal), "title": child.Summary.MovieWorkKey.Title}
				if child.Summary.MovieWorkKey.Year != nil {
					evidence["year"] = *child.Summary.MovieWorkKey.Year
				}
				assignments[child.Summary.StoragePath] = pathTreeWorkGroupAssignment{StoragePath: child.Summary.StoragePath, AssignmentType: pathTreeAssignmentVersion, TargetKey: targetKey, Confidence: 0.88, ReviewState: "auto", Evidence: evidence}
			}
		}
	}
	if len(assignments) == 0 {
		return nil
	}
	return assignments
}

func compileMovieCollectionAssignmentsFromFiles(files []database.InventoryFile, indexedSignals map[string]filenameSignalModel, tokenCache *filenameTokenProfileCache) map[string]pathTreeWorkGroupAssignment {
	childrenByParent := make(map[string][]pathTreeMovieCollectionChild)
	for _, file := range files {
		storagePath := strings.TrimSpace(file.StoragePath)
		if file.ID == 0 || storagePath == "" || file.Status != inventory.FileStatusAvailable || file.ContentClass != SourceContentClassVideo || !isVideoFile(storagePath) {
			continue
		}
		signal, ok := indexedSignals[storagePath]
		if !ok {
			signal = filenameTokenProfileForPath(tokenCache, storagePath)
		}
		if signal.VideoSignal.IsExtra || signal.VideoSignal.Episode != nil || len(signal.VideoSignal.EpisodeNumbers) > 0 {
			continue
		}
		workKey := normalizedMovieWorkKeyFromSignal(signal.VideoSignal, signal.RawPathData, signal.Identity.TitleCandidate, signal.Identity.Year)
		if strings.TrimSpace(workKey.Normalized) == "" || workKey.Year == nil {
			continue
		}
		childDir := path.Dir(storagePath)
		parentDir := pathTreeMovieCollectionParentDir(childDir, workKey)
		childrenByParent[parentDir] = append(childrenByParent[parentDir], pathTreeMovieCollectionChild{File: file, Summary: pathTreeFileSummary{StoragePath: storagePath, ParentPath: childDir, Basename: path.Base(storagePath), Signal: signal, MovieWorkKey: workKey, IsVideo: true}})
	}
	assignments := make(map[string]pathTreeWorkGroupAssignment)
	for parentDir, children := range childrenByParent {
		if !pathTreeMovieCollectionEvidence(children) {
			continue
		}
		for _, child := range children {
			targetKey := pathTreeMovieVersionTargetPath(parentDir, child.Summary.MovieWorkKey)
			assignments[child.Summary.StoragePath] = pathTreeWorkGroupAssignment{StoragePath: child.Summary.StoragePath, AssignmentType: pathTreeAssignmentMovie, TargetKey: targetKey, Confidence: 0.86, ReviewState: "auto", Evidence: map[string]any{"source": "path_tree_work_group", "shape": pathTreeWorkGroupShapeMovieCollection, "work_key": child.Summary.MovieWorkKey.Normalized, "parent_path": parentDir}}
		}
	}
	if len(assignments) == 0 {
		return nil
	}
	return assignments
}

func pathTreeMovieCollectionParentDir(childDir string, workKey pathTreeWorkKey) string {
	trimmed := strings.TrimSpace(childDir)
	if trimmed == "" || trimmed == "." {
		return trimmed
	}
	dirSignal := extractFilenameSignalModel(path.Base(trimmed) + ".mkv")
	dirKey := normalizedMovieWorkKeyFromSignal(dirSignal.VideoSignal, dirSignal.RawPathData, dirSignal.Identity.TitleCandidate, dirSignal.Identity.Year)
	if strings.TrimSpace(dirKey.Normalized) == strings.TrimSpace(workKey.Normalized) {
		parent := path.Dir(trimmed)
		if parent != "." && parent != trimmed {
			return parent
		}
	}
	return trimmed
}

func compilePathTreeMovieAssignmentsFromFiles(files []database.InventoryFile, indexedSignals map[string]filenameSignalModel, tokenCache *filenameTokenProfileCache) map[string]pathTreeWorkGroupAssignment {
	assignments := compileMovieCollectionAssignmentsFromFiles(files, indexedSignals, tokenCache)
	if assignments == nil {
		assignments = make(map[string]pathTreeWorkGroupAssignment)
	}
	for storagePath, assignment := range compileSiblingMovieVersionAssignmentsFromFiles(files, indexedSignals, tokenCache) {
		assignments[storagePath] = assignment
	}
	if len(assignments) == 0 {
		return nil
	}
	return assignments
}

func compilePathTreeSeriesAssignmentsFromFiles(files []database.InventoryFile, libraryRoot string, indexedSignals map[string]filenameSignalModel, tokenCache *filenameTokenProfileCache) map[string]pathTreeWorkGroupAssignment {
	type seriesChild struct {
		File          database.InventoryFile
		StoragePath   string
		SeriesTitle   string
		SeasonNumber  *int
		EpisodeNumber *int
	}
	childrenByParent := make(map[string][]seriesChild)
	for _, file := range files {
		storagePath := strings.TrimSpace(file.StoragePath)
		if file.ID == 0 || storagePath == "" || file.Status != inventory.FileStatusAvailable || file.ContentClass != SourceContentClassVideo || !isVideoFile(storagePath) {
			continue
		}
		childDir := path.Dir(storagePath)
		folderSignal := scanrecognition.ParseFolderName(path.Base(childDir))
		season := folderSignal.Season
		seriesTitle := firstScanTitle(folderSignal.TitleCandidates)
		if season == nil || strings.TrimSpace(seriesTitle) == "" {
			continue
		}
		signal, ok := indexedSignals[storagePath]
		if !ok {
			signal = filenameTokenProfileForPath(tokenCache, storagePath)
		}
		episode := signal.VideoSignal.Episode
		if episode == nil || signal.VideoSignal.IsExtra {
			continue
		}
		parentDir := path.Dir(childDir)
		childrenByParent[parentDir] = append(childrenByParent[parentDir], seriesChild{File: file, StoragePath: storagePath, SeriesTitle: seriesTitle, SeasonNumber: season, EpisodeNumber: episode})
	}
	assignments := make(map[string]pathTreeWorkGroupAssignment)
	for parentDir, children := range childrenByParent {
		if len(children) < 2 {
			continue
		}
		seriesTitle := children[0].SeriesTitle
		seasons := make(map[int]struct{})
		for _, child := range children {
			if !strings.EqualFold(titleclean.NormalizeMovieWorkTitle(child.SeriesTitle), titleclean.NormalizeMovieWorkTitle(seriesTitle)) || child.SeasonNumber == nil {
				seriesTitle = ""
				break
			}
			seasons[*child.SeasonNumber] = struct{}{}
		}
		if strings.TrimSpace(seriesTitle) == "" || len(seasons) < 2 {
			continue
		}
		for _, child := range children {
			assignments[child.StoragePath] = pathTreeWorkGroupAssignment{StoragePath: child.StoragePath, AssignmentType: pathTreeAssignmentEpisode, TargetKey: canonicalSeriesPath(seriesTitle), SeriesTitle: seriesTitle, SeasonNumber: child.SeasonNumber, EpisodeNumber: child.EpisodeNumber, Confidence: 0.88, ReviewState: "auto", Evidence: map[string]any{"source": "path_tree_work_group", "shape": pathTreeWorkGroupShapeSeries, "parent_path": parentDir, "series_title": seriesTitle}}
		}
	}
	if len(assignments) == 0 {
		return nil
	}
	return assignments
}

func compilePathTreeAssignmentsFromFiles(files []database.InventoryFile, libraryRoot string, indexedSignals map[string]filenameSignalModel, tokenCache *filenameTokenProfileCache) map[string]pathTreeWorkGroupAssignment {
	assignments := compilePathTreeMovieAssignmentsFromFiles(files, indexedSignals, tokenCache)
	if assignments == nil {
		assignments = make(map[string]pathTreeWorkGroupAssignment)
	}
	for storagePath, assignment := range compilePathTreeSeriesAssignmentsFromFiles(files, libraryRoot, indexedSignals, tokenCache) {
		assignments[storagePath] = assignment
	}
	if len(assignments) == 0 {
		return nil
	}
	return assignments
}

func applyPathTreeClassificationRules(files []database.InventoryFile, rules []database.ClassificationRule, indexedSignals map[string]filenameSignalModel, tokenCache *filenameTokenProfileCache) map[string]pathTreeWorkGroupAssignment {
	if len(files) == 0 || len(rules) == 0 {
		return nil
	}
	assignments := make(map[string]pathTreeWorkGroupAssignment)
	for _, rule := range rules {
		if !rule.Enabled || strings.TrimSpace(rule.RuleType) != pathTreeWorkGroupRuleType {
			continue
		}
		scopedFiles := pathTreeFilesInRuleScope(files, rule.PathPattern)
		if len(scopedFiles) == 0 {
			continue
		}
		for storagePath, assignment := range pathTreeAssignmentsForRule(scopedFiles, rule, indexedSignals, tokenCache) {
			assignments[storagePath] = assignment
		}
	}
	if len(assignments) == 0 {
		return nil
	}
	return assignments
}

func pathTreeFilesInRuleScope(files []database.InventoryFile, pattern string) []database.InventoryFile {
	scope := strings.TrimRight(strings.TrimSpace(pattern), "/")
	if scope == "" {
		return nil
	}
	matched := make([]database.InventoryFile, 0, len(files))
	for _, file := range files {
		storagePath := strings.TrimSpace(file.StoragePath)
		if storagePath == scope || strings.HasPrefix(storagePath, scope+"/") {
			matched = append(matched, file)
		}
	}
	return matched
}

func pathTreeFileMatchesAnyRule(storagePath string, rules []database.ClassificationRule) bool {
	for _, rule := range rules {
		if !rule.Enabled || strings.TrimSpace(rule.RuleType) != pathTreeWorkGroupRuleType {
			continue
		}
		scope := strings.TrimRight(strings.TrimSpace(rule.PathPattern), "/")
		trimmed := strings.TrimSpace(storagePath)
		if scope != "" && (trimmed == scope || strings.HasPrefix(trimmed, scope+"/")) {
			return true
		}
	}
	return false
}

func pathTreeAssignmentsForRule(files []database.InventoryFile, rule database.ClassificationRule, indexedSignals map[string]filenameSignalModel, tokenCache *filenameTokenProfileCache) map[string]pathTreeWorkGroupAssignment {
	switch strings.TrimSpace(rule.CandidateType) {
	case pathTreeWorkGroupShapeMovieVersionGroup:
		return pathTreeMovieVersionAssignmentsForRule(files, rule, indexedSignals, tokenCache)
	case pathTreeWorkGroupShapeMovieCollection:
		return compileMovieCollectionAssignmentsFromFiles(files, indexedSignals, tokenCache)
	case "independent_movies":
		return pathTreeIndependentMovieAssignmentsForRule(files, rule, indexedSignals, tokenCache)
	case pathTreeWorkGroupShapeSeries:
		return pathTreeSeriesAssignmentsForRule(files, rule, indexedSignals, tokenCache)
	default:
		return nil
	}
}

func pathTreeMovieVersionAssignmentsForRule(files []database.InventoryFile, rule database.ClassificationRule, indexedSignals map[string]filenameSignalModel, tokenCache *filenameTokenProfileCache) map[string]pathTreeWorkGroupAssignment {
	var payload struct {
		Title string `json:"title"`
		Year  *int   `json:"year"`
	}
	_ = json.Unmarshal([]byte(strings.TrimSpace(rule.PayloadJSON)), &payload)
	if strings.TrimSpace(payload.Title) == "" {
		payload.Title = scanrecognition.CleanTitle(path.Base(strings.TrimSpace(rule.PathPattern)))
	}
	target := pathTreeMovieVersionTargetPath(strings.TrimSpace(rule.PathPattern), pathTreeWorkKey{Title: payload.Title, Year: payload.Year, Normalized: normalizedMovieWorkKeyString(payload.Title, payload.Year)})
	assignments := make(map[string]pathTreeWorkGroupAssignment)
	for _, file := range files {
		if !pathTreeRuleFileEligible(file) {
			continue
		}
		evidence := map[string]any{"source": "classification_rule", "rule_id": rule.ID, "shape": pathTreeWorkGroupShapeMovieVersionGroup, "parent_path": strings.TrimSpace(rule.PathPattern), "title": strings.TrimSpace(payload.Title)}
		if payload.Year != nil {
			evidence["year"] = *payload.Year
		}
		assignments[file.StoragePath] = pathTreeWorkGroupAssignment{StoragePath: file.StoragePath, AssignmentType: pathTreeAssignmentVersion, TargetKey: target, Confidence: 0.95, ReviewState: "auto", Evidence: evidence}
	}
	return assignments
}

func pathTreeIndependentMovieAssignmentsForRule(files []database.InventoryFile, rule database.ClassificationRule, indexedSignals map[string]filenameSignalModel, tokenCache *filenameTokenProfileCache) map[string]pathTreeWorkGroupAssignment {
	assignments := make(map[string]pathTreeWorkGroupAssignment)
	for _, file := range files {
		if !pathTreeRuleFileEligible(file) {
			continue
		}
		signal := pathTreeSignalForFile(file.StoragePath, indexedSignals, tokenCache)
		workKey := normalizedMovieWorkKeyFromSignal(signal.VideoSignal, signal.RawPathData, signal.Identity.TitleCandidate, signal.Identity.Year)
		if strings.TrimSpace(workKey.Normalized) == "" {
			continue
		}
		assignments[file.StoragePath] = pathTreeWorkGroupAssignment{StoragePath: file.StoragePath, AssignmentType: pathTreeAssignmentMovie, TargetKey: pathTreeMovieVersionTargetPath(strings.TrimSpace(rule.PathPattern), workKey), Confidence: 0.95, ReviewState: "auto", Evidence: map[string]any{"source": "classification_rule", "rule_id": rule.ID, "shape": "independent_movies", "parent_path": strings.TrimSpace(rule.PathPattern)}}
	}
	return assignments
}

func pathTreeSeriesAssignmentsForRule(files []database.InventoryFile, rule database.ClassificationRule, indexedSignals map[string]filenameSignalModel, tokenCache *filenameTokenProfileCache) map[string]pathTreeWorkGroupAssignment {
	seriesTitle := strings.TrimSpace(rule.SeriesTitle)
	if seriesTitle == "" {
		seriesTitle = scanrecognition.CleanTitle(path.Base(strings.TrimSpace(rule.PathPattern)))
	}
	assignments := make(map[string]pathTreeWorkGroupAssignment)
	for _, file := range files {
		if !pathTreeRuleFileEligible(file) {
			continue
		}
		signal := pathTreeSignalForFile(file.StoragePath, indexedSignals, tokenCache)
		season := firstNonNilInt(signal.VideoSignal.Season, signal.PathHints.SeasonNumber, scanrecognition.ParseFolderName(path.Base(path.Dir(file.StoragePath))).Season, rule.SeasonNumber)
		episode := signal.VideoSignal.Episode
		if season == nil || episode == nil {
			continue
		}
		assignments[file.StoragePath] = pathTreeWorkGroupAssignment{StoragePath: file.StoragePath, AssignmentType: pathTreeAssignmentEpisode, TargetKey: canonicalSeriesPath(seriesTitle), SeriesTitle: seriesTitle, SeasonNumber: season, EpisodeNumber: episode, Confidence: 0.95, ReviewState: "auto", Evidence: map[string]any{"source": "classification_rule", "rule_id": rule.ID, "shape": pathTreeWorkGroupShapeSeries, "parent_path": strings.TrimSpace(rule.PathPattern)}}
	}
	return assignments
}

func pathTreeRuleFileEligible(file database.InventoryFile) bool {
	return file.ID != 0 && strings.TrimSpace(file.StoragePath) != "" && file.Status == inventory.FileStatusAvailable && file.ContentClass == SourceContentClassVideo && isVideoFile(file.StoragePath)
}

func pathTreeSignalForFile(storagePath string, indexedSignals map[string]filenameSignalModel, tokenCache *filenameTokenProfileCache) filenameSignalModel {
	if signal, ok := indexedSignals[strings.TrimSpace(storagePath)]; ok {
		return signal
	}
	return filenameTokenProfileForPath(tokenCache, storagePath)
}

func compileAmbiguousPathTreeReviewAssignmentsFromFiles(files []database.InventoryFile, indexedSignals map[string]filenameSignalModel, tokenCache *filenameTokenProfileCache) map[string]pathTreeWorkGroupAssignment {
	type candidate struct {
		StoragePath string
		ParentPath  string
		WorkKey     string
		Signal      filenameSignalModel
	}
	candidatesByParent := make(map[string][]candidate)
	for _, file := range files {
		storagePath := strings.TrimSpace(file.StoragePath)
		if file.ID == 0 || storagePath == "" || file.Status != inventory.FileStatusAvailable || file.ContentClass != SourceContentClassVideo || !isVideoFile(storagePath) {
			continue
		}
		signal, ok := indexedSignals[storagePath]
		if !ok {
			signal = filenameTokenProfileForPath(tokenCache, storagePath)
		}
		if signal.VideoSignal.IsExtra {
			continue
		}
		workKey := normalizedMovieWorkKeyFromSignal(signal.VideoSignal, signal.RawPathData, signal.Identity.TitleCandidate, signal.Identity.Year)
		if strings.TrimSpace(workKey.Normalized) == "" || workKey.Year != nil || signal.VideoSignal.Episode != nil || len(signal.VideoSignal.EpisodeNumbers) > 0 {
			continue
		}
		parentPath := path.Dir(path.Dir(storagePath))
		if parentPath == "." {
			parentPath = path.Dir(storagePath)
		}
		candidatesByParent[parentPath] = append(candidatesByParent[parentPath], candidate{StoragePath: storagePath, ParentPath: parentPath, WorkKey: workKey.Normalized, Signal: signal})
	}
	reviews := make(map[string]pathTreeWorkGroupAssignment)
	for parentPath, candidates := range candidatesByParent {
		if len(candidates) < 2 {
			continue
		}
		uniqueKeys := make(map[string]struct{})
		for _, candidate := range candidates {
			uniqueKeys[candidate.WorkKey] = struct{}{}
		}
		if len(uniqueKeys) == 1 || len(uniqueKeys) == len(candidates) {
			continue
		}
		for _, candidate := range candidates {
			reviews[candidate.StoragePath] = pathTreeWorkGroupAssignment{StoragePath: candidate.StoragePath, AssignmentType: pathTreeAssignmentReview, TargetKey: parentPath, Confidence: 0.5, ReviewState: "review_required", Evidence: map[string]any{"source": "path_tree_work_group", "shape": pathTreeWorkGroupShapeReview, "parent_path": parentPath, "work_key": candidate.WorkKey}}
		}
	}
	if len(reviews) == 0 {
		return nil
	}
	return reviews
}

func pathTreeContentShapePlanForAssignments(parentPath string, assignments map[string]pathTreeWorkGroupAssignment) contentShapeDirectoryPlan {
	if len(assignments) == 0 {
		return contentShapeDirectoryPlan{}
	}
	shapeCounts := make(map[string]int)
	confidence := 1.0
	affected := make([]string, 0, len(assignments))
	for storagePath, assignment := range assignments {
		shape := pathTreeShapeForAssignment(assignment)
		shapeCounts[shape]++
		if assignment.Confidence > 0 && assignment.Confidence < confidence {
			confidence = assignment.Confidence
		}
		affected = append(affected, storagePath)
	}
	if confidence == 1.0 {
		confidence = 0.86
	}
	sort.Strings(affected)
	shape := pathTreeDominantShape(shapeCounts)
	return contentShapeDirectoryPlan{Shape: shape, Confidence: confidence, ReviewState: "auto", Evidence: map[string]any{"source": "path_tree_work_group", "parent_path": strings.TrimSpace(parentPath), "affected_files": affected, "assignment_count": len(assignments)}, Alternatives: []contentShapePlanAlternative{{Shape: contentShapeUnknownReview, Confidence: 1 - confidence, Reason: "fallback when path-tree work-group evidence is insufficient"}}}
}

func contentShapeAssignmentsFromPathTree(assignments map[string]pathTreeWorkGroupAssignment) []contentShapeFileAssignment {
	if len(assignments) == 0 {
		return nil
	}
	paths := make([]string, 0, len(assignments))
	for storagePath := range assignments {
		paths = append(paths, storagePath)
	}
	sort.Strings(paths)
	items := make([]contentShapeFileAssignment, 0, len(paths))
	for _, storagePath := range paths {
		assignment := assignments[storagePath]
		items = append(items, contentShapeFileAssignment{StoragePath: assignment.StoragePath, AssignmentType: assignment.AssignmentType, TargetKey: assignment.TargetKey, SeriesTitle: assignment.SeriesTitle, SeasonNumber: assignment.SeasonNumber, EpisodeNumber: assignment.EpisodeNumber, AbsoluteNumber: assignment.AbsoluteNumber, AssetRole: assignment.AssetRole, Confidence: assignment.Confidence, ReviewState: assignment.ReviewState, Evidence: assignment.Evidence})
	}
	return items
}

func pathTreeAssignmentsFromContentShapeRecords(rows []database.ContentShapeAssignment) map[string]pathTreeWorkGroupAssignment {
	if len(rows) == 0 {
		return nil
	}
	assignments := make(map[string]pathTreeWorkGroupAssignment, len(rows))
	for _, row := range rows {
		confidence := 0.0
		if row.Confidence != nil {
			confidence = *row.Confidence
		}
		assignments[strings.TrimSpace(row.StoragePath)] = pathTreeWorkGroupAssignment{StoragePath: strings.TrimSpace(row.StoragePath), AssignmentType: strings.TrimSpace(row.AssignmentType), TargetKey: strings.TrimSpace(row.TargetKey), SeriesTitle: strings.TrimSpace(row.SeriesTitle), SeasonNumber: row.SeasonNumber, EpisodeNumber: row.EpisodeNumber, AbsoluteNumber: row.AbsoluteNumber, AssetRole: strings.TrimSpace(row.AssetRole), Confidence: confidence, ReviewState: strings.TrimSpace(row.ReviewState)}
	}
	return assignments
}

func pathTreeShapeForAssignment(assignment pathTreeWorkGroupAssignment) string {
	if shape, ok := assignment.Evidence["shape"].(string); ok && strings.TrimSpace(shape) != "" {
		return strings.TrimSpace(shape)
	}
	switch assignment.AssignmentType {
	case pathTreeAssignmentVersion:
		return pathTreeWorkGroupShapeMovieVersionGroup
	case pathTreeAssignmentEpisode:
		return pathTreeWorkGroupShapeSeries
	case pathTreeAssignmentMovie:
		return pathTreeWorkGroupShapeMovieCollection
	default:
		return pathTreeWorkGroupShapeReview
	}
}

func pathTreeDominantShape(counts map[string]int) string {
	shape := ""
	count := 0
	for candidate, candidateCount := range counts {
		if candidateCount > count || (candidateCount == count && candidate < shape) {
			shape = candidate
			count = candidateCount
		}
	}
	return shape
}

func pathTreeMovieCollectionEvidence(children []pathTreeMovieCollectionChild) bool {
	if len(children) < 2 {
		return false
	}
	uniqueKeys := make(map[string]struct{}, len(children))
	for _, child := range children {
		key := strings.TrimSpace(child.Summary.MovieWorkKey.Normalized)
		if key == "" || child.Summary.MovieWorkKey.Year == nil || child.Summary.Signal.VideoSignal.Episode != nil || len(child.Summary.Signal.VideoSignal.EpisodeNumbers) > 0 {
			return false
		}
		uniqueKeys[key] = struct{}{}
	}
	return len(uniqueKeys) >= 2 && float64(len(uniqueKeys))/float64(len(children)) >= 0.5
}

func pathTreeSiblingMovieVersionsHaveReleaseEvidence(children []pathTreeSiblingMovieVersionChild) bool {
	if len(children) < 2 {
		return false
	}
	withReleaseHints := 0
	basenames := make(map[string]struct{}, len(children))
	for _, child := range children {
		if pathTreeReleaseHintCount(child.Summary.Signal.VideoSignal) > 0 {
			withReleaseHints++
		}
		base := titleclean.NormalizeMovieWorkTitle(strings.TrimSuffix(path.Base(child.Summary.StoragePath), path.Ext(child.Summary.StoragePath)))
		if base != "" {
			basenames[base] = struct{}{}
		}
	}
	return withReleaseHints > 0 && len(basenames) == 1
}

func pathTreeMovieVersionTargetPath(parentPath string, key pathTreeWorkKey) string {
	title := strings.TrimSpace(key.Title)
	if title == "" {
		title = scanrecognition.CleanTitle(key.Normalized)
	}
	if key.Year != nil {
		return path.Join(strings.TrimSpace(parentPath), fmt.Sprintf("%s (%d)", title, *key.Year))
	}
	return path.Join(strings.TrimSpace(parentPath), title)
}
