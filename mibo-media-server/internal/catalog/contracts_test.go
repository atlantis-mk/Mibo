package catalog

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
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
