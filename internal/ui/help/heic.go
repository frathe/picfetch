package help

import (
	_ "embed"
	"errors"
	"io"
	"os"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

//go:embed heic_guide.md
var heicGuideEN string

//go:embed heic_guide_de.md
var heicGuideDE string

type heicGuideSystem struct {
	goos, goarch, osRelease, language string
}

// ShowHEICGuide opens the embedded installation instructions without checking
// support or opening a browser. The singleton supplies raising and Escape.
func (h *Help) ShowHEICGuide() {
	if h.stopped {
		return
	}
	h.heicGuideWin.Show(h.app, lang.L("HEIC installation instructions"), fyne.NewSize(720, 640), func() fyne.CanvasObject {
		system := h.guideSystem()
		source := heicGuideEN
		if system.language == "de" || strings.HasPrefix(system.language, "de-") {
			source = heicGuideDE
		}
		body := heicGuideSection(source, "intro")
		switch system.goos {
		case "darwin":
			body += heicGuideSection(source, "macos")
		case "windows":
			body += heicGuideSection(source, "windows")
		case "linux":
			body += heicGuideSection(source, "linux") + heicGuideSection(source, system.linuxSection())
		default:
			body += heicGuideSection(source, "unknown")
		}
		body += heicGuideSection(source, "after")
		text := widget.NewRichTextFromMarkdown(body)
		text.Wrapping = fyne.TextWrapWord
		return container.NewVScroll(text)
	}, nil)
}

func heicGuideSection(source, name string) string {
	_, section, _ := strings.Cut(source, "<!-- "+name+" -->")
	section, _, _ = strings.Cut(section, "<!-- ")
	return section
}

func (s heicGuideSystem) linuxSection() string {
	id, idValid := osReleaseValue(s.osRelease, "ID")
	version, versionValid := osReleaseValue(s.osRelease, "VERSION_ID")
	if !idValid || !versionValid {
		return "generic"
	}
	if s.goarch == "amd64" || s.goarch == "arm64" {
		if id == "debian" && version == "13" {
			return "debian13"
		}
		if id == "ubuntu" && version == "24.04" {
			return "ubuntu2404"
		}
	}
	if id == "arch" && version == "" && s.goarch == "amd64" {
		return "arch"
	}
	if id == "fedora" {
		return "fedora"
	}
	return "generic"
}

// Only exact ID/VERSION_ID matches select recipes. ID_LIKE is insufficient:
// derivatives need their own package, release and architecture evidence.
func osReleaseValue(source, key string) (string, bool) {
	var value string
	for _, line := range strings.Split(source, "\n") {
		name, candidate, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || name != key {
			continue
		}
		value = strings.TrimSpace(candidate)
		if len(value) > 0 && (value[0] == '\'' || value[0] == '"') {
			if len(value) < 2 || value[len(value)-1] != value[0] {
				return "", false
			}
			value = value[1 : len(value)-1]
		}
	}
	return value, true
}

func currentHEICGuideSystem() heicGuideSystem {
	system := heicGuideSystem{goos: runtime.GOOS, goarch: runtime.GOARCH, language: lang.SystemLocale().LanguageString()}
	if system.goos == "linux" {
		system.osRelease = readHEICOSRelease()
	}
	return system
}

func readHEICOSRelease() string {
	file, err := os.Open("/etc/os-release")
	if errors.Is(err, os.ErrNotExist) {
		file, err = os.Open("/usr/lib/os-release")
	}
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }()
	const maxSize = 64 * 1024
	content, err := io.ReadAll(io.LimitReader(file, maxSize+1))
	if err != nil || len(content) > maxSize {
		return ""
	}
	return string(content)
}
