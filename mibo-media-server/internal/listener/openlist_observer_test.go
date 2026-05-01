package listener

import (
	"context"
	"errors"
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"github.com/atlan/mibo-media-server/internal/library"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

func TestPollLibraryWithProviderPlansCreateDeleteAndStableMove(t *testing.T) {
	t.Parallel()

	svc, db, record := newListenerIntegrationService(t)
	svc.planner.SetStabilityWindow(0)
	ctx := context.Background()
	provider := &pollFakeProvider{name: "local", root: record.RootPath}

	provider.setFiles(storage.Object{Name: "MovieA.mkv", Path: record.RootPath + "/MovieA.mkv", Size: 100, StableIdentity: "stable-a"})
	if err := svc.PollLibraryWithProvider(ctx, record, provider, false); err != nil {
		t.Fatalf("poll create: %v", err)
	}
	assertQueuedRefresh(t, ctx, db, library.JobKindTargetedRefresh)
	clearJobs(t, ctx, db)

	provider.setFiles(storage.Object{Name: "MovieB.mkv", Path: record.RootPath + "/MovieB.mkv", Size: 100, StableIdentity: "stable-a"})
	if err := svc.PollLibraryWithProvider(ctx, record, provider, false); err != nil {
		t.Fatalf("poll stable move: %v", err)
	}
	assertQueuedRefresh(t, ctx, db, library.JobKindTargetedRefresh)
	clearJobs(t, ctx, db)

	provider.setFiles()
	if err := svc.PollLibraryWithProvider(ctx, record, provider, false); err != nil {
		t.Fatalf("poll delete: %v", err)
	}
	assertQueuedRefresh(t, ctx, db, library.JobKindTargetedRefresh)
}

func assertQueuedRefresh(t *testing.T, ctx context.Context, db *gorm.DB, kind string) {
	t.Helper()
	var queued []database.Job
	if err := db.WithContext(ctx).Where("kind = ? AND status = ?", kind, jobs.StatusQueued).Find(&queued).Error; err != nil {
		t.Fatalf("list queued refresh jobs: %v", err)
	}
	if len(queued) != 1 {
		t.Fatalf("expected one queued %s job, got %#v", kind, queued)
	}
}

func clearJobs(t *testing.T, ctx context.Context, db *gorm.DB) {
	t.Helper()
	if err := db.WithContext(ctx).Where("1 = 1").Delete(&database.Job{}).Error; err != nil {
		t.Fatalf("clear jobs: %v", err)
	}
}

type pollFakeProvider struct {
	name  string
	root  string
	files []storage.Object
}

func (p *pollFakeProvider) setFiles(files ...storage.Object) {
	p.files = append([]storage.Object(nil), files...)
}

func (p *pollFakeProvider) Name() string { return p.name }

func (p *pollFakeProvider) List(_ context.Context, req storage.ListRequest) ([]storage.Object, error) {
	if req.Path != p.root {
		return nil, nil
	}
	return append([]storage.Object(nil), p.files...), nil
}

func (p *pollFakeProvider) Get(_ context.Context, req storage.GetRequest) (storage.Object, error) {
	if req.Path == p.root {
		return storage.Object{Name: "Library", Path: p.root, IsDir: true}, nil
	}
	for _, file := range p.files {
		if file.Path == req.Path {
			return file, nil
		}
	}
	return storage.Object{}, errors.New("not found")
}

func (p *pollFakeProvider) Link(context.Context, storage.LinkRequest) (storage.LinkResult, error) {
	return storage.LinkResult{}, storage.ErrNotImplemented
}

func (p *pollFakeProvider) ResolveStorage(ctx context.Context, req storage.ResolveStorageRequest) (storage.ResolvedStorage, error) {
	object, err := p.Get(ctx, storage.GetRequest{Path: req.Path})
	if err != nil {
		return storage.ResolvedStorage{}, err
	}
	return storage.ResolvedStorage{Provider: p.Name(), Path: req.Path, Object: object}, nil
}

func (p *pollFakeProvider) Capabilities(context.Context) (storage.Capabilities, error) {
	return storage.Capabilities{CanList: true, CanGet: true}, nil
}
