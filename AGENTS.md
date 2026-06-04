# Repository Guidelines

## Project Structure & Module Organization
This repository is organized around the open frontend at the root and a backend service directory:

- Root app: Vite + React 19 + TypeScript UI. Main code lives in `src`, organized by feature under `src/features`, routes under `src/routes`, shared UI in `src/components`, and helpers in `src/lib`, `src/hooks`, and `src/stores`.
- `mibo-server/`: Go service submodule pointing at `https://github.com/atlantis-mk/mibo-server.git`. Entrypoints live in `cmd/`, core packages in `internal/`, persistent data in `data/`, and built binaries in `bin/`.

Supporting material lives in `docs/`, `openspec/`, and `scripts/`. Treat `dist*` and `mibo-server/internal/webui/dist` as build output.

## Build, Test, and Development Commands
- `pnpm install`: install frontend dependencies.
- `pnpm dev`: start the UI locally with Vite.
- `pnpm build`: type-check and build the frontend.
- `pnpm lint`: run ESLint.
- `pnpm test`: run Vitest in headless browser mode.
- `git submodule update --init --recursive`: fetch the backend service submodule.
- `cd mibo-server && go run ./cmd/mibo-media-server`: start the API server.
- `cd mibo-server && go test ./...`: run backend tests.
- `./scripts/build-with-frontend.sh`: build static frontend assets and embed them into the Go server binary.

## Coding Style & Naming Conventions
Frontend formatting is enforced by Prettier (`2` spaces, `80` columns, single quotes, no semicolons) and ESLint. Use path aliases like `@/features/...` where appropriate. Name React components in `PascalCase`, hooks as `use-*`/`use*`, and keep tests adjacent to the code they cover.

Backend code should stay `gofmt`-clean and follow standard Go layout: exported identifiers in `CamelCase`, internal helpers in `camelCase`, and tests in `*_test.go`.

## Pipeline Data Ownership
Derived fields must have exactly one owner. Once a derived field enters a pipeline artifact, downstream stages must consume that value and must not re-derive the same field from raw input.

## Testing Guidelines
Frontend tests use Vitest with Playwright browser runners; common patterns include `*.test.ts` and `*.test.tsx`. Backend tests are extensive and live beside packages in `internal/**`. Prefer focused test runs while iterating, then finish with `pnpm test` and `go test ./...` before opening a PR.

## Commit & Pull Request Guidelines
Recent history follows Conventional Commit style such as `feat:`, `chore:`, and `test:` with optional scopes. Keep commit subjects imperative and concise, for example `feat: expand media catalog playback`.

PRs should include a short summary, affected areas (`frontend`, `mibo-server`, or both), test evidence, and screenshots or recordings for UI changes. Call out config or migration impacts early, especially changes involving media scanning, metadata, or embedded web assets.

# AI Coding Constitution

## Supreme Principles

1. **Never sacrifice the long-term structure of the system for short-term task completion.**
2. **Never bypass the existing architecture, layering, naming conventions, error handling, logging, testing, or data conventions.**
3. **Every code change must make the system clearer, or at minimum not make it more chaotic.**
4. **Prefer local fixes over broad rewrites. Prefer compatibility over breaking changes.**
5. **When uncertain, first read the existing code patterns, then follow the project's established style.**

---

## 1. Boundary Rules

1. Modules must have a single responsibility. Do not create god Services, god Managers, or god Utils.
2. Do not violate layer boundaries. Controllers must not contain business logic. Repositories must not contain business logic. Domain code must not depend on frameworks or infrastructure.
3. Core business logic must live in a clearly owned location. It must not be scattered across Controllers, Jobs, Consumers, SQL, Hooks, or Callbacks.
4. Third-party SDKs, databases, caches, message queues, and external APIs must be isolated behind adapter layers. Core business logic must not depend on them directly.
5. Do not introduce implicit global dependencies. Dependencies should be passed explicitly or managed through the project's existing dependency injection mechanism.

---

## 2. Code Rules

1. Names must express business meaning. Avoid vague names such as `data`, `info`, `handle`, `process`, `doSomething`, `common`, `helper`, and `temp`.
2. A function should do one thing: either orchestrate a flow or handle details. Do not mix multiple levels of abstraction in one function.
3. Do not copy and paste core business rules. Repeated business rules must be consolidated into one authoritative location.
4. Do not abstract prematurely. Abstractions are allowed only when the concept is stable, the repetition is clear, and the boundary is well understood.
5. New code must follow the current project style, including directory structure, naming, exceptions, logging, tests, and dependency patterns.
6. Before deleting code, confirm that there are no callers, configurations, migrations, jobs, scripts, or external systems depending on it.

---

## 3. Business Rules

1. Core rules involving pricing, permissions, state transitions, inventory, payments, refunds, coupons, risk control, and quotas must be explicitly modeled.
2. State changes must happen through methods with business meaning. Do not arbitrarily set `status`.
3. All write operations must consider idempotency, especially payment callbacks, webhooks, message consumption, scheduled jobs, order creation, refunds, coupon issuance, and inventory deduction.
4. Do not represent business states with magic numbers or magic strings. Use enums, constants, types, or explicit models.
5. Do not hide core business fields inside JSON blobs, comments, extension fields, or unconstrained maps.

---

## 4. Data Rules

1. Database changes must be forward-compatible. Avoid directly deleting fields, changing field meanings, or changing the semantics of historical data.
2. Schemas, indexes, unique constraints, and transaction boundaries must match the business consistency requirements.
3. Data changes involving money, inventory, permissions, or state must consider concurrency, duplicate submissions, failed retries, and rollback.
4. Migration scripts must be repeatable, traceable, and rollbackable, or at least compensatable.
5. Do not introduce new tables, fields, cache keys, or message formats without a clear reason.

---

## 5. Error and Logging Rules

1. Errors must be clearly classified: validation errors, business errors, permission errors, external dependency errors, and system errors.
2. Do not return `null`, `false`, `-1`, or an empty string to hide errors.
3. Logs must include debugging context: `requestId`/`traceId`, `userId`, `resourceId`, `operation`, `reason`, `duration`, and `external dependency`.
4. Do not swallow exceptions. If an exception is caught, it must be handled, transformed, logged, or rethrown.
5. External dependency calls must consider timeouts, retries, degradation, rate limiting, and error wrapping.

---

## 6. Testing Rules

1. Logic involving money, permissions, state, inventory, payments, refunds, or data consistency must have tests.
2. Tests must cover the happy path, boundary conditions, failure paths, and repeated execution scenarios.
3. When fixing a bug, add a test that reproduces the bug.
4. Do not weaken assertions, delete tests, or mock away the logic that actually needs to be verified just to make tests pass.
5. Tests should verify business behavior, not merely implementation details.

---

## 7. Change Rules

1. Each change should be as small as possible. Do not mix unrelated refactors, formatting changes, or renames into the same change.
2. Do not rewrite large sections of stable code unless explicitly required.
3. Before introducing a new dependency, framework, or pattern, explain why it is necessary and what alternatives were considered.
4. When modifying public APIs, data structures, message formats, or configuration items, consider compatibility and migration paths.
5. Every temporary solution must leave a clear marker explaining the reason, impact, and follow-up plan.

---

## 8. AI Execution Rules

1. Before writing code, understand the existing structure. Do not invent a new architecture out of thin air.
2. Prefer the smallest necessary change. Do not expand the scope of the problem.
3. After generating code, self-check whether boundaries are correct, names are clear, errors are handled, logs are sufficient, and tests cover the risks.
4. Explicitly state assumptions where uncertainty exists. Do not present guesses as facts.
5. Do not generate code that appears complete but cannot run, integrate, or be tested.
6. Do not sacrifice security, data consistency, or maintainability to satisfy a surface-level requirement.
7. Every implementation must answer: Why does this belong here? What happens if it fails? What happens if it runs twice? How will it change in the future?

---

## Final Checklist

Before submitting any code, confirm that:

- Responsibility boundaries have not been violated.
- Core business rules are not duplicated or scattered.
- State changes are explicit and controlled.
- Write operations are idempotent, or there is a clear explanation for why idempotency is not required.
- Error handling and logging follow project conventions.
- Data changes are compatible and traceable.
- Core risk paths are covered by tests.
- No meaningless abstractions, god classes, implicit dependencies, or magic values have been introduced.
- The change scope is small enough and contains no unrelated modifications.
- The code follows the project's existing style.

## One-Sentence Constitution

AI-generated code must prioritize system boundaries, semantics, consistency, and evolvability. Any code that merely makes the current requirement appear complete while making future maintenance harder is unacceptable.
