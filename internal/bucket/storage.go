package bucket

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/mirkobrombin/dabadee/pkg/storage"
)

type Backend interface {
	Store(ctx context.Context, srcPath, filename, revisionID string) (storagePath string, err error)
	Stream(ctx context.Context, storagePath string, w io.Writer) error
	Size(ctx context.Context, storagePath string) (int64, error)
}

// LocalBackend stores files in bucket dirs with dabadee deduplication.
type LocalBackend struct {
	BucketRoot string
	dedup      *storage.Storage
}

func NewLocalBackend(cfg models.LocalStorageConfig) (*LocalBackend, error) {
	for _, c := range AllChars() {
		if err := os.MkdirAll(filepath.Join(cfg.BucketRoot, c), 0755); err != nil {
			return nil, err
		}
	}
	dedup, err := storage.NewStorage(storage.StorageOptions{
		Root:         cfg.DedupRoot,
		WithMetadata: true,
	})
	if err != nil {
		return nil, err
	}
	return &LocalBackend{BucketRoot: cfg.BucketRoot, dedup: dedup}, nil
}

func (b *LocalBackend) Store(_ context.Context, srcPath, filename, _ string) (string, error) {
	hash, err := fileHash(srcPath)
	if err != nil {
		return "", err
	}
	dest := BucketPath(b.BucketRoot, filename)
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return "", err
	}
	if err := copyFile(srcPath, dest); err != nil {
		return "", err
	}
	if err := b.dedup.MoveFileToStorage(dest, hash); err != nil {
		// dedup failure is non-fatal
		_ = err
	}
	return dest, nil
}

func (b *LocalBackend) Stream(_ context.Context, storagePath string, w io.Writer) error {
	f, err := os.Open(storagePath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

func (b *LocalBackend) Size(_ context.Context, storagePath string) (int64, error) {
	info, err := os.Stat(storagePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// S3Backend stores files in an S3-compatible bucket.
type S3Backend struct {
	client *s3.Client
	bucket string
	prefix string
}

func NewS3Backend(ctx context.Context, cfg models.S3StorageConfig) (*S3Backend, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}
	var clientOpts []func(*s3.Options)
	if cfg.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		})
	}
	return &S3Backend{
		client: s3.NewFromConfig(awsCfg, clientOpts...),
		bucket: cfg.Bucket,
		prefix: cfg.Prefix,
	}, nil
}

func (b *S3Backend) key(filename, revisionID string) string {
	c := Char(filename)
	k := fmt.Sprintf("%s/%s/%s/%s", b.prefix, c, filename, revisionID)
	if b.prefix == "" {
		k = fmt.Sprintf("%s/%s/%s", c, filename, revisionID)
	}
	return k
}

func (b *S3Backend) Store(ctx context.Context, srcPath, filename, revisionID string) (string, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, _ := f.Stat()
	key := b.key(filename, revisionID)
	_, err = b.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(b.bucket),
		Key:           aws.String(key),
		Body:          f,
		ContentLength: aws.Int64(info.Size()),
	})
	if err != nil {
		return "", err
	}
	return key, nil
}

func (b *S3Backend) Stream(ctx context.Context, storagePath string, w io.Writer) error {
	out, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(storagePath),
	})
	if err != nil {
		return err
	}
	defer out.Body.Close()
	_, err = io.Copy(w, out.Body)
	return err
}

func (b *S3Backend) Size(ctx context.Context, storagePath string) (int64, error) {
	out, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(storagePath),
	})
	if err != nil {
		return 0, err
	}
	if out.ContentLength != nil {
		return *out.ContentLength, nil
	}
	return 0, nil
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func FileHash(path string) (string, error) { return fileHash(path) }
