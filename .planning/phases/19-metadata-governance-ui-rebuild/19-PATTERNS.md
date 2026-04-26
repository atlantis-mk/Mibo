# Phase 19 Pattern Map

## Purpose

Concrete analogs and contracts for the Phase 19 planner/executor. Use these
patterns directly; do not re-explore the codebase unless a task explicitly asks
for a new file.

## Target Files And Closest Analogs

| Planned file | Role | Closest analog | Pattern to reuse |
|--------------|------|----------------|------------------|
| `web/src/lib/mibo-api.ts` | typed catalog governance API boundary | current `web/src/lib/mibo-api.ts` | keep all HTTP methods in one factory returned by `createMiboApi(...)`; export snake_case backend-shaped types |
| `web/src/lib/mibo-query.ts` | query keys and queryOptions helpers | current `web/src/lib/mibo-query.ts` | use `queryOptions(...)`, stable tuple keys, and `createAuthedMiboApi(token)` |
| `web/src/features/metadata-governance/hooks/use-governance-workspace.ts` | feature-private workspace query/mutation hook | current workspace query in `web/src/features/metadata-governance/workspace.tsx` | move raw query wiring out of the page and return a small view model |
| `web/src/features/metadata-governance/hooks/use-governance-detail.ts` | feature-private detail orchestration hook | `web/src/features/metadata-governance/detail.tsx` | preserve mutation/error/dirty-state orchestration but pivot from legacy media-item draft state to catalog field-state panels |
| `web/src/features/metadata-governance/workspace.tsx` | workspace entry page | current `web/src/features/metadata-governance/workspace.tsx` | keep card-based workspace layout, badge-heavy summaries, and thin route entry |
| `web/src/features/metadata-governance/detail.tsx` | catalog governance detail shell | current `web/src/features/metadata-governance/detail.tsx` | keep async action banners, leave-confirmation guard, and dialog-based preview patterns |
| `web/src/features/metadata-governance/components/*.tsx` | focused governance panels | current `detail-panels.tsx` and `detail-sections.tsx` | extract feature-private subpanels instead of growing the top-level detail file |
| `web/src/routes/settings.metadata*.tsx` | route param parsing | current `web/src/routes/settings.metadata.index.tsx` and `web/src/routes/settings.metadata.$id.tsx` | thin `createFileRoute(...)` files only |

## Key Code Excerpts

### API client pattern

From `web/src/lib/mibo-api.ts`:

```ts
export function createMiboApi(options: ApiOptions) {
  async function request<T>(pathname: string, init?: RequestInit): Promise<T> {
    const headers = new Headers(init?.headers)
    if (options.token) {
      headers.set('Authorization', `Bearer ${options.token}`)
    }
    const response = await fetch(`${baseUrl}${pathname}`, { ...init, headers })
    ...
    return payload.data
  }
}
```

**Use for Phase 19:** keep every new governance call inside `createMiboApi(...)`.
Do not add raw `fetch` to feature components.

### Query helper pattern

From `web/src/lib/mibo-query.ts`:

```ts
export const miboQueryKeys = {
  metadataWorkspace: (token: string) => ['metadata', 'workspace', token] as const,
}

export function createAuthedMiboApi(token: string) {
  return createMiboApi({ baseUrl: getApiBaseUrl(), token })
}
```

**Use for Phase 19:** define new catalog-governance keys in the same tuple-key
style, then consume them from feature hooks and mutations.

### Detail orchestration pattern

From `web/src/features/metadata-governance/detail.tsx`:

```tsx
const itemQuery = useQuery({
  queryKey: miboQueryKeys.mediaItemDetail(token, mediaItemId),
  queryFn: () => createAuthedMiboApi(token).getMediaItem(mediaItemId),
})

const saveDraftMutation = useMutation({
  mutationFn: () => createAuthedMiboApi(token).updateMediaItemMetadata(...),
  onSuccess: async (item) => {
    queryClient.setQueryData(...)
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: workspaceQueryKey }),
      queryClient.invalidateQueries({ queryKey: miboQueryKeys.homeData(token) }),
    ])
  },
})
```

**Use for Phase 19:** preserve this query + mutation + invalidation structure,
but switch the source of truth from a legacy draft blob to catalog field states,
image candidates, and asset-link mutations.

### Thin route pattern

From `web/src/routes/settings.metadata.$id.tsx`:

```tsx
export const Route = createFileRoute('/settings/metadata/$id')({
  component: SettingsMetadataDetailRoute,
})

function SettingsMetadataDetailRoute() {
  const { id } = Route.useParams()
  return <MetadataGovernancePage mediaItemId={Number(id)} />
}
```

**Use for Phase 19:** keep route files equally thin; only rename the prop/id
semantics to catalog item ids.

### Catalog governance contract pattern

From `mibo-media-server/internal/catalog/contracts.go`:

```go
type CatalogGovernanceWorkspace struct {
    ItemID              uint                    `json:"item_id"`
    LibraryID           uint                    `json:"library_id"`
    Type                string                  `json:"type"`
    Title               string                  `json:"title"`
    AvailabilityStatus  string                  `json:"availability_status"`
    GovernanceStatus    string                  `json:"governance_status"`
    SelectedImages      []CatalogSelectedImage  `json:"selected_images,omitempty"`
    ExternalIdentities  []CatalogExternalIdentity `json:"external_identities,omitempty"`
    SourceEvidence      []CatalogSourceEvidence `json:"source_evidence"`
    FieldStates         []CatalogFieldState     `json:"field_states"`
    Assets              []CatalogAssetDetail    `json:"assets"`
    RecommendedChildren []CatalogListItem       `json:"recommended_children"`
}
```

**Use for Phase 19:** drive the UI directly from this catalog vocabulary. Do not
translate it back into legacy `MediaItemDetail` shapes.

## Rules To Preserve

1. **Frontend/server calls stay inside `mibo-api.ts` and React Query helpers.**
2. **Route files stay thin; feature state lives under `features/metadata-governance`.**
3. **Use catalog item ids, not legacy media item ids, throughout the governance flow.**
4. **Render provider evidence as text/JSON only — never with `dangerouslySetInnerHTML`.**
5. **Keep image selection non-destructive; select one candidate, preserve the rest.**
6. **Do not manually edit generated router output files such as `routeTree.gen.ts`.**
