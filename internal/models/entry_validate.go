package models

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func validKind(k Kind) bool {
	for _, v := range Kinds {
		if k == v {
			return true
		}
	}
	return false
}

func validSlot(s Slot) bool {
	for _, v := range Slots {
		if s == v {
			return true
		}
	}
	return false
}

func slotNames() string {
	names := make([]string, len(Slots))
	for i, s := range Slots {
		names[i] = string(s)
	}
	return strings.Join(names, ", ")
}

// Validate reports whether the entry satisfies the catalog schema for the
// given kind. Component entries must carry a slot; dependency entries must
// not.
//
// Beyond the schema, the server requires at least one artifact: an entry with
// none has nothing to build or serve.
func (e CatalogEntry) Validate(kind Kind) error {
	if !validKind(kind) {
		return fmt.Errorf("kind must be one of component, dependency")
	}
	if _, err := uuid.Parse(strings.TrimSpace(e.ID)); err != nil {
		return fmt.Errorf("id must be a uuid")
	}
	if strings.TrimSpace(e.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(e.Version) == "" {
		return fmt.Errorf("version is required")
	}

	switch kind {
	case KindComponent:
		if e.Slot == nil {
			return fmt.Errorf("slot is required for components (one of %s)", slotNames())
		}
		if !validSlot(*e.Slot) {
			return fmt.Errorf("unknown slot %q (want one of %s)", *e.Slot, slotNames())
		}
	case KindDependency:
		if e.Slot != nil {
			return fmt.Errorf("slot is only valid for components")
		}
	}

	if len(e.Artifacts) == 0 {
		return fmt.Errorf("at least one artifact is required")
	}
	seen := make(map[string]struct{}, len(e.Artifacts))
	for n, a := range e.Artifacts {
		if err := a.Validate(); err != nil {
			return fmt.Errorf("artifact %d: %w", n, err)
		}
		key := a.URL + "\x00" + a.FileName + "\x00" + a.Platform.String()
		if _, dup := seen[key]; dup {
			return fmt.Errorf("artifact %d: duplicate artifact %s", n, a.FileName)
		}
		seen[key] = struct{}{}
	}

	for n, r := range e.Requirements {
		if err := r.Validate(); err != nil {
			return fmt.Errorf("requirement %d: %w", n, err)
		}
	}
	return nil
}

func (a CatalogArtifact) Validate() error {
	if strings.TrimSpace(a.URL) == "" {
		return fmt.Errorf("url is required")
	}
	if strings.TrimSpace(a.FileName) == "" {
		return fmt.Errorf("file_name is required")
	}
	if err := a.Checksum.Validate(); err != nil {
		return fmt.Errorf("checksum: %w", err)
	}
	if a.Platform != nil {
		if err := a.Platform.Validate(); err != nil {
			return fmt.Errorf("platform: %w", err)
		}
	}
	return nil
}

func (c Checksum) Validate() error {
	algo := strings.TrimSpace(c.Algorithm)
	supported := false
	for _, s := range ChecksumAlgorithms {
		if algo == s {
			supported = true
			break
		}
	}
	if !supported {
		return fmt.Errorf("algorithm must be one of %s", strings.Join(ChecksumAlgorithms, ", "))
	}
	if strings.TrimSpace(c.Value) == "" {
		return fmt.Errorf("value is required")
	}
	// The schema stores values without validating encoding, but verification
	// compares case-sensitively against a lowercase hex digest, so anything
	// else could never match. Reject it at intake rather than at build time.
	want := map[string]int{"sha256": 64, "sha512": 128}[algo]
	if len(c.Value) != want || !isLowerHex(c.Value) {
		return fmt.Errorf("value must be %d lowercase hex characters for %s", want, algo)
	}
	return nil
}

func isLowerHex(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func (t Target) Validate() error {
	okOS := false
	for _, v := range OperatingSystems {
		if t.OS == v {
			okOS = true
			break
		}
	}
	if !okOS {
		return fmt.Errorf("unknown os %q", t.OS)
	}
	okArch := false
	for _, v := range Architectures {
		if t.Arch == v {
			okArch = true
			break
		}
	}
	if !okArch {
		return fmt.Errorf("unknown arch %q", t.Arch)
	}
	return nil
}

// Validate enforces the oneOf: exactly one of name, slot or id is set.
func (r Requirement) Validate() error {
	set := 0
	if strings.TrimSpace(r.Name) != "" {
		set++
	}
	if r.Slot != nil {
		set++
		if !validSlot(*r.Slot) {
			return fmt.Errorf("unknown slot %q (want one of %s)", *r.Slot, slotNames())
		}
	}
	if strings.TrimSpace(r.ID) != "" {
		set++
		if _, err := uuid.Parse(strings.TrimSpace(r.ID)); err != nil {
			return fmt.Errorf("id must be a uuid")
		}
	}
	if set != 1 {
		return fmt.Errorf("exactly one of name, slot or id must be set")
	}
	return nil
}
