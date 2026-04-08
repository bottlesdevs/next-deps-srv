package store

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/mirkobrombin/go-slipstream/pkg/engine"
	"github.com/mirkobrombin/go-slipstream/pkg/wal"
)

type Store struct {
	Users   *engine.Bitcask[models.User]
	Deps    *engine.Bitcask[models.Dependency]
	Jobs    *engine.Bitcask[models.BuildJob]
	Files   *engine.Bitcask[models.BucketFile]
	Revs    *engine.Bitcask[models.FileRevision]
	Posts   *engine.Bitcask[models.CommunityPost]
	Audit   *engine.Bitcask[models.AuditEntry]
	Config  *engine.Bitcask[string]
	dataDir string
}

func Open(dataDir string) (*Store, error) {
	s := &Store{dataDir: dataDir}
	var err error

	if s.Users, err = openBitcask[models.User](filepath.Join(dataDir, "users")); err != nil {
		return nil, fmt.Errorf("users: %w", err)
	}
	s.Users.AddIndex("email", func(u models.User) string { return u.Email })
	s.Users.AddIndex("username", func(u models.User) string { return u.Username })
	s.Users.AddIndex("all", func(models.User) string { return "all" })
	if err = s.Users.Engine().Recover(); err != nil {
		return nil, fmt.Errorf("users recover: %w", err)
	}

	if s.Deps, err = openBitcask[models.Dependency](filepath.Join(dataDir, "deps")); err != nil {
		return nil, fmt.Errorf("deps: %w", err)
	}
	s.Deps.AddIndex("status", func(d models.Dependency) string { return d.Status })
	s.Deps.AddIndex("submitted_by", func(d models.Dependency) string { return d.SubmittedBy })
	s.Deps.AddIndex("name", func(d models.Dependency) string { return d.Name })
	s.Deps.AddIndex("all", func(models.Dependency) string { return "all" })
	if err = s.Deps.Engine().Recover(); err != nil {
		return nil, fmt.Errorf("deps recover: %w", err)
	}

	if s.Jobs, err = openBitcask[models.BuildJob](filepath.Join(dataDir, "jobs")); err != nil {
		return nil, fmt.Errorf("jobs: %w", err)
	}
	s.Jobs.AddIndex("dep_id", func(j models.BuildJob) string { return j.DependencyID })
	s.Jobs.AddIndex("status", func(j models.BuildJob) string { return j.Status })
	s.Jobs.AddIndex("all", func(models.BuildJob) string { return "all" })
	if err = s.Jobs.Engine().Recover(); err != nil {
		return nil, fmt.Errorf("jobs recover: %w", err)
	}

	if s.Files, err = openBitcask[models.BucketFile](filepath.Join(dataDir, "files")); err != nil {
		return nil, fmt.Errorf("files: %w", err)
	}
	s.Files.AddIndex("name", func(f models.BucketFile) string { return f.Name })
	s.Files.AddIndex("bucket_char", func(f models.BucketFile) string { return f.BucketChar })
	s.Files.AddIndex("all", func(models.BucketFile) string { return "all" })
	if err = s.Files.Engine().Recover(); err != nil {
		return nil, fmt.Errorf("files recover: %w", err)
	}

	if s.Revs, err = openBitcask[models.FileRevision](filepath.Join(dataDir, "revisions")); err != nil {
		return nil, fmt.Errorf("revisions: %w", err)
	}
	s.Revs.AddIndex("file_id", func(r models.FileRevision) string { return r.FileID })
	s.Revs.AddIndex("job_id", func(r models.FileRevision) string { return r.SourceJobID })
	s.Revs.AddIndex("all", func(models.FileRevision) string { return "all" })
	if err = s.Revs.Engine().Recover(); err != nil {
		return nil, fmt.Errorf("revisions recover: %w", err)
	}

	if s.Posts, err = openBitcask[models.CommunityPost](filepath.Join(dataDir, "posts")); err != nil {
		return nil, fmt.Errorf("posts: %w", err)
	}
	s.Posts.AddIndex("parent_id", func(p models.CommunityPost) string { return p.ParentID })
	s.Posts.AddIndex("author_id", func(p models.CommunityPost) string { return p.AuthorID })
	s.Posts.AddIndex("all", func(models.CommunityPost) string { return "all" })
	if err = s.Posts.Engine().Recover(); err != nil {
		return nil, fmt.Errorf("posts recover: %w", err)
	}

	if s.Audit, err = openBitcask[models.AuditEntry](filepath.Join(dataDir, "audit")); err != nil {
		return nil, fmt.Errorf("audit: %w", err)
	}
	s.Audit.AddIndex("user_id", func(a models.AuditEntry) string { return a.UserID })
	s.Audit.AddIndex("action", func(a models.AuditEntry) string { return a.Action })
	s.Audit.AddIndex("all", func(models.AuditEntry) string { return "all" })
	if err = s.Audit.Engine().Recover(); err != nil {
		return nil, fmt.Errorf("audit recover: %w", err)
	}

	if s.Config, err = openBitcask[string](filepath.Join(dataDir, "config")); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	s.Config.AddIndex("all", func(string) string { return "all" })
	if err = s.Config.Engine().Recover(); err != nil {
		return nil, fmt.Errorf("config recover: %w", err)
	}

	return s, nil
}

func (s *Store) Close() {
	s.Users.Close()
	s.Deps.Close()
	s.Jobs.Close()
	s.Files.Close()
	s.Revs.Close()
	s.Posts.Close()
	s.Audit.Close()
	s.Config.Close()
}

func (s *Store) GetConfig(ctx context.Context) (models.AppConfig, error) {
	raw, err := s.Config.Get(ctx, "app_config")
	if err != nil {
		return defaultConfig(), nil
	}
	var cfg models.AppConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return defaultConfig(), nil
	}
	return cfg, nil
}

func (s *Store) SaveConfig(ctx context.Context, cfg models.AppConfig) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.Config.Put(ctx, "app_config", string(b), 0)
}

func defaultConfig() models.AppConfig {
	return models.AppConfig{
		RegistrationOpen: true,
		RateLimit: models.RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 60,
			BurstSize:         10,
		},
	}
}

func (s *Store) DataDir() string { return s.dataDir }

func openBitcask[T any](dir string) (*engine.Bitcask[T], error) {
	w, err := wal.NewManager(dir)
	if err != nil {
		return nil, err
	}
	return engine.NewBitcask[T](w,
		func(v T) ([]byte, error) { return json.Marshal(v) },
		func(b []byte) (T, error) { var v T; return v, json.Unmarshal(b, &v) },
	), nil
}
