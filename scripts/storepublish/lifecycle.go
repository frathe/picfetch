package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type options struct {
	root, tag, candidate, snapshot, stateDir string
	runID                                    int64
}

func noteRange(ctx context.Context, rt runtime, root, base string, r release) ([]noteRelease, error) {
	items := []noteRelease{{r.Version, r.Notes}}
	match := regexp.MustCompile(`compare/v([0-9]+\.[0-9]+\.[0-9]+)\.\.\.v`).FindStringSubmatch(r.Notes)
	if len(match) != 2 {
		return nil, fmt.Errorf("release notes have no predecessor")
	}
	previous := match[1]
	seen := map[string]bool{r.Version: true}
	for previous != base {
		cmp, err := compareVersion(previous, base)
		if err != nil || cmp < 0 || seen[previous] || len(items) >= 100 {
			return nil, fmt.Errorf("release-note chain does not reach the Store version")
		}
		tag := "refs/tags/v" + previous
		commit, err := rt.Git(ctx, root, "rev-parse", "--verify", tag+"^{commit}")
		if err != nil {
			return nil, err
		}
		if _, err = rt.Git(ctx, root, "merge-base", "--is-ancestor", strings.TrimSpace(string(commit)), r.Commit); err != nil {
			return nil, fmt.Errorf("note predecessor is not in release history")
		}
		data, err := rt.Git(ctx, root, "show", tag+":.github/release-notes.md")
		if err != nil {
			return nil, err
		}
		if err = validateNotes(previous, string(data)); err != nil {
			return nil, err
		}
		items = append(items, noteRelease{previous, string(data)})
		seen[previous] = true
		match = regexp.MustCompile(`compare/v([0-9]+\.[0-9]+\.[0-9]+)\.\.\.v`).FindStringSubmatch(string(data))
		if len(match) != 2 {
			return nil, fmt.Errorf("note predecessor is missing")
		}
		previous = match[1]
	}
	return items, nil
}
func emit(rt runtime, v any) error { return json.NewEncoder(rt.Out).Encode(v) }
func previewRelease(ctx context.Context, rt runtime, o options) error {
	var r release
	var published object
	var saved *receipt
	var err error
	if o.candidate != "" || o.snapshot != "" {
		if o.candidate == "" || o.snapshot == "" {
			return fmt.Errorf("offline preview needs both candidate and snapshot")
		}
		r, err = loadRelease(o.candidate)
		if err != nil {
			return err
		}
		b, err := readLimited(o.snapshot, 8<<20)
		if err != nil {
			return err
		}
		published, err = parseObject(b)
		if err != nil {
			return err
		}
	} else {
		s := storeClient{rt: rt}
		app, err := s.application(ctx)
		if err != nil {
			return err
		}
		published, err = s.published(ctx, app)
		if err != nil {
			return err
		}
		g := githubAPI{rt: rt}
		saved, err = g.loadReceipt(ctx)
		if err != nil {
			return err
		}
		if err = validReceipt(saved); err != nil {
			return err
		}
		base, err := publishedVersion(published, saved)
		if err != nil {
			return err
		}
		candidates, err := g.candidates(ctx, o.root, o.tag, base)
		if err != nil {
			return err
		}
		if len(candidates) == 0 {
			return fmt.Errorf("no eligible validated Store release")
		}
		r = candidates[0]
	}
	base, err := publishedVersion(published, saved)
	if err != nil {
		return err
	}
	if err = admit(r, base); err != nil {
		return err
	}
	if err = advanceStoreVersion(r, published); err != nil {
		return err
	}
	items := []noteRelease{{r.Version, r.Notes}}
	if o.candidate == "" {
		items, err = noteRange(ctx, rt, o.root, base, r)
		if err != nil {
			return err
		}
	}
	notes, err := generateNotes(base, r.Version, items)
	if err != nil {
		return err
	}
	submission, err := prepareSubmission(published, notes)
	if err != nil {
		return err
	}
	return emit(rt, object{"state": "preview", "version": r.Version, "tag": r.Tag, "run_id": r.RunID, "artifact_id": r.ArtifactID, "bundle_sha256": r.BundleSHA, "notes": notes, "submission": publicSubmission(submission)})
}
func publicSubmission(source object) object {
	b, _ := json.Marshal(source)
	out, _ := parseObject(b)
	delete(out, "fileUploadUrl")
	delete(out, "statusDetails")
	return out
}
func checkStore(ctx context.Context, rt runtime) error {
	s := storeClient{rt: rt}
	app, err := s.application(ctx)
	if err != nil {
		return err
	}
	published, err := s.published(ctx, app)
	if err != nil {
		return err
	}
	g := githubAPI{rt: rt}
	saved, err := g.loadReceipt(ctx)
	if err != nil {
		return err
	}
	if err = validReceipt(saved); err != nil {
		return err
	}
	version, err := publishedVersion(published, saved)
	if err != nil {
		return err
	}
	result := object{"product_id": productID, "published_version": version, "published_submission_id": submissionRef(app, "lastPublishedApplicationSubmission"), "state": "Published"}
	if pending := submissionRef(app, "pendingApplicationSubmission"); pending != "" {
		status, err := s.request(ctx, http.MethodGet, "/submissions/"+pending+"/status", nil)
		if err != nil {
			return err
		}
		result["pending_submission_id"] = pending
		result["pending_state"] = stringField(status, "status")
		if saved != nil && pending == saved.SubmissionID && statusKind(stringField(status, "status")) == "pending" {
			report, err := pendingDiagnostics(ctx, &s, saved, published)
			if err != nil {
				return err
			}
			result["pending_validation"] = report
		}
	}
	return emit(rt, result)
}

func pendingDiagnostics(ctx context.Context, s *storeClient, saved *receipt, base object) (object, error) {
	current, err := s.submission(ctx, saved.SubmissionID)
	if err != nil {
		return nil, err
	}
	if stringField(base, "id") != saved.BaseSubmission {
		base, err = s.submission(ctx, saved.BaseSubmission)
		if err != nil {
			return nil, err
		}
	}
	prepared, err := prepareSubmission(base, saved.Notes)
	if err != nil {
		return nil, err
	}
	currentMatches, err := metadataMatches(current, saved.MetadataSHA)
	if err != nil {
		return nil, err
	}
	preparedMatches, err := metadataMatches(prepared, saved.MetadataSHA)
	if err != nil {
		return nil, err
	}
	return object{
		"receipt_phase":                  saved.Phase,
		"base_submission_id":             saved.BaseSubmission,
		"metadata_matches_recorded":      currentMatches,
		"packages_match_recorded":        pendingPackagesMatch(current),
		"prepared_base_matches_recorded": preparedMatches,
		"changes_from_base":              metadataDifferences(base, current),
		"changes_from_prepared":          metadataDifferences(prepared, current),
	}, nil
}

func validReceipt(r *receipt) error {
	if r == nil {
		return nil
	}
	if r.Schema != 1 || r.Release.Repository != repository || r.Release.ArtifactID <= 0 || r.NotesSHA != digest([]byte(r.Notes)) || r.Release.Tag != "v"+r.Release.Version {
		return fmt.Errorf("invalid durable Store receipt")
	}
	if err := validReleaseProvenance(r.Release); err != nil {
		return err
	}
	for _, sha := range []string{r.Release.BundleSHA, r.Release.WACKSHA, r.MetadataSHA} {
		if !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(sha) {
			return fmt.Errorf("invalid receipt digest")
		}
	}
	if !regexp.MustCompile(`^[A-Za-z0-9-]+$`).MatchString(r.BaseSubmission) {
		return fmt.Errorf("invalid base submission identifier")
	}
	if _, err := versionParts(r.BaseVersion); err != nil {
		return err
	}
	if _, err := versionParts(r.BasePackageVersion); err != nil {
		return err
	}
	if _, err := publicVersion(r.Release.BundleVersion); err != nil {
		return err
	}
	switch r.Phase {
	case "creating", "created", "updated", "uploaded", "committing", "observing", "published", "failed":
	default:
		return fmt.Errorf("unknown receipt phase")
	}
	if r.Phase != "creating" && r.SubmissionID == "" {
		return fmt.Errorf("receipt has no submission identifier")
	}
	return nil
}
func statusKind(status string) string {
	switch status {
	case "Published":
		return "published"
	case "PendingCommit":
		return "pending"
	case "CommitStarted", "PreProcessing", "Certification", "Release", "Publishing":
		return "processing"
	case "Canceled", "CommitFailed", "PreProcessingFailed", "CertificationFailed", "ReleaseFailed", "PublishFailed":
		return "failed"
	default:
		return "conflict"
	}
}
func receiptResult(r *receipt, status string) object {
	return object{"state": status, "tag": r.Release.Tag, "version": r.Release.Version, "run_id": r.Release.RunID, "artifact_id": r.Release.ArtifactID, "bundle_sha256": r.Release.BundleSHA, "notes_sha256": r.NotesSHA, "submission_id": r.SubmissionID}
}
func drive(ctx context.Context, rt runtime, o options, approved approval) error {
	if rt.Env("PICFETCH_STORE_SERIALIZED") != "1" {
		return fmt.Errorf("submit/reconcile must run in the serialized Microsoft Store publisher workflow")
	}
	if o.candidate != "" || o.snapshot != "" {
		return fmt.Errorf("submission must discover its original artifact from GitHub")
	}
	if err := os.MkdirAll(o.stateDir, 0o700); err != nil {
		return err
	}
	lock := filepath.Join(o.stateDir, "picfetch-store.lock")
	if err := os.Mkdir(lock, 0o700); err != nil {
		return fmt.Errorf("another Store publisher owns the local claim; check its completion before removing a stale claim")
	}
	defer func() { _ = os.Remove(lock) }()
	g := githubAPI{rt: rt}
	saved, err := g.loadReceipt(ctx)
	if err != nil {
		return err
	}
	if err = validReceipt(saved); err != nil {
		return err
	}
	if receiptDigest(saved) != approved.ReceiptSHA {
		return fmt.Errorf("Store receipt changed after preparation; prepare a fresh approval")
	}
	if approved.Mode == "reconcile" && (saved == nil || saved.Release.Tag != approved.Release.Tag || saved.Release.ArtifactID != approved.Release.ArtifactID || saved.NotesSHA != approved.NotesSHA) {
		return fmt.Errorf("approval does not match the receipt to reconcile")
	}
	if err = verifyReceiptTag(ctx, rt, o.root, approved.Release); err != nil {
		return err
	}
	s := storeClient{rt: rt}
	app, err := s.application(ctx)
	if err != nil {
		return err
	}
	if saved != nil && saved.Phase != "published" && saved.Phase != "failed" {
		if approved.Mode != "reconcile" {
			return fmt.Errorf("active Store receipt requires a separate reconcile approval")
		}
		if saved.Phase == "creating" {
			return fmt.Errorf("Store creation outcome is ambiguous; inspect the draft and receipt before any recovery")
		}
		if err = verifyReceiptTag(ctx, rt, o.root, saved.Release); err != nil {
			return err
		}
		current, err := s.submission(ctx, saved.SubmissionID)
		if err != nil {
			return err
		}
		status, err := s.request(ctx, http.MethodGet, "/submissions/"+saved.SubmissionID+"/status", nil)
		if err != nil {
			return err
		}
		state := stringField(status, "status")
		switch statusKind(state) {
		case "published":
			actual, err := publishedVersion(current, saved)
			if err != nil || actual != saved.Release.Version {
				return fmt.Errorf("published submission version differs from receipt")
			}
			saved.Phase = "published"
			if err = g.saveReceipt(ctx, saved); err != nil {
				return err
			}
			return emit(rt, receiptResult(saved, state))
		case "failed":
			saved.Phase = "failed"
			if err = g.saveReceipt(ctx, saved); err != nil {
				return err
			}
			_ = emit(rt, receiptResult(saved, state))
			return fmt.Errorf("Microsoft Store submission %s failed (%s); inspect certification in Partner Center", saved.SubmissionID, state)
		case "processing":
			if saved.Phase != "observing" {
				saved.Phase = "observing"
				if err = g.saveReceipt(ctx, saved); err != nil {
					return err
				}
			}
			return emit(rt, receiptResult(saved, state))
		case "pending":
			if saved.Phase == "committing" || saved.Phase == "observing" {
				return fmt.Errorf("commit outcome is ambiguous while Store reports PendingCommit; no commit was repeated")
			}
			if submissionRef(app, "pendingApplicationSubmission") != saved.SubmissionID {
				return fmt.Errorf("pending Store submission differs from durable receipt")
			}
			restored, err := g.restore(ctx, o.root, saved.Release)
			if err != nil {
				return err
			}
			saved.Release = restored
			return finishPending(ctx, rt, &s, &g, saved, current)
		default:
			_ = emit(rt, receiptResult(saved, state))
			return fmt.Errorf("Store state %q requires inspection before mutation", state)
		}
	}
	if approved.Mode == "reconcile" {
		if saved.Phase == "failed" {
			_ = emit(rt, receiptResult(saved, "failed"))
			return fmt.Errorf("recorded Store submission failed; inspect certification in Partner Center")
		}
		return emit(rt, receiptResult(saved, saved.Phase))
	}
	published, err := s.published(ctx, app)
	if err != nil {
		return err
	}
	base, err := publishedVersion(published, saved)
	if err != nil {
		return err
	}
	if pending := submissionRef(app, "pendingApplicationSubmission"); pending != "" {
		return fmt.Errorf("Store has an existing draft or submission %s; it was preserved", pending)
	}
	if saved != nil && o.tag == saved.Release.Tag {
		if saved.Phase == "published" && base == saved.Release.Version {
			if err = verifyReceiptTag(ctx, rt, o.root, saved.Release); err != nil {
				return err
			}
			return emit(rt, receiptResult(saved, "Published"))
		}
		if saved.Phase == "failed" {
			return fmt.Errorf("this release was rejected; identical bits will not be submitted again")
		}
	}
	if base != approved.Base {
		return fmt.Errorf("published Store base changed after preparation; prepare a fresh approval")
	}
	r, err := g.restore(ctx, o.root, approved.Release)
	if err != nil {
		return err
	}
	if err = admit(r, base); err != nil {
		return err
	}
	notes := approved.Notes
	prepared, err := prepareSubmission(published, notes)
	if err != nil {
		return err
	}
	metadata, err := metadataDigest(prepared)
	if err != nil {
		return err
	}
	basePackageVersion, err := packageVersion(published)
	if err != nil {
		return err
	}
	if err = advanceStoreVersion(r, published); err != nil {
		return err
	}
	saved = &receipt{Schema: 1, Release: r, Notes: notes, NotesSHA: digest([]byte(notes)), BaseVersion: base, BasePackageVersion: basePackageVersion, BaseSubmission: submissionRef(app, "lastPublishedApplicationSubmission"), Phase: "creating", MetadataSHA: metadata}
	// Persist intent BEFORE the bodyless, non-idempotent creation call.
	if err = g.saveReceipt(ctx, saved); err != nil {
		return err
	}
	created, err := s.request(ctx, http.MethodPost, "/submissions", nil)
	if err != nil {
		return err
	}
	saved.SubmissionID = stringField(created, "id")
	if saved.SubmissionID == "" {
		return fmt.Errorf("Store create response has no submission ID; reconcile before recovery")
	}
	saved.Phase = "created"
	if err = g.saveReceipt(ctx, saved); err != nil {
		return err
	}
	return finishPending(ctx, rt, &s, &g, saved, created)
}
func advanceStoreVersion(r release, published object) error {
	base, err := packageVersion(published)
	if err != nil {
		return err
	}
	outer, err := publicVersion(r.BundleVersion)
	if err != nil {
		return err
	}
	if cmp, _ := compareVersion(outer, base); cmp <= 0 {
		return fmt.Errorf("candidate bundle version does not advance the published Store package")
	}
	return nil
}
func verifyReceiptTag(ctx context.Context, rt runtime, root string, r release) error {
	if r.Tag != "v"+r.Version {
		return fmt.Errorf("receipt tag does not match version")
	}
	if _, err := versionParts(r.Version); err != nil {
		return err
	}
	b, err := rt.Git(ctx, root, "rev-parse", "--verify", "refs/tags/"+r.Tag+"^{commit}")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(b)) != r.Commit {
		return fmt.Errorf("release tag moved after submission")
	}
	_, err = rt.Git(ctx, root, "merge-base", "--is-ancestor", r.Commit, "origin/main")
	return err
}
func pendingPackagesMatch(current object) bool {
	entries, ok := current["applicationPackages"].([]any)
	if !ok {
		return false
	}
	newCount := 0
	for _, entry := range entries {
		p := asObject(entry)
		if stringField(p, "fileStatus") == "PendingDelete" {
			continue
		}
		if stringField(p, "fileName") != bundleName {
			return false
		}
		status := stringField(p, "fileStatus")
		if status != "PendingUpload" && status != "Uploaded" {
			return false
		}
		newCount++
	}
	return newCount == 1
}
func finishPending(ctx context.Context, rt runtime, s *storeClient, g *githubAPI, r *receipt, current object) error {
	currentMatches, err := metadataMatches(current, r.MetadataSHA)
	if err != nil {
		return err
	}
	if r.Phase == "created" {
		base, err := s.submission(ctx, r.BaseSubmission)
		if err != nil {
			return err
		}
		baseSHA, err := metadataDigest(base)
		if err != nil {
			return err
		}
		matchesBase, err := metadataMatches(current, baseSHA)
		if err != nil {
			return err
		}
		if matchesBase {
			prepared, err := prepareSubmission(current, r.Notes)
			if err != nil {
				return err
			}
			matchesPrepared, err := metadataMatches(prepared, r.MetadataSHA)
			if err != nil {
				return err
			}
			if !matchesPrepared {
				return fmt.Errorf("submission metadata changed from preview")
			}
			current, err = s.request(ctx, http.MethodPut, "/submissions/"+r.SubmissionID, prepared)
			if err != nil {
				return err
			}
			currentMatches, err = metadataMatches(current, r.MetadataSHA)
			if err != nil {
				return err
			}
		}
		if !currentMatches || !pendingPackagesMatch(current) {
			return fmt.Errorf("pending submission has changes outside the recorded update")
		}
		r.Phase = "updated"
		if err = g.saveReceipt(ctx, r); err != nil {
			return err
		}
	}
	if !currentMatches || !pendingPackagesMatch(current) {
		return fmt.Errorf("pending submission no longer matches the recorded update")
	}
	if r.Phase == "updated" {
		// Always retrieve a current upload URI; it is never put in the journal.
		current, err = s.submission(ctx, r.SubmissionID)
		if err != nil {
			return err
		}
		currentMatches, err = metadataMatches(current, r.MetadataSHA)
		if err != nil || !currentMatches || !pendingPackagesMatch(current) {
			return fmt.Errorf("pending submission changed before upload; inspect Partner Center")
		}
		if err = s.upload(ctx, stringField(current, "fileUploadUrl"), r.Release.Directory); err != nil {
			return err
		}
		r.Phase = "uploaded"
		if err = g.saveReceipt(ctx, r); err != nil {
			return err
		}
	}
	if r.Phase != "uploaded" {
		return fmt.Errorf("unexpected pending submission phase")
	}
	current, err = s.submission(ctx, r.SubmissionID)
	if err != nil {
		return err
	}
	currentMatches, err = metadataMatches(current, r.MetadataSHA)
	if err != nil || !currentMatches || !pendingPackagesMatch(current) {
		return fmt.Errorf("pending submission changed before commit; inspect Partner Center")
	}
	r.Phase = "committing"
	if err = g.saveReceipt(ctx, r); err != nil {
		return err
	}
	committed, err := s.request(ctx, http.MethodPost, "/submissions/"+r.SubmissionID+"/commit", nil)
	if err != nil {
		return err
	}
	state := stringField(committed, "status")
	if statusKind(state) != "processing" {
		return fmt.Errorf("unexpected commit response; reconcile the recorded submission")
	}
	r.Phase = "observing"
	if err = g.saveReceipt(ctx, r); err != nil {
		return err
	}
	return emit(rt, receiptResult(r, state))
}
