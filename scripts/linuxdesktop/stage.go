package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/frathe/picfetch/internal/imaging"
)

type appMetadata struct{ Name, ID, Icon string }

//go:embed install.sh
var installerTemplate string

func stage(root, executable, out string) error {
	if !safeBasename(executable, true) {
		return fmt.Errorf("unsafe executable basename %q", executable)
	}
	meta, err := readMetadata(root)
	if err != nil {
		return err
	}
	mimes, err := mimeTypes()
	if err != nil {
		return err
	}
	command := executable
	if strings.Contains(command, " ") {
		command = `"` + command + `"`
	}
	entry := "[Desktop Entry]\nType=Application\nName=" + meta.Name + "\nIcon=" + meta.ID + "\nExec=" + command + " %F\nTerminal=false\nCategories=Graphics;Photography;Viewer;\nMimeType=" + strings.Join(mimes, ";") + ";\n"
	icon, err := os.ReadFile(filepath.Join(root, meta.Icon))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, meta.ID+".desktop"), []byte(entry), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, meta.ID+".png"), icon, 0644); err != nil {
		return err
	}
	installer := strings.NewReplacer("@APP_ID@", meta.ID, "@EXECUTABLE@", executable).Replace(installerTemplate)
	return os.WriteFile(filepath.Join(out, "install.sh"), []byte(installer), 0755)
}

// Exec is a desktop-entry argument vector, not a shell command. Restrict the
// basename so only whole-argument quoting for an interior space is necessary.
func safeBasename(value string, spaces bool) bool {
	if value == "" || len(value) > 255 || value != strings.TrimSpace(value) || value[0] == '.' || value[0] == '-' {
		return false
	}
	for _, char := range value {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '.' || char == '-' || char == '_' || spaces && char == ' ' {
			continue
		}
		return false
	}
	return true
}

func readMetadata(root string) (appMetadata, error) {
	data, err := os.ReadFile(filepath.Join(root, "FyneApp.toml"))
	if err != nil {
		return appMetadata{}, err
	}
	var meta appMetadata
	fields := map[string]*string{"Name": &meta.Name, "ID": &meta.ID, "Icon": &meta.Icon}
	section := ""
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") {
			section = line
			continue
		}
		if section != "[Details]" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		field := fields[strings.TrimSpace(key)]
		if !ok || field == nil {
			continue
		}
		if *field != "" {
			return appMetadata{}, fmt.Errorf("duplicate Details.%s in FyneApp.toml", key)
		}
		*field, err = strconv.Unquote(strings.TrimSpace(value))
		if err != nil {
			return appMetadata{}, fmt.Errorf("invalid Details.%s: %w", key, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return appMetadata{}, err
	}
	if meta.Name == "" || meta.ID == "" || meta.Icon == "" {
		return appMetadata{}, fmt.Errorf("FyneApp.toml must declare Details.Name, ID and Icon")
	}
	if meta.Name != strings.TrimSpace(meta.Name) || strings.ContainsAny(meta.Name, "\r\n\t\x00\\") {
		return appMetadata{}, fmt.Errorf("unsafe Details.Name in FyneApp.toml")
	}
	if !safeBasename(meta.ID, false) {
		return appMetadata{}, fmt.Errorf("unsafe Details.ID in FyneApp.toml")
	}
	if !filepath.IsLocal(meta.Icon) {
		return appMetadata{}, fmt.Errorf("Details.Icon must remain within the repository")
	}
	return meta, nil
}

// freedesktop shared-mime-info supplies the canonical MIME names. Iterate the
// static recognition list, never the build host's optional decoder capability.
func mimeTypes() ([]string, error) {
	mapping := map[string]string{
		".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".jpe": "image/jpeg", ".jfif": "image/jpeg",
		".png": "image/png", ".gif": "image/gif", ".webp": "image/webp", ".bmp": "image/bmp",
		".tif": "image/tiff", ".tiff": "image/tiff", ".ico": "image/vnd.microsoft.icon", ".xpm": "image/x-xpixmap",
		".avif": "image/avif", ".svg": "image/svg+xml", ".heic": "image/heic", ".heif": "image/heif",
		".cr2": "image/x-canon-cr2", ".cr3": "image/x-canon-cr3", ".nef": "image/x-nikon-nef", ".nrw": "image/x-nikon-nrw",
		".arw": "image/x-sony-arw", ".dng": "image/x-adobe-dng", ".orf": "image/x-olympus-orf", ".rw2": "image/x-panasonic-rw2",
		".raf": "image/x-fuji-raf", ".pef": "image/x-pentax-pef", ".srw": "image/x-samsung-srw", ".raw": "image/x-panasonic-rw",
	}
	var result []string
	for _, ext := range imaging.RecognizedExtensions() {
		mime := mapping[ext]
		if mime == "" {
			return nil, fmt.Errorf("no Linux MIME mapping for recognized extension %q", ext)
		}
		result = append(result, mime)
	}
	slices.Sort(result)
	return slices.Compact(result), nil
}
