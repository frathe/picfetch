package locationmap

import (
	"context"
	"math"

	"github.com/frathe/picfetch/internal/imaging"
)

type LocationCandidate struct {
	Index      int
	PixelCount int64
	Metadata   imaging.Metadata
	ReadError  bool
}

type LocationOutcome int

const (
	LocationLocated LocationOutcome = iota
	LocationUnlocated
	LocationUnreadable
	LocationConflict
)

type LocationResolution struct {
	Metadata   imaging.Metadata
	DonorIndex int
	Outcome    LocationOutcome
}

// Refuse unproven fallback rather than hold a worker on quadratic agreement.
const maxLocationComparisons = 1000000

// ResolveLocation uses a representative's location, or a location on which all
// readable donors agree. It leaves non-location metadata with the representative.
// Canceled or over-budget agreement stays unreadable, without borrowing GPS.
func ResolveLocation(ctx context.Context, representative LocationCandidate, others []LocationCandidate) LocationResolution {
	result := LocationResolution{Metadata: representative.Metadata, DonorIndex: -1}
	if ctx.Err() != nil {
		result.Outcome = LocationUnreadable
		return result
	}
	if validLocation(representative.Metadata) {
		result.Outcome = LocationLocated
		return result
	}
	if representative.ReadError {
		result.Outcome = LocationUnreadable
		return result
	}

	var donors []LocationCandidate
	var best LocationCandidate
	seen := make(map[[2]float64]bool)
	for _, candidate := range others {
		if ctx.Err() != nil || candidate.ReadError {
			result.Outcome = LocationUnreadable
			return result
		}
		if validLocation(candidate.Metadata) {
			if len(donors) == 0 || candidate.PixelCount > best.PixelCount || (candidate.PixelCount == best.PixelCount && candidate.Index < best.Index) {
				best = candidate
			}
			position := [2]float64{candidate.Metadata.Latitude, candidate.Metadata.Longitude}
			if !seen[position] {
				seen[position] = true
				donors = append(donors, candidate)
			}
		}
	}
	if len(donors) == 0 {
		result.Outcome = LocationUnlocated
		return result
	}
	// Every pair is at most the sum of its distances to this anchor. This
	// proves tight groups in linear work, even with many distinct coordinates.
	tight := true
	for _, donor := range donors[1:] {
		if ctx.Err() != nil {
			result.Outcome = LocationUnreadable
			return result
		}
		if locationDistance(donors[0].Metadata, donor.Metadata) > 50 {
			tight = false
			break
		}
	}
	comparisons := 0
	for i, first := range donors {
		if tight {
			break
		}
		for _, second := range donors[i+1:] {
			if ctx.Err() != nil || comparisons >= maxLocationComparisons {
				result.Outcome = LocationUnreadable
				return result
			}
			comparisons++
			if locationDistance(first.Metadata, second.Metadata) > 100+0.000001 {
				result.Outcome = LocationConflict
				return result
			}
		}
	}

	result.Metadata.Latitude = best.Metadata.Latitude
	result.Metadata.Longitude = best.Metadata.Longitude
	result.Metadata.HasGPS = true
	result.DonorIndex = best.Index
	result.Outcome = LocationLocated
	return result
}

func validLocation(metadata imaging.Metadata) bool {
	return metadata.HasGPS &&
		!math.IsNaN(metadata.Latitude) && !math.IsInf(metadata.Latitude, 0) && metadata.Latitude >= -90 && metadata.Latitude <= 90 &&
		!math.IsNaN(metadata.Longitude) && !math.IsInf(metadata.Longitude, 0) && metadata.Longitude >= -180 && metadata.Longitude <= 180
}

func locationDistance(first, second imaging.Metadata) float64 {
	const earthRadius = 6371008.8
	latitude1 := first.Latitude * math.Pi / 180
	latitude2 := second.Latitude * math.Pi / 180
	latitudeDelta := (second.Latitude - first.Latitude) * math.Pi / 180
	longitudeDelta := (second.Longitude - first.Longitude) * math.Pi / 180
	sinLatitude := math.Sin(latitudeDelta / 2)
	sinLongitude := math.Sin(longitudeDelta / 2)
	haversine := sinLatitude*sinLatitude + math.Cos(latitude1)*math.Cos(latitude2)*sinLongitude*sinLongitude
	haversine = math.Max(0, math.Min(1, haversine))
	return 2 * earthRadius * math.Atan2(math.Sqrt(haversine), math.Sqrt(1-haversine))
}
