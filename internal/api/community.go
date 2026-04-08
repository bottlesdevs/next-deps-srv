package api

import (
	"net/http"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/middleware"
	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/google/uuid"
)

func (srv *Server) listPosts(w http.ResponseWriter, r *http.Request) {
	page, limit := pageLimit(r)
	posts, err := srv.store.ListTopPosts(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store error"})
		return
	}
	out := paginate(posts, page, limit)
	if items, ok := out["items"].([]models.CommunityPost); ok {
		type postOut struct {
			models.CommunityPost
			ReplyCount int `json:"reply_count"`
		}
		var rich []postOut
		for _, p := range items {
			n, _ := srv.store.CountReplies(r.Context(), p.ID)
			rich = append(rich, postOut{p, n})
		}
		out["items"] = rich
	}
	writeJSON(w, http.StatusOK, out)
}

func (srv *Server) createPost(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	var body struct {
		Body string `json:"body"`
	}
	if err := readJSON(r, &body); err != nil || body.Body == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body required"})
		return
	}
	post, err := srv.store.CreatePost(r.Context(), models.CommunityPost{
		ID:        uuid.NewString(),
		AuthorID:  claims.UserID,
		ParentID:  "",
		Body:      body.Body,
		CreatedAt: time.Now(),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store error"})
		return
	}
	writeJSON(w, http.StatusCreated, post)
}

func (srv *Server) listReplies(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	page, limit := pageLimit(r)
	replies, err := srv.store.ListReplies(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store error"})
		return
	}
	writeJSON(w, http.StatusOK, paginate(replies, page, limit))
}

func (srv *Server) createReply(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	parentID := r.PathValue("id")
	_, err := srv.store.GetPost(r.Context(), parentID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "parent post not found"})
		return
	}
	var body struct {
		Body string `json:"body"`
	}
	if err := readJSON(r, &body); err != nil || body.Body == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body required"})
		return
	}
	post, err := srv.store.CreatePost(r.Context(), models.CommunityPost{
		ID:        uuid.NewString(),
		AuthorID:  claims.UserID,
		ParentID:  parentID,
		Body:      body.Body,
		CreatedAt: time.Now(),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store error"})
		return
	}
	writeJSON(w, http.StatusCreated, post)
}

func (srv *Server) deletePost(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	id := r.PathValue("id")
	post, err := srv.store.GetPost(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	isAdmin := false
	for _, role := range claims.Roles {
		if role == "admin" || role == "mod" {
			isAdmin = true
			break
		}
	}
	if post.AuthorID != claims.UserID && !isAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	post.Deleted = true
	post.Body = ""
	if err := srv.store.UpdatePost(r.Context(), post); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
