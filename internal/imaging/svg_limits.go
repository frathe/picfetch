package imaging

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"

	"golang.org/x/net/html/charset"
)

const (
	maxSVGBytes            = 8 * 1024 * 1024
	maxSVGDepth            = 64
	maxSVGExpandedElements = 100_000
	maxSVGExpandedBytes    = 16 * 1024 * 1024
)

var errSVGComplexity = errors.New("SVG exceeds supported parsing or definition reuse limits")

// validateSVG bounds work before oksvg allocates paths. Its saved definitions
// are cumulative lists, not XML subtrees. Only allowing use outside defs means
// no saved definition can invoke use recursively. Counting the entire source
// once per use conservatively bounds even that cumulative expansion behavior.
// The returned expanded source size is also charged to the vector cache.
func validateSVG(ctx context.Context, data []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if len(data) > maxSVGBytes {
		return 0, errSVGComplexity
	}
	dec := xml.NewDecoder(bytes.NewReader(data))
	// Use the same encoding conversion as oksvg so both parsers see the same
	// element names and attributes, including non-UTF-8 documents.
	dec.CharsetReader = charset.NewReaderLabel
	depth, definitions, elements, uses := 0, 0, 0, 0
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		token, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return 0, err
		}
		switch element := token.(type) {
		case xml.StartElement:
			depth++
			elements++
			if depth > maxSVGDepth || elements > maxSVGExpandedElements {
				return 0, errSVGComplexity
			}
			switch element.Name.Local {
			case "defs":
				if definitions != 0 {
					return 0, errSVGComplexity
				}
				definitions++
			case "use":
				if definitions != 0 {
					return 0, errSVGComplexity
				}
				uses++
			}
		case xml.EndElement:
			depth--
			if element.Name.Local == "defs" {
				definitions--
			}
		}
		if elements > maxSVGExpandedElements/(uses+1) || len(data) > maxSVGExpandedBytes/(uses+1) {
			return 0, errSVGComplexity
		}
	}
	return len(data) * (uses + 1), nil
}
