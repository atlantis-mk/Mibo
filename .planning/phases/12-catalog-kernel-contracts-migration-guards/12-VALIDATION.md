---
phase: 12
slug: catalog-kernel-contracts-migration-guards
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-25
---

# Phase 12 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — Go package tests under `mibo-media-server/internal/...` |
| **Quick run command** | `go test ./internal/catalog ./internal/settings ./internal/database ./internal/worker -run 'Test(Catalog|DatabaseOpen|RunOnce.*Catalog|CatalogMigration)'` |
| **Full suite command** | `go test ./internal/database ./internal/catalog ./internal/inventory ./internal/settings ./internal/httpapi ./internal/worker` |
| **Estimated runtime** | ~25 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/catalog ./internal/settings ./internal/database ./internal/worker -run 'Test(Catalog|DatabaseOpen|RunOnce.*Catalog|CatalogMigration)'`
- **After every plan wave:** Run `go test ./internal/database ./internal/catalog ./internal/inventory ./internal/settings ./internal/httpapi ./internal/worker`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 12-01-01 | 01 | 1 | KERN-01 | T-12-01 | DTO layer omits raw GORM rows and stable JSON keys are explicit | unit | `go test ./internal/catalog -run 'TestCatalog.*Contract'` | ✅ | ⬜ pending |
| 12-01-02 | 01 | 1 | KERN-01 | T-12-01 | DTO mappers/builders serialize only contract-approved fields | unit | `go test ./internal/catalog -run 'TestCatalog.*Contract'` | ✅ | ⬜ pending |
| 12-02-01 | 02 | 1 | KERN-02 | T-12-02 | Migration settings parse and persist only typed allowed keys | unit | `go test ./internal/settings -run 'TestCatalogMigration'` | ✅ | ⬜ pending |
| 12-02-02 | 02 | 1 | KERN-02 | T-12-03 | Migration state endpoints require auth and validate input | integration | `go test ./internal/httpapi -run 'TestCatalogMigration'` | ✅ | ⬜ pending |
| 12-03-01 | 03 | 1 | PROD-01 | T-12-04 | Projection job payloads stay scoped and worker dispatch is explicit | unit | `go test ./internal/worker -run 'TestRunOnce.*Catalog'` | ✅ | ⬜ pending |
| 12-03-02 | 03 | 1 | PROD-01 | T-12-04 | Library/worker queue entrypoints exist for item and library projection refresh | integration | `go test ./internal/worker -run 'TestRunOnce.*Catalog'` | ✅ | ⬜ pending |
| 12-04-01 | 04 | 2 | PROD-01 | T-12-05 | Additive indexes preserve startup safety for empty and legacy DBs | unit | `go test ./internal/database -run 'Test(CatalogKernelTablesAreMigrated|DatabaseOpen.*Catalog)'` | ✅ | ⬜ pending |
| 12-04-02 | 04 | 2 | PROD-01 | T-12-05 | HTTP readiness and startup still work after migration guard changes | integration | `go test ./internal/httpapi -run TestReadyz` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
