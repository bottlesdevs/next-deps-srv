package bucket

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
)

// HashFileWith hashes a file with the named algorithm and returns the digest
// as lowercase hex. Supported algorithms match
// models.SupportedChecksumAlgorithms.
func HashFileWith(path, algorithm string) (string, error) {
	var h hash.Hash
	switch strings.ToLower(strings.TrimSpace(algorithm)) {
	case "md5":
		h = md5.New()
	case "sha1":
		h = sha1.New()
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	default:
		return "", fmt.Errorf("unsupported checksum algorithm %q", algorithm)
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ChecksumEqual compares two digests case-insensitively, as hex digests are
// commonly published in either case.
func ChecksumEqual(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}
