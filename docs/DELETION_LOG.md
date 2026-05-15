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
- web/src/components/ui/accordion.tsx
- web/src/components/ui/aspect-ratio.tsx
- web/src/components/ui/breadcrumb.tsx
- web/src/components/ui/button-group.tsx
- web/src/components/ui/calendar.tsx
- web/src/components/ui/carousel.tsx
- web/src/components/ui/chart.tsx
- web/src/components/ui/collapsible.tsx
- web/src/components/ui/combobox.tsx
- web/src/components/ui/command.tsx
- web/src/components/ui/context-menu.tsx
- web/src/components/ui/direction.tsx
- web/src/components/ui/form.tsx
- web/src/components/ui/hover-card.tsx
- web/src/components/ui/input-group.tsx
- web/src/components/ui/input-otp.tsx
- web/src/components/ui/item.tsx
- web/src/components/ui/kbd.tsx
- web/src/components/ui/menubar.tsx
- web/src/components/ui/navigation-menu.tsx
- web/src/components/ui/popover.tsx
- web/src/components/ui/progress.tsx
- web/src/components/ui/radio-group.tsx
- web/src/components/ui/resizable.tsx
- web/src/components/ui/toggle-group.tsx
- web/src/components/ui/toggle.tsx
- web/src/components/version-switcher.tsx
- web/src/features/settings/components/settings-menu-trigger.tsx

### Duplicate Code Consolidated
- None in this pass; limited changes to verified dead code only

### Unused Exports Removed
- web/src/components/media-poster-card.tsx - MediaRail
- web/src/lib/library-presentation.ts - formatLibraryType
- web/src/lib/media-presentation.ts - getMediaCardAvailabilityStatus(), isMediaCardPlayable()
- web/src/lib/mibo-query.ts - libraryMetadataStrategyQueryOptions(), scheduleDetailQueryOptions(), workflowDiagnosticsQueryOptions()
- web/src/features/settings/components/settings-aside-card.tsx - SettingsAsideCard
- web/src/features/media/components/standalone-media-detail-utils.ts - formatAudioTrackLabel(), formatDate(), describeMatchStatus()
- web/src/features/home/home-sections.tsx - StatCard(), formatMediaType()
- web/src/features/library/index.tsx - LIBRARY_PAGE_SIZE_OPTIONS
- web/src/features/settings/components/library-form.tsx - deriveLibraryNameFromPath()
- web/src/lib/mibo-api.ts - TOKEN_STORAGE_KEY

### Additional Internalized Exports
- web/src/features/media/components/standalone-media-detail-utils.ts - fileNameFromStoragePath()
- web/src/features/schedules/components/schedule-list.tsx - formatScope(), formatFrequency(), formatLatestResult()
- web/src/features/settings/components/library-scan-exclusion-rules-editor.tsx - EMPTY_SCAN_EXCLUSION_RULE_DRAFT
- web/src/features/play/components/AppSidebar.tsx - PlaybackSidebarItem
- web/src/features/settings/sections.ts - SettingsSectionPath

### Additional Dead Code Removed
- web/src/features/media/components/standalone-media-detail-utils.ts - formatChannels()

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
