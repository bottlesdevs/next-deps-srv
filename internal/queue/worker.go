package queue

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/bucket"
	"github.com/bottlesdevs/next-deps-srv/internal/email"
	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/bottlesdevs/next-deps-srv/internal/store"
	"github.com/google/uuid"
)

func runJob(ctx context.Context, job models.BuildJob, dep models.Dependency, s *store.Store, backend bucket.Backend, mailer *email.Mailer, h *logHub) {
	log := func(format string, args ...any) {
		line := fmt.Sprintf(format, args...)
		h.emit(line)
		job.Logs = append(job.Logs, line)
		_ = s.UpdateJob(ctx, job)
	}

	fail := func(err error) {
		log("❌ %s", err.Error())
		job.Status = "failed"
		job.Error = err.Error()
		job.FinishedAt = time.Now()
		_ = s.UpdateJob(ctx, job)
		_ = s.UpdateDep(ctx, func() models.Dependency { dep.Status = "approved"; return dep }())
		notifyBuildResult(ctx, job, dep, s, mailer)
	}

	log("⬇️  Downloading %s", dep.Manifest.URL)
	tmp, err := os.MkdirTemp("", "ndeps-job-*")
	if err != nil {
		fail(err)
		return
	}
	defer os.RemoveAll(tmp)

	archivePath := filepath.Join(tmp, "archive")
	if err := downloadFile(dep.Manifest.URL, archivePath); err != nil {
		fail(err)
		return
	}
	log("✅ Download complete")

	log("🔍 Verifying MD5 hash...")
	hash, err := bucket.FileHash(archivePath)
	if err != nil {
		fail(err)
		return
	}
	if hash != dep.Manifest.ExpectedHash {
		fail(fmt.Errorf("hash mismatch: got %s want %s", hash, dep.Manifest.ExpectedHash))
		return
	}
	log("✅ Hash OK: %s", hash)

	// Index the archive itself as a bucket file.
	log("📁 Indexing source archive...")
	archiveFilename := path.Base(dep.Manifest.URL)
	if archiveFilename == "" || archiveFilename == "." {
		archiveFilename = "archive_" + job.ID
	}
	if n, err := indexOneFile(ctx, archivePath, archiveFilename, job, dep, hash, s, backend); err != nil {
		log("⚠️  Could not index archive: %v", err)
	} else {
		log("  📦 %s [%s]", archiveFilename, n)
	}

	extractDir := filepath.Join(tmp, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		fail(err)
		return
	}
	log("📦 Extracting archive (recursive)...")
	if err := bucket.ExtractAll(archivePath, extractDir, 4, log); err != nil {
		fail(err)
		return
	}
	log("✅ Extraction complete")

	log("🗂️  Indexing files...")
	count, err := indexFiles(ctx, extractDir, job, dep, hash, s, backend, log)
	if err != nil {
		fail(err)
		return
	}

	log("✅ Indexed %d file(s)", count)
	job.FilesIndexed = count
	job.Status = "done"
	job.FinishedAt = time.Now()
	_ = s.UpdateJob(ctx, job)

	dep.Status = "built"
	_ = s.UpdateDep(ctx, dep)

	notifyBuildResult(ctx, job, dep, s, mailer)
}

func indexFiles(ctx context.Context, dir string, job models.BuildJob, dep models.Dependency, archiveHash string, s *store.Store, backend bucket.Backend, log func(string, ...any)) (int, error) {
	count := 0
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		action, err := indexOneFile(ctx, p, info.Name(), job, dep, archiveHash, s, backend)
		if err != nil {
			return err
		}
		log("  📄 %s [%s]", info.Name(), action)
		count++
		return nil
	})
	return count, err
}

// indexOneFile stores a single file in the bucket and creates/updates its
// BucketFile + FileRevision records. Returns a human-readable action string
// ("new", "rev N", "skip – same hash") and any error.
func indexOneFile(ctx context.Context, srcPath, filename string, job models.BuildJob, dep models.Dependency, archiveHash string, s *store.Store, backend bucket.Backend) (string, error) {
	info, err := os.Stat(srcPath)
	if err != nil {
		return "", err
	}

	fileHash, err := bucket.FileHash(srcPath)
	if err != nil {
		return "", err
	}

	existing, err := s.GetFileByName(ctx, filename)
	found := err == nil

	var fileID string
	var revNum int

	if !found {
		newFile, err := s.CreateFile(ctx, models.BucketFile{
			Name:       filename,
			BucketChar: bucket.Char(filename),
		})
		if err != nil {
			return "", err
		}
		fileID = newFile.ID
		revNum = 1
	} else {
		fileID = existing.ID
		existingRevs, _ := s.RevisionsByFile(ctx, fileID)
		revNum = len(existingRevs) + 1

		for _, r := range existingRevs {
			if r.Hash == fileHash {
				return "skip – same hash", nil
			}
		}
	}

	revID := uuid.NewString()
	storagePath, err := backend.Store(ctx, srcPath, filename, revID)
	if err != nil {
		return "", err
	}

	rev, err := s.CreateRevision(ctx, models.FileRevision{
		ID:          revID,
		FileID:      fileID,
		RevisionNum: revNum,
		Hash:        fileHash,
		SourceJobID: job.ID,
		SourceDepID: dep.ID,
		ArchiveURL:  dep.Manifest.URL,
		ArchiveHash: archiveHash,
		StoragePath: storagePath,
		SizeBytes:   info.Size(),
	})
	if err != nil {
		return "", err
	}

	if found {
		existing.LatestRevID = rev.ID
		_ = s.UpdateFile(ctx, existing)
	} else {
		if newFile, err := s.GetFileByName(ctx, filename); err == nil {
			newFile.LatestRevID = rev.ID
			_ = s.UpdateFile(ctx, newFile)
		}
	}

	action := "new"
	if found {
		action = fmt.Sprintf("rev %d", revNum)
	}
	return action, nil
}

func notifyBuildResult(ctx context.Context, job models.BuildJob, dep models.Dependency, s *store.Store, mailer *email.Mailer) {
	if mailer == nil {
		return
	}
	mods, _ := s.AdminAndModUsers(ctx)
	var modEmails []string
	for _, u := range mods {
		modEmails = append(modEmails, u.Email)
	}
	submitter, err := s.GetUser(ctx, dep.SubmittedBy)
	all := modEmails
	if err == nil {
		all = append(all, submitter.Email)
	}
	if job.Status == "done" {
		_ = mailer.BuildDone(dep, job, all)
	} else {
		_ = mailer.BuildFailed(dep, job, all)
	}
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
