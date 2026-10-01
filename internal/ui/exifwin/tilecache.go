package exifwin

import (
	"image"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type cachedTile struct {
	data     []byte
	pixels   *image.NRGBA
	header   http.Header
	expires  time.Time
	received time.Time
	noStore  bool
}

func tileWeight(entry *cachedTile) int64 {
	size := int64(len(entry.data))
	if entry.pixels != nil {
		size += int64(len(entry.pixels.Pix))
	}
	for name, values := range entry.header {
		for _, value := range values {
			size += int64(len(name) + len(value) + 16)
		}
	}
	return size
}

func cacheTile(data []byte, previous, response http.Header, now time.Time) *cachedTile {
	header := make(http.Header)
	size := 0
	for _, name := range []string{"Cache-Control", "Expires", "Date", "Age", "ETag", "Last-Modified"} {
		values, exists := response[http.CanonicalHeaderKey(name)]
		if !exists && name != "Date" && name != "Age" {
			values = previous.Values(name)
		}
		for _, value := range values {
			size += len(name) + len(value) + 16
			if size > 16*1024 {
				return &cachedTile{data: data, noStore: true, received: now}
			}
			header.Add(name, strings.Clone(value))
		}
	}
	entry := &cachedTile{data: data, header: header, received: now}
	age := time.Duration(0)
	if seconds, err := strconv.ParseInt(header.Get("Age"), 10, 64); err == nil && seconds > 0 {
		age = time.Duration(min(seconds, int64((1<<63-1)/time.Second))) * time.Second
	}
	date, dateErr := http.ParseTime(header.Get("Date"))
	if dateErr == nil {
		age = max(age, now.Sub(date))
	}
	lifetime := 7 * 24 * time.Hour
	if _, present := header["Expires"]; present {
		lifetime = 0
		if expires, err := http.ParseTime(header.Get("Expires")); err == nil {
			if dateErr == nil {
				lifetime = expires.Sub(date)
			} else {
				lifetime = expires.Sub(now)
			}
		}
	}
	noCache, maxAgeSeen := false, false
	for _, value := range header.Values("Cache-Control") {
		for _, directive := range strings.Split(value, ",") {
			name, argument, _ := strings.Cut(strings.TrimSpace(directive), "=")
			switch strings.ToLower(strings.TrimSpace(name)) {
			case "no-store":
				entry.noStore = true
			case "no-cache":
				noCache = true
			case "max-age":
				if maxAgeSeen {
					noCache = true
				}
				maxAgeSeen = true
				argument = strings.TrimSpace(argument)
				if len(argument) >= 2 && argument[0] == '"' && argument[len(argument)-1] == '"' {
					argument = argument[1 : len(argument)-1]
				}
				invalidDigit := strings.IndexFunc(argument, func(value rune) bool { return value < '0' || value > '9' }) >= 0
				seconds, err := strconv.ParseInt(argument, 10, 64)
				if err != nil || invalidDigit || seconds < 0 || seconds > int64((1<<63-1)/time.Second) {
					noCache = true
				} else {
					lifetime = time.Duration(seconds) * time.Second
				}
			}
		}
	}
	entry.expires = now
	if !noCache && lifetime > age {
		entry.expires = now.Add(lifetime - age)
	}
	return entry
}

func freshTileLocked(fetcher *tileFetcher, url string) bool {
	entry, ok := fetcher.cache.Get(url)
	return ok && fetcher.now().Before(entry.expires)
}
