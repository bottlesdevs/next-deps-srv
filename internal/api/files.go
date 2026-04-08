package api

import (
	"fmt"
	"net/http"
	"path/filepath"
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
