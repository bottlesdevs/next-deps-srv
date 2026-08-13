package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
)

func (srv *Server) getFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	file, err := srv.store.GetFileByName(r.Context(), name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	revs, _ := srv.store.RevisionsByFile(r.Context(), file.ID)
	type revOut struct {
		ID          string `json:"id"`
		RevisionNum int    `json:"revision_num"`
		Hash        string `json:"hash"`
		SizeBytes   int64  `json:"size_bytes"`
		SourceDep   string `json:"source_dep"`
		ArchiveURL  string `json:"archive_url"`
		DownloadURL string `json:"download_url"`
	}
	var outs []revOut
	for _, rev := range revs {
		outs = append(outs, revOut{
			ID:          rev.ID,
			RevisionNum: rev.RevisionNum,
			Hash:        rev.Hash,
			SizeBytes:   rev.SizeBytes,
			SourceDep:   rev.SourceDepID,
			ArchiveURL:  rev.ArchiveURL,
			DownloadURL: fmt.Sprintf("/api/v1/files/download/%s", rev.ID),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"file": file, "revisions": outs})
}

func (srv *Server) downloadFile(w http.ResponseWriter, r *http.Request) {
	revID := r.PathValue("rev_id")
	rev, err := srv.store.GetRevision(r.Context(), revID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filepath.Base(rev.StoragePath)+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	if err := srv.backend.Stream(r.Context(), rev.StoragePath, w); err != nil {
		// headers already sent, can't write error
		return
	}
}

// depFiles lists the bucket files produced by builds of one dependency, with
// how many revisions of each came from it.
func (srv *Server) depFiles(w http.ResponseWriter, r *http.Request) {
	depID := r.PathValue("id")
	if _, err := srv.store.GetDep(r.Context(), depID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	revs, err := srv.store.RevisionsByDep(r.Context(), depID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store error"})
		return
	}

	type fileOut struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		RevisionCount int    `json:"revision_count"`
		LatestRevID   string `json:"latest_rev_id"`
		SizeBytes     int64  `json:"size_bytes"`
		Platform      string `json:"platform,omitempty"`
	}

	byFile := make(map[string]*fileOut)
	order := make([]string, 0, len(revs))
	for _, rev := range revs {
		out, seen := byFile[rev.FileID]
		if !seen {
			file, err := srv.store.GetFile(r.Context(), rev.FileID)
			if err != nil {
				continue // revision outlived its file record
			}
			out = &fileOut{ID: file.ID, Name: file.Name, LatestRevID: file.LatestRevID}
			byFile[rev.FileID] = out
			order = append(order, rev.FileID)
		}
		out.RevisionCount++
		// Report the newest revision this dependency contributed.
		if rev.SizeBytes > 0 && out.SizeBytes == 0 {
			out.SizeBytes = rev.SizeBytes
		}
		if out.Platform == "" {
			out.Platform = rev.Platform
		}
	}

	items := make([]fileOut, 0, len(order))
	for _, id := range order {
		items = append(items, *byFile[id])
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	writeJSON(w, http.StatusOK, map[string]any{"total": len(items), "items": items})
}
