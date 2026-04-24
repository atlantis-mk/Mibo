package providers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/storage"
	"github.com/atlan/mibo-media-server/internal/storage/local"
	"github.com/atlan/mibo-media-server/internal/storage/openlist"
)

type Registry struct {
	cfg   config.Config
	local storage.Provider
}

func NewRegistry(cfg config.Config) *Registry {
	return &Registry{cfg: cfg, local: local.New(cfg.Local)}
}

func (r *Registry) Get(name string) (storage.Provider, error) {
	providerName := strings.ToLower(strings.TrimSpace(name))
	if providerName == "" {
		return nil, fmt.Errorf("storage provider is required")
	}
	if providerName == "local" {
		return r.local, nil
	}
	if providerName == "openlist" {
		cfg, err := NormalizeSourceConfig("openlist", nil, r.cfg)
		if err != nil {
			return nil, err
		}
		return r.buildOpenList(cfg, r.cfg.OpenList.RootPath)
	}
	return nil, fmt.Errorf("unsupported storage provider %q", providerName)
}

func (r *Registry) Build(providerName string, sourceConfig *SourceConfig, rootPath string) (storage.Provider, error) {
	normalizedName := normalizeProviderName(providerName)
	if normalizedName == "local" {
		return r.local, nil
	}
	if normalizedName == "openlist" {
		cfg, err := NormalizeSourceConfig("openlist", sourceConfig, r.cfg)
		if err != nil {
			return nil, err
		}
		return r.buildOpenList(cfg, rootPath)
	}
	return nil, fmt.Errorf("unsupported storage provider %q", providerName)
}

func (r *Registry) BuildForSource(source database.MediaSource) (storage.Provider, error) {
	sourceConfig, err := ParseSourceConfig(source.ConfigJSON)
	if err != nil {
		return nil, err
	}
	return r.Build(source.Provider, &sourceConfig, source.RootPath)
}

func (r *Registry) buildOpenList(sourceConfig SourceConfig, rootPath string) (storage.Provider, error) {
	adapterConfig, err := sourceConfig.OpenListConfig(rootPath)
	if err != nil {
		return nil, err
	}
	return openlist.New(adapterConfig), nil
}

func (r *Registry) Names() []string {
	names := []string{"local", "openlist"}
	sort.Strings(names)
	return names
}
