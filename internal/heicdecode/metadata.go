package heicdecode

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"time"
	"unicode/utf8"
)

// Metadata contains normalized values only, never embedded Exif/TIFF bytes.
// Orientation describes the Exif tag; container transforms are already applied
// by the guest. The integration must resolve their precedence before exposure.
type Metadata struct {
	Orientation      int
	Make             string
	Model            string
	Software         string
	DateTime         string
	DateTimeOriginal string
	ExposureTime     float64
	FNumber          float64
	ISOSpeed         int
	FocalLength      float64
	Flash            int
	GPSLatitude      float64
	GPSLongitude     float64
	GPSAltitude      float64
	Copyright        string
	Artist           string
}

func (m Metadata) validate() error {
	for _, s := range []string{m.Make, m.Model, m.Software, m.DateTime, m.DateTimeOriginal, m.Copyright, m.Artist} {
		if len(s) > 1024 || !utf8.ValidString(s) {
			return errors.New("invalid metadata text")
		}
	}
	for _, n := range []float64{m.ExposureTime, m.FNumber, m.FocalLength, m.GPSLatitude, m.GPSLongitude, m.GPSAltitude} {
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return errors.New("non-finite metadata value")
		}
	}
	for _, date := range []string{m.DateTime, m.DateTimeOriginal} {
		if date != "" {
			if _, err := time.Parse("2006:01:02 15:04:05", date); err != nil {
				return errors.New("invalid metadata date")
			}
		}
	}
	if m.Orientation < 0 || m.Orientation > 8 || m.ExposureTime < 0 || m.FNumber < 0 || m.FocalLength < 0 || m.ISOSpeed < 0 || m.ISOSpeed > 65535 || m.Flash < 0 || m.Flash > 65535 || m.GPSLatitude < -90 || m.GPSLatitude > 90 || m.GPSLongitude < -180 || m.GPSLongitude > 180 {
		return errors.New("metadata value outside its range")
	}
	return nil
}

func readMetadata(data []byte) (*Metadata, error) {
	if len(data) == 0 {
		return nil, nil
	}
	if !utf8.Valid(data) {
		return nil, errors.New("invalid metadata encoding")
	}
	// encoding/json accepts duplicate keys; reject them before decoding the fixed
	// schema. Raw values remain bounded by the already checked metadata length.
	d := json.NewDecoder(bytes.NewReader(data))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return nil, errors.New("metadata must be an object")
	}
	seen := make(map[string]bool)
	for d.More() {
		token, err = d.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok || seen[key] {
			return nil, errors.New("duplicate metadata field")
		}
		switch key {
		case "Orientation", "Make", "Model", "Software", "DateTime", "DateTimeOriginal", "ExposureTime", "FNumber", "ISOSpeed", "FocalLength", "Flash", "GPSLatitude", "GPSLongitude", "GPSAltitude", "Copyright", "Artist":
		default:
			return nil, errors.New("unknown metadata field")
		}
		seen[key] = true
		var value json.RawMessage
		if err = d.Decode(&value); err != nil {
			return nil, err
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, errors.New("null metadata field")
		}
	}
	var m Metadata
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err = d.Decode(&m); err != nil {
		return nil, err
	}
	var extra any
	if err = d.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing metadata")
	}
	if err = m.validate(); err != nil {
		return nil, err
	}
	return &m, nil
}
