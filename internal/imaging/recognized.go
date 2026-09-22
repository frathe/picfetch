package imaging

// RecognizedExtensions returns the static file types accepted at desktop entry
// points and in portable rules, including formats requiring an optional native
// decoder. It does not report whether this machine can decode those formats.
// The returned slice is independent of all previous calls.
// Qodana's PR analysis misses the packaging and Explorer preset callers.
//
//goland:noinspection GoUnusedExportedFunction
func RecognizedExtensions() []string {
	return append(SupportedExtensions(), ".heic", ".heif")
}
