package library

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/storage"
)

const browsePerPage = 500

type BrowseResult struct {
	Provider    string           `json:"provider"`
	RootPath    string           `json:"root_path"`
	CurrentPath string           `json:"current_path"`
	ParentPath  string           `json:"parent_path,omitempty"`
	Items       []storage.Object `json:"items"`
}

type OpenListTestResult struct {
	Status   string `json:"status"`
	Provider string `json:"provider"`
	Message  string `json:"message"`
	RootPath string `json:"root_path"`
}

func (s *Service) BrowseProviderPath(ctx context.Context, providerName, inputPath string, refresh bool) (BrowseResult, error) {
	provider, err := s.storage.Get(providerName)
	if err != nil {
		return BrowseResult{}, err
	}

	rootPath := s.providerRootPath(provider.Name())
	return s.browsePath(ctx, provider, rootPath, inputPath, refresh)
}

func (s *Service) BrowseMediaSourcePath(ctx context.Context, sourceID uint, inputPath string, refresh bool) (BrowseResult, error) {
	source, provider, err := s.providerForSource(ctx, sourceID)
	if err != nil {
		return BrowseResult{}, err
	}

	return s.browsePath(ctx, provider, source.RootPath, inputPath, refresh)
}

func (s *Service) BrowseTemporaryOpenListPath(ctx context.Context, cfg providers.OpenListSourceConfig, inputPath string, refresh bool) (BrowseResult, error) {
	provider, err := s.storage.Build("openlist", &providers.SourceConfig{OpenList: &cfg}, "/")
	if err != nil {
		return BrowseResult{}, err
	}
	return s.browsePath(ctx, provider, "/", inputPath, refresh)
}

func (s *Service) TestTemporaryOpenListConnection(ctx context.Context, cfg providers.OpenListSourceConfig) (OpenListTestResult, error) {
	provider, err := s.storage.Build("openlist", &providers.SourceConfig{OpenList: &cfg}, "/")
	if err != nil {
		return OpenListTestResult{}, err
	}

	resolved, err := provider.ResolveStorage(ctx, storage.ResolveStorageRequest{Path: "/"})
	if err != nil {
		return OpenListTestResult{}, err
	}

	return OpenListTestResult{
		Status:   "ok",
		Provider: provider.Name(),
		Message:  "OpenList 连接成功，可以继续选择根路径",
		RootPath: resolved.Path,
	}, nil
}

func (s *Service) browsePath(ctx context.Context, provider storage.Provider, rootPath, inputPath string, refresh bool) (BrowseResult, error) {
	providerName := provider.Name()
	normalizedRootPath := normalizePathForProvider(providerName, rootPath)
	targetPath := normalizePathForProvider(providerName, inputPath)
	if strings.TrimSpace(inputPath) == "" {
		targetPath = normalizedRootPath
	}

	if !isPathWithinRoot(providerName, normalizedRootPath, targetPath) {
		return BrowseResult{}, fmt.Errorf("path %s is outside browse root %s", targetPath, normalizedRootPath)
	}

	resolved, err := provider.ResolveStorage(ctx, storage.ResolveStorageRequest{Path: targetPath})
	if err != nil {
		return BrowseResult{}, err
	}
	if !resolved.Object.IsDir {
		return BrowseResult{}, fmt.Errorf("path %s is not a directory", targetPath)
	}

	objects, err := provider.List(ctx, storage.ListRequest{
		Path:    targetPath,
		Refresh: refresh,
		Page:    1,
		PerPage: browsePerPage,
	})
	if err != nil {
		return BrowseResult{}, err
	}

	directories := make([]storage.Object, 0, len(objects))
	for _, object := range objects {
		if object.IsDir {
			directories = append(directories, object)
		}
	}
	sort.Slice(directories, func(i, j int) bool {
		return directories[i].Name < directories[j].Name
	})

	result := BrowseResult{
		Provider:    providerName,
		RootPath:    normalizedRootPath,
		CurrentPath: targetPath,
		Items:       directories,
	}
	if parentPath, ok := parentPathWithinRoot(providerName, normalizedRootPath, targetPath); ok {
		result.ParentPath = parentPath
	}

	return result, nil
}

func (s *Service) providerRootPath(providerName string) string {
	if strings.EqualFold(strings.TrimSpace(providerName), "local") {
		return strings.TrimSpace(s.cfg.Local.RootPath)
	}
	if strings.EqualFold(strings.TrimSpace(providerName), "openlist") {
		return normalizePath(strings.TrimSpace(s.cfg.OpenList.RootPath))
	}
	return "/"
}

func isPathWithinRoot(providerName, rootPath, targetPath string) bool {
	if strings.EqualFold(strings.TrimSpace(providerName), "local") {
		cleanRoot := filepath.Clean(rootPath)
		cleanTarget := filepath.Clean(targetPath)
		if cleanRoot == string(filepath.Separator) {
			return true
		}
		rel, err := filepath.Rel(cleanRoot, cleanTarget)
		if err != nil {
			return false
		}
		return rel == "." || (!strings.HasPrefix(rel, "..") && rel != "..")
	}

	cleanRoot := normalizePath(rootPath)
	cleanTarget := normalizePath(targetPath)
	if cleanRoot == "/" {
		return true
	}
	return cleanTarget == cleanRoot || strings.HasPrefix(cleanTarget, cleanRoot+"/")
}

func parentPathWithinRoot(providerName, rootPath, targetPath string) (string, bool) {
	if !isPathWithinRoot(providerName, rootPath, targetPath) {
		return "", false
	}

	if strings.EqualFold(strings.TrimSpace(providerName), "local") {
		cleanRoot := filepath.Clean(rootPath)
		cleanTarget := filepath.Clean(targetPath)
		if cleanTarget == cleanRoot {
			return "", false
		}
		parent := filepath.Dir(cleanTarget)
		if !isPathWithinRoot(providerName, cleanRoot, parent) {
			return "", false
		}
		return parent, true
	}

	cleanRoot := normalizePath(rootPath)
	cleanTarget := normalizePath(targetPath)
	if cleanTarget == cleanRoot {
		return "", false
	}
	parent := path.Dir(cleanTarget)
	if !isPathWithinRoot(providerName, cleanRoot, parent) {
		return "", false
	}
	return parent, true
}
