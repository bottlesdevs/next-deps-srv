package api

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/auth"
	"github.com/bottlesdevs/next-deps-srv/internal/middleware"
	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/google/uuid"
)

func (srv *Server) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	user, err := srv.store.GetUserByUsername(r.Context(), body.Username)
	if err != nil || !auth.CheckPassword(user.PasswordHash, body.Password) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}
	token, err := auth.IssueToken(user, srv.secret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (srv *Server) register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &body); err != nil || body.Username == "" || body.Email == "" || body.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username, email and password required"})
		return
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	user, err := srv.store.CreateUser(r.Context(), models.User{
		ID:           uuid.NewString(),
		Username:     body.Username,
		Email:        body.Email,
		PasswordHash: hash,
		Roles:        []string{"contributor"},
		CreatedAt:    time.Now(),
	})
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "user already exists"})
		return
	}
	token, _ := auth.IssueToken(user, srv.secret)
	if srv.mailer != nil {
		u := user
		go func() {
			admins, _ := srv.store.AdminAndModUsers(context.Background())
			var emails []string
			for _, a := range admins {
				emails = append(emails, a.Email)
			}
			if err := srv.mailer.UserRegistered(u, emails); err != nil {
				log.Printf("mail: %v", err)
			}
		}()
	}
	writeJSON(w, http.StatusCreated, map[string]string{"token": token})
}

func (srv *Server) getMe(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	if claims == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	user, err := srv.store.GetUser(r.Context(), claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, safeUser(user))
}

func (srv *Server) updateMe(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	if claims == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	user, err := srv.store.GetUser(r.Context(), claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	var body struct {
		Email   string `json:"email"`
		Bio     string `json:"bio"`
		Website string `json:"website"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if body.Email != "" {
		user.Email = body.Email
	}
	user.Bio = body.Bio
	user.Website = body.Website
	user.UpdatedAt = time.Now()
	if err := srv.store.UpdateUser(r.Context(), user); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	writeJSON(w, http.StatusOK, safeUser(user))
}

func (srv *Server) uploadAvatar(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	if claims == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20) // 2 MB
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file too large or invalid form"})
		return
	}
	file, hdr, err := r.FormFile("avatar")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no file"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(hdr.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !allowed[ext] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported file type"})
		return
	}

	avatarDir := filepath.Join(srv.dataDir, "avatars")
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "storage error"})
		return
	}
	dest := filepath.Join(avatarDir, claims.UserID+ext)
	f, err := os.Create(dest)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "storage error"})
		return
	}
	defer f.Close()
	if _, err := io.Copy(f, file); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "write error"})
		return
	}
	user, _ := srv.store.GetUser(r.Context(), claims.UserID)
	user.AvatarExt = ext
	user.UpdatedAt = time.Now()
	_ = srv.store.UpdateUser(r.Context(), user)
	writeJSON(w, http.StatusOK, map[string]string{"url": fmt.Sprintf("/api/v1/users/%s/avatar", claims.UserID)})
}

func (srv *Server) serveAvatar(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	user, err := srv.store.GetUser(r.Context(), id)
	if err != nil || user.AvatarExt == "" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(srv.dataDir, "avatars", id+user.AvatarExt))
}

type safeUserOut struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	Bio       string    `json:"bio"`
	Website   string    `json:"website"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

func safeUser(u models.User) safeUserOut {
	av := ""
	if u.AvatarExt != "" {
		av = fmt.Sprintf("/api/v1/users/%s/avatar", u.ID)
	}
	return safeUserOut{
		ID: u.ID, Username: u.Username, Email: u.Email, Roles: u.Roles,
		Bio: u.Bio, Website: u.Website, AvatarURL: av, CreatedAt: u.CreatedAt,
	}
}
