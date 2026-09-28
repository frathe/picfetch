package launch

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
)

// ErrInvalidPolicy identifies a missing or rejected launch policy.
var ErrInvalidPolicy = errors.New("launch: invalid policy")

// Purpose identifies an ordinary launch or the isolated trial being recorded.
type Purpose uint8

const (
	Ordinary Purpose = iota
	ExplorerTrial
	LocationMapTrial
)

// UpdateReason explains why GitHub self-update effects are unavailable.
type UpdateReason string

const (
	MissingPolicy       UpdateReason = "missing policy"
	StoreManagedUpdates UpdateReason = "store-managed updates"
	TrialUpdates        UpdateReason = "trial updates"
)

// UpdatePermission is a copyable decision captured at launch construction.
type UpdatePermission struct {
	valid        bool
	storeManaged bool
	trial        bool
}

// Allowed reports whether this launch may perform GitHub self-update effects.
func (p UpdatePermission) Allowed() bool { return p.valid && !p.storeManaged && !p.trial }

// Reasons returns a fresh slice of all applicable refusal reasons.
func (p UpdatePermission) Reasons() []UpdateReason {
	if !p.valid {
		return []UpdateReason{MissingPolicy}
	}
	var reasons []UpdateReason
	if p.storeManaged {
		reasons = append(reasons, StoreManagedUpdates)
	}
	if p.trial {
		reasons = append(reasons, TrialUpdates)
	}
	return reasons
}

// Storage names the four launch-selected roots used by application features.
type Storage struct {
	FavoritesDir string
	PresetsDir   string
	AnalysisDir  string
	UpdatesDir   string
}

// Policy is an immutable, resource-free launch decision. Its zero value is invalid.
type Policy struct {
	valid         bool
	purpose       Purpose
	storeManaged  bool
	applicationID string
	trialDir      string
}

// NewPolicy captures launch facts without probing or reserving trial resources.
func NewPolicy(opts Options, normalID string, storeManaged bool) (Policy, error) {
	if normalID == "" {
		return Policy{}, fmt.Errorf("%w: application identity is empty", ErrInvalidPolicy)
	}
	if opts.ExplorerTrial != "" && opts.LocationMapTrial != "" {
		return Policy{}, fmt.Errorf("%w: choose only one native trial mode", ErrInvalidPolicy)
	}
	purpose := Ordinary
	if opts.ExplorerTrial != "" {
		purpose = ExplorerTrial
	} else if opts.LocationMapTrial != "" {
		purpose = LocationMapTrial
	}
	policy := Policy{valid: true, purpose: purpose, storeManaged: storeManaged, applicationID: normalID}
	if purpose == Ordinary {
		return policy, nil
	}
	trialDir := opts.ExplorerTrial
	if purpose == LocationMapTrial {
		trialDir = opts.LocationMapTrial
	}
	absolute, err := filepath.Abs(trialDir)
	if err != nil {
		return Policy{}, fmt.Errorf("%w: resolve trial directory: %w", ErrInvalidPolicy, err)
	}
	policy.trialDir = filepath.Clean(absolute)
	digest := sha256.Sum256([]byte(policy.trialDir))
	if purpose == ExplorerTrial {
		// Match explorertrial.Identity's namespace without resolving the root again.
		policy.applicationID = fmt.Sprintf("io.picfetch.explorer-trial.%x", digest)
	} else {
		policy.applicationID = fmt.Sprintf("%s.location-map-trial.%x", normalID, digest[:12])
	}
	return policy, nil
}

// Valid reports whether NewPolicy successfully constructed this value.
func (p Policy) Valid() bool { return p.valid }

// ApplicationID is the captured identity for Fyne preferences and sessions.
func (p Policy) ApplicationID() string { return p.applicationID }

// Purpose identifies the captured ordinary or trial launch mode.
func (p Policy) Purpose() Purpose { return p.purpose }

// TrialDir is the absolute trial root, or empty for an ordinary launch.
func (p Policy) TrialDir() string { return p.trialDir }

// StoreManaged reports the captured distribution input.
func (p Policy) StoreManaged() bool { return p.storeManaged }

// Updates returns the launch's fixed self-update permission.
func (p Policy) Updates() UpdatePermission {
	return UpdatePermission{valid: p.valid, storeManaged: p.storeManaged, trial: p.purpose != Ordinary}
}

// ResolveStorage preserves ordinary roots or selects children of the trial root.
func (p Policy) ResolveStorage(ordinary Storage) (Storage, error) {
	if !p.valid {
		return Storage{}, ErrInvalidPolicy
	}
	if p.purpose == Ordinary {
		return ordinary, nil
	}
	return Storage{
		FavoritesDir: filepath.Join(p.trialDir, "favorites"),
		PresetsDir:   filepath.Join(p.trialDir, "presets"),
		AnalysisDir:  filepath.Join(p.trialDir, "image-analysis"),
		UpdatesDir:   filepath.Join(p.trialDir, "updates"),
	}, nil
}
