package storage

import (
	"context"
	"errors"
	"time"
)

var ErrNotImplemented = errors.New("storage operation not implemented")

type Provider interface {
	Name() string
	List(ctx context.Context, req ListRequest) ([]Object, error)
	Get(ctx context.Context, req GetRequest) (Object, error)
	Link(ctx context.Context, req LinkRequest) (LinkResult, error)
	ResolveStorage(ctx context.Context, req ResolveStorageRequest) (ResolvedStorage, error)
	Capabilities(ctx context.Context) (Capabilities, error)
}

type ListRequest struct {
	Path    string
	Refresh bool
	Page    int
	PerPage int
}

type GetRequest struct {
	Path string
}

type LinkRequest struct {
	Path string
}

type ResolveStorageRequest struct {
	Path string
}

type Capabilities struct {
	CanList bool `json:"can_list"`
	CanGet  bool `json:"can_get"`
	CanLink bool `json:"can_link"`
}

type Object struct {
	Name           string            `json:"name"`
	Path           string            `json:"path"`
	IsDir          bool              `json:"is_dir"`
	Size           int64             `json:"size"`
	Created        *time.Time        `json:"created,omitempty"`
	Modified       *time.Time        `json:"modified,omitempty"`
	RawURL         string            `json:"raw_url,omitempty"`
	ThumbnailURL   string            `json:"thumbnail_url,omitempty"`
	StableIdentity string            `json:"stable_identity,omitempty"`
	Provider       string            `json:"provider,omitempty"`
	HashInfo       map[string]string `json:"hash_info,omitempty"`
	ObjectType     string            `json:"object_type,omitempty"`
	Sign           string            `json:"-"`
	Related        []Object          `json:"-"`
	ProviderMeta   map[string]string `json:"-"`
}

func (o Object) CloneProviderMeta() map[string]string {
	return CloneStringMap(o.ProviderMeta)
}

func (o Object) SanitizedProviderMeta() map[string]string {
	return CloneStringMap(o.ProviderMeta)
}

func CloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		if key == "" || value == "" {
			continue
		}
		cloned[key] = value
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

type LinkResult struct {
	URL string `json:"url"`
}

type ResolvedStorage struct {
	Provider string       `json:"provider"`
	Path     string       `json:"path"`
	Object   Object       `json:"object"`
	Caps     Capabilities `json:"capabilities"`
}
