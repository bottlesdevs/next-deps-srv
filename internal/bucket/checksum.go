package bucket

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
)

// HashFileWith hashes a file with the named algorithm and returns the digest
// as lowercase hex. Only the algorithms the catalog schema admits are
// supported (see models.ChecksumAlgorithms).
func HashFileWith(path, algorithm string) (string, error) {
	var h hash.Hash
	switch algorithm {
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

// ChecksumEqual compares a computed digest with a declared one. The catalog
// schema specifies an exact, case-sensitive comparison against a lowercase
// hexadecimal digest, so this deliberately does not fold case.
func ChecksumEqual(computed, declared string) bool {
	return computed == declared
}
