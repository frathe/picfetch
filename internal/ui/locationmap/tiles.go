package locationmap

import (
	"bytes"
	"container/list"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	maxTileResponse = 1024 * 1024
	defaultEncoded  = 32 * 1024 * 1024
	defaultDecoded  = 64 * 1024 * 1024
	maxTileFailures = 1024
)

// TileKey identifies one OpenStreetMap raster tile.
type TileKey struct{ Z, X, Y int }

// TileOptions configures an in-memory tile store. URL is a fmt.Sprintf template
// with three integer placeholders, in Z, X, Y order.
type TileOptions struct {
	Client       *http.Client
	Now          func() time.Time
	URL          string
	EncodedBytes int64
	DecodedBytes int64
}

type tileEntry struct {
	encoded      []byte
	decoded      image.Image
	expires      time.Time
	etag         string
	lastModified string
	freshness    http.Header
	encodedNode  *list.Element
	decodedNode  *list.Element
}

type tileFailure struct {
	err   error
	retry time.Time
	count int
	node  *list.Element
}

// TileStore retains bounded encoded and decoded tiles. It owns no workers.
type TileStore struct {
	client       *http.Client
	now          func() time.Time
	url          string
	encodedLimit int64
	decodedLimit int64
	mu           sync.Mutex
	entries      map[TileKey]*tileEntry
	encodedLRU   list.List
	decodedLRU   list.List
	encodedBytes int64
	decodedBytes int64
	failures     map[TileKey]*tileFailure
	failureLRU   list.List
}

// NewTileStore constructs a goroutine-safe, memory-only tile store.
func NewTileStore(options TileOptions) *TileStore {
	if options.Client == nil {
		options.Client = &http.Client{Timeout: 15 * time.Second}
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.URL == "" {
		options.URL = "https://tile.openstreetmap.org/%d/%d/%d.png"
	}
	if options.EncodedBytes <= 0 {
		options.EncodedBytes = defaultEncoded
	}
	if options.DecodedBytes <= 0 {
		options.DecodedBytes = defaultDecoded
	}
	return &TileStore{client: options.Client, now: options.Now, url: options.URL,
		encodedLimit: options.EncodedBytes, decodedLimit: options.DecodedBytes,
		entries: make(map[TileKey]*tileEntry), failures: make(map[TileKey]*tileFailure)}
}

// Usage reports bytes retained by the encoded and decoded LRUs.
func (s *TileStore) Usage() (encodedBytes, decodedBytes int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.encodedBytes, s.decodedBytes
}

// Fetch returns a tile or the earliest retry time after a failure.
func (s *TileStore) Fetch(ctx context.Context, key TileKey) (image.Image, time.Time, error) {
	if err := ctx.Err(); err != nil {
		return nil, time.Time{}, err
	}
	now := s.now()
	s.mu.Lock()
	if entry := s.entries[key]; entry != nil && now.Before(entry.expires) {
		if entry.decoded != nil {
			s.decodedLRU.MoveToFront(entry.decodedNode)
			img := entry.decoded
			s.mu.Unlock()
			return img, time.Time{}, nil
		}
		if entry.encoded != nil {
			s.encodedLRU.MoveToFront(entry.encodedNode)
			encoded := entry.encoded
			s.mu.Unlock()
			return s.decodeCached(ctx, key, encoded)
		}
	}
	if failure := s.failures[key]; failure != nil && now.Before(failure.retry) {
		s.failureLRU.MoveToFront(failure.node)
		retry, err := failure.retry, failure.err
		s.mu.Unlock()
		return nil, retry, err
	}
	var etag, modified string
	if entry := s.entries[key]; entry != nil {
		etag, modified = entry.etag, entry.lastModified
	}
	s.mu.Unlock()

	url := fmt.Sprintf(s.url, key.Z, key.X, key.Y)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return s.failed(ctx, key, now, time.Time{}, err)
	}
	req.Header.Set("User-Agent", "PicFetch (+https://github.com/frathe/picfetch)")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if modified != "" {
		req.Header.Set("If-Modified-Since", modified)
	}
	response, err := s.client.Do(req)
	if err != nil {
		return s.failed(ctx, key, s.now(), time.Time{}, err)
	}
	defer func() { _ = response.Body.Close() }()
	now = s.now()
	if response.StatusCode == http.StatusNotModified {
		return s.notModified(ctx, key, response.Header, now)
	}
	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("tile HTTP status %d", response.StatusCode)
		return s.failed(ctx, key, now, retryAfter(response.Header.Get("Retry-After"), now), err)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxTileResponse+1))
	if err == nil && len(data) > maxTileResponse {
		err = errors.New("tile response exceeds 1 MiB")
	}
	if err != nil {
		return s.failed(ctx, key, now, time.Time{}, err)
	}
	img, err := decodeTile(data)
	if err != nil {
		return s.failed(ctx, key, now, time.Time{}, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, time.Time{}, err
	}
	expires, noStore := tileFreshness(response.Header, now)
	s.mu.Lock()
	if err := ctx.Err(); err != nil {
		s.mu.Unlock()
		return nil, time.Time{}, err
	}
	s.clearFailure(key)
	if noStore {
		s.dropEntry(key)
	} else {
		s.dropEntry(key)
		entry := &tileEntry{expires: expires, etag: response.Header.Get("ETag"), lastModified: response.Header.Get("Last-Modified"), freshness: response.Header.Clone()}
		s.entries[key] = entry
		s.addEncoded(key, entry, data)
		s.addDecoded(key, entry, img)
		s.discardEmpty(key, entry)
	}
	s.mu.Unlock()
	return img, time.Time{}, nil
}

func (s *TileStore) decodeCached(ctx context.Context, key TileKey, encoded []byte) (image.Image, time.Time, error) {
	img, err := decodeTile(encoded)
	if err != nil {
		return s.failed(ctx, key, s.now(), time.Time{}, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, time.Time{}, err
	}
	s.mu.Lock()
	if err := ctx.Err(); err != nil {
		s.mu.Unlock()
		return nil, time.Time{}, err
	}
	if entry := s.entries[key]; entry != nil && bytes.Equal(entry.encoded, encoded) {
		s.addDecoded(key, entry, img)
	}
	s.mu.Unlock()
	return img, time.Time{}, nil
}

func (s *TileStore) notModified(ctx context.Context, key TileKey, header http.Header, now time.Time) (image.Image, time.Time, error) {
	s.mu.Lock()
	entry := s.entries[key]
	if entry == nil || (entry.encoded == nil && entry.decoded == nil) {
		s.mu.Unlock()
		return s.failed(ctx, key, now, time.Time{}, errors.New("tile returned 304 without cached pixels"))
	}
	img, data := entry.decoded, entry.encoded
	merged := entry.freshness.Clone()
	merged.Del("Age")
	merged.Del("Date")
	for key, values := range header {
		merged[key] = append([]string(nil), values...)
	}
	s.mu.Unlock()
	if img == nil {
		var err error
		img, err = decodeTile(data)
		if err != nil {
			return s.failed(ctx, key, now, time.Time{}, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, time.Time{}, err
	}
	expires, noStore := tileFreshness(merged, now)
	s.mu.Lock()
	if err := ctx.Err(); err != nil {
		s.mu.Unlock()
		return nil, time.Time{}, err
	}
	s.clearFailure(key)
	if current := s.entries[key]; current == entry {
		if noStore {
			s.dropEntry(key)
		} else {
			entry.expires = expires
			entry.freshness = merged
			if value := header.Get("ETag"); value != "" {
				entry.etag = value
			}
			if value := header.Get("Last-Modified"); value != "" {
				entry.lastModified = value
			}
			if entry.decoded == nil {
				s.addDecoded(key, entry, img)
			}
		}
	}
	s.mu.Unlock()
	return img, time.Time{}, nil
}

func decodeTile(data []byte) (image.Image, error) {
	if len(data) < 24 || !bytes.Equal(data[:8], []byte("\x89PNG\r\n\x1a\n")) ||
		binary.BigEndian.Uint32(data[8:12]) != 13 || string(data[12:16]) != "IHDR" ||
		binary.BigEndian.Uint32(data[16:20]) != 256 || binary.BigEndian.Uint32(data[20:24]) != 256 {
		return nil, errors.New("tile is not a 256x256 PNG")
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	// PNG can decode to an eight-byte-per-pixel 16-bit image. Normalize every
	// tile to the four-byte representation charged by the decoded budget.
	pixels := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	draw.Draw(pixels, pixels.Bounds(), img, img.Bounds().Min, draw.Src)
	return pixels, nil
}

func tileFreshness(header http.Header, now time.Time) (time.Time, bool) {
	var maxAge *time.Duration
	noStore, noCache := false, false
	ageSeconds, _ := strconv.ParseInt(header.Get("Age"), 10, 64)
	if ageSeconds < 0 {
		ageSeconds = 0
	}
	const maxSeconds = int64(1<<63-1) / int64(time.Second)
	if ageSeconds > maxSeconds {
		ageSeconds = maxSeconds
	}
	age := time.Duration(ageSeconds) * time.Second
	for _, value := range header.Values("Cache-Control") {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(strings.ToLower(part))
			switch part {
			case "no-store":
				noStore = true
			case "no-cache":
				noCache = true
			default:
				if raw, ok := strings.CutPrefix(part, "max-age="); ok {
					seconds, err := strconv.ParseInt(strings.Trim(raw, `"`), 10, 64)
					if err == nil && seconds >= 0 && seconds <= maxSeconds {
						duration := time.Duration(seconds) * time.Second
						maxAge = &duration
					}
				}
			}
		}
	}
	if noCache {
		return now, noStore
	}
	if maxAge != nil {
		return now.Add(*maxAge - age), noStore
	}
	if expires, err := http.ParseTime(header.Get("Expires")); err == nil {
		if date, err := http.ParseTime(header.Get("Date")); err == nil {
			return now.Add(expires.Sub(date) - age), noStore
		}
		return expires.Add(-age), noStore
	}
	return now.Add(7 * 24 * time.Hour), noStore
}

func retryAfter(value string, now time.Time) time.Time {
	if seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil && seconds >= 0 {
		seconds = min(seconds, int64((1<<63-1)/time.Second))
		return now.Add(time.Duration(seconds) * time.Second)
	}
	if date, err := http.ParseTime(value); err == nil {
		return date
	}
	return time.Time{}
}

func (s *TileStore) failed(ctx context.Context, key TileKey, now, serverRetry time.Time, err error) (image.Image, time.Time, error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, time.Time{}, ctxErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, time.Time{}, ctxErr
	}
	failure := s.failures[key]
	if failure == nil {
		failure = &tileFailure{node: s.failureLRU.PushFront(key)}
		s.failures[key] = failure
	} else {
		s.failureLRU.MoveToFront(failure.node)
	}
	failure.count++
	backoff := time.Second << min(failure.count-1, 6)
	if backoff > time.Minute {
		backoff = time.Minute
	}
	failure.retry = now.Add(backoff)
	if serverRetry.After(failure.retry) {
		failure.retry = serverRetry
	}
	failure.err = err
	for oldest := s.failureLRU.Back(); oldest != nil && len(s.failures) > maxTileFailures; oldest = s.failureLRU.Back() {
		delete(s.failures, oldest.Value.(TileKey))
		s.failureLRU.Remove(oldest)
	}
	return nil, failure.retry, err
}

func (s *TileStore) clearFailure(key TileKey) {
	if failure := s.failures[key]; failure != nil {
		delete(s.failures, key)
		s.failureLRU.Remove(failure.node)
	}
}

func (s *TileStore) addEncoded(key TileKey, entry *tileEntry, data []byte) {
	if int64(len(data)) > s.encodedLimit {
		return
	}
	entry.encoded = data
	entry.encodedNode = s.encodedLRU.PushFront(key)
	s.encodedBytes += int64(len(data))
	for oldest := s.encodedLRU.Back(); oldest != nil && s.encodedBytes > s.encodedLimit; oldest = s.encodedLRU.Back() {
		oldKey := oldest.Value.(TileKey)
		old := s.entries[oldKey]
		s.encodedBytes -= int64(len(old.encoded))
		old.encoded, old.encodedNode = nil, nil
		s.encodedLRU.Remove(oldest)
		s.discardEmpty(oldKey, old)
	}
}

func (s *TileStore) addDecoded(key TileKey, entry *tileEntry, img image.Image) {
	const size = 256 * 256 * 4
	if size > s.decodedLimit {
		return
	}
	if entry.decodedNode != nil {
		s.decodedLRU.MoveToFront(entry.decodedNode)
		return
	}
	entry.decoded = img
	entry.decodedNode = s.decodedLRU.PushFront(key)
	s.decodedBytes += size
	for oldest := s.decodedLRU.Back(); oldest != nil && s.decodedBytes > s.decodedLimit; oldest = s.decodedLRU.Back() {
		oldKey := oldest.Value.(TileKey)
		old := s.entries[oldKey]
		s.decodedBytes -= size
		old.decoded, old.decodedNode = nil, nil
		s.decodedLRU.Remove(oldest)
		s.discardEmpty(oldKey, old)
	}
}

func (s *TileStore) dropEntry(key TileKey) {
	entry := s.entries[key]
	if entry == nil {
		return
	}
	if entry.encodedNode != nil {
		s.encodedBytes -= int64(len(entry.encoded))
		s.encodedLRU.Remove(entry.encodedNode)
	}
	if entry.decodedNode != nil {
		s.decodedBytes -= 256 * 256 * 4
		s.decodedLRU.Remove(entry.decodedNode)
	}
	delete(s.entries, key)
}

func (s *TileStore) discardEmpty(key TileKey, entry *tileEntry) {
	if entry.encoded == nil && entry.decoded == nil {
		delete(s.entries, key)
	}
}
