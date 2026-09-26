package locationtrial

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRecorderKeepsLatestStateAndRequiresNewDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "native")
	recorder, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := New(dir); err == nil {
		t.Fatal("existing evidence overwritten")
	}
	formats := map[string]int{"jpg": 3}
	recorder.Publish(State{Images: 3, Formats: formats, Ready: true})
	formats["jpg"] = 4
	for i := range 100 {
		recorder.Publish(State{Sequence: i, Images: 3, Ready: true, Stages: []Stage{{Kind: "cold", StartNS: 1, EndNS: 10, PreparationNS: 2, ScanNS: 7, Complete: true}}})
	}
	recorder.Stop()
	if err := recorder.Wait(); err != nil {
		t.Fatal(err)
	}
	recorder.Publish(State{Images: 999})
	data, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	if state.Sequence != 99 || state.Images != 3 || len(state.Stages) != 1 || !state.Stages[0].Complete {
		t.Fatalf("lost final admitted state: %+v", state)
	}
}

func TestRecorderReportsWriteFailure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "native")
	recorder, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "state.json"), 0o700); err != nil {
		t.Fatal(err)
	}
	recorder.Publish(State{Images: 1})
	recorder.Stop()
	if err := recorder.Wait(); err == nil {
		t.Fatal("failed observation write accepted")
	}
}
