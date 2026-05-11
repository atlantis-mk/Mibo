package playback

type ClientProfile string

const (
	ClientProfileWeb    ClientProfile = "web"
	ClientProfileMobile ClientProfile = "mobile"
	ClientProfileTV     ClientProfile = "tv"
)

type PlaybackRequest struct {
	ItemID         uint
	MetadataItemID uint
	ResourceID     uint
	LibraryID      uint
	UserID         *uint
	ClientProfile  ClientProfile
}

type PlaybackDecision struct {
	Kind          string           `json:"kind"`
	ClientProfile ClientProfile    `json:"client_profile"`
	SelectedBy    string           `json:"selected_by"`
	FallbackKind  string           `json:"fallback_kind,omitempty"`
	Reasons       []DecisionReason `json:"reasons"`
}

type DecisionReason struct {
	Code     string `json:"code"`
	Category string `json:"category"`
	Message  string `json:"message"`
}
