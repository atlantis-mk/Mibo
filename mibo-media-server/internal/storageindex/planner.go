package storageindex

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
)

const (
	ChangeKindCreate          = "create"
	ChangeKindUpdate          = "update"
	ChangeKindDelete          = "delete"
	ChangeKindDirectoryDelete = "directory_delete"
	ChangeKindMove            = "move"
	ChangeKindUncertain       = "uncertain"

	defaultPlannerStabilityWindow = 30 * time.Second
	defaultPlannerMaxScopedPaths  = 20
)

type Planner struct {
	now             func() time.Time
	stabilityWindow time.Duration
	maxScopedPaths  int
}

type PlanInput struct {
	LibraryID   uint
	LibraryRoot string
	Previous    []database.StorageIndexEntry
	Current     []database.StorageIndexEntry
}

type Change struct {
	Kind        string
	Path        string
	OldPath     string
	IsDir       bool
	Deferred    bool
	Reason      string
	StableMatch bool
}

type RefreshPlan struct {
	LibraryID uint
	RootPath  string
	FullSync  bool
	Reason    string
}

type PlanResult struct {
	Changes []Change
	Plans   []RefreshPlan
}

func NewPlanner() *Planner {
	return &Planner{
		now:             func() time.Time { return time.Now().UTC() },
		stabilityWindow: defaultPlannerStabilityWindow,
		maxScopedPaths:  defaultPlannerMaxScopedPaths,
	}
}

func (p *Planner) SetStabilityWindow(window time.Duration) {
	p.stabilityWindow = window
}

func (p *Planner) SetMaxScopedPaths(maxPaths int) {
	p.maxScopedPaths = maxPaths
}

func (p *Planner) Plan(input PlanInput) PlanResult {
	root := normalizePath(input.LibraryRoot)
	if root == "" {
		root = string(filepath.Separator)
	}
	previousByPath := entriesByPath(input.Previous)
	currentByPath := entriesByPath(input.Current)
	deleted := make([]database.StorageIndexEntry, 0)
	created := make([]database.StorageIndexEntry, 0)
	changes := make([]Change, 0)

	for pathValue, previous := range previousByPath {
		current, ok := currentByPath[pathValue]
		if !ok || current.ObservationStatus == ObservationStatusMissing {
			deleted = append(deleted, previous)
			continue
		}
		if entryChanged(previous, current) {
			change := Change{Kind: ChangeKindUpdate, Path: current.StoragePath, IsDir: current.IsDir, Reason: "provider_metadata_changed"}
			if p.isUnstable(current) {
				change.Deferred = true
				change.Reason = "file_still_changing"
			}
			changes = append(changes, change)
		}
	}

	for pathValue, current := range currentByPath {
		if _, ok := previousByPath[pathValue]; ok {
			continue
		}
		if current.ObservationStatus == ObservationStatusMissing {
			continue
		}
		created = append(created, current)
	}

	created, deleted, moveChanges := p.detectStableMoves(created, deleted)
	changes = append(changes, moveChanges...)
	for _, entry := range created {
		change := Change{Kind: ChangeKindCreate, Path: entry.StoragePath, IsDir: entry.IsDir, Reason: "new_path"}
		if p.isUnstable(entry) {
			change.Deferred = true
			change.Reason = "file_still_changing"
		}
		changes = append(changes, change)
	}
	for _, entry := range deleted {
		kind := ChangeKindDelete
		if entry.IsDir {
			kind = ChangeKindDirectoryDelete
		}
		changes = append(changes, Change{Kind: kind, Path: entry.StoragePath, IsDir: entry.IsDir, Reason: "missing_from_provider"})
	}

	return PlanResult{Changes: changes, Plans: p.refreshPlans(input.LibraryID, root, changes)}
}

func (p *Planner) detectStableMoves(created []database.StorageIndexEntry, deleted []database.StorageIndexEntry) ([]database.StorageIndexEntry, []database.StorageIndexEntry, []Change) {
	deletedByIdentity := make(map[string]database.StorageIndexEntry)
	for _, entry := range deleted {
		identity := strings.TrimSpace(entry.StableIdentityKey)
		if identity == "" {
			continue
		}
		deletedByIdentity[identity] = entry
	}
	remainingCreated := make([]database.StorageIndexEntry, 0, len(created))
	matchedDeleted := make(map[string]struct{})
	moves := make([]Change, 0)
	for _, entry := range created {
		identity := strings.TrimSpace(entry.StableIdentityKey)
		old, ok := deletedByIdentity[identity]
		if identity == "" || !ok || old.StorageProvider != entry.StorageProvider || old.LibraryID != entry.LibraryID {
			remainingCreated = append(remainingCreated, entry)
			continue
		}
		moves = append(moves, Change{Kind: ChangeKindMove, Path: entry.StoragePath, OldPath: old.StoragePath, IsDir: entry.IsDir, Reason: "stable_identity_match", StableMatch: true})
		matchedDeleted[old.StoragePath] = struct{}{}
	}
	remainingDeleted := make([]database.StorageIndexEntry, 0, len(deleted))
	for _, entry := range deleted {
		if _, ok := matchedDeleted[entry.StoragePath]; ok {
			continue
		}
		remainingDeleted = append(remainingDeleted, entry)
	}
	return remainingCreated, remainingDeleted, moves
}

func (p *Planner) refreshPlans(libraryID uint, libraryRoot string, changes []Change) []RefreshPlan {
	scopes := make([]string, 0)
	seen := make(map[string]struct{})
	for _, change := range changes {
		if change.Deferred {
			continue
		}
		for _, scope := range changeScopes(change, libraryRoot) {
			if _, ok := seen[scope]; ok {
				continue
			}
			seen[scope] = struct{}{}
			scopes = append(scopes, scope)
		}
	}
	if len(scopes) == 0 {
		return nil
	}
	if len(scopes) > p.maxScopedPaths {
		return []RefreshPlan{{LibraryID: libraryID, RootPath: libraryRoot, FullSync: true, Reason: "storage_index_diff"}}
	}
	ancestor := commonAncestor(scopes, libraryRoot)
	if ancestor == libraryRoot && len(scopes) > 1 && len(scopes) > p.maxScopedPaths/2 {
		return []RefreshPlan{{LibraryID: libraryID, RootPath: libraryRoot, FullSync: true, Reason: "storage_index_diff"}}
	}
	return []RefreshPlan{{LibraryID: libraryID, RootPath: ancestor, Reason: "storage_index_diff"}}
}

func (p *Planner) isUnstable(entry database.StorageIndexEntry) bool {
	if entry.IsDir || p.stabilityWindow <= 0 {
		return false
	}
	if entry.FirstObservedAt.IsZero() || entry.LastObservedAt.IsZero() {
		return false
	}
	return entry.LastObservedAt.Sub(entry.FirstObservedAt) < p.stabilityWindow && p.now().Sub(entry.LastObservedAt) < p.stabilityWindow
}

func entriesByPath(entries []database.StorageIndexEntry) map[string]database.StorageIndexEntry {
	result := make(map[string]database.StorageIndexEntry, len(entries))
	for _, entry := range entries {
		pathValue := normalizePath(entry.StoragePath)
		if pathValue == "" {
			continue
		}
		entry.StoragePath = pathValue
		result[pathValue] = entry
	}
	return result
}

func entryChanged(previous database.StorageIndexEntry, current database.StorageIndexEntry) bool {
	if previous.IsDir != current.IsDir || previous.SizeBytes != current.SizeBytes || strings.TrimSpace(previous.StableIdentityKey) != strings.TrimSpace(current.StableIdentityKey) || strings.TrimSpace(previous.HashesJSON) != strings.TrimSpace(current.HashesJSON) || strings.TrimSpace(previous.ProviderMetaJSON) != strings.TrimSpace(current.ProviderMetaJSON) {
		return true
	}
	if previous.ModifiedAt == nil || current.ModifiedAt == nil {
		return previous.ModifiedAt != current.ModifiedAt
	}
	return !previous.ModifiedAt.Equal(*current.ModifiedAt)
}

func changeScopes(change Change, libraryRoot string) []string {
	switch change.Kind {
	case ChangeKindMove:
		return []string{pathParent(change.OldPath, libraryRoot), pathParent(change.Path, libraryRoot)}
	case ChangeKindDirectoryDelete:
		return []string{pathParent(change.Path, libraryRoot)}
	default:
		if change.IsDir {
			return []string{clampPath(change.Path, libraryRoot)}
		}
		return []string{pathParent(change.Path, libraryRoot)}
	}
}

func pathParent(pathValue string, libraryRoot string) string {
	clean := clampPath(pathValue, libraryRoot)
	if clean == libraryRoot {
		return libraryRoot
	}
	parent := normalizePath(filepath.Dir(clean))
	return clampPath(parent, libraryRoot)
}

func commonAncestor(paths []string, libraryRoot string) string {
	if len(paths) == 0 {
		return libraryRoot
	}
	ancestor := clampPath(paths[0], libraryRoot)
	for _, pathValue := range paths[1:] {
		ancestor = commonAncestorPair(ancestor, clampPath(pathValue, libraryRoot), libraryRoot)
	}
	return clampPath(ancestor, libraryRoot)
}

func commonAncestorPair(left string, right string, libraryRoot string) string {
	left = clampPath(left, libraryRoot)
	right = clampPath(right, libraryRoot)
	for left != libraryRoot {
		if right == left || strings.HasPrefix(right, left+string(filepath.Separator)) {
			return left
		}
		left = normalizePath(filepath.Dir(left))
	}
	return libraryRoot
}

func clampPath(pathValue string, libraryRoot string) string {
	clean := normalizePath(pathValue)
	root := normalizePath(libraryRoot)
	if root == "" || root == string(filepath.Separator) {
		if clean == "" {
			return string(filepath.Separator)
		}
		return clean
	}
	if clean == "" || clean == string(filepath.Separator) {
		return root
	}
	rel, err := filepath.Rel(root, clean)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return root
	}
	return clean
}
