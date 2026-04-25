package catalog

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestCatalogDTOContractExportsRequiredTypes(t *testing.T) {
	exported := []any{
		CatalogListItem{},
		CatalogItemDetail{},
		CatalogSeasonDetail{},
		CatalogEpisodeDetail{},
		CatalogAssetDetail{},
		CatalogGovernanceWorkspace{},
	}

	if len(exported) != 6 {
		t.Fatalf("expected 6 exported catalog DTO types, got %d", len(exported))
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
	forbidden := []string{
		"gorm:\"",
		"database.CatalogItem",
		"database.MediaAsset",
	}
	for _, fragment := range forbidden {
		if strings.Contains(text, fragment) {
			t.Fatalf("expected contracts.go to exclude %q", fragment)
		}
	}
	if !strings.Contains(text, "series") {
		t.Fatalf("expected contracts.go to include canonical series type")
	}
	if strings.Contains(text, "show") {
		t.Fatalf("expected contracts.go to avoid legacy show naming")
	}
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
