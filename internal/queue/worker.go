package queue

import (
	"context"
	"fmt"
	"io"
	"log"
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

	if len(dep.Entry.Artifacts) == 0 {
		fail(fmt.Errorf("entry %s has no artifacts to build", dep.Entry.ID))
		return
	}

	tmp, err := os.MkdirTemp("", "ndeps-job-*")
	if err != nil {
		fail(err)
		return
	}
	defer os.RemoveAll(tmp)

	log("🧩 %s %s (%s) — %d artifact(s)", dep.Entry.Name, dep.Entry.Version, dep.Kind, len(dep.Entry.Artifacts))

	total := 0
	for n, art := range dep.Entry.Artifacts {
		label := art.FileName
		if p := art.Platform.String(); p != "" {
			label = fmt.Sprintf("%s (%s)", art.FileName, p)
		}
		log("── artifact %d/%d: %s", n+1, len(dep.Entry.Artifacts), label)

		count, err := buildArtifact(ctx, filepath.Join(tmp, fmt.Sprintf("artifact-%d", n)), art, job, dep, s, backend, log)
		if err != nil {
			fail(fmt.Errorf("artifact %s: %w", art.FileName, err))
			return
		}
		total += count
	}

	log("✅ Indexed %d file(s) across %d artifact(s)", total, len(dep.Entry.Artifacts))
	job.FilesIndexed = total
	job.Status = "done"
	job.FinishedAt = time.Now()
	_ = s.UpdateJob(ctx, job)

	dep.Status = "built"
	_ = s.UpdateDep(ctx, dep)

	notifyBuildResult(ctx, job, dep, s, mailer)
}

// buildArtifact downloads one artifact, verifies it against its declared
// checksum, extracts it, and indexes the result. Returns the number of files
// indexed.
func buildArtifact(ctx context.Context, workDir string, art models.CatalogArtifact, job models.BuildJob, dep models.Dependency, s *store.Store, backend bucket.Backend, log func(string, ...any)) (int, error) {
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return 0, err
	}

	log("⬇️  Downloading %s", art.URL)
	archivePath := filepath.Join(workDir, "archive")
	if err := downloadFile(art.URL, archivePath); err != nil {
		return 0, err
	}
	log("✅ Download complete")

	// The schema makes the checksum mandatory, so this always runs.
	log("🔍 Verifying %s checksum...", art.Checksum.Algorithm)
	sum, err := bucket.HashFileWith(archivePath, art.Checksum.Algorithm)
	if err != nil {
		return 0, err
	}
	if !bucket.ChecksumEqual(sum, art.Checksum.Value) {
		return 0, fmt.Errorf("checksum mismatch: got %s want %s", sum, art.Checksum.Value)
	}
	log("✅ Checksum OK: %s", sum)

	// The archive's own content hash, used as the provenance marker on every
	// revision extracted from it. Independent of the declared checksum.
	archiveHash, err := bucket.FileHash(archivePath)
	if err != nil {
		return 0, err
	}

	// Index the archive itself as a bucket file.
	log("📁 Indexing source archive...")
	archiveFilename := art.FileName
	if archiveFilename == "" {
		archiveFilename = path.Base(art.URL)
	}
	if archiveFilename == "" || archiveFilename == "." || archiveFilename == "/" {
		archiveFilename = "archive_" + job.ID
	}
	if n, err := indexOneFile(ctx, archivePath, archiveFilename, job, dep, art, archiveHash, s, backend); err != nil {
		log("⚠️  Could not index archive: %v", err)
	} else {
		log("  📦 %s [%s]", archiveFilename, n)
	}

	extractDir := filepath.Join(workDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return 0, err
	}
	log("📦 Extracting archive (recursive)...")
	if err := bucket.ExtractAll(archivePath, extractDir, 4, log); err != nil {
		return 0, err
	}
	log("✅ Extraction complete")

	log("🗂️  Indexing files...")
	return indexFiles(ctx, extractDir, job, dep, art, archiveHash, s, backend, log)
}

func indexFiles(ctx context.Context, dir string, job models.BuildJob, dep models.Dependency, art models.CatalogArtifact, archiveHash string, s *store.Store, backend bucket.Backend, log func(string, ...any)) (int, error) {
	count := 0
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		action, err := indexOneFile(ctx, p, info.Name(), job, dep, art, archiveHash, s, backend)
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
func indexOneFile(ctx context.Context, srcPath, filename string, job models.BuildJob, dep models.Dependency, art models.CatalogArtifact, archiveHash string, s *store.Store, backend bucket.Backend) (string, error) {
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
		ArchiveURL:  art.URL,
		ArchiveHash: archiveHash,
		Platform:    art.Platform.String(),
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
		if err := mailer.BuildDone(dep, job, all); err != nil {
			log.Printf("mail: %v", err)
		}
	} else {
		if err := mailer.BuildFailed(dep, job, all); err != nil {
			log.Printf("mail: %v", err)
		}
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
