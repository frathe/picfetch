//go:build windows && (amd64 || arm64)

package heic

// Minimal COM ABI from Microsoft's public wincodec.h declarations. Source:
// microsoft/win32metadata 5c5efbc01d4c87f6830ec304d42777991d533154.
type wicUnknown struct{ vtable *wicUnknownMethods }
type wicUnknownMethods struct{ queryInterface, addRef, release uintptr }

type wicFactory struct{ vtable *wicFactoryMethods }
type wicFactoryMethods struct {
	wicUnknownMethods
	createDecoderFromFilename, createDecoderFromStream, createDecoderFromFileHandle uintptr
	createComponentInfo, createDecoder, createEncoder, createPalette                uintptr
	createFormatConverter, createBitmapScaler, createBitmapClipper                  uintptr
	createBitmapFlipRotator, createStream                                           uintptr
}

type wicStream struct{ vtable *wicStreamMethods }
type wicStreamMethods struct {
	wicUnknownMethods
	read, write, seek, setSize, copyTo, commit, revert, lockRegion, unlockRegion, stat, clone uintptr
	initializeFromIStream, initializeFromFilename, initializeFromMemory                       uintptr
}

type wicDecoder struct{ vtable *wicDecoderMethods }
type wicDecoderMethods struct {
	wicUnknownMethods
	queryCapability, initialize, getContainerFormat, getDecoderInfo uintptr
	copyPalette, getMetadataQueryReader, getPreview                 uintptr
	getColorContexts, getThumbnail, getFrameCount, getFrame         uintptr
}

type wicSource struct{ vtable *wicSourceMethods }
type wicSourceMethods struct {
	wicUnknownMethods
	getSize, getPixelFormat, getResolution, copyPalette, copyPixels uintptr
}

type wicFrame struct{ vtable *wicFrameMethods }
type wicFrameMethods struct {
	wicSourceMethods
	getMetadataQueryReader, getColorContexts, getThumbnail uintptr
}

type wicMetadataReader struct{ vtable *wicMetadataReaderMethods }
type wicMetadataReaderMethods struct {
	wicUnknownMethods
	getContainerFormat, getLocation, getMetadataByName, getEnumerator uintptr
}

type wicInfo struct{ vtable *wicInfoMethods }
type wicInfoMethods struct {
	wicUnknownMethods
	getComponentType, getCLSID, getSigningStatus, getAuthor    uintptr
	getVendorGUID, getVersion, getSpecVersion, getFriendlyName uintptr
}

type wicChainReader struct{ vtable *wicChainReaderMethods }
type wicChainReaderMethods struct {
	wicUnknownMethods
	getChainedFrameCount, getChainedFrame uintptr
}
