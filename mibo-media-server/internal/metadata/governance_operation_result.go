package metadata

type GovernanceOperationResultInput struct {
	Operation            string
	OriginMetadataItemID uint
	TargetMetadataItemID uint
	TargetType           string
	Status               string
	GovernanceStatus     string
	Plan                 MetadataExecutionPlan
	MetadataItemIDs      []uint
	LibraryID            uint
	MetadataRootID       *uint
	AppliedFields        []MetadataAppliedField
	SkippedFields        []MetadataSkippedField
	Warnings             []MetadataOperationWarning
}

func newGovernanceOperationResult(input GovernanceOperationResultInput) MetadataOperationResult {
	return MetadataOperationResult{
		Operation:            input.Operation,
		OriginMetadataItemID: input.OriginMetadataItemID,
		TargetMetadataItemID: input.TargetMetadataItemID,
		TargetType:           input.TargetType,
		Status:               input.Status,
		GovernanceStatus:     input.GovernanceStatus,
		Plan:                 metadataExecutionPlanSummary(input.Plan),
		AffectedScope: MetadataAffectedScope{
			MetadataItemIDs: input.MetadataItemIDs,
			LibraryID:       input.LibraryID,
			MetadataRootID:  input.MetadataRootID,
		},
		AppliedFields: input.AppliedFields,
		SkippedFields: input.SkippedFields,
		Warnings:      input.Warnings,
	}
}
