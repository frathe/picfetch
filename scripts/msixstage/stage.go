package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/image/draw"

	"github.com/frathe/picfetch/internal/imaging"
)

const (
	storeIdentityName  = "OpenSourceDeveloperFloria.PicFetch"
	storePublisher     = "CN=D9654E56-586C-4C1E-ABC8-71CCDC33B78F"
	storePublisherName = "Open Source Developer Florian Rathe"
)

type appMetadata struct {
	Version string
	Build   int
}

type stageOptions struct {
	Root       string
	Arch       string
	Executable string
	Out        string
}

type assetSpec struct {
	name string
	size int
}

var packageAssets = []assetSpec{
	{"StoreLogo.png", 50},
	{"Square44x44Logo.png", 44},
	{"Square44x44Logo.scale-200.png", 88},
	{"Square44x44Logo.scale-400.png", 176},
	{"Square44x44Logo.targetsize-16_altform-unplated.png", 16},
	{"Square44x44Logo.targetsize-24_altform-unplated.png", 24},
	{"Square44x44Logo.targetsize-32_altform-unplated.png", 32},
	{"Square44x44Logo.targetsize-48_altform-unplated.png", 48},
	{"Square44x44Logo.targetsize-256_altform-unplated.png", 256},
	{"Square150x150Logo.png", 150},
	{"Square150x150Logo.scale-200.png", 300},
	{"Square150x150Logo.scale-400.png", 600},
}

func stage(opts stageOptions) error {
	if opts.Root == "" || opts.Arch == "" || opts.Executable == "" || opts.Out == "" {
		return fmt.Errorf("root, arch, exe, and out are required")
	}

	metaFile, err := os.Open(filepath.Join(opts.Root, "FyneApp.toml"))
	if err != nil {
		return err
	}
	meta, err := readAppMetadata(metaFile)
	closeErr := metaFile.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}

	manifest, err := renderManifest(meta, opts.Arch)
	if err != nil {
		return err
	}
	exe, err := os.ReadFile(opts.Executable)
	if err != nil {
		return err
	}
	iconFile, err := os.Open(filepath.Join(opts.Root, "assets", "appIcon.png"))
	if err != nil {
		return err
	}
	icon, err := png.Decode(iconFile)
	closeErr = iconFile.Close()
	if err != nil {
		return fmt.Errorf("decode app icon: %w", err)
	}
	if closeErr != nil {
		return closeErr
	}
	if icon.Bounds().Dx() != icon.Bounds().Dy() {
		return fmt.Errorf("app icon is %dx%d, want square", icon.Bounds().Dx(), icon.Bounds().Dy())
	}

	assetsDir := filepath.Join(opts.Out, "Assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(opts.Out, "AppxManifest.xml"), []byte(manifest), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(opts.Out, "picfetch.exe"), exe, 0o755); err != nil {
		return err
	}

	return renderAssets(icon, assetsDir)
}

func readAppMetadata(r io.Reader) (appMetadata, error) {
	var meta appMetadata
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "Version":
			meta.Version = strings.Trim(strings.TrimSpace(value), `"`)
		case "Build":
			build, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return appMetadata{}, fmt.Errorf("invalid Fyne build: %w", err)
			}
			meta.Build = build
		}
	}
	if err := scanner.Err(); err != nil {
		return appMetadata{}, err
	}
	if meta.Version == "" {
		return appMetadata{}, fmt.Errorf("FyneApp.toml has no Version")
	}
	if _, err := storeVersion(meta.Build); err != nil {
		return appMetadata{}, err
	}
	return meta, nil
}

func storeVersion(build int) (string, error) {
	if build < 1 || build > 65535 {
		return "", fmt.Errorf("Fyne Build %d is outside the MSIX range 1..65535", build)
	}
	return fmt.Sprintf("1.0.%d.0", build), nil
}

func renderManifest(meta appMetadata, arch string) (string, error) {
	var msixArch string
	switch arch {
	case "amd64":
		msixArch = "x64"
	case "arm64":
		msixArch = "arm64"
	default:
		return "", fmt.Errorf("unsupported architecture %q (want amd64 or arm64)", arch)
	}

	version, err := storeVersion(meta.Build)
	if err != nil {
		return "", err
	}

	var fileTypes strings.Builder
	for _, ext := range imaging.SupportedExtensions() {
		fmt.Fprintf(&fileTypes, "              <uap:FileType>%s</uap:FileType>\n", ext)
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<Package
  xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10"
  xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10"
  xmlns:uap10="http://schemas.microsoft.com/appx/manifest/uap/windows10/10"
  xmlns:rescap="http://schemas.microsoft.com/appx/manifest/foundation/windows10/restrictedcapabilities"
  IgnorableNamespaces="uap uap10 rescap">
  <Identity Name="%s" Publisher="%s" Version="%s" ProcessorArchitecture="%s" />
  <Properties>
    <DisplayName>PicFetch</DisplayName>
    <PublisherDisplayName>%s</PublisherDisplayName>
    <Description>A small, fast desktop image viewer.</Description>
    <Logo>Assets\StoreLogo.png</Logo>
  </Properties>
  <Resources>
    <Resource Language="en-us" />
    <Resource Language="de-de" />
  </Resources>
  <Dependencies>
    <TargetDeviceFamily Name="Windows.Desktop" MinVersion="10.0.19041.0" MaxVersionTested="10.0.26100.0" />
  </Dependencies>
  <Capabilities>
    <rescap:Capability Name="runFullTrust" />
  </Capabilities>
  <Applications>
    <Application Id="PicFetch" Executable="picfetch.exe" uap10:RuntimeBehavior="packagedClassicApp" uap10:TrustLevel="mediumIL">
      <uap:VisualElements DisplayName="PicFetch" Description="A small, fast desktop image viewer." BackgroundColor="transparent" Square150x150Logo="Assets\Square150x150Logo.png" Square44x44Logo="Assets\Square44x44Logo.png">
        <uap:DefaultTile ShortName="PicFetch" />
      </uap:VisualElements>
      <Extensions>
        <uap:Extension Category="windows.fileTypeAssociation">
          <uap:FileTypeAssociation Name="picfetch.image">
            <uap:SupportedFileTypes>
%s            </uap:SupportedFileTypes>
          </uap:FileTypeAssociation>
        </uap:Extension>
      </Extensions>
    </Application>
  </Applications>
</Package>
`, storeIdentityName, storePublisher, version, msixArch, storePublisherName, fileTypes.String()), nil
}

func renderAssets(src image.Image, dir string) error {
	for _, spec := range packageAssets {
		dst := image.NewNRGBA(image.Rect(0, 0, spec.size, spec.size))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

		var buf bytes.Buffer
		if err := png.Encode(&buf, dst); err != nil {
			return fmt.Errorf("encode %s: %w", spec.name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, spec.name), buf.Bytes(), 0o644); err != nil {
			return err
		}
	}
	return nil
}
