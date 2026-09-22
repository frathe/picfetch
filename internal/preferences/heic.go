package preferences

import (
	"encoding/json"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/heic"
)

const heicObservationKey = "heicObservation.v1"

// ClearHEICObservation retires a genuinely invalidated provider observation.
func ClearHEICObservation(app fyne.App) {
	prefsWriteMu.Lock()
	defer prefsWriteMu.Unlock()
	app.Preferences().RemoveValue(heicObservationKey)
}

// LoadHEICObservation reads the independently persisted capability record.
func LoadHEICObservation(app fyne.App) heic.Observation {
	prefsWriteMu.Lock()
	defer prefsWriteMu.Unlock()
	text := app.Preferences().String(heicObservationKey)
	var observation heic.Observation
	if len(text) > 4096 || json.Unmarshal([]byte(text), &observation) != nil || !observation.Matches(observation.Identity) {
		return heic.Observation{}
	}
	return observation
}

// SaveHEICObservation cannot be overwritten by an older full settings snapshot.
func SaveHEICObservation(app fyne.App, observation heic.Observation) {
	if !observation.Matches(observation.Identity) {
		return
	}
	data, err := json.Marshal(observation)
	if err != nil {
		fyne.LogError("save HEIC capability observation", err)
		return
	}
	prefsWriteMu.Lock()
	defer prefsWriteMu.Unlock()
	app.Preferences().SetString(heicObservationKey, string(data))
}
