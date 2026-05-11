package metadata

func OperationResponseFromResult(value MetadataOperationResult) MetadataOperationResponse {
	return MetadataOperationResponse{
		Operation:            value.Operation,
		OriginMetadataItemID: value.OriginMetadataItemID,
		TargetMetadataItemID: value.TargetMetadataItemID,
		TargetType:           value.TargetType,
		Status:               value.Status,
		GovernanceStatus:     value.GovernanceStatus,
		Plan:                 value.Plan,
		ProviderAttempts:     value.ProviderAttempts,
		MetadataSourceIDs:    value.MetadataSourceIDs,
		AppliedFields:        value.AppliedFields,
		SkippedFields:        value.SkippedFields,
		AffectedScope:        value.AffectedScope,
		Warnings:             value.Warnings,
	}
}

func derefUintForOperation(value *uint) uint {
	if value == nil {
		return 0
	}
	return *value
}

func appendUniqueUint(values []uint, additions ...uint) []uint {
	seen := make(map[uint]struct{}, len(values)+len(additions))
	result := make([]uint, 0, len(values)+len(additions))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	for _, value := range additions {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func searchCandidateFromNormalized(value NormalizedMetadataCandidate) SearchCandidate {
	return SearchCandidate{Provider: value.Provider, MediaType: value.ProviderType, ExternalID: value.ExternalID, Title: value.Title, OriginalTitle: value.OriginalTitle, Overview: value.Overview, PosterURL: value.PosterURL, BackdropURL: value.BackdropURL, ReleaseDate: value.ReleaseDate, Year: value.Year, Confidence: value.Confidence, MatchedQuery: value.MatchedQuery, ReasonSummary: value.ReasonSummary}
}

func searchCandidatesFromNormalized(values []NormalizedMetadataCandidate) []SearchCandidate {
	items := make([]SearchCandidate, 0, len(values))
	for _, value := range values {
		items = append(items, searchCandidateFromNormalized(value))
	}
	return items
}
