// Package display owns single-image presentation: source-bound observations and
// captures, surface publication, rotation/fades, GIF playback, SVG sharpening,
// foreground loading and bounded preloads. Root composes its surface with zoom
// and retains collection, window and cross-feature policy.
package display

func normalizedRotation(steps int) int { return ((steps % 4) + 4) % 4 }

// Count reports the number of decoded frames, without exposing their storage.
func (f *Feature) Count() int { return len(f.frames) }

// Rotation reports the pending clockwise quarter turns.
func (f *Feature) Rotation() int { return f.rotation }
