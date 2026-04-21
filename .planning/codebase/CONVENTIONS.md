# Coding Conventions

**Analysis Date:** 2026-04-21

## Naming Patterns

**Files:**
- Frontend files in `web/src/` use kebab-case names such as `web/src/components/setup-wizard.tsx`, `web/src/features/app/components/browse-app-shell.tsx`, and `web/src/features/app/hooks/use-app-controller.ts`.
- Frontend hook files use the `use-*.ts` / `use-*.tsx` pattern in `web/src/features/app/hooks/`, such as `web/src/features/app/hooks/use-auth-state.ts` and `web/src/features/app/hooks/use-library-data-state.ts`.
- Backend files in `mibo-media-server/internal/` use lowercase package-oriented names such as `mibo-media-server/internal/httpapi/router.go`, `mibo-media-server/internal/library/service.go`, and `mibo-media-server/internal/worker/worker.go`.
- Go tests use Go’s standard `*_test.go` naming, for example `mibo-media-server/internal/httpapi/router_test.go` and `mibo-media-server/internal/worker/worker_test.go`.

**Functions:**
- React components use PascalCase function names, for example `SetupWizard` in `web/src/components/setup-wizard.tsx` and `BrowseAppShell` in `web/src/features/app/components/browse-app-shell.tsx`.
- React hooks use the `useX` prefix, for example `useAppController` in `web/src/features/app/hooks/use-app-controller.ts`.
- Frontend event handlers and helpers use camelCase verbs such as `handleRegister`, `handleCreateSource`, `loadPath`, and `refreshStatus` in `web/src/components/setup-wizard.tsx` and `web/src/components/path-picker.tsx`.
- Go exported functions use PascalCase (`New`, `Run`, `CreateMediaSource`) and unexported helpers use lowerCamelCase (`writeJSON`, `decodeJSON`, `bearerToken`) in `mibo-media-server/internal/httpapi/router.go` and `mibo-media-server/internal/worker/worker.go`.

**Variables:**
- Frontend local state and variables use camelCase, such as `apiBaseUrl`, `draftApiBaseUrl`, `selectedLibraryType`, and `errorMessage` in `web/src/components/setup-wizard.tsx` and `web/src/components/path-picker.tsx`.
- Frontend constants use SCREAMING_SNAKE_CASE when shared, such as `DEFAULT_OPENLIST_BASE_URL` and `API_BASE_STORAGE_KEY` in `web/src/features/app/constants.ts` and `web/src/lib/client-config.ts`.
- Backend locals use short camelCase names such as `cfg`, `db`, `jobsSvc`, `librarySvc`, and `requestID` in `mibo-media-server/internal/app/app.go` and `mibo-media-server/internal/httpapi/router.go`.

**Types:**
- Frontend prop and model aliases use PascalCase, often with a `Props` suffix, for example `PathPickerProps` in `web/src/components/path-picker.tsx`, `BrowseAppShellProps` in `web/src/features/app/components/browse-app-shell.tsx`, and `AppRouteState` in `web/src/features/app/hooks/use-app-controller.ts`.
- API payload and domain types in `web/src/lib/mibo-api.ts` use PascalCase names and snake_case field names that mirror backend JSON, such as `SetupStatus`, `MediaSource`, and `request_id`.
- Go structs use PascalCase field names plus explicit JSON tags, for example `CreateMediaSourceInput` in `mibo-media-server/internal/library/service.go` and `envelope` in `mibo-media-server/internal/httpapi/router.go`.

## Code Style

**Formatting:**
- Frontend formatting is controlled by Prettier in `web/.prettierrc`.
- Use no semicolons, double quotes, `tabWidth: 2`, `trailingComma: "es5"`, and `printWidth: 80` as defined in `web/.prettierrc`.
- Tailwind classes are auto-sorted by `prettier-plugin-tailwindcss` configured in `web/.prettierrc`.
- Backend Go code follows `gofmt` style; no separate formatter config is detected under `mibo-media-server/`.

**Linting:**
- Frontend linting is ESLint flat config in `web/eslint.config.js`.
- Apply `@eslint/js`, `typescript-eslint`, `eslint-plugin-react-hooks`, and `eslint-plugin-react-refresh` to `web/**/*.{ts,tsx}` via `web/eslint.config.js`.
- Ignore `web/dist/` through `globalIgnores(['dist'])` in `web/eslint.config.js`.
- No dedicated Go linter config such as `.golangci.yml` is detected under `mibo-media-server/`.

## Import Organization

**Order:**
1. External libraries first, for example React, router, icons, and `sonner` in `web/src/components/setup-wizard.tsx` and `web/src/router.tsx`.
2. Internal alias imports from `@/…` next, for example `@/components/ui/button`, `@/features/app/constants`, and `@/lib/mibo-api` in `web/src/components/setup-wizard.tsx`.
3. Relative imports last when staying inside a feature folder, for example `../formatters` and `./use-auth-state` in `web/src/features/app/hooks/use-app-controller.ts`.

**Path Aliases:**
- Frontend uses the `@/*` alias from `web/tsconfig.json`.
- `web/components.json` standardizes aliases for `@/components`, `@/components/ui`, `@/lib`, `@/lib/utils`, and `@/hooks`.
- Backend imports use full module paths rooted at `github.com/atlan/mibo-media-server/...`, for example in `mibo-media-server/cmd/mibo-media-server/main.go` and `mibo-media-server/internal/app/app.go`.

## Error Handling

**Patterns:**
- Frontend API errors are normalized through `ApiError` in `web/src/lib/mibo-api.ts` and handled with `instanceof ApiError` checks in `web/src/components/setup-wizard.tsx` and `web/src/features/app/hooks/use-app-controller.ts`.
- Frontend async flows use `try/catch/finally` with loading flags, for example `handleRegister` and `handleCreateSource` in `web/src/components/setup-wizard.tsx` and `loadPath` in `web/src/components/path-picker.tsx`.
- Use user-facing fallback strings when the thrown value is unknown, for example `error instanceof Error ? error.message : "无法浏览路径"` in `web/src/components/path-picker.tsx` and `error instanceof Error ? error.message : "无法加载媒体详情"` in `web/src/router.tsx`.
- Backend returns errors instead of panicking. Wrap context with `fmt.Errorf("...: %w", err)` when adding call-site detail, as in `mibo-media-server/internal/app/app.go`, `mibo-media-server/internal/library/service.go`, and `mibo-media-server/internal/library/scan.go`.
- HTTP handlers validate early and return immediately through `writeError(...)`, `decodeJSON(...)`, and `requireUser(...)` in `mibo-media-server/internal/httpapi/router.go`.
- Request decoding is strict: `decodeJSON` in `mibo-media-server/internal/httpapi/router.go` uses `decoder.DisallowUnknownFields()` and rejects multiple JSON documents.

## Logging

**Framework:** log / toast

**Patterns:**
- Frontend uses `toast.success(...)` and `toast.error(...)` from `sonner`, for example in `web/src/components/setup-wizard.tsx` and `web/src/features/app/hooks/use-app-controller.ts`.
- No `console.*` logging is detected under `web/src/`.
- Backend uses the standard library `log` package, for example `log.Fatalf(...)` in `mibo-media-server/cmd/mibo-media-server/main.go`, `log.Printf(...)` in `mibo-media-server/internal/app/app.go`, and request logging in `mibo-media-server/internal/httpapi/router.go`.
- HTTP logging is centralized in `loggingMiddleware` in `mibo-media-server/internal/httpapi/router.go`; include request ID, method, path, status, duration, and recovered panics there instead of ad hoc per-handler logging.

## Comments

**When to Comment:**
- Comment sparingly. Most application files in `web/src/` and `mibo-media-server/internal/` rely on descriptive names instead of inline comments.
- Inline comments mainly appear in generated or framework-heavy UI primitives such as `web/src/components/ui/sidebar.tsx` and `web/src/components/ui/chart.tsx`.

**JSDoc/TSDoc:**
- Not detected in `web/src/`.
- Go doc comments for exported symbols are not a consistent pattern in `mibo-media-server/internal/`.

## Function Design

**Size:**
- Frontend keeps reusable UI elements as focused components like `PathPicker` in `web/src/components/path-picker.tsx`, but orchestration hooks can be large and stateful, especially `useAppController` in `web/src/features/app/hooks/use-app-controller.ts`.
- Backend packages group related methods on service structs such as `Service` in `mibo-media-server/internal/library/service.go` and `Runner` in `mibo-media-server/internal/worker/worker.go`.

**Parameters:**
- Frontend component parameters are typed props objects, for example `BrowseAppShellProps` in `web/src/features/app/components/browse-app-shell.tsx`.
- Frontend callbacks are passed explicitly as props, for example `onOpenLibrary`, `onRefresh`, and `onSaveProgress` in `web/src/features/app/components/browse-app-shell.tsx`.
- Backend functions usually accept `context.Context` first and domain-specific input structs next, for example `CreateMediaSource(ctx context.Context, input CreateMediaSourceInput)` in `mibo-media-server/internal/library/service.go`.

**Return Values:**
- Frontend helper APIs favor concrete typed returns and nullable state, for example `StorageBrowseResult | null` in `web/src/components/path-picker.tsx` and typed API calls from `web/src/lib/mibo-api.ts`.
- Backend methods follow Go’s `(value, error)` convention throughout `mibo-media-server/internal/library/service.go`, `mibo-media-server/internal/database/database.go`, and `mibo-media-server/internal/httpapi/router.go`.
- HTTP responses are always wrapped in a JSON envelope with `request_id`, `data`, and optional `error` via `writeResponse(...)` in `mibo-media-server/internal/httpapi/router.go`.

## Module Design

**Exports:**
- Frontend favors named exports for components, hooks, and helpers, for example `export function SetupWizard()` in `web/src/components/setup-wizard.tsx` and `export function BrowseAppShell()` in `web/src/features/app/components/browse-app-shell.tsx`.
- The main exception is `web/src/App.tsx`, which uses `export default App`.
- Backend packages expose constructors and methods from package files, for example `NewService` in `mibo-media-server/internal/library/service.go` and `NewRunner` in `mibo-media-server/internal/worker/worker.go`.

**Barrel Files:**
- Not detected in `web/src/`; imports point directly at concrete files such as `@/components/ui/button` and `@/features/app/pages/home-page`.
- Not applicable in `mibo-media-server/`; Go packages are imported by package path instead of TypeScript-style barrels.

---

*Convention analysis: 2026-04-21*
