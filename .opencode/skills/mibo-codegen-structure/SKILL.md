---
name: mibo-codegen-structure
description: Generate or modify code in Mibo using the repository's real structure rules for routes, features, services, storage adapters, file splitting, and file size thresholds.
license: MIT
compatibility: opencode
metadata:
  audience: maintainers
  workflow: code-generation
---

## What I do

- Apply Mibo's project-specific code generation rules before writing code.
- Choose the correct directory and file placement for frontend and backend changes.
- Keep route, feature, component, API client, handler, service, and adapter responsibilities separate.
- Use file splitting and file size guidance as soft thresholds instead of rigid rules.
- Prevent generic abstractions that conflict with this repository's current architecture.

## When to use me

Use this when generating new code for Mibo.
Use this when adding a new frontend page, feature, component, store, API call, backend service, HTTP endpoint, worker task, or storage adapter.
Use this when refactoring a growing file and deciding whether to split it or keep it intact.
Use this when reviewing whether generated code matches the repo's structure conventions.

## Core repository boundaries

- Product code belongs in `web/` and `mibo-media-server/`.
- `OpenList/` is an upstream boundary and must not be used as the default location for new Mibo product logic.
- Root `package.json` is tooling-only and is not the frontend app manifest.
- Frontend requests must go through `web/src/lib/mibo-api.ts` and related query helpers, not raw `fetch` in features.
- Backend storage access must go through `mibo-media-server/internal/storage/provider.go` and concrete adapters.
- Slow backend work must go through `jobs + worker` instead of being completed inline in request handlers.

## Frontend placement rules

Current frontend structure:

```text
web/src/
├── routes/
├── features/
├── components/
├── components/ui/
├── lib/
├── stores/
└── hooks/
```

Use these rules:

1. Route files go in `web/src/routes/`.
   - Keep them thin.
   - Limit them to `createFileRoute(...)`, parameter parsing, search validation, and rendering a feature entry component.
   - Do not put large UI trees or request orchestration in route files.
2. Business pages go in `web/src/features/<feature>/index.tsx`.
   - Route files should usually import the feature entry and pass parsed params into it.
3. Feature-private UI goes in `web/src/features/<feature>/components/`.
4. Shared product UI goes in `web/src/components/`.
   - Use this only when the component is reused across multiple features.
5. UI primitives go in `web/src/components/ui/`.
   - Keep them free of business fetch logic and feature-specific state.
6. Shared API boundary code goes in `web/src/lib/`.
   - Put endpoint methods and response types in `mibo-api.ts`.
   - Put query keys and `queryOptions` helpers in `mibo-query.ts`.
7. Cross-page client state goes in `web/src/stores/`.
   - Use this only for truly shared client state such as auth.

## Frontend generation rules

- Prefer `routes -> features -> lib/stores/components`.
- Do not generate raw `fetch` calls inside feature pages unless there is a very strong repo-specific reason.
- Prefer React Query for server state.
- Prefer Zustand only for client state that must survive navigation or hydration.
- Keep feature-specific helpers close to the feature instead of promoting them immediately to shared code.
- Preserve existing path alias usage such as `#/*`.
- Preserve existing design system patterns based on shadcn and `components/ui`.

## Backend placement rules

Current backend structure:

```text
mibo-media-server/internal/
├── app/
├── httpapi/
├── config/
├── database/
├── auth/
├── library/
├── metadata/
├── playback/
├── progress/
├── settings/
├── jobs/
├── worker/
├── providers/
└── storage/
```

Use these rules:

1. Executable startup code goes in `cmd/mibo-media-server/main.go`.
   - Keep it thin.
2. App wiring goes in `internal/app/app.go`.
   - Construct services and infrastructure here.
3. HTTP handlers go in `internal/httpapi/`.
   - Keep handlers thin.
   - Limit them to auth, request decode, response mapping, and calling services.
4. Business logic goes in domain packages such as `internal/library/`, `internal/metadata/`, and `internal/playback/`.
5. Storage abstractions go in `internal/storage/`.
   - Concrete implementations belong in provider-specific subdirectories such as `local/` and `openlist/`.
6. Provider registration and source-config normalization go in `internal/providers/`.
7. Async orchestration goes in `internal/jobs/` and `internal/worker/`.

## Backend generation rules

- Prefer one domain directory per business capability.
- Inside one domain package, split by sub-responsibility rather than creating extra packages too early.
- Typical files inside a domain package may include:
  - `service.go`
  - `query.go`
  - `scan.go`
  - `browse.go`
  - `enrichment.go`
- Do not default to a global `controller/service/repository/model` directory architecture for this repo.
- Do not put core business workflows inside `httpapi/router.go`.
- Do not spread provider-specific branching across domain services when the abstraction belongs in `storage.Provider` or `providers.Registry`.

## File splitting rules

Use file splitting when it improves clarity, not just to reduce line count.

Split a file when one or more of these are true:

- It contains multiple independent responsibilities.
- A route file is accumulating page-level data loading and UI.
- A feature entry is holding several clearly independent regions or interactions.
- A backend file mixes request handling, business orchestration, and storage details.
- A domain package now contains several sub-flows that can be named clearly.
- A single edit commonly touches unrelated areas of the same file.

Do not split just because a file is not tiny.
Keep a file intact when it still represents one coherent flow and splitting would only add navigation overhead.

## File size guidance

These are warning thresholds, not hard limits.

Frontend:

- `routes/*.tsx`: 5-30 lines preferred
- `features/*/index.tsx`: 80-250 lines preferred
- `features/*/components/*.tsx`: 50-200 lines preferred
- `components/*.tsx`: 40-180 lines preferred
- `components/ui/*.tsx`: 30-220 lines preferred
- `stores/*.ts`: 20-120 lines preferred
- `hooks/*.ts`: 20-150 lines preferred
- `lib/mibo-query.ts`: 50-200 lines preferred
- `lib/mibo-api.ts`: 200-800 lines acceptable if well organized by protocol sections

Backend:

- `cmd/*/main.go`: 20-60 lines preferred
- `internal/app/app.go`: 60-180 lines preferred
- `internal/*/service.go`: 80-250 lines preferred
- `internal/*/{query,scan,browse,enrichment}.go`: 80-300 lines preferred
- `internal/storage/*/adapter.go`: 100-300 lines preferred
- `internal/database/models.go`: 150-400 lines acceptable
- `internal/httpapi/router.go`: very large files are tolerated temporarily, but topic-based splitting is preferred once it grows unwieldy

General thresholds:

- Under 200 lines is usually healthy.
- 200-400 lines should trigger a structure review.
- 400-700 lines usually means splitting is worth evaluating seriously.
- Above 700 lines is usually too large unless the file is a model or protocol aggregation file with one stable responsibility.

## Reuse and abstraction rules

- Prefer the closest valid location first.
- Promote code to a shared location only after there is stable reuse or a clear shared boundary.
- Do not create broad `utils`, `shared`, or `common` directories as the default destination.
- Do not move single-use logic into a shared helper layer just because it looks reusable.
- Delay abstraction until there are real repetition points or a clear architectural boundary.

## Output requirements

When generating code, make sure the result:

- Lands in the correct package or feature.
- Preserves the dependency direction of the current architecture.
- Keeps route files thin and feature files focused.
- Keeps backend handlers thin and service-centered.
- Uses the storage provider boundary instead of OpenList-specific behavior in domain logic.
- Uses jobs and worker for slow flows when applicable.
- Avoids introducing generic project-wide helper buckets without a strong reason.

## Guardrails

- Do not add new Mibo product logic under `OpenList/` unless the task explicitly targets upstream OpenList.
- Do not generate raw API calls in frontend features when `mibo-api.ts` or `mibo-query.ts` is the correct boundary.
- Do not create a global `controller/service/repository/model` structure that conflicts with the repo's domain-based backend layout.
- Do not create `misc`, `helpers`, `common`, or `shared` dumping-ground files.
- Do not split a coherent file into many tiny wrappers with no independent value.
- Do not use file length alone as the reason to refactor.
- Do not move logic upward into global shared layers before local placement has failed.

## Quick decision heuristics

Before writing or moving code, ask:

1. Which package or feature owns this behavior?
2. Is this route-only, feature-only, shared product UI, or UI primitive?
3. Is this handler logic, domain logic, async job logic, or storage adapter logic?
4. Is this actually reused, or only hypothetically reusable?
5. Will splitting this file reduce cognitive load, or just increase file hopping?

If the answers are unclear, prefer the smallest correct change in the nearest valid location.
