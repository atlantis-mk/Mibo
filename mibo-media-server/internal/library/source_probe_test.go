package library

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/storage"
)

func TestProbeSourceClassifiesSampledEntries(t *testing.T) {
	t.Parallel()

	provider := fakeProbeProvider{objects: map[string][]storage.Object{
		"/media": {
			{Path: "/media/movie.mkv"},
			{Path: "/media/song.flac"},
			{Path: "/media/book.epub"},
			{Path: "/media/poster.jpg"},
			{Path: "/media/archive.bin"},
		},
	}}
	summary := probeSource(context.Background(), provider, "/media", sourceProbeOptions{MaxDuration: time.Second, MaxObjects: 20, MaxDepth: 1, PageSize: 20})

	if summary.Status != SourceProbeStatusReady {
		t.Fatalf("expected ready probe, got %#v", summary)
	}
	for _, className := range []string{SourceContentClassVideo, SourceContentClassAudio, SourceContentClassText, SourceContentClassImage, SourceContentClassOther} {
		if summary.Classes[className] != 1 {
			t.Fatalf("expected one %s file, got %#v", className, summary.Classes)
		}
	}
	if summary.SampledObjects != 5 || summary.SampledFiles != 5 {
		t.Fatalf("unexpected sample counts: %#v", summary)
	}
}

func TestProbeSourceStopsWhenBudgetLimited(t *testing.T) {
	t.Parallel()

	provider := fakeProbeProvider{objects: map[string][]storage.Object{
		"/media": {
			{Path: "/media/one.mkv"},
			{Path: "/media/two.mkv"},
			{Path: "/media/three.mkv"},
		},
	}}
	summary := probeSource(context.Background(), provider, "/media", sourceProbeOptions{MaxDuration: time.Second, MaxObjects: 2, MaxDepth: 1, PageSize: 20})

	if summary.Status != SourceProbeStatusPartial || !summary.BudgetLimited {
		t.Fatalf("expected budget-limited partial probe, got %#v", summary)
	}
	if summary.SampledObjects != 2 {
		t.Fatalf("expected exactly two sampled objects, got %#v", summary)
	}
}

func TestProbeSourceReportsInitialListError(t *testing.T) {
	t.Parallel()

	summary := probeSource(context.Background(), fakeProbeProvider{err: errors.New("boom")}, "/media", sourceProbeOptions{MaxDuration: time.Second, MaxObjects: 20, MaxDepth: 1, PageSize: 20})
	if summary.Status != SourceProbeStatusError || summary.Error == "" {
		t.Fatalf("expected probe error summary, got %#v", summary)
	}
}

type fakeProbeProvider struct {
	objects map[string][]storage.Object
	err     error
}

func (p fakeProbeProvider) Name() string { return "fake" }

func (p fakeProbeProvider) List(ctx context.Context, req storage.ListRequest) ([]storage.Object, error) {
	if p.err != nil {
		return nil, p.err
	}
	return append([]storage.Object(nil), p.objects[req.Path]...), nil
}

func (p fakeProbeProvider) Get(ctx context.Context, req storage.GetRequest) (storage.Object, error) {
	return storage.Object{}, storage.ErrNotImplemented
}

func (p fakeProbeProvider) Link(ctx context.Context, req storage.LinkRequest) (storage.LinkResult, error) {
	return storage.LinkResult{}, storage.ErrNotImplemented
}

func (p fakeProbeProvider) ResolveStorage(ctx context.Context, req storage.ResolveStorageRequest) (storage.ResolvedStorage, error) {
	return storage.ResolvedStorage{Provider: p.Name(), Path: req.Path, Object: storage.Object{Path: req.Path, IsDir: true}}, nil
}

func (p fakeProbeProvider) Capabilities(ctx context.Context) (storage.Capabilities, error) {
	return storage.Capabilities{CanList: true}, nil
}
