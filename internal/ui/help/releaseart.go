package help

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

type releaseImageRequest struct {
	url     string
	segment *releaseImageSegment
}

// Replace every markdown image before Fyne can synchronously load its URI.
// Images nested in lists/paragraphs follow the same worker and UI path.
func prepareReleaseImages(segments []widget.RichTextSegment, requests *[]releaseImageRequest) []widget.RichTextSegment {
	out := make([]widget.RichTextSegment, 0, len(segments))
	for _, segment := range segments {
		switch s := segment.(type) {
		case *widget.ImageSegment:
			art := &releaseImageSegment{message: lang.L("Loading image...")}
			if s.Source == nil {
				art.message = lang.L("Image unavailable")
			} else {
				*requests = append(*requests, releaseImageRequest{url: s.Source.String(), segment: art})
			}
			out = append(out, art)
			continue
		case *widget.ParagraphSegment:
			s.Texts = prepareReleaseImages(s.Texts, requests)
		case *widget.ListSegment:
			s.Items = prepareReleaseImages(s.Items, requests)
		case *widget.TableSegment:
			for cell := range s.Headers {
				s.Headers[cell] = prepareReleaseImages(s.Headers[cell], requests)
			}
			for row := range s.Rows {
				for cell := range s.Rows[row] {
					s.Rows[row][cell] = prepareReleaseImages(s.Rows[row][cell], requests)
				}
			}
		}
		out = append(out, segment)
	}
	return out
}

func (h *Help) loadReleaseImages(session *releaseNotesSession, text *widget.RichText, requests []releaseImageRequest) {
	client, queue := h.imageClient, h.imageUI
	workers := min(3, len(requests))
	for first := range workers {
		h.imageWorkers.Add(1)
		go func() {
			defer h.imageWorkers.Done()
			for index := first; index < len(requests); index += workers {
				if session.ctx.Err() != nil {
					return
				}
				request := requests[index]
				picture, err := loadReleaseImage(session.ctx, client, request.url)
				if session.ctx.Err() != nil {
					return
				}
				queue.Do(func() {
					if session.ctx.Err() != nil {
						return
					}
					if err != nil {
						request.segment.message = lang.L("Image unavailable")
						fyne.LogError("load release notes image", err)
					} else {
						request.segment.picture = picture
						request.segment.message = ""
					}
					text.Refresh()
				})
			}
		}()
	}
}

// Decoded pixels are published only on UI. The fixed-height slot keeps the
// document steady while downloads finish, with image proportions preserved.
type releaseImageSegment struct {
	picture image.Image
	message string
}

func (s *releaseImageSegment) Inline() bool    { return false }
func (s *releaseImageSegment) Textual() string { return s.message }

func (s *releaseImageSegment) Visual() fyne.CanvasObject {
	picture := canvas.NewImageFromImage(s.picture)
	picture.FillMode = canvas.ImageFillContain
	picture.ScaleMode = canvas.ImageScaleSmooth
	picture.SetMinSize(fyne.NewSize(220, 220))
	status := widget.NewLabel(s.message)
	status.Alignment = fyne.TextAlignCenter
	status.Wrapping = fyne.TextWrapWord
	content := container.NewStack(picture, container.NewCenter(status))
	s.Update(content)
	return content
}

func (s *releaseImageSegment) Update(object fyne.CanvasObject) {
	content := object.(*fyne.Container)
	picture := content.Objects[0].(*canvas.Image)
	status := content.Objects[1].(*fyne.Container).Objects[0].(*widget.Label)
	picture.Image = s.picture
	picture.Refresh()
	status.SetText(s.message)
	if s.picture == nil {
		status.Show()
	} else {
		status.Hide()
	}
}

func (s *releaseImageSegment) Select(_, _ fyne.Position) {}
func (s *releaseImageSegment) SelectedText() string      { return "" }
func (s *releaseImageSegment) Unselect()                 {}
