## Why

Mibo can currently scan content successfully while still showing an empty home page when a media source or library later enters an error state. Users need a clear, actionable explanation that content still exists, which dependency failed, what is affected, and how to recover.

## What Changes

- Introduce a structured health diagnostics model for media sources, libraries, jobs, and external dependencies.
- Add a Health Center UI that groups active issues, explains impact in user-friendly language, exposes technical details, and presents recovery actions.
- Surface blocking health issues globally through navigation badges, library badges, and home-page empty-state explanations.
- Classify common backend failure strings into stable diagnostic reason codes, including OpenList/PikPak authentication or captcha expiration.
- Preserve detailed job errors for troubleshooting while translating known failures into operator-facing remediation guidance.
- Add recovery flows for validating affected media sources and re-running scans for affected libraries.

## Capabilities

### New Capabilities
- `health-center-diagnostics`: Structured health events, diagnostics APIs, user-facing issue summaries, and guided recovery actions for media sources, libraries, jobs, and external dependencies.

### Modified Capabilities
- `homepage-media-library-dashboard`: Home page empty and degraded states must explain when scanned content is unavailable because affected libraries or media sources have active blocking health issues.

## Impact

- Backend: add health diagnosis classification, persistence or derived health views, diagnostics endpoints, and source validation / affected-library recovery actions.
- Frontend: add Health Center screens, global issue indicators, side-bar/library status badges, and richer home-page empty/degraded states.
- APIs: extend library/media source responses or add health-specific endpoints with stable reason codes, severity, impact, and recovery actions.
- Jobs and scanning: record failed job context in a way diagnostics can associate with media sources, libraries, and affected content.
- Tests: add backend classification/API tests and frontend state rendering tests for empty, degraded, and recoverable-health scenarios.
