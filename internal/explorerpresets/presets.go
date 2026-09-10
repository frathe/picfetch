// Package explorerpresets owns reusable local image rules and their library.
package explorerpresets

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/similarity"
)

// Rule combines all enabled conditions. Empty values leave a field unconstrained.
type Rule struct {
	Tags                                     []string
	Format, Orientation, Make, Model         string
	MinWidth, MaxWidth, MinHeight, MaxHeight int
	CaptureFrom, CaptureThrough              string
}

// Preset has a stable identity independent of its name or current members.
type Preset struct {
	ID, Name string
	Rule     Rule
	Versions Versions
}

type Versions struct {
	Rule, Facts                      int
	Model, Representation, Catalogue string
}

func CurrentVersions() Versions {
	return Versions{Rule: 1, Facts: similarity.FactsVersion, Model: similarity.ModelRevision,
		Representation: similarity.RepresentationVersion, Catalogue: similarity.TagCatalogueVersion()}
}

func (p Preset) Compatible() bool { return p.Versions == CurrentVersions() }

func ValidName(name string) bool {
	name = strings.TrimSpace(name)
	return name != "" && utf8.RuneCountInString(name) <= 80 && !strings.ContainsAny(name, "\r\n\t")
}

func (r Rule) Validate() error {
	if r.empty() {
		return errors.New("choose at least one rule")
	}
	for _, pair := range [][2]int{{r.MinWidth, r.MaxWidth}, {r.MinHeight, r.MaxHeight}} {
		if pair[0] < 0 || pair[1] < 0 || (pair[0] > 0 && pair[1] > 0 && pair[0] > pair[1]) {
			return errors.New("invalid dimension range")
		}
	}
	if r.Format != "" {
		valid := false
		for _, ext := range imaging.SupportedExtensions() {
			ext = strings.TrimPrefix(ext, ".")
			if ext == "jpeg" {
				ext = "jpg"
			}
			if ext == "tif" {
				ext = "tiff"
			}
			if ext == r.Format {
				valid = true
				break
			}
		}
		if !valid {
			return errors.New("unknown file type")
		}
	}
	if r.Orientation != "" && r.Orientation != "portrait" && r.Orientation != "landscape" && r.Orientation != "square" {
		return errors.New("invalid orientation")
	}
	for _, date := range []string{r.CaptureFrom, r.CaptureThrough} {
		if date != "" {
			if _, err := time.Parse("2006-01-02", date); err != nil {
				return errors.New("invalid capture date")
			}
		}
	}
	if r.CaptureFrom != "" && r.CaptureThrough != "" && r.CaptureFrom > r.CaptureThrough {
		return errors.New("invalid capture date range")
	}
	known := similarity.TagIDs()
	for _, tag := range r.Tags {
		if !slices.Contains(known, tag) {
			return errors.New("unknown visual tag")
		}
	}
	return nil
}

func (r Rule) empty() bool {
	return len(r.Tags) == 0 && strings.TrimSpace(r.Make) == "" && strings.TrimSpace(r.Model) == "" && r.CaptureFrom == "" && r.CaptureThrough == "" && r.Format == "" && r.Orientation == "" && r.MinWidth == 0 && r.MaxWidth == 0 && r.MinHeight == 0 && r.MaxHeight == 0
}

// Matches reads only immutable analysis facts, never source files or pixels.
func (r Rule) Matches(item similarity.Item) bool {
	if item.Error != "" || r.empty() {
		return false
	}
	for _, tag := range r.Tags {
		if !slices.Contains(item.Tags, tag) {
			return false
		}
	}
	f := item.Facts
	metadata := r.Make != "" || r.Model != "" || r.CaptureFrom != "" || r.CaptureThrough != "" || r.Format != "" || r.Orientation != "" || r.MinWidth != 0 || r.MaxWidth != 0 || r.MinHeight != 0 || r.MaxHeight != 0
	if metadata && f.Version != similarity.FactsVersion {
		return false
	}
	if r.Make != "" && !strings.EqualFold(strings.TrimSpace(r.Make), f.Make) {
		return false
	}
	if r.Model != "" && !strings.EqualFold(strings.TrimSpace(r.Model), f.Model) {
		return false
	}
	if (r.CaptureFrom != "" || r.CaptureThrough != "") && (f.CaptureDate == "" || (r.CaptureFrom != "" && f.CaptureDate < r.CaptureFrom) || (r.CaptureThrough != "" && f.CaptureDate > r.CaptureThrough)) {
		return false
	}
	if r.Format != "" && r.Format != f.Format {
		return false
	}
	if (r.Orientation != "" || r.MinWidth != 0 || r.MaxWidth != 0 || r.MinHeight != 0 || r.MaxHeight != 0) && (f.Width <= 0 || f.Height <= 0) {
		return false
	}
	if (r.MinWidth > 0 && f.Width < r.MinWidth) || (r.MaxWidth > 0 && f.Width > r.MaxWidth) || (r.MinHeight > 0 && f.Height < r.MinHeight) || (r.MaxHeight > 0 && f.Height > r.MaxHeight) {
		return false
	}
	orientation := "square"
	if f.Width < f.Height {
		orientation = "portrait"
	} else if f.Width > f.Height {
		orientation = "landscape"
	}
	return r.Orientation == "" || r.Orientation == orientation
}

// Store serializes this viewer's library operations. Call methods on a worker.
type Store struct {
	Dir string
	mu  sync.Mutex
}

func DefaultDir() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "picfetch", "presets")
}

type library struct {
	Version int
	Presets []Preset
}

func (s *Store) load(ctx context.Context) ([]Preset, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.Dir == "" {
		return nil, errors.New("preset directory unavailable")
	}
	data, err := os.ReadFile(filepath.Join(s.Dir, "presets.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var lib library
	if err := json.Unmarshal(data, &lib); err != nil {
		return nil, err
	}
	if lib.Version != 1 {
		return nil, errors.New("unsupported preset library version")
	}
	ids, names := map[string]bool{}, map[string]bool{}
	for _, p := range lib.Presets {
		name := strings.ToLower(p.Name)
		if p.ID == "" || !ValidName(p.Name) || ids[p.ID] || names[name] {
			return nil, errors.New("invalid preset library")
		}
		ids[p.ID], names[name] = true, true
	}
	return lib.Presets, ctx.Err()
}

func (s *Store) Load(ctx context.Context) ([]Preset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load(ctx)
}

// Delete removes only the reusable definition; callers own existing cohorts.
func (s *Store) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load(ctx)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(all, func(p Preset) bool { return p.ID == id })
	if i < 0 {
		return errors.New("preset was removed")
	}
	return s.write(ctx, slices.Delete(all, i, i+1))
}

// Save returns the stored identity only after atomic replacement succeeds.
func (s *Store) Save(ctx context.Context, p Preset) (Preset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.Name = strings.TrimSpace(p.Name)
	if !ValidName(p.Name) {
		return p, errors.New("invalid preset name")
	}
	if err := p.Rule.Validate(); err != nil {
		return p, err
	}
	all, err := s.load(ctx)
	if err != nil {
		return p, err
	}
	index := -1
	for i, old := range all {
		if old.ID == p.ID {
			index = i
		} else if strings.EqualFold(old.Name, p.Name) {
			return p, errors.New("preset name already exists")
		}
	}
	if p.ID != "" && index < 0 {
		return p, errors.New("preset was removed")
	}
	if p.ID == "" {
		p.ID = rand.Text()
	}
	p.Versions = CurrentVersions()
	if index < 0 {
		all = append(all, p)
	} else {
		all[index] = p
	}
	return p, s.write(ctx, all)
}

func (s *Store) write(ctx context.Context, all []Preset) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.Dir == "" {
		return errors.New("preset directory unavailable")
	}
	if err := os.MkdirAll(s.Dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(s.Dir, ".presets-*.tmp")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(file.Name()) }()
	err = json.NewEncoder(file).Encode(library{Version: 1, Presets: all})
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), filepath.Join(s.Dir, "presets.json")); err != nil {
		return fmt.Errorf("replace preset library: %w", err)
	}
	return nil
}
