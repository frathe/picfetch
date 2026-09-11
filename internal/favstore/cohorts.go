package favstore

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Cohort records explicit membership, rather than a rule for future images.
type Cohort struct {
	Name     string   `json:"name"`
	PresetID string   `json:"preset_id,omitempty"`
	Paths    []string `json:"paths"`
}

// CohortState retains explicit groups and reviewed returns to Unassigned.
type CohortState struct {
	Groups     []Cohort `json:"cohorts"`
	Unassigned []string `json:"unassigned,omitempty"`
}

type cohortDocument struct {
	Version int `json:"version"`
	CohortState
}

// CohortStore binds writes to the file list observed at open. It owns no open
// handles; each operation checks identity through a fresh directory handle.
type CohortStore struct {
	dir     string
	list    os.FileInfo
	members map[string]bool
}

// OpenCohorts loads favorite-owned groups on a worker. A missing cohort file is
// an empty collection; unreadable/corrupt data is an error, never overwritten.
func OpenCohorts(ctx context.Context, dir string) (*CohortStore, CohortState, error) {
	if err := ctx.Err(); err != nil {
		return nil, CohortState{}, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, CohortState{}, err
	}
	defer func() { _ = root.Close() }()
	info, err := root.Stat(fileListName)
	if err != nil {
		return nil, CohortState{}, err
	}
	data, err := root.ReadFile(fileListName)
	if err != nil {
		return nil, CohortState{}, err
	}
	var files map[string]string
	if err := json.Unmarshal(data, &files); err != nil {
		return nil, CohortState{}, err
	}
	s := &CohortStore{dir: dir, list: info, members: map[string]bool{}}
	for _, path := range files {
		s.members[path] = true
	}
	data, err = root.ReadFile("cohorts.json")
	var state CohortState
	if err == nil {
		if bytes.HasPrefix(bytes.TrimSpace(data), []byte("[")) {
			err = json.Unmarshal(data, &state.Groups)
		} else {
			var doc cohortDocument
			err = json.Unmarshal(data, &doc)
			if err == nil && doc.Version != 2 {
				err = errors.New("unsupported cohort state version")
			}
			state = doc.CohortState
		}
	} else if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	if err != nil {
		return nil, CohortState{}, err
	}
	if err := s.current(ctx, root); err != nil {
		return nil, CohortState{}, err
	}
	// Replacing a favorite may retain some old images; only these survive.
	for i := range state.Groups {
		var paths []string
		for _, path := range state.Groups[i].Paths {
			if s.members[path] {
				paths = append(paths, path)
			}
		}
		state.Groups[i].Paths = paths
	}
	var released []string
	for _, path := range state.Unassigned {
		if s.members[path] {
			released = append(released, path)
		}
	}
	state.Unassigned = released
	return s, state, nil
}

// Contains reports whether every current source belongs to this favorite.
func (s *CohortStore) Contains(paths []string) bool {
	for _, path := range paths {
		if !s.members[path] {
			return false
		}
	}
	return true
}

func (s *CohortStore) current(ctx context.Context, root *os.Root) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now, err := root.Stat(fileListName)
	if err != nil {
		return err
	}
	if !os.SameFile(s.list, now) || s.list.Size() != now.Size() || !s.list.ModTime().Equal(now.ModTime()) {
		return errors.New("favorite file list changed")
	}
	return nil
}

// Save atomically replaces group metadata only for the same favorite file list.
// It never creates a favorite directory removed while this map was open.
func (s *CohortStore) Save(ctx context.Context, state CohortState) error {
	root, err := os.OpenRoot(s.dir)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	if err := s.current(ctx, root); err != nil {
		return err
	}
	for _, group := range state.Groups {
		if !s.Contains(group.Paths) {
			return fmt.Errorf("cohort %q contains images outside the favorite", group.Name)
		}
	}
	if !s.Contains(state.Unassigned) {
		return errors.New("unassigned images are outside the favorite")
	}
	name := ".cohorts-" + rand.Text() + ".tmp"
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = root.Remove(name) }()
	err = json.NewEncoder(file).Encode(cohortDocument{Version: 2, CohortState: state})
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
	if err := s.current(ctx, root); err != nil {
		return err
	}
	return root.Rename(name, "cohorts.json")
}
