package api

import (
	"net/http"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/middleware"
	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/google/uuid"
)

func (srv *Server) listDeps(w http.ResponseWriter, r *http.Request) {
	page, limit := pageLimit(r)
	deps, err := srv.store.ListApprovedDeps(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store error"})
		return
	}
	writeJSON(w, http.StatusOK, paginate(deps, page, limit))
}

func (srv *Server) getDep(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	dep, err := srv.store.GetDep(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, dep)
}

func (srv *Server) submitDep(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	var body models.Manifest
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if body.Name == "" || body.URL == "" || body.ExpectedHash == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, url and expected_hash required"})
		return
	}
	dep, err := srv.store.CreateDep(r.Context(), models.Dependency{
		ID:          uuid.NewString(),
		Name:        body.Name,
		Status:      "pending_review",
		SubmittedBy: claims.UserID,
		Manifest:    body,
		CreatedAt:   time.Now(),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store error"})
		return
	}
	// notify mods
	go func() {
		if srv.mailer != nil {
			mods, _ := srv.store.AdminAndModUsers(r.Context())
			var emails []string
			for _, m := range mods {
				emails = append(emails, m.Email)
			}
			_ = srv.mailer.DepSubmitted(dep, emails)
		}
	}()
	logAudit(r.Context(), srv.store, claims.UserID, claims.Username, "submit_dep", dep.ID, dep.Name, ipFrom(r))
	writeJSON(w, http.StatusCreated, dep)
}

func (srv *Server) pendingDeps(w http.ResponseWriter, r *http.Request) {
	page, limit := pageLimit(r)
	deps, err := srv.store.ListPendingDeps(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store error"})
		return
	}
	writeJSON(w, http.StatusOK, paginate(deps, page, limit))
}

func (srv *Server) approveDep(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	id := r.PathValue("id")
	dep, err := srv.store.GetDep(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	dep.Status = "approved"
	dep.ReviewedBy = claims.UserID
	dep.UpdatedAt = time.Now()
	if err := srv.store.UpdateDep(r.Context(), dep); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	// launch build
	job, err := srv.queue.Submit(r.Context(), dep)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "queue error"})
		return
	}
	logAudit(r.Context(), srv.store, claims.UserID, claims.Username, "approve_dep", dep.ID, dep.Name, ipFrom(r))
	writeJSON(w, http.StatusOK, map[string]any{"dep": dep, "job": job})
}

func (srv *Server) rejectDep(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	id := r.PathValue("id")
	dep, err := srv.store.GetDep(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = readJSON(r, &body)
	dep.Status = "rejected"
	dep.ReviewedBy = claims.UserID
	dep.RejectReason = body.Reason
	dep.UpdatedAt = time.Now()
	if err := srv.store.UpdateDep(r.Context(), dep); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	logAudit(r.Context(), srv.store, claims.UserID, claims.Username, "reject_dep", dep.ID, dep.Name, ipFrom(r))
	writeJSON(w, http.StatusOK, dep)
}

func paginate[T any](items []T, page, limit int) map[string]any {
	total := len(items)
	start := page * limit
	end := start + limit
	if start >= total {
		items = []T{}
	} else {
		if end > total {
			end = total
		}
		items = items[start:end]
	}
	return map[string]any{"total": total, "page": page, "limit": limit, "items": items}
}
