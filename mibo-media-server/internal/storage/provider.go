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
	Modified       *time.Time        `json:"modified,omitempty"`
	RawURL         string            `json:"raw_url,omitempty"`
	StableIdentity string            `json:"stable_identity,omitempty"`
	Provider       string            `json:"provider,omitempty"`
	HashInfo       map[string]string `json:"hash_info,omitempty"`
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
