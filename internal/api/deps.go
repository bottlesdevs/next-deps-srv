package api

import (
	"context"
	"log"
	"net/http"
	"sort"
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

// catalog serves the published catalog document: schema_version plus every
// built dependency's item. This is the endpoint consumers fetch.
func (srv *Server) catalog(w http.ResponseWriter, r *http.Request) {
	deps, err := srv.store.ListApprovedDeps(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store error"})
		return
	}
	sort.Slice(deps, func(i, j int) bool { return deps[i].UpdatedAt.After(deps[j].UpdatedAt) })
	doc := models.Catalog{SchemaVersion: models.SchemaVersion, Items: make([]models.Item, 0, len(deps))}
	seen := make(map[string]struct{}, len(deps))
	for _, d := range deps {
		// items must be unique; the newest build of an id+version wins.
		key := d.Item.ID + "\x00" + d.Item.Version
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		doc.Items = append(doc.Items, d.Item)
	}
	writeJSON(w, http.StatusOK, doc)
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

// submitDepBody is a catalog item plus the descriptive metadata the site
// shows but the published catalog document does not carry.
type submitDepBody struct {
	models.Item
	Category    string `json:"category"`
	Description string `json:"description"`
	License     string `json:"license"`
}

func (srv *Server) submitDep(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	var body submitDepBody
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := body.Item.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	dep, err := srv.store.CreateDep(r.Context(), models.Dependency{
		ID:          uuid.NewString(),
		Name:        body.Item.Name,
		Category:    body.Category,
		Description: body.Description,
		License:     body.License,
		Status:      "pending_review",
		SubmittedBy: claims.UserID,
		Item:        body.Item,
		CreatedAt:   time.Now(),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store error"})
		return
	}
	// notify mods
	go func() {
		if srv.mailer != nil {
			mods, _ := srv.store.AdminAndModUsers(context.Background())
			var emails []string
			for _, m := range mods {
				emails = append(emails, m.Email)
			}
			if err := srv.mailer.DepSubmitted(dep, emails); err != nil {
				log.Printf("mail: %v", err)
			}
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
	if srv.mailer != nil {
		d := dep
		go func() {
			submitter, err := srv.store.GetUser(context.Background(), d.SubmittedBy)
			if err == nil {
				if err := srv.mailer.DepApproved(d, submitter.Email); err != nil {
					log.Printf("mail: %v", err)
				}
			}
		}()
	}
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
	if srv.mailer != nil {
		d := dep
		reason := body.Reason
		go func() {
			submitter, err := srv.store.GetUser(context.Background(), d.SubmittedBy)
			if err == nil {
				if err := srv.mailer.DepRejected(d, submitter.Email, reason); err != nil {
					log.Printf("mail: %v", err)
				}
			}
		}()
	}
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
