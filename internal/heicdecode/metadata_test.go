package heicdecode

import "testing"

func TestMetadataRejectsUnvalidatedFields(t *testing.T) {
	for _, data := range []string{
		`{"Orientation":9}`, `{"GPSLatitude":91}`, `{"GPSLongitude":-181}`, `{"FNumber":-1}`,
		`{"DateTime":"yesterday"}`, `{"Make":null}`, `{"Orientation":1,"orientation":8}`, `{"Unknown":true}`, `{"Orientation":1,"Orientation":8}`, `{"Make":"ok"} {}`,
	} {
		if _, err := readMetadata([]byte(data)); err == nil {
			t.Fatalf("accepted %s", data)
		}
	}
	got, err := readMetadata([]byte(`{"Orientation":6,"GPSLatitude":0,"GPSLongitude":0,"DateTimeOriginal":"2026:09:15 12:30:00","Make":"Camera"}`))
	if err != nil || got == nil || got.Orientation != 6 || got.Make != "Camera" {
		t.Fatalf("metadata = %+v, %v", got, err)
	}
}
