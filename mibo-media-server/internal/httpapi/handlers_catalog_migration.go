package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/jobs"
	"gorm.io/gorm"
)

type queueCatalogLegacyBackfillInput struct {
	LibraryID *uint `json:"library_id,omitempty"`
}

func (r *Router) handleQueueCatalogLegacyBackfill(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	if r.jobs == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("jobs service unavailable"))
		return
	}

	var input queueCatalogLegacyBackfillInput
	if err := decodeOptionalJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	scope, jobKey, err := r.resolveLegacyBackfillScope(req.Context(), input.LibraryID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	if existingRun, found, err := r.findActiveLegacyBackfillRun(req.Context(), jobKey); err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	} else if found {
		writeJSON(req.Context(), w, http.StatusAccepted, existingRun)
		return
	}

	run, err := r.catalog.CreateLegacyBackfillRun(req.Context(), catalog.CreateLegacyBackfillRunInput{
		Scope:             scope,
		TriggeredByUserID: user.ID,
	})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	job, err := r.jobs.EnqueueUnique(req.Context(), catalog.JobKindLegacyBackfill, jobKey, catalog.LegacyBackfillPayload{RunID: run.ID, LibraryID: scope.LibraryID})
	if err != nil {
		r.markLegacyBackfillRunFailed(req.Context(), run.ID, err)
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	queuedRun, err := r.resolveQueuedLegacyBackfillRun(req.Context(), run.ID, job)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusAccepted, queuedRun)
}

func (r *Router) handleListCatalogMigrationRuns(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}

	runs, err := r.catalog.ListLegacyBackfillRuns(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, runs)
}

func (r *Router) handleGetCatalogMigrationRun(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}

	runID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	run, err := r.catalog.GetLegacyBackfillRun(req.Context(), runID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(req.Context(), w, http.StatusNotFound, err)
			return
		}
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, run)
}

func (r *Router) resolveLegacyBackfillScope(ctx context.Context, libraryID *uint) (catalog.LegacyBackfillScope, string, error) {
	if libraryID == nil {
		return catalog.LegacyBackfillScope{Kind: catalog.LegacyBackfillScopeAll}, "catalog-backfill-legacy:all", nil
	}
	if *libraryID == 0 {
		return catalog.LegacyBackfillScope{}, "", errors.New("library_id must be greater than zero")
	}
	if r.library == nil {
		return catalog.LegacyBackfillScope{}, "", errors.New("library service unavailable")
	}
	if _, err := r.library.GetLibrary(ctx, *libraryID); err != nil {
		return catalog.LegacyBackfillScope{}, "", err
	}
	return catalog.LegacyBackfillScope{Kind: catalog.LegacyBackfillScopeLibrary, LibraryID: libraryID}, fmt.Sprintf("catalog-backfill-legacy:library:%d", *libraryID), nil
}

func (r *Router) findActiveLegacyBackfillRun(ctx context.Context, jobKey string) (catalog.LegacyBackfillRun, bool, error) {
	var job database.Job
	err := r.db.WithContext(ctx).
		Where("job_key = ? AND status IN ?", jobKey, []string{jobs.StatusQueued, jobs.StatusRunning}).
		Order("id desc").
		First(&job).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return catalog.LegacyBackfillRun{}, false, nil
		}
		return catalog.LegacyBackfillRun{}, false, err
	}
	run, err := r.loadLegacyBackfillRunFromJob(ctx, job)
	if err != nil {
		return catalog.LegacyBackfillRun{}, false, err
	}
	return run, true, nil
}

func (r *Router) resolveQueuedLegacyBackfillRun(ctx context.Context, createdRunID uint, job database.Job) (catalog.LegacyBackfillRun, error) {
	payload, err := decodeLegacyBackfillPayload(job.PayloadJSON)
	if err != nil {
		return catalog.LegacyBackfillRun{}, err
	}
	if payload.RunID == createdRunID {
		return r.catalog.GetLegacyBackfillRun(ctx, createdRunID)
	}
	if err := r.db.WithContext(ctx).Delete(&database.CatalogMigrationRun{}, createdRunID).Error; err != nil {
		return catalog.LegacyBackfillRun{}, err
	}
	return r.catalog.GetLegacyBackfillRun(ctx, payload.RunID)
}

func (r *Router) loadLegacyBackfillRunFromJob(ctx context.Context, job database.Job) (catalog.LegacyBackfillRun, error) {
	payload, err := decodeLegacyBackfillPayload(job.PayloadJSON)
	if err != nil {
		return catalog.LegacyBackfillRun{}, err
	}
	return r.catalog.GetLegacyBackfillRun(ctx, payload.RunID)
}

func (r *Router) markLegacyBackfillRunFailed(ctx context.Context, runID uint, runErr error) {
	if runID == 0 {
		return
	}
	message := "failed to queue backfill run"
	if runErr != nil {
		message = runErr.Error()
	}
	_ = r.db.WithContext(ctx).Model(&database.CatalogMigrationRun{}).Where("id = ?", runID).Updates(map[string]any{
		"status":      catalog.LegacyBackfillStatusFailed,
		"fatal_error": message,
		"finished_at": time.Now().UTC(),
	}).Error
}

func decodeOptionalJSON(req *http.Request, out any) error {
	if req.Body == nil {
		return nil
	}
	defer req.Body.Close()

	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	if decoder.More() {
		return errors.New("request body must contain a single JSON document")
	}
	return nil
}

func decodeLegacyBackfillPayload(raw string) (catalog.LegacyBackfillPayload, error) {
	var payload catalog.LegacyBackfillPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return catalog.LegacyBackfillPayload{}, err
	}
	if payload.RunID == 0 {
		return catalog.LegacyBackfillPayload{}, errors.New("legacy backfill job missing run id")
	}
	return payload, nil
}
