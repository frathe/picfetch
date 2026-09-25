package locationmap

import (
	"context"
	"sync"

	"github.com/frathe/picfetch/internal/imaging"
)

// Fact is completed raw GPS metadata for one source version.
type Fact struct {
	Version  string
	Metadata imaging.Metadata
}

// locationMetadata keeps only the fixed-size fields used by the map. Camera
// and other EXIF strings can be large and belong to the EXIF view, not this cache.
func locationMetadata(metadata imaging.Metadata) imaging.Metadata {
	return imaging.Metadata{HasGPS: metadata.HasGPS, Latitude: metadata.Latitude, Longitude: metadata.Longitude}
}

type factEntry struct {
	version  string
	revision uint64
	fact     *Fact
}

// FactCache retains completed metadata only for current collection members.
type FactCache struct {
	mu      sync.Mutex
	entries map[string]*factEntry
}

func NewFactCache() *FactCache {
	return &FactCache{entries: make(map[string]*factEntry)}
}

// Keep replaces the live membership and retains facts for surviving sources.
func (c *FactCache) Keep(keys []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	next := make(map[string]*factEntry, len(keys))
	for _, key := range keys {
		if entry := c.entries[key]; entry != nil {
			next[key] = entry
		} else if next[key] == nil {
			next[key] = &factEntry{}
		}
	}
	c.entries = next
}

func (c *FactCache) Get(key, version string) (imaging.Metadata, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry := c.entries[key]; entry != nil && entry.fact != nil && entry.fact.Version == version {
		return entry.fact.Metadata, true
	}
	return imaging.Metadata{}, false
}

// FactWriter is authority captured for one member and version.
type FactWriter struct {
	cache    *FactCache
	key      string
	version  string
	entry    *factEntry
	revision uint64
}

func (c *FactCache) Capture(key, version string) FactWriter {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := c.entries[key]
	if entry == nil || version == "" {
		return FactWriter{}
	}
	if entry.version != version {
		entry.version = version
		entry.revision++
		entry.fact = nil
	}
	return FactWriter{cache: c, key: key, version: version, entry: entry, revision: entry.revision}
}

// Store publishes a successful metadata read while its capture remains current.
func (w FactWriter) Store(ctx context.Context, metadata imaging.Metadata) bool {
	if w.cache == nil {
		return false
	}
	w.cache.mu.Lock()
	defer w.cache.mu.Unlock()

	entry := w.cache.entries[w.key]
	if entry != w.entry || entry.version != w.version || entry.revision != w.revision || ctx.Err() != nil {
		return false
	}
	entry.fact = &Fact{Version: w.version, Metadata: locationMetadata(metadata)}
	return true
}

// Invalidate retires completed facts while preserving membership.
func (c *FactCache) Invalidate(keys []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, key := range keys {
		if entry := c.entries[key]; entry != nil {
			entry.fact = nil
			entry.revision++
		}
	}
}

// Snapshot copies the completed facts without exposing cache storage.
func (c *FactCache) Snapshot() map[string]Fact {
	c.mu.Lock()
	defer c.mu.Unlock()

	facts := make(map[string]Fact)
	for key, entry := range c.entries {
		if entry.fact != nil {
			facts[key] = *entry.fact
		}
	}
	return facts
}
