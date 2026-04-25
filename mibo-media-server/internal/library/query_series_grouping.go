package library

import "strings"

func normalizeSeriesGroupingTitle(input string) string {
	cleaned := cleanTitle(input)
	normalized := strings.Join(strings.Fields(strings.TrimSpace(cleaned)), " ")
	return strings.TrimSpace(normalized)
}
