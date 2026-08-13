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

func (srv *Server) componentCatalog(w http.ResponseWriter, r *http.Request) {
	srv.serveCatalog(w, r, models.KindComponent)
}

func (srv *Server) dependencyCatalog(w http.ResponseWriter, r *http.Request) {
	srv.serveCatalog(w, r, models.KindDependency)
}

// serveCatalog builds one published catalog document: schema_version plus the
// entries of every built dependency of the requested kind. Components and
// dependencies are served from separate endpoints because only component
// entries carry a slot.
func (srv *Server) serveCatalog(w http.ResponseWriter, r *http.Request, kind models.Kind) {
	deps, err := srv.store.ListApprovedDeps(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store error"})
		return
	}
	sort.Slice(deps, func(i, j int) bool { return deps[i].UpdatedAt.After(deps[j].UpdatedAt) })

	doc := models.Catalog{SchemaVersion: models.SchemaVersion, Entries: []models.CatalogEntry{}}
	seen := make(map[string]struct{}, len(deps))
	for _, d := range deps {
		if d.Kind != kind {
			continue
		}
		// One release per name+version; the newest build wins.
		key := d.Entry.Name + "\x00" + d.Entry.Version
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		doc.Entries = append(doc.Entries, d.Entry.NormalizeForCatalog())
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

// submitDepBody is a catalog entry plus the kind that selects its catalog and
// the descriptive metadata the site shows but the document does not carry.
// The entry id is assigned by the server, so any client-supplied id is
// ignored.
type submitDepBody struct {
	models.CatalogEntry
	Kind        models.Kind `json:"kind"`
	Category    string      `json:"category"`
	Description string      `json:"description"`
	License     string      `json:"license"`
}

func (srv *Server) submitDep(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	var body submitDepBody
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	// The entry id is the record's own uuid: the schema types it as a uuid and
	// one catalog entry corresponds to exactly one record.
	id := uuid.NewString()
	entry := body.CatalogEntry
	entry.ID = id

	if err := entry.Validate(body.Kind); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	dep, err := srv.store.CreateDep(r.Context(), models.Dependency{
		ID:          id,
		Name:        entry.Name,
		Kind:        body.Kind,
		Category:    body.Category,
		Description: body.Description,
		License:     body.License,
		Status:      "pending_review",
		SubmittedBy: claims.UserID,
		Entry:       entry,
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
