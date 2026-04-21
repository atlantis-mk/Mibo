# Testing Patterns

**Analysis Date:** 2026-04-21

## Test Framework

**Runner:**
- Backend: Go standard testing package via `go test`.
- Config: No dedicated Go test config file is required; tests live directly beside packages in `mibo-media-server/internal/httpapi/router_test.go`, `mibo-media-server/internal/metadata/service_test.go`, and `mibo-media-server/internal/worker/worker_test.go`.
- Frontend: No automated test runner is detected under `web/`; no `vitest.config.*`, `jest.config.*`, or frontend `*.test.*` / `*.spec.*` files are present.

**Assertion Library:**
- Backend uses the standard library only: `testing`, `httptest`, manual `if ... { t.Fatalf(...) }`, and `t.Fatal(...)` assertions in `mibo-media-server/internal/httpapi/router_test.go`, `mibo-media-server/internal/metadata/service_test.go`, and `mibo-media-server/internal/worker/worker_test.go`.
- No `testify`, `gomock`, or frontend assertion library is detected.

**Run Commands:**
```bash
cd mibo-media-server && go test ./...                                # Run all backend tests
cd mibo-media-server && go test ./internal/httpapi -run TestReadyz   # Run focused HTTP API test
cd mibo-media-server && go test ./internal/worker -run TestRunOnceProcessesSyncLibraryJob  # Run focused worker test
cd mibo-media-server && go test ./... -cover                         # View backend coverage locally
```

## Test File Organization

**Location:**
- Backend tests are package-local and sit next to implementation files, for example `mibo-media-server/internal/httpapi/router.go` with `mibo-media-server/internal/httpapi/router_test.go`.
- Frontend automated tests are not detected under `web/src/`.

**Naming:**
- Go test files use `*_test.go`, such as `mibo-media-server/internal/metadata/service_test.go` and `mibo-media-server/internal/worker/worker_test.go`.
- Test functions use descriptive `TestXxx` names that spell out the behavior, for example `TestSearchCandidatesReturnsHelpfulTMDBAuthError` in `mibo-media-server/internal/metadata/service_test.go` and `TestBrowseMediaSourceEndpointRestrictsToSourceRoot` in `mibo-media-server/internal/httpapi/router_test.go`.

**Structure:**
```
mibo-media-server/internal/httpapi/router.go
mibo-media-server/internal/httpapi/router_test.go
mibo-media-server/internal/metadata/service.go
mibo-media-server/internal/metadata/service_test.go
mibo-media-server/internal/worker/worker.go
mibo-media-server/internal/worker/worker_test.go
web/src/                       # No automated test files detected
```

## Test Structure

**Suite Organization:**
```typescript
// Pattern mirrored from `mibo-media-server/internal/httpapi/router_test.go`
func TestReadyz(t *testing.T) {
    openList := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(map[string]any{ ... })
    }))
    defer openList.Close()

    db, err := database.Open(config.DatabaseConfig{
        Driver: "sqlite",
        DSN: filepath.Join(t.TempDir(), "mibo.db"),
    })
    if err != nil {
        t.Fatalf("open database: %v", err)
    }

    request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
    recorder := httptest.NewRecorder()
    router.ServeHTTP(recorder, request)

    if recorder.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
    }
}
```

**Patterns:**
- Arrange everything inline inside each test: fake servers, sqlite database, config, services, and router setup are all explicit in `mibo-media-server/internal/httpapi/router_test.go` and `mibo-media-server/internal/worker/worker_test.go`.
- Assert with direct guard clauses (`if got != want { t.Fatalf(...) }`) instead of matcher libraries.
- Decode JSON responses into local anonymous structs or package types, for example in `mibo-media-server/internal/httpapi/router_test.go`.
- Helper functions are extracted only when repeated setup becomes heavy, for example `newDeleteTestRouter`, `seedLibraryData`, `createAuthHeader`, and `assertDeletedCount` in `mibo-media-server/internal/httpapi/router_test.go`.
- Subtests with `t.Run(...)` are not used in the current backend test files.

## Mocking

**Framework:** Standard library fakes only

**Patterns:**
```typescript
// HTTP dependency stub pattern from `mibo-media-server/internal/worker/worker_test.go`
openList := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]any{ ... })
}))
defer openList.Close()

// Binary stub pattern from `mibo-media-server/internal/worker/worker_test.go`
func writeFakeFFprobe(t *testing.T) string {
    t.Helper()
    path := filepath.Join(t.TempDir(), "ffprobe")
    if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
        t.Fatalf("write fake ffprobe: %v", err)
    }
    return path
}
```

**What to Mock:**
- External HTTP services with `httptest.NewServer(...)`, especially OpenList and TMDB in `mibo-media-server/internal/httpapi/router_test.go`, `mibo-media-server/internal/metadata/service_test.go`, and `mibo-media-server/internal/worker/worker_test.go`.
- OS-level tool dependencies by writing a temporary executable, as done for `ffprobe` in `mibo-media-server/internal/worker/worker_test.go` and `mibo-media-server/internal/httpapi/router_test.go`.
- Authentication setup through real service calls plus helper wrappers such as `createAuthHeader(...)` in `mibo-media-server/internal/httpapi/router_test.go`.

**What NOT to Mock:**
- The database is usually real sqlite opened against `t.TempDir()` via `database.Open(...)`, not mocked, in all three backend test files.
- Service wiring is typically real. Tests construct `auth.NewService(...)`, `library.NewService(...)`, `metadata.NewService(...)`, and `playback.NewService(...)` directly in `mibo-media-server/internal/httpapi/router_test.go` and `mibo-media-server/internal/worker/worker_test.go`.

## Fixtures and Factories

**Test Data:**
```typescript
// Factory/helper pattern from `mibo-media-server/internal/httpapi/router_test.go`
func seedLibraryData(t *testing.T, ctx context.Context, db *gorm.DB, authSvc *auth.Service, libraryID uint, rootDir, name string) (uint, uint, uint) {
    t.Helper()
    user, err := authSvc.Register(ctx, fmt.Sprintf("%s-user", strings.ToLower(name)), "password123")
    if err != nil {
        t.Fatalf("register user: %v", err)
    }
    // create item, file, and progress rows
    return user.ID, item.ID, file.ID
}
```

**Location:**
- Fixtures are inline inside each test file rather than centralized in a shared `testdata/` package.
- Reusable helpers live at the bottom of the owning test file, for example in `mibo-media-server/internal/httpapi/router_test.go` and `mibo-media-server/internal/worker/worker_test.go`.

## Coverage

**Requirements:** None enforced

**View Coverage:**
```bash
cd mibo-media-server && go test ./... -cover
```

## Test Types

**Unit Tests:**
- Service-level behavior is tested in package scope with real sqlite plus stubbed HTTP providers, for example `TestMatchItemUsesDatabaseTMDBConfig` and `TestSearchCandidatesReturnsHelpfulTMDBAuthError` in `mibo-media-server/internal/metadata/service_test.go`.

**Integration Tests:**
- Router tests exercise full request/response flows across handlers, auth, database, storage, metadata, playback, and jobs in `mibo-media-server/internal/httpapi/router_test.go`.
- Worker tests run end-to-end job processing using real services and temp infrastructure in `mibo-media-server/internal/worker/worker_test.go`.

**E2E Tests:**
- Frontend/browser E2E framework is not detected under `web/`.
- Backend black-box deployment tests are not detected beyond package-level integration tests.

## Common Patterns

**Async Testing:**
```typescript
// Context timeout pattern from `mibo-media-server/internal/worker/worker_test.go`
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

runner := NewRunner(cfg.Worker, jobsSvc, librarySvc, metadataSvc, probeSvc)
runner.RunOnce(ctx)
```

**Error Testing:**
```typescript
// Error assertion pattern from `mibo-media-server/internal/metadata/service_test.go`
_, err = svc.SearchCandidates(ctx, item.ID, ManualSearchInput{Title: "MovieA"})
if err == nil {
    t.Fatal("expected auth error")
}
if !strings.Contains(err.Error(), "TMDB 认证失败") {
    t.Fatalf("unexpected error: %v", err)
}
```

---

*Testing analysis: 2026-04-21*
