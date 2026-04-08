package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/api"
	"github.com/bottlesdevs/next-deps-srv/internal/auth"
	"github.com/bottlesdevs/next-deps-srv/internal/bucket"
	"github.com/bottlesdevs/next-deps-srv/internal/email"
	"github.com/bottlesdevs/next-deps-srv/internal/middleware"
	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/bottlesdevs/next-deps-srv/internal/queue"
	"github.com/bottlesdevs/next-deps-srv/internal/store"
	"github.com/google/uuid"
	"github.com/mirkobrombin/go-cli-builder/v2/pkg/cli"
)

type CLI struct {
	Serve ServeCmd `cmd:"serve" help:"Start the server"`
	cli.Base
}

type ServeCmd struct {
	Port      int    `cli:"port,p" help:"Port to listen on" default:"8080" env:"PORT"`
	DataDir   string `cli:"data,d" help:"Data directory" default:"./data" env:"DATA_DIR"`
	JWTSecret string `cli:"jwt-secret,s" help:"JWT signing secret" env:"JWT_SECRET"`
	cli.Base
}

func (c *ServeCmd) Run() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("--jwt-secret is required (or set JWT_SECRET)")
	}

	dataDir := c.DataDir
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}

	s, err := store.Open(dataDir)
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	defer s.Close()

	ctx := context.Background()
	if err := bootstrap(ctx, s, c.JWTSecret); err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}

	cfg, _ := s.GetConfig(ctx)

	var backend bucket.Backend
	switch cfg.Storage.Backend {
	case "s3":
		backend, err = bucket.NewS3Backend(ctx, cfg.Storage.S3)
		if err != nil {
			return fmt.Errorf("s3 backend: %w", err)
		}
	default:
		local := cfg.Storage.Local
		if local.BucketRoot == "" {
			local.BucketRoot = dataDir + "/bucket"
		}
		if local.DedupRoot == "" {
			local.DedupRoot = dataDir + "/dedup"
		}
		backend, err = bucket.NewLocalBackend(local)
		if err != nil {
			return fmt.Errorf("local backend: %w", err)
		}
	}

	var mailer *email.Mailer
	if cfg.SMTP.Host != "" {
		mailer = email.New(cfg.SMTP)
	}

	rl := middleware.NewRateLimiter(cfg.RateLimit)
	bq := queue.New(s, backend, mailer)
	srv := api.NewServer(s, bq, backend, mailer, c.JWTSecret, dataDir)

	addr := fmt.Sprintf(":%d", c.Port)
	c.Logger.Success("next-deps-srv listening on %s", addr)
	return http.ListenAndServe(addr, srv.Handler(rl))
}

func bootstrap(ctx context.Context, s *store.Store, secret string) error {
	users, err := s.ListUsers(ctx)
	if err != nil || len(users) > 0 {
		return err
	}

	hash, err := auth.HashPassword("admin")
	if err != nil {
		return err
	}
	admin := models.User{
		ID:           uuid.NewString(),
		Username:     "admin",
		Email:        "admin@localhost",
		PasswordHash: hash,
		Roles:        []string{"admin"},
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if _, err := s.CreateUser(ctx, admin); err != nil {
		return err
	}

	token, _ := auth.IssueToken(admin, secret)
	log.Printf("[bootstrap] Created default admin user. Token: %s", token)
	return nil
}

func main() {
	app := &CLI{}
	if err := cli.Run(app); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
