package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/atlan/mibo-media-server/internal/library"
)

func (r *Router) handleSetScanExclusionEnabled(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	exclusionID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input scanExclusionEnabledInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	exclusion, err := r.library.SetScanExclusionEnabled(req.Context(), library.SetScanExclusionEnabledInput{ExclusionID: exclusionID, Enabled: input.Enabled, UserID: &user.ID})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, exclusion)
}

func (r *Router) handleListScanExclusions(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	libraryID, err := parseOptionalUintQuery(req, "library_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	enabled, err := parseOptionalEnabledQuery(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	exclusions, err := r.library.ListScanExclusionsView(req.Context(), library.ListScanExclusionsInput{LibraryID: libraryID, Enabled: enabled})
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, exclusions)
}

func (r *Router) handleListFilenameExclusionRules(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	libraryID, err := parseOptionalUintQuery(req, "library_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	enabled, err := parseOptionalEnabledQuery(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	rules, err := r.library.ListFilenameExclusionRules(req.Context(), library.ListScanExclusionsInput{LibraryID: libraryID, Enabled: enabled})
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, rules)
}

func (r *Router) handleSetFilenameExclusionRuleEnabled(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	ruleID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input scanExclusionEnabledInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	rule, err := r.library.SetFilenameExclusionRuleEnabled(req.Context(), library.SetFilenameExclusionRuleEnabledInput{RuleID: ruleID, Enabled: input.Enabled, UserID: &user.ID})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, rule)
}

func (r *Router) handleRestoreFilenameExclusionMatch(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	ruleID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var input filenameExclusionRestoreInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	restore, err := r.library.RestoreFilenameExclusionMatch(req.Context(), library.RestoreFilenameExclusionMatchInput{RuleID: ruleID, InventoryFileID: input.InventoryFileID, UserID: &user.ID})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, restore)
}

func (r *Router) handleListScanExclusionRules(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	rules, err := r.library.ListScanExclusionRules(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, rules)
}

func (r *Router) handleCreateScanExclusionRule(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	var body scanExclusionRuleInput
	if err := decodeJSON(req, &body); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	rule, err := r.library.CreateScanExclusionRule(req.Context(), library.ScanExclusionRuleInput{LibraryID: body.LibraryID, Name: body.Name, Description: body.Description, RuleType: body.RuleType, Value: body.Value, Reason: body.Reason, Enabled: enabled, UserID: &user.ID})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusCreated, rule)
}

func (r *Router) handleUpdateScanExclusionRule(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	ruleID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var body scanExclusionRuleInput
	if err := decodeJSON(req, &body); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if body.Enabled != nil && body.LibraryID == nil && strings.TrimSpace(body.Name) == "" && strings.TrimSpace(body.RuleType) == "" && strings.TrimSpace(body.Value) == "" && strings.TrimSpace(body.Reason) == "" && strings.TrimSpace(body.Description) == "" {
		rule, err := r.library.SetScanExclusionRuleEnabled(req.Context(), library.SetScanExclusionRuleEnabledInput{RuleID: ruleID, Enabled: *body.Enabled, UserID: &user.ID})
		if err != nil {
			writeError(req.Context(), w, http.StatusBadRequest, err)
			return
		}
		writeJSON(req.Context(), w, http.StatusOK, rule)
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	rule, err := r.library.UpdateScanExclusionRule(req.Context(), library.UpdateScanExclusionRuleInput{RuleID: ruleID, LibraryID: body.LibraryID, Name: body.Name, Description: body.Description, RuleType: body.RuleType, Value: body.Value, Reason: body.Reason, Enabled: enabled, UserID: &user.ID})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, rule)
}

func (r *Router) handleDeleteScanExclusionRule(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	ruleID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if err := r.library.DeleteScanExclusionRule(req.Context(), ruleID); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (r *Router) handleReplaceLibraryScanExclusionRules(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.library == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("library service unavailable"))
		return
	}
	libraryID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	var body replaceLibraryScanExclusionRulesInput
	if err := decodeJSON(req, &body); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	inputs := make([]library.ScanExclusionRuleInput, 0, len(body.Rules))
	for _, rule := range body.Rules {
		enabled := true
		if rule.Enabled != nil {
			enabled = *rule.Enabled
		}
		inputs = append(inputs, library.ScanExclusionRuleInput{Name: rule.Name, Description: rule.Description, RuleType: rule.RuleType, Value: rule.Value, Reason: rule.Reason, Enabled: enabled})
	}
	rules, err := r.library.ReplaceLibraryScanExclusionRules(req.Context(), libraryID, inputs, &user.ID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, rules)
}

func parseOptionalEnabledQuery(req *http.Request) (*bool, error) {
	switch strings.ToLower(strings.TrimSpace(req.URL.Query().Get("enabled"))) {
	case "true", "1", "yes":
		value := true
		return &value, nil
	case "false", "0", "no":
		value := false
		return &value, nil
	case "", "all":
		return nil, nil
	default:
		return nil, errors.New("invalid query parameter \"enabled\"")
	}
}
