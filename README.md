# Mibo

Mibo is a React + Vite frontend for managing and browsing a media library.

## Project Layout

- `src/`: frontend source code, with features under `src/features`, routes under `src/routes`, shared UI under `src/components`, and helpers under `src/lib`, `src/hooks`, and `src/stores`.
- `public/`: static frontend assets.
- `mibo-server/`: backend service Git submodule, pointing at `https://github.com/atlantis-mk/mibo-server.git`.
- `scripts/`: helper scripts for embedding the built frontend into the backend service.
- `docs/` and `openspec/`: project documentation and change specifications.

## Frontend Development

```bash
pnpm install
pnpm dev
```

Useful commands:

```bash
pnpm build
pnpm lint
pnpm test
```

## Full-Stack Internal Build

When `mibo-server/` is available, build the frontend and embed it into the backend binary:

```bash
git submodule update --init --recursive
./scripts/build-with-frontend.sh
```

The script writes frontend assets to `mibo-server/internal/webui/dist` and builds the backend binary under `build/`.
