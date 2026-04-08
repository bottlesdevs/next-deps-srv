package models

import "time"

type AuditEntry struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address"`
	CreatedAt time.Time `json:"created_at"`
}

type AppConfig struct {
	SMTP             SMTPConfig      `json:"smtp"`
	RateLimit        RateLimitConfig `json:"rate_limit"`
	Storage          StorageConfig   `json:"storage"`
	RegistrationOpen bool            `json:"registration_open"`
}

type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	UseTLS   bool   `json:"use_tls"`
}

type RateLimitConfig struct {
	Enabled           bool `json:"enabled"`
	RequestsPerMinute int  `json:"requests_per_minute"`
	BurstSize         int  `json:"burst_size"`
}

type StorageConfig struct {
	Backend string             `json:"backend"` // "local" or "s3"
	Local   LocalStorageConfig `json:"local"`
	S3      S3StorageConfig    `json:"s3"`
}

type LocalStorageConfig struct {
	BucketRoot string `json:"bucket_root"`
	DedupRoot  string `json:"dedup_root"`
}

type S3StorageConfig struct {
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
	Endpoint  string `json:"endpoint,omitempty"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Prefix    string `json:"prefix"`
}
