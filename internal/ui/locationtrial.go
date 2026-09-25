package ui

import (
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/locationtrial"
)

type locationTrialSession struct {
	recorder  *locationtrial.Recorder
	state     locationtrial.State
	scanStart int64
}

func (v *viewer) configureLocationTrial(dir string) error {
	if dir == "" {
		return nil
	}
	recorder, err := locationtrial.New(dir)
	if err != nil {
		return err
	}
	v.locationTrial = &locationTrialSession{recorder: recorder}
	return nil
}

func (v *viewer) beginLocationTrial() {
	t := v.locationTrial
	if t == nil {
		return
	}
	t.state.Images = v.FileCount()
	t.state.Formats = map[string]int{}
	for _, source := range v.state.files {
		t.state.Formats[strings.ToLower(strings.TrimPrefix(filepath.Ext(source.Path()), "."))]++
	}
	kind := "warm"
	if len(t.state.Stages) == 0 {
		kind = "cold"
	}
	t.state.Stages = append(t.state.Stages, locationtrial.Stage{Kind: kind, StartNS: time.Now().UnixNano()})
	t.scanStart = 0
	// Fixed qualification geometry affects only this isolated launch.
	v.win.Resize(fyne.NewSize(1200, 800))
}

func (v *viewer) scanLocationTrial() {
	t := v.locationTrial
	if t == nil || len(t.state.Stages) == 0 {
		return
	}
	t.scanStart = time.Now().UnixNano()
	stage := &t.state.Stages[len(t.state.Stages)-1]
	stage.PreparationNS = t.scanStart - stage.StartNS
}

func (v *viewer) recordLocationTrial() {
	t := v.locationTrial
	if t == nil {
		return
	}
	t.state.Sequence++
	t.state.Ready = v.FileCount() > 0 && !v.scanOp.active && !v.sortOp.active
	t.state.Active, t.state.Visible = v.locationMap.Active(), v.locationMap.Visible()
	if t.scanStart != 0 && t.state.Active && v.locationMap.Counts().Complete && len(t.state.Stages) > 0 {
		stage := &t.state.Stages[len(t.state.Stages)-1]
		if !stage.Complete {
			stage.EndNS = time.Now().UnixNano()
			stage.ScanNS = stage.EndNS - t.scanStart
			stage.Complete = true
		}
	}
	t.recorder.Publish(t.state)
}

func (v *viewer) stopLocationTrial() {
	if v.locationTrial != nil {
		v.locationTrial.recorder.Stop()
	}
}

func (v *viewer) waitLocationTrial() error {
	if v.locationTrial == nil {
		return nil
	}
	return v.locationTrial.recorder.Wait()
}
