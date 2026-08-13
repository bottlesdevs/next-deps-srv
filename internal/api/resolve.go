package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/bottlesdevs/next-deps-srv/internal/models"
)

type resolveSelector struct {
	ID      string       `json:"id,omitempty"`
	Name    string       `json:"name,omitempty"`
	Slot    *models.Slot `json:"slot,omitempty"`
	Version string       `json:"version,omitempty"`
}

type resolveRequest struct {
	Components   []resolveSelector `json:"components,omitempty"`
	Dependencies []resolveSelector `json:"dependencies,omitempty"`
	Platform     *models.Target    `json:"platform,omitempty"`
}

type resolvedEntry struct {
	Kind models.Kind `json:"kind"`
	models.CatalogEntry
}

type resolveResponse struct {
	SchemaVersion int             `json:"schema_version"`
	Platform      *models.Target  `json:"platform,omitempty"`
	Entries       []resolvedEntry `json:"entries"`
}

type downloadSource struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Version string      `json:"version"`
	Kind    models.Kind `json:"kind"`
}

type downloadItem struct {
	Sources  []downloadSource  `json:"sources"`
	URL      string            `json:"url"`
	FileName string            `json:"file_name"`
	Checksum models.Checksum   `json:"checksum"`
	Platform *models.Target    `json:"platform,omitempty"`
	Steps    []json.RawMessage `json:"steps,omitempty"`
}

type downloadBatchResponse struct {
	SchemaVersion int            `json:"schema_version"`
	Platform      *models.Target `json:"platform,omitempty"`
	Downloads     []downloadItem `json:"downloads"`
}

type catalogResolver struct {
	deps []models.Dependency
}

func (srv *Server) resolve(w http.ResponseWriter, r *http.Request) {
	req, err := readResolveRequest(w, r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	resolved, err := srv.resolveCatalog(r, req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resolved)
}

func (srv *Server) downloadBatch(w http.ResponseWriter, r *http.Request) {
	req, err := readResolveRequest(w, r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	resolved, err := srv.resolveCatalog(r, req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	out := downloadBatchResponse{
		SchemaVersion: models.SchemaVersion,
		Platform:      resolved.Platform,
		Downloads:     []downloadItem{},
	}
	seen := make(map[string]int)
	for _, entry := range resolved.Entries {
		source := downloadSource{
			ID: entry.ID, Name: entry.Name, Version: entry.Version, Kind: entry.Kind,
		}
		for _, artifact := range entry.Artifacts {
			raw, err := json.Marshal(artifact)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid stored artifact"})
				return
			}
			key := string(raw)
			if n, ok := seen[key]; ok {
				out.Downloads[n].Sources = append(out.Downloads[n].Sources, source)
				continue
			}
			seen[key] = len(out.Downloads)
			out.Downloads = append(out.Downloads, downloadItem{
				Sources: []downloadSource{source}, URL: artifact.URL,
				FileName: artifact.FileName, Checksum: artifact.Checksum,
				Platform: artifact.Platform, Steps: artifact.Steps,
			})
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func readResolveRequest(w http.ResponseWriter, r *http.Request) (resolveRequest, error) {
	var req resolveRequest
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return req, fmt.Errorf("invalid body")
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return req, fmt.Errorf("invalid body")
	}
	if len(req.Components)+len(req.Dependencies) == 0 {
		return req, fmt.Errorf("at least one component or dependency is required")
	}
	if len(req.Components)+len(req.Dependencies) > 100 {
		return req, fmt.Errorf("too many requested entries")
	}
	if req.Platform != nil {
		if err := req.Platform.Validate(); err != nil {
			return req, fmt.Errorf("platform: %w", err)
		}
	}
	for i := range req.Components {
		req.Components[i].normalize()
		if err := req.Components[i].validate(models.KindComponent); err != nil {
			return req, fmt.Errorf("component: %w", err)
		}
	}
	for i := range req.Dependencies {
		req.Dependencies[i].normalize()
		if err := req.Dependencies[i].validate(models.KindDependency); err != nil {
			return req, fmt.Errorf("dependency: %w", err)
		}
	}
	return req, nil
}

func (s *resolveSelector) normalize() {
	s.ID = strings.TrimSpace(s.ID)
	s.Name = strings.TrimSpace(s.Name)
	s.Version = strings.TrimSpace(s.Version)
}

func (s resolveSelector) validate(kind models.Kind) error {
	if s.Slot != nil && kind == models.KindDependency {
		return fmt.Errorf("slot is only valid for components")
	}
	requirement := models.Requirement{ID: s.ID, Name: s.Name, Slot: s.Slot}
	if err := requirement.Validate(); err != nil {
		return err
	}
	return nil
}

func (srv *Server) resolveCatalog(r *http.Request, req resolveRequest) (resolveResponse, error) {
	deps, err := srv.store.ListApprovedDeps(r.Context())
	if err != nil {
		return resolveResponse{}, fmt.Errorf("store error")
	}
	resolver := newCatalogResolver(deps)

	roots := make([]models.Dependency, 0, len(req.Components)+len(req.Dependencies))
	for _, selector := range req.Components {
		dep, ok := resolver.selectEntry(models.KindComponent, selector)
		if !ok {
			return resolveResponse{}, fmt.Errorf("component not found: %s", selector.label())
		}
		roots = append(roots, dep)
	}
	for _, selector := range req.Dependencies {
		dep, ok := resolver.selectEntry(models.KindDependency, selector)
		if !ok {
			return resolveResponse{}, fmt.Errorf("dependency not found: %s", selector.label())
		}
		roots = append(roots, dep)
	}

	out := resolveResponse{
		SchemaVersion: models.SchemaVersion,
		Platform:      req.Platform,
		Entries:       []resolvedEntry{},
	}
	state := make(map[string]uint8)
	var visit func(models.Dependency) error
	visit = func(dep models.Dependency) error {
		switch state[dep.ID] {
		case 1:
			return fmt.Errorf("requirement cycle at %s %s", dep.Entry.Name, dep.Entry.Version)
		case 2:
			return nil
		}
		state[dep.ID] = 1
		for _, requirement := range dep.Entry.Requirements {
			required, ok := resolver.selectRequirement(requirement)
			if !ok {
				return fmt.Errorf("requirement not found for %s: %s", dep.Entry.Name, requirementLabel(requirement))
			}
			if err := visit(required); err != nil {
				return err
			}
		}

		entry, err := entryForPlatform(dep.Entry, req.Platform)
		if err != nil {
			return err
		}
		out.Entries = append(out.Entries, resolvedEntry{Kind: dep.Kind, CatalogEntry: entry})
		state[dep.ID] = 2
		return nil
	}
	for _, root := range roots {
		if err := visit(root); err != nil {
			return resolveResponse{}, err
		}
	}
	return out, nil
}

func newCatalogResolver(deps []models.Dependency) catalogResolver {
	sort.Slice(deps, func(i, j int) bool {
		if deps[i].UpdatedAt.Equal(deps[j].UpdatedAt) {
			return deps[i].ID < deps[j].ID
		}
		return deps[i].UpdatedAt.After(deps[j].UpdatedAt)
	})
	canonical := make([]models.Dependency, 0, len(deps))
	seen := make(map[string]struct{}, len(deps))
	for _, dep := range deps {
		key := string(dep.Kind) + "\x00" + dep.Entry.Name + "\x00" + dep.Entry.Version
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		canonical = append(canonical, dep)
	}
	return catalogResolver{deps: canonical}
}

func (r catalogResolver) selectEntry(kind models.Kind, selector resolveSelector) (models.Dependency, bool) {
	for _, dep := range r.deps {
		if dep.Kind != kind || selector.Version != "" && dep.Entry.Version != selector.Version {
			continue
		}
		switch {
		case selector.ID != "" && dep.Entry.ID == selector.ID:
			return dep, true
		case selector.Name != "" && dep.Entry.Name == selector.Name:
			return dep, true
		case selector.Slot != nil && dep.Entry.Slot != nil && *dep.Entry.Slot == *selector.Slot:
			return dep, true
		}
	}
	return models.Dependency{}, false
}

func (r catalogResolver) selectRequirement(requirement models.Requirement) (models.Dependency, bool) {
	for _, dep := range r.deps {
		switch {
		case requirement.ID != "" && dep.Entry.ID == requirement.ID:
			return dep, true
		case requirement.Name != "" && dep.Entry.Name == requirement.Name:
			return dep, true
		case requirement.Slot != nil && dep.Kind == models.KindComponent && dep.Entry.Slot != nil && *dep.Entry.Slot == *requirement.Slot:
			return dep, true
		}
	}
	return models.Dependency{}, false
}

func entryForPlatform(entry models.CatalogEntry, target *models.Target) (models.CatalogEntry, error) {
	if target == nil {
		return entry.NormalizeForCatalog(), nil
	}
	artifacts := make([]models.CatalogArtifact, 0, len(entry.Artifacts))
	for _, artifact := range entry.Artifacts {
		if artifact.Platform == nil || artifact.Platform.OS == target.OS && artifact.Platform.Arch == target.Arch {
			artifacts = append(artifacts, artifact)
		}
	}
	if len(artifacts) == 0 {
		return models.CatalogEntry{}, fmt.Errorf("no artifacts for %s %s on %s", entry.Name, entry.Version, target.String())
	}
	entry.Artifacts = artifacts
	return entry, nil
}

func (s resolveSelector) label() string {
	switch {
	case s.ID != "":
		return "id " + s.ID
	case s.Name != "":
		if s.Version != "" {
			return s.Name + " " + s.Version
		}
		return "name " + s.Name
	case s.Slot != nil:
		if s.Version != "" {
			return "slot " + string(*s.Slot) + " " + s.Version
		}
		return "slot " + string(*s.Slot)
	default:
		return "invalid selector"
	}
}

func requirementLabel(requirement models.Requirement) string {
	switch {
	case requirement.ID != "":
		return "id " + requirement.ID
	case requirement.Name != "":
		return "name " + requirement.Name
	case requirement.Slot != nil:
		return "slot " + string(*requirement.Slot)
	default:
		return "invalid requirement"
	}
}
