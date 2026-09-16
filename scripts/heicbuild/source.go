package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

const maintainedDir = "third_party/h265"
const maintainedRecord = "PICFETCH-SOURCE.json"
const maintainedRevision = "fc127b44e1c99447b8d150256563c6d4f99b8ad3"
const maintainedRecordSHA = "583c5ab3b021675acfac0bc7344a7ff7acdc8f0ae1ab94e804d7b931e3618b9b"

// The baseline records the exact previously maintained production copy. A
// decoder update must intentionally review this record as well as the guest.
func verifyMaintainedSource(root string) error {
	dir := filepath.Join(root, maintainedDir)
	path := filepath.Join(dir, maintainedRecord)
	digest, err := fileDigest(path)
	if err != nil {
		return err
	}
	if digest != maintainedRecordSHA {
		return errors.New("maintained decoder source record changed without review")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var record struct {
		Files map[string]string
	}
	if err = json.Unmarshal(data, &record); err != nil {
		return err
	}
	if err = checkDigests(dir, record.Files); err != nil {
		return err
	}
	return filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == maintainedRecord || rel == "PICFETCH.md" {
			return nil
		}
		if _, ok := record.Files[rel]; !ok {
			return fmt.Errorf("unreviewed maintained decoder file: %s", rel)
		}
		return nil
	})
}

func verifyReplacements(root string) error {
	for _, name := range []string{"go.mod", guestDir + "/go.mod"} {
		path := filepath.Join(root, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		module, err := modfile.Parse(path, data, nil)
		if err != nil {
			return err
		}
		found := false
		for _, replacement := range module.Replace {
			if replacement.Old.Path != decoderModule {
				continue
			}
			if replacement.Old.Version != "" || replacement.New.Version != "" ||
				filepath.Clean(filepath.Join(filepath.Dir(path), replacement.New.Path)) != filepath.Join(root, maintainedDir) {
				return fmt.Errorf("%s does not select the maintained decoder source", name)
			}
			found = true
		}
		if !found {
			return fmt.Errorf("%s lacks the maintained decoder replacement", name)
		}
	}
	return nil
}
