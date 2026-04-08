package bucket

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// archiveKind identifies the container format by magic bytes.
type archiveKind int

const (
	kindUnknown archiveKind = iota
	kindZip
	kindCab
	kind7z
	kindTarGz
	kindTarBz2
	kindTarXz
	kindMSI  // OLE compound (also used for .msi)
	kindRar
)

var magicTable = []struct {
	magic []byte
	kind  archiveKind
}{
	{[]byte{0x50, 0x4B, 0x03, 0x04}, kindZip},         // ZIP
	{[]byte{0x4D, 0x53, 0x43, 0x46}, kindCab},         // MSCF – Cabinet
	{[]byte{0x49, 0x53, 0x63, 0x28}, kindCab},         // ISc( – older InstallShield CAB
	{[]byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}, kind7z}, // 7-Zip
	{[]byte{0x1F, 0x8B}, kindTarGz},                    // gzip
	{[]byte{0x42, 0x5A, 0x68}, kindTarBz2},             // bzip2
	{[]byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00}, kindTarXz}, // xz
	{[]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, kindMSI}, // OLE/MSI
	{[]byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07}, kindRar}, // RAR
}

func detectKind(path string) (archiveKind, error) {
	f, err := os.Open(path)
	if err != nil {
		return kindUnknown, err
	}
	defer f.Close()
	buf := make([]byte, 8)
	n, _ := f.Read(buf)
	buf = buf[:n]
	for _, m := range magicTable {
		if len(buf) >= len(m.magic) {
			ok := true
			for i, b := range m.magic {
				if buf[i] != b {
					ok = false
					break
				}
			}
			if ok {
				return m.kind, nil
			}
		}
	}
	// Check MZ (PE/EXE/DLL) - treat as "try 7z"
	if len(buf) >= 2 && buf[0] == 0x4D && buf[1] == 0x5A {
		return kind7z, nil
	}
	return kindUnknown, nil
}

// kindName returns a human-readable label for logging.
func kindName(k archiveKind) string {
	switch k {
	case kindZip:
		return "ZIP"
	case kindCab:
		return "CAB"
	case kind7z:
		return "7z/PE/SFX"
	case kindTarGz:
		return "tar.gz"
	case kindTarBz2:
		return "tar.bz2"
	case kindTarXz:
		return "tar.xz"
	case kindMSI:
		return "MSI/OLE"
	case kindRar:
		return "RAR"
	default:
		return "unknown"
	}
}

// Extract unpacks src into destDir, detecting the format from magic bytes.
// logf receives verbose progress lines; pass nil to suppress.
func Extract(src, destDir string, logf func(string, ...any)) error {
	if logf == nil {
		logf = func(string, ...any) {}
	}

	kind, err := detectKind(src)
	if err != nil {
		return fmt.Errorf("detect format: %w", err)
	}
	logf("🔍 Detected format: %s", kindName(kind))

	switch kind {
	case kindZip:
		return extractZip(src, destDir, logf)
	case kindTarGz:
		return extractTarGz(src, destDir, logf)
	case kindTarBz2:
		return extractTarBz2(src, destDir, logf)
	case kindTarXz:
		return extractCmd(logf, destDir, "tar", "-xJf", src, "-C", destDir)
	case kindCab:
		return extractCab(src, destDir, logf)
	case kindMSI, kind7z, kindRar:
		return extract7z(src, destDir, logf)
	default:
		logf("⚠️  Unrecognised format - storing file as-is")
		return copyFile(src, filepath.Join(destDir, filepath.Base(src)))
	}
}

// isArchiveKind returns true for kinds that should be extracted recursively.
func isArchiveKind(k archiveKind) bool {
	return k != kindUnknown
}

// ExtractAll extracts src into destDir and then recursively extracts any
// nested archives found in the output, up to maxDepth levels deep.
// After each nested extraction the container archive is removed so that only
// leaf (non-archive) files remain in destDir when the function returns.
func ExtractAll(src, destDir string, maxDepth int, logf func(string, ...any)) error {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	if err := Extract(src, destDir, logf); err != nil {
		return err
	}
	return expandArchivesInDir(destDir, maxDepth, 1, logf)
}

// expandArchivesInDir walks dir and extracts any archive files it finds,
// replacing each archive with its contents, up to maxDepth recursion.
func expandArchivesInDir(dir string, maxDepth, depth int, logf func(string, ...any)) error {
	if depth > maxDepth {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.IsDir() {
			// recurse into already-existing subdirs
			if err := expandArchivesInDir(filepath.Join(dir, e.Name()), maxDepth, depth+1, logf); err != nil {
				logf("⚠️  recurse %s: %v", e.Name(), err)
			}
			continue
		}

		path := filepath.Join(dir, e.Name())
		kind, err := detectKind(path)
		if err != nil || !isArchiveKind(kind) {
			continue
		}

		// create a sibling subdir named after the archive (without extension)
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		subDir := filepath.Join(dir, base)
		// avoid collision if a plain file has the same stem as a directory
		if _, err := os.Stat(subDir); err == nil {
			subDir = subDir + "_extracted"
		}
		if err := os.MkdirAll(subDir, 0755); err != nil {
			logf("⚠️  mkdir %s: %v", subDir, err)
			continue
		}

		logf("📂 [depth %d] Expanding %s (%s) → %s/", depth, e.Name(), kindName(kind), base)
		if err := Extract(path, subDir, logf); err != nil {
			logf("⚠️  Extract %s: %v", e.Name(), err)
			// remove empty subdir; keep original archive intact
			os.Remove(subDir)
			continue
		}

		// remove the container archive - its contents are now in subDir
		os.Remove(path)

		// recurse into the newly created subdir
		if err := expandArchivesInDir(subDir, maxDepth, depth+1, logf); err != nil {
			logf("⚠️  recurse %s: %v", base, err)
		}
	}
	return nil
}

// --- Zip ---

func extractZip(src, destDir string, logf func(string, ...any)) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		logf("  ↳ %s", f.Name)
		if err := extractZipFile(f, destDir); err != nil {
			return err
		}
	}
	return nil
}

func extractZipFile(f *zip.File, destDir string) error {
	path := filepath.Join(destDir, filepath.Clean(f.Name))
	if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("zip slip: %s", f.Name)
	}
	if f.FileInfo().IsDir() {
		return os.MkdirAll(path, 0755)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}

// --- Tar ---

func extractTarGz(src, destDir string, logf func(string, ...any)) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	return extractTar(tar.NewReader(gz), destDir, logf)
}

func extractTarBz2(src, destDir string, logf func(string, ...any)) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	return extractTar(tar.NewReader(bzip2.NewReader(f)), destDir, logf)
}

func extractTar(tr *tar.Reader, destDir string, logf func(string, ...any)) error {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		path := filepath.Join(destDir, filepath.Clean(hdr.Name))
		if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("tar slip: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			logf("  ↳ %s", hdr.Name)
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			out, err := os.Create(path)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

// --- CAB via cabextract ---

func extractCab(src, destDir string, logf func(string, ...any)) error {
	bin, err := exec.LookPath("cabextract")
	if err != nil {
		logf("⚠️  cabextract not found, falling back to 7z")
		return extract7z(src, destDir, logf)
	}
	logf("  ↳ running cabextract")
	return extractCmd(logf, destDir, bin, "-d", destDir, src)
}

// --- 7z (MSI, EXE/SFX, 7z, RAR) ---

func extract7z(src, destDir string, logf func(string, ...any)) error {
	bin, err := exec.LookPath("7z")
	if err != nil {
		return fmt.Errorf("7z not found: %w", err)
	}
	logf("  ↳ running 7z x")
	return extractCmd(logf, destDir, bin, "x", "-y", "-o"+destDir, src)
}

// extractCmd runs an external command and streams its stdout/stderr to logf.
func extractCmd(logf func(string, ...any), _ string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			logf("    [%s] %s", filepath.Base(name), line)
		}
	}
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(name), err)
	}
	return nil
}

// --- helpers ---

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
