package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/locationtrial"
)

func TestSustainedMemoryDoesNotSubstituteSlowStartupForBrowsing(t *testing.T) {
	memory := &memoryObservation{samples: []MemorySample{{AtNS: 1, RSSBytes: 1}, {AtNS: int64(2 * time.Minute), RSSBytes: 1}}}
	if !memory.needsSustainedBrowsing() {
		t.Fatal("startup/scan duration substituted for sustained browsing")
	}
}

func TestProcessDriverReadsPublishedState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"images":7,"ready":true,"formats":{"jpg":7}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	driver := &processDriver{app: &nativeProcess{done: make(chan struct{})}, statePath: path}
	state, err := driver.State(context.Background())
	if err != nil || !state.Ready || state.Images != 7 || state.Formats["jpg"] != 7 {
		t.Fatalf("published state not returned: %+v, %v", state, err)
	}
}

type fixtureNativeDriver struct {
	state       locationtrial.State
	inputs      int
	failGesture int
	gestures    int
}

func (d *fixtureNativeDriver) State(_ context.Context) (locationtrial.State, error) {
	return d.state, nil
}
func (d *fixtureNativeDriver) Input(_ context.Context, command nativeCommand) (nativeObservation, error) {
	d.inputs++
	if command.Kind == "open" {
		d.state.Active, d.state.Visible = true, true
		kind := "warm"
		if len(d.state.Stages) == 0 {
			kind = "cold"
		}
		start := int64(d.inputs) * 1_000_000_000
		d.state.Stages = append(d.state.Stages, Stage{Kind: kind, StartNS: start, EndNS: start + 20, PreparationNS: 5, ScanNS: 15, Complete: true})
	} else if command.Kind == "cancel" || command.Kind == "close" {
		d.state.Active, d.state.Visible = false, false
	}
	observation := nativeObservation{Kind: command.Kind, InputNS: int64(d.inputs) * 1_000_000_000, VisibleNS: int64(d.inputs)*1_000_000_000 + 10, Before: fmt.Sprintf("%s-before.png", command.Name), After: fmt.Sprintf("%s-after.png", command.Name)}
	if command.Kind == "pan" || command.Kind == "zoom" {
		d.gestures++
		if d.gestures == d.failGesture {
			observation.VisibleNS = 0
			observation.Skipped = true
			return observation, errors.New("controlled native observation failure")
		}
	}
	return observation, nil
}

func TestNativeProtocolUsesObservedCountAndRetainsFailedSample(t *testing.T) {
	for _, fail := range []int{0, 3} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			driver := &fixtureNativeDriver{state: locationtrial.State{Images: 7, Formats: map[string]int{"jpg": 7}, Ready: true}, failGesture: fail}
			var report Report
			err := collectNative(context.Background(), driver, &report, func() bool { return false })
			if report.Images != 7 || report.Formats["jpg"] != 7 {
				t.Fatal("dataset count substituted for actual admission")
			}
			if fail != 0 {
				if err == nil || len(report.Gestures) != fail || !report.Gestures[fail-1].Skipped {
					t.Fatal("failed native sample discarded")
				}
				return
			}
			if err != nil || len(report.Gestures) != 40 || len(report.Stages) != 2 || len(report.Cancellations) != 1 || report.OpenCloseCycles < 2 {
				t.Fatalf("incomplete native protocol: %+v %v", report, err)
			}
		})
	}
}

func TestNativeProtocolCancelsWithoutFabricatingStages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	driver := &fixtureNativeDriver{}
	var report Report
	if err := collectNative(ctx, driver, &report, func() bool { return false }); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	if len(report.Stages) != 0 || len(report.Gestures) != 0 {
		t.Fatal("interrupted protocol fabricated observations")
	}
}
