package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/storage"
)

type Adapter struct {
	rootPath string
}

func New(cfg config.LocalStorageConfig) *Adapter {
	return &Adapter{rootPath: normalizeRootPath(cfg.RootPath)}
}

func (a *Adapter) Name() string {
	return "local"
}

func (a *Adapter) List(_ context.Context, req storage.ListRequest) ([]storage.Object, error) {
	resolved, err := a.resolvePath(req.Path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(resolved)
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	objects := make([]storage.Object, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		itemPath := filepath.Join(resolved, entry.Name())
		object := storage.Object{
			Name:         entry.Name(),
			Path:         itemPath,
			IsDir:        entry.IsDir(),
			RawURL:       itemPath,
			ProviderMeta: localProviderMeta(info),
		}
		object.StableIdentity = localStableIdentity(info)
		if !entry.IsDir() {
			object.Size = info.Size()
		}
		modified := info.ModTime().UTC()
		object.Modified = &modified
		objects = append(objects, object)
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	perPage := req.PerPage
	if perPage <= 0 {
		perPage = len(objects)
		if perPage == 0 {
			perPage = 1000
		}
	}
	start := (page - 1) * perPage
	if start >= len(objects) {
		return []storage.Object{}, nil
	}
	end := start + perPage
	if end > len(objects) {
		end = len(objects)
	}
	return objects[start:end], nil
}

func (a *Adapter) Get(_ context.Context, req storage.GetRequest) (storage.Object, error) {
	resolved, err := a.resolvePath(req.Path)
	if err != nil {
		return storage.Object{}, err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return storage.Object{}, err
	}

	object := storage.Object{
		Name:           filepath.Base(resolved),
		Path:           resolved,
		IsDir:          info.IsDir(),
		Size:           info.Size(),
		RawURL:         resolved,
		StableIdentity: localStableIdentity(info),
		ProviderMeta:   localProviderMeta(info),
	}
	modified := info.ModTime().UTC()
	object.Modified = &modified
	return object, nil
}

func (a *Adapter) Link(context.Context, storage.LinkRequest) (storage.LinkResult, error) {
	return storage.LinkResult{}, storage.ErrNotImplemented
}

func (a *Adapter) ResolveStorage(ctx context.Context, req storage.ResolveStorageRequest) (storage.ResolvedStorage, error) {
	object, err := a.Get(ctx, storage.GetRequest{Path: req.Path})
	if err != nil {
		return storage.ResolvedStorage{}, err
	}

	caps, err := a.Capabilities(ctx)
	if err != nil {
		return storage.ResolvedStorage{}, err
	}

	return storage.ResolvedStorage{
		Provider: a.Name(),
		Path:     object.Path,
		Object:   object,
		Caps:     caps,
	}, nil
}

func (a *Adapter) Capabilities(context.Context) (storage.Capabilities, error) {
	return storage.Capabilities{CanList: true, CanGet: true, CanLink: false}, nil
}

func (a *Adapter) resolvePath(input string) (string, error) {
	resolved := strings.TrimSpace(input)
	if resolved == "" {
		resolved = a.rootPath
	}
	if !filepath.IsAbs(resolved) {
		return "", fmt.Errorf("local storage path must be absolute")
	}
	resolved = filepath.Clean(resolved)
	if !isWithinRoot(a.rootPath, resolved) {
		return "", fmt.Errorf("path %s is outside local storage root %s", resolved, a.rootPath)
	}
	return resolved, nil
}

func normalizeRootPath(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return string(filepath.Separator)
	}
	if !filepath.IsAbs(trimmed) {
		if absolute, err := filepath.Abs(trimmed); err == nil {
			trimmed = absolute
		}
	}
	return filepath.Clean(trimmed)
}

func isWithinRoot(rootPath, targetPath string) bool {
	if rootPath == string(filepath.Separator) {
		return true
	}
	rel, err := filepath.Rel(rootPath, targetPath)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && rel != "..")
}

func localStableIdentity(info os.FileInfo) string {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return ""
	}
	return fmt.Sprintf("local:%d:%d", stat.Dev, stat.Ino)
}

func localProviderMeta(info os.FileInfo) map[string]string {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return nil
	}
	return map[string]string{
		"device": fmt.Sprintf("%d", stat.Dev),
		"inode":  fmt.Sprintf("%d", stat.Ino),
	}
}
