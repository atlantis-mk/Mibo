package catalog

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestCatalogDTOContractExportsRequiredTypes(t *testing.T) {
	exported := []any{
		CatalogListItem{},
		CatalogItemDetail{},
		CatalogSeasonDetail{},
		CatalogEpisodeDetail{},
		CatalogEpisodeParentContext{},
		CatalogEpisodeShelfItem{},
		CatalogAssetDetail{},
		CatalogAssetFileSummary{},
		CatalogMediaStreamSummary{},
		CatalogGovernanceWorkspace{},
		CatalogMetadataOperationResult{},
	}

	if len(exported) != 11 {
		t.Fatalf("expected 11 exported catalog DTO types, got %d", len(exported))
	}
}

func TestCatalogFileContractUsesJSONTagsOnly(t *testing.T) {
	content, err := os.ReadFile("contracts.go")
	if err != nil {
		t.Fatalf("read contracts.go: %v", err)
	}

	text := string(content)
	required := []string{
		"json:\"availability_status\"",
		"json:\"governance_status\"",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected contracts.go to contain %q", fragment)
		}
	}
	if strings.Contains(text, "gorm:\"") {
		t.Fatalf("expected contracts.go to exclude gorm tags")
	}
	if !strings.Contains(text, "series") {
		t.Fatalf("expected contracts.go to include canonical series type")
	}
	if strings.Contains(text, "show") {
		t.Fatalf("expected contracts.go to avoid legacy show naming")
	}

	assertDTOsDoNotExposeRawRows(t, "contracts.go")
}

func TestCatalogTypeContractKeepsSeriesCanonical(t *testing.T) {
	payload, err := json.Marshal(CatalogListItem{Type: ItemTypeSeries})
	if err != nil {
		t.Fatalf("marshal list item: %v", err)
	}

	text := string(payload)
	if !strings.Contains(text, `"type":"series"`) {
		t.Fatalf("expected marshaled dto to use series type, got %s", text)
	}
	if strings.Contains(text, `"type":"show"`) {
		t.Fatalf("expected marshaled dto to avoid legacy show type, got %s", text)
	}
}

func TestCatalogJSONContractShapeAndMapperBehavior(t *testing.T) {
	now := time.Date(2026, time.April, 25, 12, 0, 0, 0, time.UTC)
	year := 2024
	seasonNumber := 1
	episodeNumber := 2
	absoluteNumber := 2
	runtimeSeconds := 3600
	seasonRuntime := 3200
	durationSeconds := 3660.5
	width := 1280
	height := 720
	confidence := 0.92
	segmentEnd := 3660.5
	sourceID := uint(44)
	sourceID2 := uint(45)
	sourceID3 := uint(46)
	sourceID4 := uint(47)
	sourceID5 := uint(48)
	editedByUserID := uint(7)

	images := []database.ItemImage{
		{ImageType: "poster", URL: "https://example.com/poster.jpg", Width: &width, Height: &height, Language: "en", IsSelected: true},
		{ImageType: "backdrop", URL: "https://example.com/backdrop.jpg", IsSelected: false},
	}
	externalIDs := []database.CatalogExternalID{{Provider: "tmdb", ProviderType: "tv", ExternalID: "999", IsPrimary: true, Source: "provider", Confidence: &confidence}}
	sources := []database.MetadataSource{
		{
			SourceType: SourceTypeProvider,
			SourceName: "tmdb",
			Language:   "en",
			ExternalID: "999",
			PayloadJSON: `{
				"title":"The Example Show",
				"name":"Example Name",
				"original_title":"Original Example",
				"overview":"Provider overview",
				"release_date":"2024-04-20",
				"first_air_date":"2024-04-21",
				"last_air_date":"2024-04-22",
				"runtime":3600,
				"status":"returning",
				"season_number":1,
				"episode_number":2,
				"provider_blob":{"raw":"secret"},
				"genres":["drama"],
				"popularity":99.9
			}`,
			Confidence: &confidence,
			FetchedAt:  now,
		},
		{
			SourceType: SourceTypeProvider,
			SourceName: "tvdb",
			Language:   "en",
			ExternalID: "888",
			PayloadJSON: `{
				"provider_blob":{"nested":true},
				"genres":["mystery"]
			}`,
			FetchedAt: now,
		},
	}
	fieldStates := []database.MetadataFieldState{
		{FieldKey: "title", SourceID: &sourceID, ValueJSON: `"The Example Show"`, IsLocked: true, LockReason: "operator lock", EditedByUserID: &editedByUserID, EditedAt: &now},
		{FieldKey: "runtime", SourceID: &sourceID2, ValueJSON: `3600`},
		{FieldKey: "is_featured", SourceID: &sourceID3, ValueJSON: `true`},
		{FieldKey: "provider_blob", SourceID: &sourceID4, ValueJSON: `{"raw":"secret"}`},
		{FieldKey: "genres", SourceID: &sourceID5, ValueJSON: `["drama"]`},
	}
	rollup := &database.ItemRollup{ChildCount: 8, AvailableCount: 6, MissingCount: 1, UnairedCount: 1, PlayedCount: 2, InProgressCount: 1, LatestAirDate: &now, LatestAddedAt: &now}

	assetDetail := BuildCatalogAssetDetail(CatalogAssetDetailInput{
		Asset:   database.MediaAsset{ID: 21, LibraryID: 3, AssetType: "main", DisplayName: "4K HDR", Edition: "Director's Cut", QualityLabel: "2160p", DurationSeconds: &durationSeconds, Status: AvailabilityAvailable, ProbeStatus: "ready"},
		Links:   []database.AssetItem{{ItemID: 13, Role: "primary", SegmentIndex: 0, EndSeconds: &segmentEnd, Confidence: &confidence, Source: "scanner"}},
		Files:   []CatalogAssetFileSummary{{FileID: 31, Role: "source", PartIndex: 0, StorageProvider: "local", StoragePath: "/media/example.mkv", SizeBytes: 1234, Container: "mkv", Status: AvailabilityAvailable}},
		Streams: []CatalogMediaStreamSummary{{FileID: 31, StreamIndex: 0, StreamType: "video", Codec: "h264", Width: &width, Height: &height}},
	})
	cast := []CatalogPersonDetail{{Name: "Actor A", Role: "Lead"}}
	directors := []CatalogPersonDetail{{Name: "Director A", Role: "Director"}}
	birthday := now

	seasonDetail := BuildCatalogSeasonDetail(CatalogSeasonDetailInput{
		Item:        database.CatalogItem{ID: 11, LibraryID: 3, Type: ItemTypeSeason, Title: "Season 1", Overview: "Season overview", IndexNumber: &seasonNumber, RuntimeSeconds: &seasonRuntime, AvailabilityStatus: AvailabilityAvailable, GovernanceStatus: GovernanceMatched},
		Rollup:      rollup,
		Images:      images,
		ExternalIDs: externalIDs,
		Sources:     sources,
		FieldStates: fieldStates,
	})

	episodeDetail := BuildCatalogEpisodeDetail(CatalogEpisodeDetailInput{
		Item: database.CatalogItem{
			ID:                 12,
			LibraryID:          3,
			Type:               ItemTypeEpisode,
			Title:              "Episode 2",
			Overview:           "Episode overview",
			Year:               &year,
			ParentIndexNumber:  &seasonNumber,
			IndexNumber:        &episodeNumber,
			AbsoluteNumber:     &absoluteNumber,
			RuntimeSeconds:     &runtimeSeconds,
			AvailabilityStatus: AvailabilityUnaired,
			GovernanceStatus:   GovernanceNeedsReview,
			FirstAirDate:       &now,
		},
		Images:      images,
		ExternalIDs: externalIDs,
		Sources:     sources,
		FieldStates: fieldStates,
		Assets:      []CatalogAssetDetail{assetDetail},
	})

	legacySeriesType := "show"
	cases := []struct {
		name      string
		value     any
		required  []string
		forbidden []string
		typeValue string
		statusKey string
		status    string
	}{
		{
			name: "list item",
			value: BuildCatalogListItem(CatalogListItemInput{
				Item:        database.CatalogItem{ID: 10, LibraryID: 3, Type: legacySeriesType, Title: "The Example Show", Year: &year, AvailabilityStatus: AvailabilityMissing, GovernanceStatus: GovernancePending},
				Rollup:      rollup,
				Images:      images,
				ExternalIDs: externalIDs,
			}),
			required:  []string{"id", "library_id", "type", "availability_status", "governance_status", "child_summary", "selected_images", "external_identities"},
			forbidden: []string{"deleted_at", "parent_id", "root_id", "payload_json", "value_json"},
			typeValue: ItemTypeSeries,
			statusKey: "availability_status",
			status:    AvailabilityMissing,
		},
		{
			name: "item detail",
			value: BuildCatalogItemDetail(CatalogItemDetailInput{
				Item:        database.CatalogItem{ID: 10, LibraryID: 3, Type: legacySeriesType, Title: "The Example Show", Year: &year, AvailabilityStatus: AvailabilityMissing, GovernanceStatus: GovernanceMatched},
				Rollup:      rollup,
				Images:      images,
				ExternalIDs: externalIDs,
				Sources:     sources,
				FieldStates: fieldStates,
				Cast:        cast,
				Directors:   directors,
				Tags:        []CatalogTagDetail{{Kind: "genre", Name: "Drama"}},
				Seasons:     []CatalogSeasonDetail{seasonDetail},
				Episodes:    []CatalogEpisodeDetail{episodeDetail},
				Assets:      []CatalogAssetDetail{assetDetail},
				Related:     []CatalogListItem{BuildCatalogListItem(CatalogListItemInput{Item: database.CatalogItem{ID: 13, LibraryID: 3, Type: ItemTypeSeries, Title: "Related Show", AvailabilityStatus: AvailabilityAvailable, GovernanceStatus: GovernanceMatched}})},
			}),
			required:  []string{"id", "library_id", "type", "availability_status", "governance_status", "tags", "genres", "related_items", "assets", "source_evidence", "field_states", "cast", "directors", "seasons", "episodes", "same_season_episodes"},
			forbidden: []string{"deleted_at", "parent_id", "root_id", "payload_json", "value_json"},
			typeValue: ItemTypeSeries,
			statusKey: "availability_status",
			status:    AvailabilityMissing,
		},
		{
			name:      "season detail",
			value:     seasonDetail,
			required:  []string{"id", "library_id", "type", "availability_status", "governance_status", "episodes", "source_evidence", "field_states"},
			forbidden: []string{"deleted_at", "parent_id", "root_id", "payload_json", "value_json"},
			typeValue: ItemTypeSeason,
			statusKey: "availability_status",
			status:    AvailabilityAvailable,
		},
		{
			name:      "episode detail",
			value:     episodeDetail,
			required:  []string{"id", "library_id", "type", "availability_status", "governance_status", "assets", "source_evidence", "field_states"},
			forbidden: []string{"deleted_at", "parent_id", "root_id", "payload_json", "value_json"},
			typeValue: ItemTypeEpisode,
			statusKey: "availability_status",
			status:    AvailabilityUnaired,
		},
		{
			name:      "asset detail",
			value:     assetDetail,
			required:  []string{"id", "library_id", "asset_type", "status", "probe_status", "links", "files", "streams"},
			forbidden: []string{"deleted_at", "payload_json", "value_json"},
			statusKey: "status",
			status:    AvailabilityAvailable,
		},
		{
			name: "person detail",
			value: CatalogPersonPageDetail{
				ID:                 91,
				Name:               "Actor A",
				SortName:           "Actor A",
				AvatarURL:          "https://example.com/actor-a.jpg",
				Biography:          "Lead performer.",
				Birthday:           &birthday,
				PlaceOfBirth:       "Seoul",
				KnownForDepartment: "Acting",
				ExternalIdentities: []CatalogExternalIdentity{{Provider: "tmdb", ProviderType: "person", ExternalID: "321", IsPrimary: true}},
				RelatedItems:       []CatalogListItem{BuildCatalogListItem(CatalogListItemInput{Item: database.CatalogItem{ID: 13, LibraryID: 3, Type: ItemTypeSeries, Title: "Related Show", AvailabilityStatus: AvailabilityAvailable, GovernanceStatus: GovernanceMatched}})},
			},
			required:  []string{"id", "name", "avatar_url", "biography", "birthday", "place_of_birth", "known_for_department", "external_identities", "related_items"},
			forbidden: []string{"deleted_at", "payload_json", "value_json"},
		},
		{
			name: "governance workspace",
			value: BuildCatalogGovernanceWorkspace(CatalogGovernanceWorkspaceInput{
				Item:                database.CatalogItem{ID: 10, LibraryID: 3, Type: legacySeriesType, Title: "The Example Show", AvailabilityStatus: AvailabilityMissing, GovernanceStatus: GovernanceManual},
				Images:              images,
				ExternalIDs:         externalIDs,
				Sources:             sources,
				FieldStates:         fieldStates,
				Assets:              []CatalogAssetDetail{assetDetail},
				RecommendedChildren: []CatalogListItem{BuildCatalogListItem(CatalogListItemInput{Item: database.CatalogItem{ID: 12, LibraryID: 3, Type: ItemTypeEpisode, Title: "Episode 2", AvailabilityStatus: AvailabilityUnaired, GovernanceStatus: GovernanceNeedsReview}})},
			}),
			required:  []string{"item_id", "library_id", "type", "availability_status", "governance_status", "assets", "source_evidence", "field_states", "recommended_children"},
			forbidden: []string{"deleted_at", "parent_id", "root_id", "payload_json", "value_json"},
			typeValue: ItemTypeSeries,
			statusKey: "availability_status",
			status:    AvailabilityMissing,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := marshalContractJSON(t, tc.value)
			decoded := unmarshalContractJSON(t, payload)

			for _, key := range tc.required {
				if _, ok := decoded[key]; !ok {
					t.Fatalf("expected %s payload to include %q, got %s", tc.name, key, payload)
				}
			}
			for _, key := range tc.forbidden {
				if strings.Contains(payload, `"`+key+`"`) {
					t.Fatalf("expected %s payload to exclude %q, got %s", tc.name, key, payload)
				}
			}
			if tc.typeValue != "" && decoded["type"] != tc.typeValue {
				t.Fatalf("expected %s type %q, got %#v", tc.name, tc.typeValue, decoded["type"])
			}
			if decoded[tc.statusKey] != tc.status {
				t.Fatalf("expected %s %s %q, got %#v", tc.name, tc.statusKey, tc.status, decoded[tc.statusKey])
			}
			assertCuratedSourceEvidenceSummary(t, decoded)
			assertCuratedFieldStateValues(t, decoded)
		})
	}
}

func assertCuratedSourceEvidenceSummary(t *testing.T, decoded map[string]any) {
	t.Helper()

	rawEvidence, ok := decoded["source_evidence"]
	if !ok {
		return
	}

	evidence, ok := rawEvidence.([]any)
	if !ok {
		t.Fatalf("expected source_evidence array, got %#v", rawEvidence)
	}
	if len(evidence) != 2 {
		t.Fatalf("expected 2 source evidence entries, got %d", len(evidence))
	}

	first := sourceEvidenceEntry(t, evidence[0])
	summary := nestedMap(t, first, "summary")
	expected := map[string]any{
		"title":          "The Example Show",
		"name":           "Example Name",
		"original_title": "Original Example",
		"overview":       "Provider overview",
		"release_date":   "2024-04-20",
		"first_air_date": "2024-04-21",
		"last_air_date":  "2024-04-22",
		"runtime":        float64(3600),
		"status":         "returning",
		"season_number":  float64(1),
		"episode_number": float64(2),
	}
	if len(summary) != len(expected) {
		t.Fatalf("expected curated summary size %d, got %#v", len(expected), summary)
	}
	for key, want := range expected {
		if got := summary[key]; got != want {
			t.Fatalf("expected source_evidence.summary[%q] = %#v, got %#v", key, want, got)
		}
	}
	for _, forbidden := range []string{"provider_blob", "genres", "popularity"} {
		if _, ok := summary[forbidden]; ok {
			t.Fatalf("expected source_evidence.summary to omit %q, got %#v", forbidden, summary)
		}
	}

	second := sourceEvidenceEntry(t, evidence[1])
	if _, ok := second["summary"]; ok {
		t.Fatalf("expected source_evidence summary to be omitted when no allowlisted scalar keys exist, got %#v", second)
	}
}

func assertCuratedFieldStateValues(t *testing.T, decoded map[string]any) {
	t.Helper()

	rawStates, ok := decoded["field_states"]
	if !ok {
		return
	}

	states, ok := rawStates.([]any)
	if !ok {
		t.Fatalf("expected field_states array, got %#v", rawStates)
	}
	if len(states) != 5 {
		t.Fatalf("expected 5 field states, got %d", len(states))
	}

	byField := make(map[string]map[string]any, len(states))
	for _, rawState := range states {
		state, ok := rawState.(map[string]any)
		if !ok {
			t.Fatalf("expected field_states object, got %#v", rawState)
		}
		fieldKey, ok := state["field_key"].(string)
		if !ok {
			t.Fatalf("expected field_key string, got %#v", state["field_key"])
		}
		byField[fieldKey] = state
	}

	assertFieldStateValue(t, byField, "title", "The Example Show")
	assertFieldStateValue(t, byField, "runtime", float64(3600))
	assertFieldStateValue(t, byField, "is_featured", true)
	assertFieldStateValueOmitted(t, byField, "provider_blob")
	assertFieldStateValueOmitted(t, byField, "genres")
}

func sourceEvidenceEntry(t *testing.T, value any) map[string]any {
	t.Helper()

	entry, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected source_evidence object, got %#v", value)
	}
	return entry
}

func nestedMap(t *testing.T, parent map[string]any, key string) map[string]any {
	t.Helper()

	value, ok := parent[key]
	if !ok {
		t.Fatalf("expected %q to exist, got %#v", key, parent)
	}
	mapped, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected %q to be an object, got %#v", key, value)
	}
	return mapped
}

func assertFieldStateValue(t *testing.T, states map[string]map[string]any, fieldKey string, want any) {
	t.Helper()

	state, ok := states[fieldKey]
	if !ok {
		t.Fatalf("expected field state %q, got %#v", fieldKey, states)
	}
	if got := state["value"]; got != want {
		t.Fatalf("expected field state %q value %#v, got %#v", fieldKey, want, got)
	}
}

func assertFieldStateValueOmitted(t *testing.T, states map[string]map[string]any, fieldKey string) {
	t.Helper()

	state, ok := states[fieldKey]
	if !ok {
		t.Fatalf("expected field state %q, got %#v", fieldKey, states)
	}
	if _, ok := state["value"]; ok {
		t.Fatalf("expected field state %q value to be omitted, got %#v", fieldKey, state)
	}
}

func marshalContractJSON(t *testing.T, value any) string {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}
	return string(payload)
}

func unmarshalContractJSON(t *testing.T, payload string) map[string]any {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("unmarshal contract: %v", err)
	}
	return decoded
}

func assertDTOsDoNotExposeRawRows(t *testing.T, path string) {
	t.Helper()

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	requiredDTOs := map[string]struct{}{
		"CatalogListItem":            {},
		"CatalogItemDetail":          {},
		"CatalogSeasonDetail":        {},
		"CatalogEpisodeDetail":       {},
		"CatalogAssetDetail":         {},
		"CatalogGovernanceWorkspace": {},
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if _, ok := requiredDTOs[typeSpec.Name.Name]; !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				t.Fatalf("expected %s to be a struct", typeSpec.Name.Name)
			}
			for _, field := range structType.Fields.List {
				if fieldUsesRawDatabaseRow(field.Type) {
					t.Fatalf("expected %s to avoid raw database row fields", typeSpec.Name.Name)
				}
			}
		}
	}
}

func fieldUsesRawDatabaseRow(expr ast.Expr) bool {
	switch typed := expr.(type) {
	case *ast.SelectorExpr:
		pkg, ok := typed.X.(*ast.Ident)
		if !ok || pkg.Name != "database" {
			return false
		}
		return typed.Sel.Name == "CatalogItem" || typed.Sel.Name == "MediaAsset"
	case *ast.ArrayType:
		return fieldUsesRawDatabaseRow(typed.Elt)
	case *ast.StarExpr:
		return fieldUsesRawDatabaseRow(typed.X)
	default:
		return false
	}
}
