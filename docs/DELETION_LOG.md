# Code Deletion Log

## [2026-05-14] Refactor Session

### Unused Dependencies Removed
- @base-ui/react - only referenced by deleted unused combobox primitive
- @hookform/resolvers - no runtime references found
- @tanstack/react-virtual - no runtime references found
- cmdk - only referenced by deleted unused command primitive
- date-fns - no runtime references found
- embla-carousel-react - only referenced by deleted unused carousel primitive
- input-otp - only referenced by deleted unused OTP primitive
- react-day-picker - only referenced by deleted unused calendar primitive
- react-hook-form - only referenced by deleted unused form primitive
- react-resizable-panels - only referenced by deleted unused resizable primitive
- recharts - only referenced by deleted unused chart primitive
- zod - no runtime references found

### Unused Files Deleted
- frontend/src/components/ui/accordion.tsx
- frontend/src/components/ui/aspect-ratio.tsx
- frontend/src/components/ui/breadcrumb.tsx
- frontend/src/components/ui/button-group.tsx
- frontend/src/components/ui/calendar.tsx
- frontend/src/components/ui/carousel.tsx
- frontend/src/components/ui/chart.tsx
- frontend/src/components/ui/collapsible.tsx
- frontend/src/components/ui/combobox.tsx
- frontend/src/components/ui/command.tsx
- frontend/src/components/ui/context-menu.tsx
- frontend/src/components/ui/direction.tsx
- frontend/src/components/ui/form.tsx
- frontend/src/components/ui/hover-card.tsx
- frontend/src/components/ui/input-group.tsx
- frontend/src/components/ui/input-otp.tsx
- frontend/src/components/ui/item.tsx
- frontend/src/components/ui/kbd.tsx
- frontend/src/components/ui/menubar.tsx
- frontend/src/components/ui/navigation-menu.tsx
- frontend/src/components/ui/popover.tsx
- frontend/src/components/ui/progress.tsx
- frontend/src/components/ui/radio-group.tsx
- frontend/src/components/ui/resizable.tsx
- frontend/src/components/ui/toggle-group.tsx
- frontend/src/components/ui/toggle.tsx
- frontend/src/components/version-switcher.tsx
- frontend/src/features/settings/components/settings-menu-trigger.tsx

### Duplicate Code Consolidated
- None in this pass; limited changes to verified dead code only

### Unused Exports Removed
- frontend/src/components/media-poster-card.tsx - MediaRail
- frontend/src/lib/library-presentation.ts - formatLibraryType
- frontend/src/lib/media-presentation.ts - getMediaCardAvailabilityStatus(), isMediaCardPlayable()
- frontend/src/lib/mibo-query.ts - libraryMetadataStrategyQueryOptions(), scheduleDetailQueryOptions(), workflowDiagnosticsQueryOptions()
- frontend/src/features/settings/components/settings-aside-card.tsx - SettingsAsideCard
- frontend/src/features/media/components/standalone-media-detail-utils.ts - formatAudioTrackLabel(), formatDate(), describeMatchStatus()
- frontend/src/features/home/home-sections.tsx - StatCard(), formatMediaType()
- frontend/src/features/library/index.tsx - LIBRARY_PAGE_SIZE_OPTIONS
- frontend/src/features/settings/components/library-form.tsx - deriveLibraryNameFromPath()
- frontend/src/lib/mibo-api.ts - TOKEN_STORAGE_KEY

### Additional Internalized Exports
- frontend/src/features/media/components/standalone-media-detail-utils.ts - fileNameFromStoragePath()
- frontend/src/features/schedules/components/schedule-list.tsx - formatScope(), formatFrequency(), formatLatestResult()
- frontend/src/features/settings/components/library-scan-exclusion-rules-editor.tsx - EMPTY_SCAN_EXCLUSION_RULE_DRAFT
- frontend/src/features/play/components/AppSidebar.tsx - PlaybackSidebarItem
- frontend/src/features/settings/sections.ts - SettingsSectionPath

### Additional Dead Code Removed
- frontend/src/features/media/components/standalone-media-detail-utils.ts - formatChannels()

### Impact
- Files deleted: 28
- Dependencies removed: 12
- Lines of code removed: not precisely measured
- Bundle size reduction: not measured

### Testing
- `pnpm typecheck` ✅
- `pnpm test` ✅ (8 files, 20 tests)
- `pnpm build` ✅
- `pnpm lint` ⚠️ still reports many pre-existing React Hooks / React Refresh issues noted in repo guidance

### Notes
- `src/types/style-imports.d.ts` is still flagged by knip, but was kept because it may still be needed for ambient Swiper CSS module declarations.
