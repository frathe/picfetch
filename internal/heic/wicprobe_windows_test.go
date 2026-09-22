//go:build windows && (amd64 || arm64)

package heic

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/frathe/picfetch/internal/wincom"
)

// This bounded experiment independently checks Microsoft's frame ordering and
// provider identity alongside the production adapter's native tests.
// SDK ABI source: microsoft/win32metadata commit
// 5c5efbc01d4c87f6830ec304d42777991d533154, wincodec.h.
func TestHEICWindowsWICProbe(t *testing.T) {
	if os.Getenv("PICFETCH_HEIC_NATIVE_TEST") != "1" {
		t.Skip("requires explicit native Windows WIC qualification")
	}
	if os.Getenv("PICFETCH_WIC_QUALIFY_CHILD") != "1" {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, executable, "-test.run=^TestHEICWindowsWICProbe$", "-test.v")
		cmd.Env = append(os.Environ(), "PICFETCH_WIC_QUALIFY_CHILD=1")
		output := &limitedLog{remaining: 64 * 1024}
		cmd.Stdout, cmd.Stderr = output, output
		if err := cmd.Run(); err != nil {
			t.Fatalf("native Windows WIC experiment: %v\n%s", err, output.String())
		}
		t.Log(output.String())
		return
	}
	wicProbeLimitProcess(t)
	for _, fixture := range []struct {
		file        string
		left, right int
	}{{"probe8", 0, 255}, {"probe10", 0, 255}, {"nonfirst-primary", 255, 0}} {
		t.Run(fixture.file, func(t *testing.T) {
			data, err := os.ReadFile("testdata/" + fixture.file + ".heic")
			if err != nil {
				t.Fatal(err)
			}
			frames := wicProbeFrames(t, data)
			// WIC orders its primary first even when pitm names a later stored
			// item. The production variants also change pitm and item IDs.
			assertGrayPixel(t, frames[0], 8, 8, fixture.left)
			assertGrayPixel(t, frames[0], 48, 8, fixture.right)
		})
	}
}

func wicProbeLimitProcess(t *testing.T) {
	t.Helper()
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = windows.CloseHandle(job) })
	var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_PROCESS_MEMORY | windows.JOB_OBJECT_LIMIT_PROCESS_TIME
	limits.BasicLimitInformation.PerProcessUserTimeLimit = 40 * 10_000_000
	limits.ProcessMemoryLimit = 4 * 1024 * 1024 * 1024
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		t.Fatal(err)
	}
	if err := windows.AssignProcessToJobObject(job, windows.CurrentProcess()); err != nil {
		t.Fatal(err)
	}
	t.Log("WIC probe process limits: 4 GiB committed memory, 40 CPU seconds; no network denial")
}

type wicProbeUnknown struct{ vtable *wicProbeUnknownMethods }
type wicProbeUnknownMethods struct{ queryInterface, addRef, release uintptr }

type wicProbeFactory struct{ vtable *wicProbeFactoryMethods }
type wicProbeFactoryMethods struct {
	wicProbeUnknownMethods
	createDecoderFromFilename, createDecoderFromStream, createDecoderFromFileHandle uintptr
	createComponentInfo, createDecoder, createEncoder, createPalette                uintptr
	createFormatConverter, createBitmapScaler, createBitmapClipper                  uintptr
	createBitmapFlipRotator, createStream                                           uintptr
}

type wicProbeStream struct{ vtable *wicProbeStreamMethods }
type wicProbeStreamMethods struct {
	wicProbeUnknownMethods
	read, write, seek, setSize, copyTo, commit, revert, lockRegion, unlockRegion, stat, clone uintptr
	initializeFromIStream, initializeFromFilename, initializeFromMemory                       uintptr
}

type wicProbeDecoder struct{ vtable *wicProbeDecoderMethods }
type wicProbeDecoderMethods struct {
	wicProbeUnknownMethods
	queryCapability, initialize, getContainerFormat, getDecoderInfo uintptr
	copyPalette, getMetadataQueryReader, getPreview                 uintptr
	getColorContexts, getThumbnail, getFrameCount, getFrame         uintptr
}

type wicProbeSource struct{ vtable *wicProbeSourceMethods }
type wicProbeSourceMethods struct {
	wicProbeUnknownMethods
	getSize, getPixelFormat, getResolution, copyPalette, copyPixels uintptr
}

type wicProbeInfo struct{ vtable *wicProbeInfoMethods }
type wicProbeInfoMethods struct {
	wicProbeUnknownMethods
	getComponentType, getCLSID, getSigningStatus, getAuthor    uintptr
	getVendorGUID, getVersion, getSpecVersion, getFriendlyName uintptr
}

func wicProbeGUID(t *testing.T, text string) windows.GUID {
	t.Helper()
	id, err := windows.GUIDFromString(text)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

//go:uintptrescapes
func wicProbeCall(t *testing.T, name string, function uintptr, arguments ...uintptr) {
	t.Helper()
	hr, _, _ := syscall.SyscallN(function, arguments...)
	if wincom.FailedHRESULT(hr) {
		t.Fatalf("%s: HRESULT 0x%08x", name, uint32(hr))
	}
}

func TestHEICWindowsPrimaryVariants(t *testing.T) {
	if os.Getenv("PICFETCH_HEIC_NATIVE_TEST") != "1" {
		t.Skip("requires native Windows WIC qualification")
	}
	data, err := os.ReadFile("testdata/nonfirst-primary.heic")
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient("")
	t.Cleanup(func() { client.Stop(); client.Wait() })
	for _, renumber := range []bool{false, true} {
		for _, primary := range []uint16{1, 2} {
			t.Run(fmt.Sprintf("primary_%d_renumber_%t", primary, renumber), func(t *testing.T) {
				input := bytes.Clone(data)
				id := primary
				if renumber {
					ids := []uint16{9, 3}
					id = ids[primary-1]
					iloc, ipma, cursor := bytes.Index(input, []byte("iloc")), bytes.Index(input, []byte("ipma")), 0
					for index, itemID := range ids {
						infe := cursor + bytes.Index(input[cursor:], []byte("infe"))
						binary.BigEndian.PutUint16(input[infe+8:], itemID)
						binary.BigEndian.PutUint16(input[iloc+12+index*14:], itemID)
						binary.BigEndian.PutUint16(input[ipma+12+index*7:], itemID)
						cursor = infe + 4
					}
				}
				binary.BigEndian.PutUint16(input[bytes.Index(input, []byte("pitm"))+8:], id)
				result, err := client.Read(t.Context(), input, Request{Pixels: true, MaxEncodedBytes: 64 * 1024, MaxPixels: 4096})
				if err != nil {
					t.Fatal(err)
				}
				left := int(primary-1) * 255
				assertGrayPixel(t, result, 8, 8, left)
				assertGrayPixel(t, result, 48, 8, 255-left)
			})
		}
	}
}

func TestHEICWindowsAlphaMetadata(t *testing.T) {
	data, err := os.ReadFile("testdata/alpha-premultiplied8.heic")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name                 string
		change               func([]byte)
		alpha, premultiplied bool
		want                 error
	}{
		{"primary alpha", func(_ []byte) {}, true, true, nil},
		{"other image alpha", func(b []byte) { binary.BigEndian.PutUint16(b[bytes.Index(b, []byte("auxl"))+8:], 3) }, false, false, nil},
		{"other image premultiplication", func(b []byte) { binary.BigEndian.PutUint16(b[bytes.Index(b, []byte("prem"))+4:], 3) }, true, false, nil},
		{"depth auxiliary", func(b []byte) { i := bytes.Index(b, []byte("auxid:1")); b[i+6] = '2' }, false, false, nil},
		{"unknown auxiliary version", func(b []byte) { b[bytes.Index(b, []byte("auxC"))+4] = 1 }, false, false, ErrUnsupported},
		{"malformed references", func(b []byte) { b[bytes.Index(b, []byte("iref"))+4] = 2 }, false, false, ErrInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := bytes.Clone(data)
			tc.change(input)
			alpha, premultiplied, err := windowsAlpha(input)
			if alpha != tc.alpha || premultiplied != tc.premultiplied || !errors.Is(err, tc.want) {
				t.Fatalf("alpha=%t premultiplied=%t error=%v, want %t %t %v", alpha, premultiplied, err, tc.alpha, tc.premultiplied, tc.want)
			}
		})
	}
}

func TestHEICWindowsWorkerRestrictions(t *testing.T) {
	if os.Getenv("PICFETCH_HEIC_NATIVE_TEST") != "1" {
		t.Skip("requires native Windows worker qualification")
	}
	if os.Getenv("PICFETCH_HEIC_RESTRICTION_TEST") == "1" {
		if err := restrictWorker(); err != nil {
			t.Fatal(err)
		}
		var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
		if err := windows.QueryInformationJobObject(0, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits)), nil); err != nil {
			t.Fatal(err)
		}
		flags := uint32(windows.JOB_OBJECT_LIMIT_PROCESS_MEMORY | windows.JOB_OBJECT_LIMIT_PROCESS_TIME | windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS)
		if limits.BasicLimitInformation.LimitFlags&flags != flags || limits.ProcessMemoryLimit != 4*1024*1024*1024 || limits.BasicLimitInformation.PerProcessUserTimeLimit != 40*10_000_000 || limits.BasicLimitInformation.ActiveProcessLimit != 1 {
			t.Fatalf("incorrect worker limits: %+v", limits)
		}
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestHEICWindowsWorkerRestrictions$", "-test.v")
	prepareWorker(cmd)
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.HideWindow || cmd.SysProcAttr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
		t.Error("HEIC workers must not open console windows")
	}
	cmd.Env = append(os.Environ(), "PICFETCH_HEIC_RESTRICTION_TEST=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("worker restriction child: %v\n%s", err, output)
	}
}

func wicProbeRelease(object unsafe.Pointer) {
	if object != nil {
		unknown := (*wicProbeUnknown)(object)
		_, _, _ = syscall.SyscallN(unknown.vtable.release, uintptr(object))
	}
}

func wicProbeFrames(t *testing.T, data []byte) []Result {
	t.Helper()
	if len(data) == 0 || len(data) > 64*1024 {
		t.Fatal("qualification fixture exceeds 64 KiB input bound")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	ole := windows.NewLazySystemDLL("ole32.dll")
	initialize, uninitialize := ole.NewProc("CoInitializeEx"), ole.NewProc("CoUninitialize")
	hr, _, _ := initialize.Call(0, wincom.COInitApartmentThreaded)
	if wincom.FailedHRESULT(hr) {
		t.Fatalf("CoInitializeEx: HRESULT 0x%08x", uint32(hr))
	}
	defer func() { _, _, _ = uninitialize.Call() }()
	create := ole.NewProc("CoCreateInstance")
	factoryClass := wicProbeGUID(t, "{cacaf262-9370-4615-a13b-9f5539da4c0a}")
	factoryID := wicProbeGUID(t, "{ec5ec8a9-c395-4314-9c77-54d7a935ff70}")
	var factory *wicProbeFactory
	wicProbeCall(t, "create WIC factory", create.Addr(), uintptr(unsafe.Pointer(&factoryClass)), 0, 1,
		uintptr(unsafe.Pointer(&factoryID)), uintptr(unsafe.Pointer(&factory)))
	if factory == nil {
		t.Fatal("WIC factory is nil")
	}
	defer wicProbeRelease(unsafe.Pointer(factory))
	var stream *wicProbeStream
	wicProbeCall(t, "create memory stream", factory.vtable.createStream, uintptr(unsafe.Pointer(factory)), uintptr(unsafe.Pointer(&stream)))
	if stream == nil {
		t.Fatal("WIC stream is nil")
	}
	var pinned runtime.Pinner
	pinned.Pin(&data[0])
	defer pinned.Unpin()
	defer wicProbeRelease(unsafe.Pointer(stream))
	wicProbeCall(t, "initialize memory stream", stream.vtable.initializeFromMemory, uintptr(unsafe.Pointer(stream)), uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)))
	// Direct class activation, never registry arbitration among matching codecs.
	decoderClass := wicProbeGUID(t, "{e9a4a80a-44fe-4de4-8971-7150b10a5199}")
	decoderID := wicProbeGUID(t, "{9edde9e7-8dee-47ea-99df-e6faf2ed44bf}")
	var decoder *wicProbeDecoder
	wicProbeCall(t, "activate Microsoft HEIF decoder", create.Addr(), uintptr(unsafe.Pointer(&decoderClass)), 0, 1,
		uintptr(unsafe.Pointer(&decoderID)), uintptr(unsafe.Pointer(&decoder)))
	if decoder == nil {
		t.Fatal("WIC HEIF decoder is nil")
	}
	defer wicProbeRelease(unsafe.Pointer(decoder))
	wicProbeCall(t, "initialize HEIF decoder", decoder.vtable.initialize, uintptr(unsafe.Pointer(decoder)), uintptr(unsafe.Pointer(stream)), 0)
	wicProbeProvider(t, decoder, decoderClass)
	var count uint32
	wicProbeCall(t, "get frame count", decoder.vtable.getFrameCount, uintptr(unsafe.Pointer(decoder)), uintptr(unsafe.Pointer(&count)))
	if count == 0 || count > 16 {
		t.Fatalf("unbounded/unusable frame count: %d", count)
	}
	t.Logf("WIC frame count: %d", count)
	frames := make([]Result, 0, count)
	for index := uint32(0); index < count; index++ {
		var frame *wicProbeSource
		wicProbeCall(t, "get indexed frame", decoder.vtable.getFrame, uintptr(unsafe.Pointer(decoder)), uintptr(index), uintptr(unsafe.Pointer(&frame)))
		if frame == nil {
			t.Fatal("WIC frame is nil")
		}
		result := wicProbePixels(t, frame)
		wicProbeRelease(unsafe.Pointer(frame))
		frames = append(frames, result)
		t.Logf("frame %d: %dx%d, left=%v right=%v", index, result.Width, result.Height, result.Pixels[8*result.Stride+8*4:][:4], result.Pixels[8*result.Stride+48*4:][:4])
	}
	var modules [1024]windows.Handle
	var needed uint32
	if err := windows.EnumProcessModules(windows.CurrentProcess(), &modules[0], uint32(unsafe.Sizeof(modules)), &needed); err != nil {
		t.Fatal(err)
	}
	if needed > uint32(unsafe.Sizeof(modules)) {
		t.Fatal("provider module list exceeds qualification bound")
	}
	for _, module := range modules[:uintptr(needed)/unsafe.Sizeof(modules[0])] {
		var path [32768]uint16
		if _, err := windows.GetModuleFileName(module, &path[0], uint32(len(path))); err != nil {
			t.Fatal(err)
		}
		name := windows.UTF16ToString(path[:])
		if lower := strings.ToLower(name); strings.Contains(lower, "heif") || strings.Contains(lower, "hevc") {
			t.Logf("loaded codec module: %s", name)
		}
	}
	runtime.KeepAlive(data)
	return frames
}

func wicProbeProvider(t *testing.T, decoder *wicProbeDecoder, expected windows.GUID) {
	t.Helper()
	var info *wicProbeInfo
	wicProbeCall(t, "get decoder info", decoder.vtable.getDecoderInfo, uintptr(unsafe.Pointer(decoder)), uintptr(unsafe.Pointer(&info)))
	if info == nil {
		t.Fatal("WIC decoder info is nil")
	}
	defer wicProbeRelease(unsafe.Pointer(info))
	var classID, vendor windows.GUID
	wicProbeCall(t, "get provider CLSID", info.vtable.getCLSID, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(&classID)))
	wicProbeCall(t, "get provider vendor", info.vtable.getVendorGUID, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(&vendor)))
	if classID != expected || vendor != wicProbeGUID(t, "{f0e749ca-edef-4589-a73a-ee0e626a2a2b}") {
		t.Fatalf("unexpected HEIF provider class=%v vendor=%v", classID, vendor)
	}
	var signing uint32
	wicProbeCall(t, "get signing status", info.vtable.getSigningStatus, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(&signing)))
	var version [256]uint16
	var written uint32
	wicProbeCall(t, "get provider version", info.vtable.getVersion, uintptr(unsafe.Pointer(info)), uintptr(len(version)), uintptr(unsafe.Pointer(&version[0])), uintptr(unsafe.Pointer(&written)))
	t.Logf("Microsoft HEIF class=%v vendor=%v signing=0x%x version=%s", classID, vendor, signing, windows.UTF16ToString(version[:]))
	var module windows.Handle
	getModule := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetModuleHandleExW")
	succeeded, _, err := getModule.Call(0x6, decoder.vtable.initialize, uintptr(unsafe.Pointer(&module)))
	if succeeded == 0 {
		t.Fatalf("provider module: %v", err)
	}
	var path [32768]uint16
	if _, err := windows.GetModuleFileName(module, &path[0], uint32(len(path))); err != nil {
		t.Fatal(err)
	}
	t.Logf("HEIF implementation module: %s", windows.UTF16ToString(path[:]))
}

func wicProbePixels(t *testing.T, frame *wicProbeSource) Result {
	t.Helper()
	var width, height uint32
	wicProbeCall(t, "get frame size", frame.vtable.getSize, uintptr(unsafe.Pointer(frame)), uintptr(unsafe.Pointer(&width)), uintptr(unsafe.Pointer(&height)))
	if width != 64 || height != 64 {
		t.Fatalf("fixture full resolution is 64x64, got %dx%d", width, height)
	}
	var format windows.GUID
	wicProbeCall(t, "get source pixel format", frame.vtable.getPixelFormat, uintptr(unsafe.Pointer(frame)), uintptr(unsafe.Pointer(&format)))
	t.Logf("source pixel format: %v", format)
	var converted *wicProbeSource
	rgba := wicProbeGUID(t, "{f5c7ad2d-6a8d-43dd-a7a8-a29935261ae9}")
	convert := windows.NewLazySystemDLL("windowscodecs.dll").NewProc("WICConvertBitmapSource")
	wicProbeCall(t, "convert SDR probe pixels", convert.Addr(), uintptr(unsafe.Pointer(&rgba)), uintptr(unsafe.Pointer(frame)), uintptr(unsafe.Pointer(&converted)))
	if converted == nil {
		t.Fatal("converted WIC source is nil")
	}
	defer wicProbeRelease(unsafe.Pointer(converted))
	result := Result{Width: 64, Height: 64, Stride: 256, Pixels: make([]byte, 64*64*4), Provider: fmt.Sprint(format)}
	wicProbeCall(t, "copy full pixels", converted.vtable.copyPixels, uintptr(unsafe.Pointer(converted)), 0, uintptr(result.Stride), uintptr(len(result.Pixels)), uintptr(unsafe.Pointer(&result.Pixels[0])))
	return result
}
