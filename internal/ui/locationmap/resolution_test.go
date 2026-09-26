package locationmap

import (
	"context"
	"math"
	"reflect"
	"testing"

	"github.com/frathe/picfetch/internal/imaging"
)

// Cancel at a deterministic checkpoint, without racing a timer or a worker.
type resolutionCancelContext struct {
	context.Context
	cancel context.CancelFunc
	checks int
}

func (c *resolutionCancelContext) Err() error {
	c.checks--
	if c.checks <= 0 {
		c.cancel()
	}
	return c.Context.Err()
}

func TestResolveLocationCancellation(t *testing.T) {
	donors := make([]LocationCandidate, 2000)
	for i := range donors {
		donors[i] = LocationCandidate{Index: i, Metadata: imaging.Metadata{HasGPS: true, Latitude: float64(i) / 3000000}}
	}
	for _, checks := range []int{1, len(donors) + 10} {
		ctx, cancel := context.WithCancel(context.Background())
		controlled := &resolutionCancelContext{Context: ctx, cancel: cancel, checks: checks}
		got := ResolveLocation(controlled, LocationCandidate{}, donors)
		canceled := ctx.Err() != nil
		cancel()
		if !canceled || got.Outcome != LocationUnreadable || got.DonorIndex != -1 || got.Metadata.HasGPS {
			t.Errorf("agreement did not observe cancellation at checkpoint %d: canceled=%v result=%+v", checks, canceled, got)
		}
	}
}

func TestResolveLocationBoundsDistinctDonorAgreement(t *testing.T) {
	donors := make([]LocationCandidate, 2000)
	for i := range donors {
		angle := 2 * math.Pi * float64(i) / float64(len(donors))
		degrees := 49.0 / 6371008.8 * 180 / math.Pi
		donors[i] = LocationCandidate{Index: i, Metadata: imaging.Metadata{
			HasGPS: true, Latitude: degrees * math.Sin(angle), Longitude: degrees * math.Cos(angle),
		}}
	}
	got := ResolveLocation(context.Background(), LocationCandidate{}, donors)
	if got.Outcome != LocationUnreadable || got.DonorIndex != -1 || got.Metadata.HasGPS {
		t.Fatalf("exhausted agreement must leave GPS unproven: %+v", got)
	}
}

func TestResolveLocationLargeRepeatedCoordinates(t *testing.T) {
	donors := make([]LocationCandidate, 2000)
	for i := range donors {
		donors[i] = LocationCandidate{Index: i, PixelCount: int64(i), Metadata: imaging.Metadata{HasGPS: true, Latitude: 52.52, Longitude: 13.405}}
	}
	got := ResolveLocation(context.Background(), LocationCandidate{}, donors)
	if got.Outcome != LocationLocated || got.DonorIndex != 1999 || got.Metadata.Latitude != 52.52 || got.Metadata.Longitude != 13.405 {
		t.Fatalf("repeated coordinates consumed agreement budget or lost best donor: %+v", got)
	}
}

func TestResolveLocationLargeTightGroup(t *testing.T) {
	donors := make([]LocationCandidate, 2000)
	for i := range donors {
		donors[i] = LocationCandidate{Index: i, Metadata: imaging.Metadata{HasGPS: true, Latitude: float64(i) / 10000000}}
	}
	got := ResolveLocation(context.Background(), LocationCandidate{}, donors)
	if got.Outcome != LocationLocated || got.DonorIndex != 0 || !got.Metadata.HasGPS {
		t.Fatalf("tightly grouped coordinates exhausted agreement: %+v", got)
	}
}

func TestResolveLocationKeepsRepresentativeGPS(t *testing.T) {
	representative := LocationCandidate{Index: 3, ReadError: true, Metadata: imaging.Metadata{DateTaken: "representative", Latitude: 12, Longitude: 34, HasGPS: true}}
	others := []LocationCandidate{{Index: 4, ReadError: true}, {Index: 5, Metadata: imaging.Metadata{Latitude: -20, Longitude: -30, HasGPS: true}}}

	got := ResolveLocation(context.Background(), representative, others)
	if got.Outcome != LocationLocated || got.DonorIndex != -1 || got.Metadata != representative.Metadata {
		t.Fatalf("representative GPS must win without inspecting donors: %+v", got)
	}
}

func TestResolveLocationMissingAndSingleDonor(t *testing.T) {
	representative := LocationCandidate{Index: 1, Metadata: imaging.Metadata{DateTaken: "representative"}}
	if got := ResolveLocation(context.Background(), representative, nil); got.Outcome != LocationUnlocated || got.DonorIndex != -1 || got.Metadata != representative.Metadata {
		t.Fatalf("no donor should leave representative unlocated: %+v", got)
	}

	donor := LocationCandidate{Index: 8, PixelCount: 100, Metadata: imaging.Metadata{DateTaken: "donor", Latitude: 52.5, Longitude: 13.4, HasGPS: true}}
	got := ResolveLocation(context.Background(), representative, []LocationCandidate{donor})
	if got.Outcome != LocationLocated || got.DonorIndex != donor.Index {
		t.Fatalf("one valid donor should locate representative: %+v", got)
	}
	want := representative.Metadata
	want.Latitude, want.Longitude, want.HasGPS = donor.Metadata.Latitude, donor.Metadata.Longitude, true
	if got.Metadata != want {
		t.Fatalf("borrow only GPS fields; got %+v, want %+v", got.Metadata, want)
	}
}

func TestResolveLocationUnreadableCandidatePreventsDonorInference(t *testing.T) {
	valid := LocationCandidate{Index: 2, Metadata: imaging.Metadata{Latitude: 1, Longitude: 2, HasGPS: true}}
	for _, tc := range []struct {
		name           string
		representative LocationCandidate
		others         []LocationCandidate
	}{
		{"representative unreadable", LocationCandidate{Index: 1, ReadError: true}, []LocationCandidate{valid}},
		{"other unreadable", LocationCandidate{Index: 1}, []LocationCandidate{valid, {Index: 3, ReadError: true}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveLocation(context.Background(), tc.representative, tc.others)
			if got.Outcome != LocationUnreadable || got.DonorIndex != -1 || got.Metadata != tc.representative.Metadata {
				t.Fatalf("unknown candidate location prevents inference: %+v", got)
			}
		})
	}
}

func TestResolveLocationDonorsMustAgreePairwise(t *testing.T) {
	near := func(index int, meters float64) LocationCandidate {
		return LocationCandidate{Index: index, Metadata: imaging.Metadata{Latitude: meters / 6371008.8 * 180 / math.Pi, HasGPS: true}}
	}
	for _, tc := range []struct {
		name    string
		others  []LocationCandidate
		outcome LocationOutcome
	}{
		{"exactly 100 meters", []LocationCandidate{near(2, 0), near(3, 100)}, LocationLocated},
		{"within roundoff tolerance", []LocationCandidate{near(2, 0), near(3, 100.0000005)}, LocationLocated},
		{"beyond roundoff tolerance", []LocationCandidate{near(2, 0), near(3, 100.000002)}, LocationConflict},
		{"over 100 meters", []LocationCandidate{near(2, 0), near(3, 100.001)}, LocationConflict},
		{"chain with distant endpoints", []LocationCandidate{near(2, 0), near(3, 70), near(4, 140)}, LocationConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveLocation(context.Background(), LocationCandidate{Index: 1}, tc.others)
			if got.Outcome != tc.outcome {
				t.Fatalf("outcome = %v, want %v", got.Outcome, tc.outcome)
			}
			if tc.outcome != LocationLocated && got.DonorIndex != -1 {
				t.Fatalf("conflict cannot select donor: %+v", got)
			}
		})
	}
}

func TestResolveLocationDatelineAndDeterministicDonor(t *testing.T) {
	representative := LocationCandidate{Index: 1, Metadata: imaging.Metadata{DateTaken: "representative"}}
	others := []LocationCandidate{
		{Index: 8, PixelCount: 10, Metadata: imaging.Metadata{Latitude: 0, Longitude: 179.9998, HasGPS: true}},
		{Index: 7, PixelCount: 20, Metadata: imaging.Metadata{Latitude: 0, Longitude: -179.9998, HasGPS: true}},
		{Index: 4, PixelCount: 20, Metadata: imaging.Metadata{Latitude: 0, Longitude: 180, HasGPS: true}},
	}
	wantOthers := append([]LocationCandidate(nil), others...)
	got := ResolveLocation(context.Background(), representative, others)
	if got.Outcome != LocationLocated || got.DonorIndex != 4 || got.Metadata.DateTaken != "representative" || got.Metadata.Longitude != 180 {
		t.Fatalf("world wrap and ranking: %+v", got)
	}
	if !reflect.DeepEqual(others, wantOthers) || representative.Metadata.DateTaken != "representative" {
		t.Fatal("resolution changed inputs")
	}
}

func TestResolveLocationIgnoresInvalidCoordinates(t *testing.T) {
	invalid := []imaging.Metadata{
		{Latitude: math.NaN(), HasGPS: true},
		{Longitude: math.Inf(1), HasGPS: true},
		{Latitude: 90.0001, HasGPS: true},
		{Longitude: -180.0001, HasGPS: true},
		{Latitude: 0, Longitude: 0, HasGPS: false},
	}
	representative := LocationCandidate{Index: 1, Metadata: imaging.Metadata{Latitude: 91, HasGPS: true}}
	for _, metadata := range invalid {
		got := ResolveLocation(context.Background(), representative, []LocationCandidate{{Index: 2, Metadata: metadata}})
		if got.Outcome != LocationUnlocated || got.DonorIndex != -1 {
			t.Fatalf("invalid GPS must be absent: %+v", got)
		}
	}
}
