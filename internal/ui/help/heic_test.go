package help

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestHEICGuideOfflineWindowLifecycle(t *testing.T) {
	app := &discussionLinkApp{App: test.NewApp()}
	h := New(app, "PicFetch", nil)
	h.guideSystem = func() heicGuideSystem {
		return heicGuideSystem{goos: "linux", goarch: "amd64", language: "en"}
	}
	initialWindows := len(app.Driver().AllWindows())
	h.SetImageClient(&http.Client{Transport: releaseImageTransport(func(_ *http.Request) (*http.Response, error) {
		t.Fatal("reading the HEIC guide must not make an HTTP request")
		return nil, nil
	})})
	h.ShowHEICGuide()
	win := heicWindow(app)
	if win == nil {
		t.Fatal("the HEIC guide action did not open its window")
	}
	t.Cleanup(func() {
		if current := heicWindow(app); current != nil {
			current.Close()
		}
	})
	if _, ok := win.Content().(*container.Scroll); !ok {
		t.Fatal("the HEIC guide must be in a scrollable window")
	}
	text := findRichText(win.Content())
	if text == nil || text.Wrapping != fyne.TextWrapWord {
		t.Fatal("the actual window tree must contain wrapped Markdown")
	}
	if heading, ok := text.Segments[0].(*widget.TextSegment); !ok || heading.Style != widget.RichTextStyleHeading {
		t.Fatal("the embedded guide must render its Markdown heading")
	}
	plain := richTextPlain(text.Segments)
	if !strings.Contains(plain, "Check HEIC support") || !strings.Contains(plain, "restart") {
		t.Fatal("the guide must explain Settings recheck and possible restart")
	}
	if app.opened != nil {
		t.Fatal("reading the HEIC guide must not launch the browser")
	}
	h.ShowHEICGuide()
	if heicWindow(app) != win || len(app.Driver().AllWindows()) != initialWindows+1 {
		t.Fatal("repeated requests must raise the same guide window")
	}
	win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if heicWindow(app) != nil {
		t.Fatal("Escape must close the HEIC guide")
	}
	h.ShowHEICGuide()
	if heicWindow(app) == nil || heicWindow(app) == win {
		t.Fatal("the guide must reopen after Escape")
	}
	heicWindow(app).Close()
	h.Stop()
	h.ShowHEICGuide()
	if heicWindow(app) != nil {
		t.Fatal("the guide must not open after shutdown")
	}
}

func TestHEICGuideCurrentSystemSelection(t *testing.T) {
	app := test.NewApp()
	for _, tc := range []struct {
		name, goos, arch, release string
		want, absent              []string
	}{
		{"macOS Intel", "darwin", "amd64", "", []string{"macOS", "10.13", "built-in"}, []string{"Microsoft Store", "sudo "}},
		{"macOS Apple Silicon", "darwin", "arm64", "", []string{"macOS", "10.13", "built-in"}, []string{"Microsoft Store", "sudo "}},
		{"Windows x64", "windows", "amd64", "", []string{"HEIF Image Extensions", "HEVC Video Extensions", "cost"}, []string{"macOS", "sudo "}},
		{"Windows ARM64", "windows", "arm64", "", []string{"Microsoft Store", "system requirements"}, []string{"macOS", "sudo "}},
		{"Debian 13", "linux", "amd64", "ID=debian\nVERSION_ID=13", []string{"Debian 13", "sudo apt install libheif1 libheif-plugin-libde265"}, []string{"Ubuntu", "pacman"}},
		{"Debian 13 ARM64", "linux", "arm64", "ID=\"debian\"\nVERSION_ID='13'", []string{"Debian 13", "sudo apt install libheif1 libheif-plugin-libde265"}, nil},
		{"Ubuntu 24.04", "linux", "amd64", "ID=ubuntu\nVERSION_ID=\"24.04\"", []string{"Ubuntu 24.04", "universe", "sudo apt install libheif1 libheif-plugin-libde265"}, []string{"Debian 13", "pacman"}},
		{"Ubuntu 24.04 ARM64", "linux", "arm64", "ID=ubuntu\nVERSION_ID=24.04", []string{"Ubuntu 24.04", "libheif-plugin-libde265"}, nil},
		{"Arch rolling x86_64", "linux", "amd64", "ID=arch\nBUILD_ID=rolling", []string{"Arch Linux", "sudo pacman -Syu libheif", "full system upgrade"}, []string{"apt install"}},
		{"Arch ARM is unqualified", "linux", "arm64", "ID=arch", []string{"No installation command"}, []string{"sudo "}},
		{"Fedora links only", "linux", "amd64", "ID=fedora\nVERSION_ID=44", []string{"Fedora", "RPM Fusion", "optional external repository", "No installation command"}, []string{"sudo "}},
		{"Fedora ARM links only", "linux", "arm64", "ID=fedora\nVERSION_ID=44", []string{"Fedora", "RPM Fusion", "No installation command"}, []string{"sudo "}},
		{"unverified Debian release", "linux", "amd64", "ID=debian\nVERSION_ID=12", []string{"No installation command"}, []string{"sudo "}},
		{"unverified Ubuntu release", "linux", "amd64", "ID=ubuntu\nVERSION_ID=26.04", []string{"No installation command"}, []string{"sudo "}},
		{"unverified architecture", "linux", "riscv64", "ID=debian\nVERSION_ID=13", []string{"No installation command"}, []string{"sudo "}},
		{"derivative is not its parent", "linux", "amd64", "ID=linuxmint\nID_LIKE=ubuntu\nVERSION_ID=24.04", []string{"No installation command"}, []string{"sudo "}},
		{"missing distribution", "linux", "amd64", "", []string{"No installation command", "system libheif", "HEVC"}, []string{"sudo "}},
		{"malformed rolling release", "linux", "amd64", "ID=arch\nVERSION_ID=\"unclosed", []string{"No installation command"}, []string{"sudo "}},
		{"unknown OS", "freebsd", "amd64", "", []string{"No installation command"}, []string{"sudo ", "Microsoft Store", "macOS"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := New(app, "PicFetch", nil)
			h.guideSystem = func() heicGuideSystem {
				return heicGuideSystem{goos: tc.goos, goarch: tc.arch, osRelease: tc.release, language: "en-US"}
			}
			h.ShowHEICGuide()
			win := heicWindow(app)
			defer win.Close()
			plain := richTextPlain(findRichText(win.Content()).Segments)
			for _, want := range tc.want {
				if !strings.Contains(plain, want) {
					t.Errorf("rendered guide omits %q", want)
				}
			}
			for _, absent := range tc.absent {
				if strings.Contains(plain, absent) {
					t.Errorf("rendered guide offers unrelated or unqualified guidance %q", absent)
				}
			}
		})
	}
}

func TestHEICGuideDocumentsStayOfflineAndRenderable(t *testing.T) {
	arrow := regexp.MustCompile(`[\x{2190}-\x{21ff}\x{27f0}-\x{27ff}\x{2900}-\x{297f}]`)
	table := regexp.MustCompile(`(?m)^\s*\||^\s*:?-{3,}:?\s*\|`)
	for name, source := range map[string]string{"English": heicGuideEN, "German": heicGuideDE} {
		t.Run(name, func(t *testing.T) {
			if arrow.MatchString(source) {
				t.Error("the guide contains a Unicode arrow unsupported by the app font")
			}
			if table.MatchString(source) {
				t.Error("the guide contains a Markdown table unsupported by the renderer")
			}
			parsed := widget.NewRichTextFromMarkdown(source)
			walkRichText(parsed.Segments, func(segment widget.RichTextSegment) {
				if _, ok := segment.(*widget.ImageSegment); ok {
					t.Error("guide images can trigger resource downloads during layout")
				}
			})
		})
	}
}

func TestHEICGuideLocaleFallback(t *testing.T) {
	app := test.NewApp()
	for _, tc := range []struct{ locale, want, absent string }{
		{"de", "HEIC-Installationsanleitung", "After installation"},
		{"de-DE", "HEIC-Installationsanleitung", "After installation"},
		{"de-AT", "HEIC-Installationsanleitung", "After installation"},
		{"de-CH", "HEIC-Installationsanleitung", "After installation"},
		{"en-US", "HEIC installation instructions", "Nach der Installation"},
		{"fr-FR", "HEIC installation instructions", "Nach der Installation"},
		{"en-DE", "HEIC installation instructions", "Nach der Installation"},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			h := New(app, "PicFetch", nil)
			h.guideSystem = func() heicGuideSystem {
				return heicGuideSystem{goos: "windows", goarch: "amd64", language: tc.locale}
			}
			h.ShowHEICGuide()
			win := heicWindow(app)
			defer win.Close()
			plain := richTextPlain(findRichText(win.Content()).Segments)
			if !strings.Contains(plain, tc.want) || strings.Contains(plain, tc.absent) {
				t.Fatalf("guide does not follow German/English manual fallback for %q", tc.locale)
			}
		})
	}
}

func heicWindow(app fyne.App) fyne.Window {
	for _, win := range app.Driver().AllWindows() {
		if win.Title() == lang.L("HEIC installation instructions") {
			return win
		}
	}
	return nil
}
