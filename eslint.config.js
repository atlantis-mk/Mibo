import globals from 'globals'
import js from '@eslint/js'
import pluginQuery from '@tanstack/eslint-plugin-query'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import { defineConfig } from 'eslint/config'
import tseslint from 'typescript-eslint'

export default defineConfig(
  { ignores: ['dist', 'src/components/ui'] },
  {
    extends: [
      js.configs.recommended,
      ...tseslint.configs.recommended,
      ...pluginQuery.configs['flat/recommended'],
    ],
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      'react-hooks/set-state-in-effect': 'off',
      'react-hooks/preserve-manual-memoization': 'off',
      'react-refresh/only-export-components': [
        'warn',
        {
          allowConstantExport: true,
          allowExportNames: [
            'Route',
            'DEFAULT_MEDIA_POSTER_DISPLAY_SETTINGS',
            'createDefaultDiscoveryFilters',
            'DEFAULT_LIBRARY_BROWSER_PAGE_SIZE',
            'isLibraryBrowserPageSize',
            'DEFAULT_LIBRARY_PAGE_SIZE',
            'isLibraryPageSize',
            'formatKind',
            'formatDateTime',
            'EMPTY_LIBRARY_FORM',
            'libraryFormScanExclusionRuleInputs',
            'libraryFormMetadataStrategyInput',
            'buildScanExclusionRuleDraft',
            'normalizeScanExclusionRuleDrafts',
            'DEFAULT_OPENLIST_BASE_URL',
            'EMPTY_SOURCE_FORM',
            'buildStorageProviderOptions',
            'deriveLocalSourceName',
            'buildMediaSourceDraft',
            'EMPTY_PLUGIN_PROVIDER_DRAFT',
            'buildProviderInstanceDraft',
            'buildProviderInstanceInput',
            'buildMetadataProfileDraft',
            'buildMetadataProfileInput',
            'buildPluginProviderDraft',
            'buildStageProviderOptions',
            'formatAvailabilityLabel',
            'buildPluginConfigurationDefaults',
          ],
        },
      ],
      'no-console': 'error',
      'no-unused-vars': 'off',
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          args: 'all',
          argsIgnorePattern: '^_',
          caughtErrors: 'all',
          caughtErrorsIgnorePattern: '^_',
          destructuredArrayIgnorePattern: '^_',
          varsIgnorePattern: '^_',
          ignoreRestSiblings: true,
        },
      ],
      // Enforce type-only imports for TypeScript types
      '@typescript-eslint/consistent-type-imports': [
        'error',
        {
          prefer: 'type-imports',
          fixStyle: 'inline-type-imports',
          disallowTypeAnnotations: false,
        },
      ],
      // Prevent duplicate imports from the same module
      'no-duplicate-imports': 'error',
    },
  },
  {
    files: ['src/routes/**/*.tsx'],
    rules: {
      'react-refresh/only-export-components': 'off',
    },
  }
)
