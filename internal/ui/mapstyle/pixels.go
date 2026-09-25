// Package mapstyle applies the same local color treatment to both OSM maps.
package mapstyle

import (
	"image"
	"image/color"

	"fyne.io/fyne/v2/theme"
)

// darkPixels invert luminance, soften chroma,
// and give neutral land a cool charcoal tint. Keep the original provider pixels
// immutable and share them with light mode, without another pixel cache.
type darkPixels struct{ image.Image }

func (darkPixels) ColorModel() color.Model { return color.NRGBAModel }

func (tile darkPixels) At(x, y int) color.Color {
	pixel := color.NRGBAModel.Convert(tile.Image.At(x, y)).(color.NRGBA)
	r, g, b := int(pixel.R), int(pixel.G), int(pixel.B)
	luminance := (54*r + 183*g + 19*b) / 256
	light := (255 - luminance) * 3 / 4
	return color.NRGBA{
		R: uint8(max(0, min(255, 24+light+(r-luminance)*3/8))),
		G: uint8(max(0, min(255, 28+light+(g-luminance)*3/8))),
		B: uint8(max(0, min(255, 34+light+(b-luminance)*3/8))),
		A: pixel.A,
	}
}

// ForTheme returns immutable display pixels for the current app appearance.
// Light mode unwraps the original pixels; repeated use never stacks filters.
func ForTheme(pixels image.Image) image.Image {
	if filtered, ok := pixels.(darkPixels); ok {
		pixels = filtered.Image
	}
	// Read resolved colors: PicFetch's forced appearance wraps the theme rather
	// than changing Fyne's system ThemeVariant value.
	r, g, b, _ := theme.Color(theme.ColorNameBackground).RGBA()
	if pixels != nil && r+g+b < 3*32768 {
		return darkPixels{pixels}
	}
	return pixels
}
