package models

import (
	"fmt"
	"strings"
)

// SupportedChecksumAlgorithms lists the algorithms an artifact checksum may
// declare. Keep in sync with bucket.HashFileWith.
var SupportedChecksumAlgorithms = []string{"md5", "sha1", "sha256", "sha512"}

// Validate reports whether the item satisfies the catalog schema's
// constraints: the required fields, their minimum lengths, and the
// uniqueness rules the published document depends on.
func (i Item) Validate() error {
	if strings.TrimSpace(i.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(i.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(i.Version) == "" {
		return fmt.Errorf("version is required")
	}
	if i.Kind != nil {
		if strings.TrimSpace(i.Kind.Type) == "" || strings.TrimSpace(i.Kind.Flavour) == "" {
			return fmt.Errorf("kind requires both type and flavour")
		}
	}
	if len(i.Artifacts) == 0 {
		return fmt.Errorf("at least one artifact is required")
	}
	seen := make(map[string]struct{}, len(i.Artifacts))
	for n, a := range i.Artifacts {
		if err := a.Validate(); err != nil {
			return fmt.Errorf("artifact %d: %w", n, err)
		}
		key := a.URL + "\x00" + a.FileName + "\x00" + a.Platform.String()
		if _, dup := seen[key]; dup {
			return fmt.Errorf("artifact %d: duplicate artifact %s", n, a.FileName)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func (a Artifact) Validate() error {
	if strings.TrimSpace(a.URL) == "" {
		return fmt.Errorf("url is required")
	}
	if strings.TrimSpace(a.FileName) == "" {
		return fmt.Errorf("file_name is required")
	}
	if strings.TrimSpace(a.ComponentRoot) == "" {
		return fmt.Errorf("component_root is required")
	}
	if a.Size < 0 {
		return fmt.Errorf("size must not be negative")
	}
	if a.Checksum != nil {
		if strings.TrimSpace(a.Checksum.Value) == "" {
			return fmt.Errorf("checksum value is required")
		}
		algo := strings.ToLower(strings.TrimSpace(a.Checksum.Algorithm))
		if algo == "" {
			return fmt.Errorf("checksum algorithm is required")
		}
		supported := false
		for _, s := range SupportedChecksumAlgorithms {
			if algo == s {
				supported = true
				break
			}
		}
		if !supported {
			return fmt.Errorf("unsupported checksum algorithm %q (want one of %s)",
				a.Checksum.Algorithm, strings.Join(SupportedChecksumAlgorithms, ", "))
		}
	}
	if a.Platform != nil {
		if strings.TrimSpace(a.Platform.OS) == "" || strings.TrimSpace(a.Platform.Arch) == "" {
			return fmt.Errorf("platform requires both os and arch")
		}
	}
	return nil
}
