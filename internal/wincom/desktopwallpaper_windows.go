//go:build windows

// Package wincom owns shared Windows COM declarations used by platform
// adapters.
package wincom

// GUID is the binary layout Windows COM functions expect for class and
// interface identifiers. Its fields stay private because callers only pass
// values returned by DesktopWallpaperIDs to COM.
type GUID struct {
	data1 uint32
	data2 uint16
	data3 uint16
	data4 [8]byte
}

// DesktopWallpaper is the leading interface pointer returned for
// IDesktopWallpaper.
type DesktopWallpaper struct {
	vtbl *desktopWallpaperVtbl
}

type desktopWallpaperVtbl struct {
	queryInterface, addRef, release                   uintptr
	setWallpaper, getWallpaper                        uintptr
	getMonitorDevicePathAt, getMonitorDevicePathCount uintptr
	getMonitorRECT                                    uintptr
}

const (
	COInitApartmentThreaded = 0x2
	CLSCTXLocalServer       = 0x4
)

// DesktopWallpaperIDs returns private copies of the IDesktopWallpaper class
// and interface identifiers so callers can safely hand their addresses to
// CoCreateInstance.
func DesktopWallpaperIDs() (GUID, GUID) {
	classID := GUID{0xc2cf3110, 0x460e, 0x4fc1, [8]byte{0xb9, 0xd0, 0x8a, 0x1c, 0x0c, 0x9c, 0xc4, 0xbd}}
	interfaceID := GUID{0xb92b56a9, 0x8b55, 0x4e14, [8]byte{0x9a, 0x89, 0x01, 0x99, 0xbb, 0xb6, 0xf9, 0x3b}}

	return classID, interfaceID
}

func (api *DesktopWallpaper) ReleaseProc() uintptr {
	return api.vtbl.release
}

func (api *DesktopWallpaper) SetWallpaperProc() uintptr {
	return api.vtbl.setWallpaper
}

func (api *DesktopWallpaper) MonitorDevicePathAtProc() uintptr {
	return api.vtbl.getMonitorDevicePathAt
}

func (api *DesktopWallpaper) MonitorDevicePathCountProc() uintptr {
	return api.vtbl.getMonitorDevicePathCount
}

func (api *DesktopWallpaper) MonitorRECTProc() uintptr {
	return api.vtbl.getMonitorRECT
}
