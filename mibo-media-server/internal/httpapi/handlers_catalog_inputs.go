package httpapi

type catalogGovernanceFieldUpdateInput struct {
	FieldKey   string `json:"field_key"`
	Value      any    `json:"value"`
	Lock       bool   `json:"lock"`
	LockReason string `json:"lock_reason"`
	Force      bool   `json:"force"`
}

type catalogGovernanceImageSelectionInput struct {
	ImageType string `json:"image_type"`
	URL       string `json:"url"`
}

type catalogGovernanceResourceLinkInput struct {
	TargetMetadataItemID uint     `json:"target_metadata_item_id"`
	SourceMetadataItemID *uint    `json:"source_metadata_item_id"`
	LibraryID            uint     `json:"library_id"`
	Mode                 string   `json:"mode"`
	Role                 string   `json:"role"`
	SegmentIndex         int      `json:"segment_index"`
	StartSeconds         *float64 `json:"start_seconds"`
	EndSeconds           *float64 `json:"end_seconds"`
}

type catalogGovernanceResourceLinkUpdateInput struct {
	LibraryID    uint   `json:"library_id"`
	Role         string `json:"role"`
	SegmentIndex int    `json:"segment_index"`
	NewRole      string `json:"new_role"`
	ReviewState  string `json:"review_state"`
}

type catalogGovernanceMetadataMergeInput struct {
	TargetMetadataItemID uint `json:"target_metadata_item_id"`
	LibraryID            uint `json:"library_id"`
}

type catalogGovernanceMetadataSplitInput struct {
	TargetMetadataItemID uint   `json:"target_metadata_item_id"`
	ResourceIDs          []uint `json:"resource_ids"`
	LibraryID            uint   `json:"library_id"`
}

type catalogGovernanceProjectionVisibilityInput struct {
	LibraryID uint `json:"library_id"`
	Hidden    bool `json:"hidden"`
}

type scanExclusionMarkInput struct {
	Reason string `json:"reason"`
}

type filenameExclusionRestoreInput struct {
	InventoryFileID uint `json:"inventory_file_id"`
}

type scanExclusionEnabledInput struct {
	Enabled bool `json:"enabled"`
}

type scanExclusionRuleInput struct {
	LibraryID   *uint  `json:"library_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RuleType    string `json:"rule_type"`
	Value       string `json:"value"`
	Reason      string `json:"reason"`
	Enabled     *bool  `json:"enabled"`
}

type replaceLibraryScanExclusionRulesInput struct {
	Rules []scanExclusionRuleInput `json:"rules"`
}
