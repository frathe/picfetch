package heic

import "fmt"

// validateDarwinContainer keeps the still-image codec boundary while leaving
// bit depth and color interpretation to ImageIO.
func validateDarwinContainer(data []byte) error {
	itemType, _, err := primaryItemProperties(data)
	if err != nil {
		return err
	}
	if itemType != "hvc1" && itemType != "grid" {
		return fmt.Errorf("%w: primary item is not an HEVC still or grid", ErrUnsupported)
	}
	return nil
}
