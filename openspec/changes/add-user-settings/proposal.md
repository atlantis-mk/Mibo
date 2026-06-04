## Why

Mibo currently lacks a dedicated user settings capability for per-user preferences, which leaves frontend clients without a stable backend contract for reading and updating account-level settings. We need a server-side user settings model now so frontend work can integrate against a supported API instead of local-only or ad hoc state.

## What Changes

- Add a new user settings capability in `mibo-media-server` for storing per-user preferences.
- Expose authenticated APIs for reading and updating the current user's settings.
- Define a persistence model and service layer that separates user-scoped settings from server-wide admin settings.
- Establish an initial settings surface tailored for frontend integration, without requiring changes in `frontend/` as part of this change.
- Add validation, defaults, and backward-safe read behavior so existing users receive sensible settings before explicit customization.

## Capabilities

### New Capabilities
- `user-settings`: Per-user settings storage and authenticated read/update APIs for frontend clients.

### Modified Capabilities
- None.

## Impact

- Affected backend code in `mibo-media-server/internal/settings`, `mibo-media-server/internal/httpapi`, and database migration/model layers.
- New authenticated API surface for frontend clients to fetch and persist user settings.
- New tests covering default resolution, authorization, validation, and persistence behavior.
- No `frontend/` implementation is included in this change; the goal is to make backend/frontend integration possible.
