package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// approval is prepared without Microsoft secrets, then retained as an immutable
// Actions artifact. Its file digest is passed between jobs, never rediscovered.
// GitHub Required reviewers authorizes the job consuming that exact artifact.
type approval struct {
	Schema       int     `json:"schema"`
	Mode         string  `json:"mode"`
	Release      release `json:"release"`
	Notes        string  `json:"notes"`
	NotesSHA     string  `json:"notes_sha256"`
	Base         string  `json:"base_version"`
	ReceiptSHA   string  `json:"previous_receipt_sha256"`
	SubmissionID string  `json:"submission_id,omitempty"`
	Phase        string  `json:"receipt_phase,omitempty"`
}

func receiptDigest(r *receipt) string {
	if r == nil {
		return digest([]byte("null"))
	}
	stable := *r
	stable.Release.Directory, stable.Release.Notes = "", ""
	b, _ := encode(stable)
	return digest(b)
}

func prepareApproval(ctx context.Context, rt runtime, o options, mode, out string) error {
	if (mode != "submit" && mode != "reconcile") || out == "" {
		return fmt.Errorf("prepare needs --mode submit|reconcile and --out")
	}
	g := githubAPI{rt: rt}
	saved, err := g.loadReceipt(ctx)
	if err != nil {
		return err
	}
	if err = validReceipt(saved); err != nil {
		return err
	}
	p := approval{Schema: 1, Mode: mode, Base: "1.0.2", ReceiptSHA: receiptDigest(saved)}
	if mode == "reconcile" || (saved != nil && saved.Phase == "published" && saved.Release.Tag == o.tag) {
		if saved == nil || (o.tag != "" && o.tag != saved.Release.Tag) {
			return fmt.Errorf("reconcile requires the selected durable receipt")
		}
		if err = verifyReceiptTag(ctx, rt, o.root, saved.Release); err != nil {
			return err
		}
		p.Release, p.Notes, p.NotesSHA, p.Base = saved.Release, saved.Notes, saved.NotesSHA, saved.BaseVersion
		p.SubmissionID, p.Phase = saved.SubmissionID, saved.Phase
	} else {
		if saved != nil {
			switch saved.Phase {
			case "published":
				p.Base = saved.Release.Version
			case "failed":
				p.Base = saved.BaseVersion
			default:
				return fmt.Errorf("a Store receipt is active; dispatch reconcile and approve that release before preparing another")
			}
		}
		candidates, err := g.candidatesForRun(ctx, o.root, o.tag, p.Base, o.runID)
		if err != nil {
			return err
		}
		if len(candidates) == 0 {
			return emit(rt, object{"state": "no_update", "published_version": p.Base})
		}
		p.Release = candidates[0]
		if saved != nil && saved.Phase == "failed" && p.Release.Version == saved.Release.Version {
			return fmt.Errorf("this release was rejected; identical bits will not be submitted again")
		}
		items, err := noteRange(ctx, rt, o.root, p.Base, p.Release)
		if err != nil {
			return err
		}
		p.Notes, err = generateNotes(p.Base, p.Release.Version, items)
		if err != nil {
			return err
		}
		p.NotesSHA = digest([]byte(p.Notes))
	}
	p.Release.Directory, p.Release.Notes = "", ""
	b, err := encode(p)
	if err != nil {
		return err
	}
	if err = os.WriteFile(out, b, 0o600); err != nil {
		return err
	}
	return emit(rt, p)
}

func loadApproval(path, sha, mode string) (approval, error) {
	var p approval
	if path == "" || sha == "" {
		return p, fmt.Errorf("submit/reconcile requires the frozen approval file and --approval-sha256")
	}
	b, err := readLimited(path, 4<<20)
	if err != nil || digest(b) != sha {
		return p, fmt.Errorf("approval file is unavailable or its digest changed")
	}
	if err = json.Unmarshal(b, &p); err != nil || p.Schema != 1 || p.Mode != mode || p.NotesSHA != digest([]byte(p.Notes)) || p.Release.ArtifactID <= 0 || p.ReceiptSHA == "" {
		return p, fmt.Errorf("invalid frozen approval")
	}
	if err = validReleaseProvenance(p.Release); err != nil {
		return p, err
	}
	if _, err = versionParts(p.Base); err != nil {
		return p, err
	}
	return p, nil
}
